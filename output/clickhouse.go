package output

import (
	"encoding/binary"
	"fmt"
	"sync"

	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	"github.com/rogpeppe/fastuuid"
	log "github.com/sirupsen/logrus"

	"github.com/ClickHouse/clickhouse-go"
	data "github.com/ClickHouse/clickhouse-go/lib/data"
)

var chstats = types.OutputStats{Name: "Clickhouse", SentToOutput: 0, Skipped: 0}
var uuidGen = fastuuid.MustNewGenerator()

func connectClickhouseRetry(chConfig types.ClickHouseConfig) clickhouse.Clickhouse {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if chConfig.ClickhouseOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		c, err := connectClickhouse(chConfig)
		if err == nil {
			return c
		}

		// Error getting connection, wait the timer or check if we are exiting
		select {

		case <-tick.C:
			continue
		}
	}
}

func connectClickhouse(chConfig types.ClickHouseConfig) (clickhouse.Clickhouse, error) {
	connection, err := clickhouse.OpenDirect(fmt.Sprintf("tcp://%v?debug=%v", chConfig.ClickhouseAddress, chConfig.ClickhouseDebug))
	//todo: there are many options that needs to be added to this. Compression, TLS, etc.
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return connection, err
}

func min(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Main handler for Clickhouse output. the data from the dispatched output channel will reach this function
// Essentially, the function is responsible to hold an available connection ready by calling another goroutine,
// maintain the incoming data batch and try to INSERT them as quick as possible into the Clickhouse table
// the table structure of Clickhouse is hardcoded into the code so before outputing to Clickhouse, the user
// needs to make sure that there is proper Database connection and table are present. Refer to the project's
// clickhouse folder for the file tables.sql
func ClickhouseOutput(chConfig types.ClickHouseConfig) {
	printStatsTicker := time.NewTicker(chConfig.General.PrintStatsDelay)
	var workerChannelList []chan types.DNSResult
	for i := 0; i < int(chConfig.ClickhouseWorkers); i++ {
		workerChannelList = append(workerChannelList, make(chan types.DNSResult, chConfig.ClickhouseWorkerChannelSize))
		go clickhouseOutputWorker(chConfig, workerChannelList[i])
	}
	var cnt uint
	for {
		cnt++
		select {
		case data := <-chConfig.ResultChannel:
			//fantout is a round-robin logic
			workerChannelList[cnt%chConfig.ClickhouseWorkers] <- data
		case <-printStatsTicker.C:
			log.Infof("output: %+v", chstats)
		}
	}
}

func clickhouseOutputWorker(chConfig types.ClickHouseConfig, workerchannel chan types.DNSResult) {
	connect := connectClickhouseRetry(chConfig)
	batch := make([]types.DNSResult, 0, chConfig.ClickhouseBatchSize)

	ticker := time.NewTicker(chConfig.ClickhouseDelay)
	for {
		select {
		case data := <-workerchannel:
			if chConfig.General.PacketLimit == 0 || len(batch) < chConfig.General.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			if err := clickhouseSendData(connect, batch, chConfig); err != nil {
				log.Warnf("Error sending data to clickhouse: %v", err)
				connect = connectClickhouseRetry(chConfig)
			} else {
				batch = make([]types.DNSResult, 0, chConfig.ClickhouseBatchSize)
			}

		}
	}
}

func clickhouseSendData(connect clickhouse.Clickhouse, batch []types.DNSResult, chConfig types.ClickHouseConfig) error {
	if len(batch) == 0 {
		return nil
	}
	// Return if the connection is null, we are exiting
	if connect == nil {
		return nil
	}
	_, err := connect.Begin()
	if err != nil {
		log.Warnf("Error starting transaction: %v", err)
		return err
	}

	_, err = connect.Prepare("INSERT INTO DNS_LOG (DnsDate, timestamp, Server, IPVersion, SrcIP, DstIP, Protocol, QR, OpCode, Class, Type, ResponseCode, Question, Size, Edns0Present, DoBit,FullQuery, ID) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}

	block, err := connect.Block()
	if err != nil {
		log.Warnf("Error getting block: %v", err)
		return err
	}

	blocks := []*data.Block{block}

	var clickhouseWaitGroup sync.WaitGroup

	for i := 0; i < len(blocks); i++ {
		clickhouseWaitGroup.Add(1)
	}

	count := len(blocks)
	for i := range blocks {

		b := blocks[i]
		start := i * (len(batch)) / count
		end := min((i+1)*(len(batch))/count, len(batch))
		go func() {
			defer clickhouseWaitGroup.Done()
			b.Reserve()
			for k := start; k < end; k++ {
				for _, dnsQuery := range batch[k].DNS.Question {

					if util.CheckIfWeSkip(chConfig.ClickhouseOutputType, dnsQuery.Name) {
						chstats.Skipped++
						continue
					}
					chstats.SentToOutput++

					var fullQuery []byte
					if chConfig.ClickhouseSaveFullQuery {
						fullQuery = []byte(batch[k].String()) //todo: check this
					}
					var SrcIP, DstIP uint64

					if batch[k].IPVersion == 4 {
						SrcIP = uint64(binary.BigEndian.Uint32(batch[k].SrcIP))
						DstIP = uint64(binary.BigEndian.Uint32(batch[k].DstIP))
					} else {
						SrcIP = binary.BigEndian.Uint64(batch[k].SrcIP[8:]) //limitation of clickhouse-go doesn't let us go more than 64 bits for ipv6 at the moment
						DstIP = binary.BigEndian.Uint64(batch[k].DstIP[8:])
					}
					QR := uint8(0)
					if batch[k].DNS.Response {
						QR = 1
					}
					edns, doBit := uint8(0), uint8(0)
					if edns0 := batch[k].DNS.IsEdns0(); edns0 != nil {
						edns = 1
						if edns0.Do() {
							doBit = 1
						}
					}

					b.NumRows++
					//writing the vars into a SQL statement
					b.WriteDate(0, batch[k].Timestamp)
					b.WriteDateTime(1, batch[k].Timestamp)
					b.WriteBytes(2, []byte(chConfig.General.ServerName))
					b.WriteUInt8(3, batch[k].IPVersion)
					b.WriteUInt64(4, SrcIP)
					b.WriteUInt64(5, DstIP)
					b.WriteFixedString(6, []byte(batch[k].Protocol))
					b.WriteUInt8(7, QR)
					b.WriteUInt8(8, uint8(batch[k].DNS.Opcode))
					b.WriteUInt16(9, uint16(dnsQuery.Qclass))
					b.WriteUInt16(10, uint16(dnsQuery.Qtype))
					b.WriteUInt8(11, uint8(batch[k].DNS.Rcode))
					b.WriteString(12, string(dnsQuery.Name))
					b.WriteUInt16(13, batch[k].PacketLength)
					b.WriteUInt8(14, edns)
					b.WriteUInt8(15, doBit)

					b.WriteFixedString(16, fullQuery)
					myUUID := uuidGen.Next()
					b.WriteFixedString(17, myUUID[:16])
				}
			}
			if err := connect.WriteBlock(b); err != nil {
				log.Warnf("Error writing block: %s", err)
				return
			}
		}()
	}
	clickhouseWaitGroup.Wait() // there is a separate waitgroup only for Clickhouse, need to investigate if this is needed or not.
	if err := connect.Commit(); err != nil {
		log.Warnf("Error writing block: %s", err)
		return err
	}

	return nil
}
