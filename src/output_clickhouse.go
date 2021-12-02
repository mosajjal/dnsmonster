package main

import (
	"encoding/binary"
	"fmt"
	"sync"

	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/rogpeppe/fastuuid"
	log "github.com/sirupsen/logrus"

	"github.com/ClickHouse/clickhouse-go"
	data "github.com/ClickHouse/clickhouse-go/lib/data"
)

var chstats = outputStats{"Clickhouse", 0, 0}
var uuidGen = fastuuid.MustNewGenerator()

func connectClickhouseRetry(chConfig clickHouseConfig) clickhouse.Clickhouse {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if chConfig.clickhouseOutputType == 0 {
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
		case <-types.GlobalExitChannel:
			// When exiting, return immediately
			return nil
		case <-tick.C:
			continue
		}
	}
}

func connectClickhouse(chConfig clickHouseConfig) (clickhouse.Clickhouse, error) {
	connection, err := clickhouse.OpenDirect(fmt.Sprintf("tcp://%v?debug=%v", chConfig.clickhouseAddress, chConfig.clickhouseDebug))
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

func clickhouseOutput(chConfig clickHouseConfig) {
	printStatsTicker := time.Tick(chConfig.general.printStatsDelay)
	var workerChannelList []chan types.DNSResult
	for i := 0; i < int(chConfig.clickhouseWorkers); i++ {
		workerChannelList = append(workerChannelList, make(chan types.DNSResult, chConfig.clickhouseWorkerChannelSize))
		go clickhouseOutputWorker(chConfig, workerChannelList[i]) //todo: fan out
		types.GlobalWaitingGroup.Add(1)
	}
	var cnt uint
	for {
		cnt++
		select {
		case data := <-chConfig.resultChannel:
			//fantout is a round-robin logic
			workerChannelList[cnt%chConfig.clickhouseWorkers] <- data
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", chstats)
		}
	}
}

func clickhouseOutputWorker(chConfig clickHouseConfig, workerchannel chan types.DNSResult) {

	connect := connectClickhouseRetry(chConfig)
	batch := make([]types.DNSResult, 0, chConfig.clickhouseBatchSize)

	ticker := time.Tick(chConfig.clickhouseDelay)
	for {
		select {
		case data := <-workerchannel:
			if chConfig.general.packetLimit == 0 || len(batch) < chConfig.general.packetLimit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := clickhouseSendData(connect, batch, chConfig); err != nil {
				log.Warnf("Error sending data to clickhouse: %#v, %v", batch, err) //todo: remove batch from this print
				connect = connectClickhouseRetry(chConfig)
			} else {
				batch = make([]types.DNSResult, 0, chConfig.clickhouseBatchSize)
			}
		case <-types.GlobalExitChannel:
			return

		}
	}
}

func clickhouseSendData(connect clickhouse.Clickhouse, batch []types.DNSResult, chConfig clickHouseConfig) error {
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

					if checkIfWeSkip(chConfig.clickhouseOutputType, dnsQuery.Name) {
						chstats.Skipped++
						continue
					}
					chstats.SentToOutput++

					var fullQuery []byte
					if chConfig.clickhouseSaveFullQuery {
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
					b.WriteBytes(2, []byte(chConfig.general.serverName))
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
	clickhouseWaitGroup.Wait() //todo: do I need to have s separate waitgroup for CH?
	if err := connect.Commit(); err != nil {
		log.Warnf("Error writing block: %s", err)
		return err
	}

	return nil
}
