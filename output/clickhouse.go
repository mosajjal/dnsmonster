package output

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/rogpeppe/fastuuid"
	log "github.com/sirupsen/logrus"

	"github.com/ClickHouse/clickhouse-go"
	data "github.com/ClickHouse/clickhouse-go/lib/data"
)

type ClickhouseConfig struct {
	ClickhouseAddress           string        `long:"clickhouseAddress"           env:"DNSMONSTER_CLICKHOUSEADDRESS"           default:"localhost:9000"                                          description:"Address of the clickhouse database to save the results"`
	ClickhouseUsername          string        `long:"clickhouseUsername"          env:"DNSMONSTER_CLICKHOUSEUSERNAME"          default:""                                                        description:"Username to connect to the clickhouse database"`
	ClickhousePassword          string        `long:"clickhousePassword"          env:"DNSMONSTER_CLICKHOUSEPASSWORD"          default:""                                                        description:"Password to connect to the clickhouse database"`
	ClickhouseDelay             time.Duration `long:"clickhouseDelay"             env:"DNSMONSTER_CLICKHOUSEDELAY"             default:"1s"                                                      description:"Interval between sending results to ClickHouse"`
	ClickhouseDebug             bool          `long:"clickhouseDebug"             env:"DNSMONSTER_CLICKHOUSEDEBUG"             description:"Debug Clickhouse connection"`
	ClickhouseCompress          bool          `long:"clickhouseCompress"          env:"DNSMONSTER_CLICKHOUSECOMPRESS"          description:"Compress Clickhouse connection"`
	ClickhouseSecure            bool          `long:"clickhouseSecure"            env:"DNSMONSTER_CLICKHOUSESECURE"            description:"Use TLS for Clickhouse connection"`
	ClickhouseSaveFullQuery     bool          `long:"clickhouseSaveFullQuery"     env:"DNSMONSTER_CLICKHOUSESAVEFULLQUERY"     description:"Save full packet query and response in JSON format."`
	ClickhouseOutputType        uint          `long:"clickhouseOutputType"        env:"DNSMONSTER_CLICKHOUSEOUTPUTTYPE"        default:"0"                                                       description:"What should be written to clickhouse. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"    choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ClickhouseBatchSize         uint          `long:"clickhouseBatchSize"         env:"DNSMONSTER_CLICKHOUSEBATCHSIZE"         default:"100000"                                                  description:"Minimun capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations"`
	ClickhouseWorkers           uint          `long:"clickhouseWorkers"           env:"DNSMONSTER_CLICKHOUSEWORKERS"           default:"1"                                                       description:"Number of Clickhouse output Workers"`
	ClickhouseWorkerChannelSize uint          `long:"clickhouseWorkerChannelSize" env:"DNSMONSTER_CLICKHOUSEWORKERCHANNELSIZE" default:"100000"                                                  description:"Channel Size for each Clickhouse Worker"`
	outputChannel               chan types.DNSResult
	closeChannel                chan bool
}

func (chConfig ClickhouseConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("clickhouse_output", "ClickHouse Output", &chConfig)

	chConfig.outputChannel = make(chan types.DNSResult, util.GeneralFlags.ResultChannelSize)

	types.GlobalDispatchList = append(types.GlobalDispatchList, &chConfig)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (chConfig ClickhouseConfig) Initialize() error {
	if chConfig.ClickhouseOutputType > 0 && chConfig.ClickhouseOutputType < 5 {
		log.Info("Creating Clickhouse Output Channel")
		go chConfig.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (chConfig ClickhouseConfig) Close() {
	//todo: implement this
	<-chConfig.closeChannel
}

func (chConfig ClickhouseConfig) OutputChannel() chan types.DNSResult {
	return chConfig.outputChannel
}

var uuidGen = fastuuid.MustNewGenerator()

func (chConfig ClickhouseConfig) connectClickhouseRetry() clickhouse.Clickhouse {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if chConfig.ClickhouseOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		c, err := chConfig.connectClickhouse()
		if err == nil {
			return c
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue
	}
}

func (chConfig ClickhouseConfig) connectClickhouse() (clickhouse.Clickhouse, error) {
	connection, err := clickhouse.OpenDirect(fmt.Sprintf("tcp://%v?debug=%v&skip_verify=%v&secure=%v&compress=%v&username=%s&password=%s", chConfig.ClickhouseAddress, chConfig.ClickhouseDebug, util.GeneralFlags.SkipTLSVerification, chConfig.ClickhouseSecure, chConfig.ClickhouseCompress, chConfig.ClickhouseUsername, chConfig.ClickhousePassword))
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
func (chConfig ClickhouseConfig) Output() {
	for i := 0; i < int(chConfig.ClickhouseWorkers); i++ {
		go chConfig.clickhouseOutputWorker()
	}
}

func (chConfig ClickhouseConfig) clickhouseOutputWorker() {
	connect := chConfig.connectClickhouseRetry()
	batch := make([]types.DNSResult, 0, chConfig.ClickhouseBatchSize)

	ticker := time.NewTicker(chConfig.ClickhouseDelay)
	for {
		select {
		case data := <-chConfig.outputChannel:
			if util.GeneralFlags.PacketLimit == 0 || len(batch) < util.GeneralFlags.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			if err := chConfig.clickhouseSendData(connect, batch); err != nil {
				log.Warnf("Error sending data to clickhouse: %v", err)
				connect = chConfig.connectClickhouseRetry()
			} else {
				batch = make([]types.DNSResult, 0, chConfig.ClickhouseBatchSize)
			}

		}
	}
}

func (chConfig ClickhouseConfig) clickhouseSendData(connect clickhouse.Clickhouse, batch []types.DNSResult) error {
	clickhouseSentToOutput := metrics.GetOrRegisterCounter("clickhouseSentToOutput", metrics.DefaultRegistry)
	clickhouseSkipped := metrics.GetOrRegisterCounter("clickhouseSkipped", metrics.DefaultRegistry)

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
						clickhouseSkipped.Inc(1)
						continue
					}
					clickhouseSentToOutput.Inc(1)

					var fullQuery []byte
					if chConfig.ClickhouseSaveFullQuery {
						fullQuery = []byte(batch[k].String())
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
					b.WriteBytes(2, []byte(util.GeneralFlags.ServerName))
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

var _ = ClickhouseConfig{}.initializeFlags()
