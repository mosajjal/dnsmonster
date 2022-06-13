package output

import (
	"context"
	"crypto/tls"
	"errors"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/compress"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type ClickhouseConfig struct {
	ClickhouseAddress           []string      `long:"clickhouseAddress"           env:"DNSMONSTER_CLICKHOUSEADDRESS"           default:"localhost:9000"                                          description:"Address of the clickhouse database to save the results. multiple values can be provided"`
	ClickhouseUsername          string        `long:"clickhouseUsername"          env:"DNSMONSTER_CLICKHOUSEUSERNAME"          default:""                                                        description:"Username to connect to the clickhouse database"`
	ClickhousePassword          string        `long:"clickhousePassword"          env:"DNSMONSTER_CLICKHOUSEPASSWORD"          default:""                                                        description:"Password to connect to the clickhouse database"`
	ClickhouseDatabase          string        `long:"clickhouseDatabase"          env:"DNSMONSTER_CLICKHOUSEDATABASE"          default:"default"                                                 description:"Database to connect to the clickhouse database"`
	ClickhouseDelay             time.Duration `long:"clickhouseDelay"             env:"DNSMONSTER_CLICKHOUSEDELAY"             default:"0s"                                                      description:"Interval between sending results to ClickHouse. If non-0, Batch size is ignored and batch delay is used"`
	ClickhouseDebug             bool          `long:"clickhouseDebug"             env:"DNSMONSTER_CLICKHOUSEDEBUG"             description:"Debug Clickhouse connection"`
	ClickhouseCompress          bool          `long:"clickhouseCompress"          env:"DNSMONSTER_CLICKHOUSECOMPRESS"          description:"Compress Clickhouse connection"`
	ClickhouseSecure            bool          `long:"clickhouseSecure"            env:"DNSMONSTER_CLICKHOUSESECURE"            description:"Use TLS for Clickhouse connection"`
	ClickhouseSaveFullQuery     bool          `long:"clickhouseSaveFullQuery"     env:"DNSMONSTER_CLICKHOUSESAVEFULLQUERY"     description:"Save full packet query and response in JSON format."`
	ClickhouseOutputType        uint          `long:"clickhouseOutputType"        env:"DNSMONSTER_CLICKHOUSEOUTPUTTYPE"        default:"0"                                                       description:"What should be written to clickhouse. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"    choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ClickhouseBatchSize         uint          `long:"clickhouseBatchSize"         env:"DNSMONSTER_CLICKHOUSEBATCHSIZE"         default:"100000"                                                  description:"Minimum capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations"`
	ClickhouseWorkers           uint          `long:"clickhouseWorkers"           env:"DNSMONSTER_CLICKHOUSEWORKERS"           default:"1"                                                       description:"Number of Clickhouse output Workers"`
	ClickhouseWorkerChannelSize uint          `long:"clickhouseWorkerChannelSize" env:"DNSMONSTER_CLICKHOUSEWORKERCHANNELSIZE" default:"100000"                                                  description:"Channel Size for each Clickhouse Worker"`
	outputChannel               chan util.DNSResult
	outputMarshaller            util.OutputMarshaller
	closeChannel                chan bool
}

func (chConfig ClickhouseConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("clickhouse_output", "ClickHouse Output", &chConfig)

	chConfig.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)

	util.GlobalDispatchList = append(util.GlobalDispatchList, &chConfig)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (chConfig ClickhouseConfig) Initialize() error {
	var err error
	chConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

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
	// todo: implement this
	<-chConfig.closeChannel
}

func (chConfig ClickhouseConfig) OutputChannel() chan util.DNSResult {
	return chConfig.outputChannel
}

func (chConfig ClickhouseConfig) connectClickhouseRetry() (driver.Conn, driver.Batch) {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if chConfig.ClickhouseOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		c, b, err := chConfig.connectClickhouse()
		if err == nil {
			return c, b
		} else {
			log.Errorf("Error connecting to Clickhouse: %s", err)
			// todo: try and create table if it doesn't exist
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue
	}
}

func (chConfig ClickhouseConfig) connectClickhouse() (driver.Conn, driver.Batch, error) {
	compressOption := &clickhouse.Compression{Method: compress.NONE}
	tlsOption := &tls.Config{InsecureSkipVerify: util.GeneralFlags.SkipTLSVerification}
	if chConfig.ClickhouseCompress {
		compressOption = &clickhouse.Compression{Method: compress.LZ4}
	}
	if !chConfig.ClickhouseSecure {
		tlsOption = nil
	}

	connection, err := clickhouse.Open(&clickhouse.Options{
		Addr: chConfig.ClickhouseAddress,
		Auth: clickhouse.Auth{
			Database: chConfig.ClickhouseDatabase,
			Username: chConfig.ClickhouseUsername,
			Password: chConfig.ClickhousePassword,
		},
		DialTimeout:     time.Second * 2,
		MaxOpenConns:    32,
		MaxIdleConns:    16,
		ConnMaxLifetime: time.Hour,
		TLS:             tlsOption,
		Debug:           chConfig.ClickhouseDebug,
		Compression:     compressOption,
	})
	// connection, err := clickhouse.Open(fmt.Sprintf("tcp://%v?debug=%v&skip_verify=%v&secure=%v&compress=%v&username=%s&password=%s&database=%s", chConfig.ClickhouseAddress, chConfig.ClickhouseDebug, util.GeneralFlags.SkipTLSVerification, chConfig.ClickhouseSecure, chConfig.ClickhouseCompress, chConfig.ClickhouseUsername, chConfig.ClickhousePassword, chConfig.ClickhouseDatabase))
	if err != nil {
		log.Error(err)
		return connection, nil, err
	}

	batch, err := connection.PrepareBatch(ctx, "INSERT INTO DNS_LOG")
	return connection, batch, err
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
	conn, batch := chConfig.connectClickhouseRetry()
	ctx = context.Background()
	clickhouseSentToOutput := metrics.GetOrRegisterCounter("clickhouseSentToOutput", metrics.DefaultRegistry)
	clickhouseSkipped := metrics.GetOrRegisterCounter("clickhouseSkipped", metrics.DefaultRegistry)
	clickhouseFailed := metrics.GetOrRegisterCounter("clickhouseFailed", metrics.DefaultRegistry)

	c := uint(0)

	now := time.Now()

	ticker := time.NewTicker(time.Second * 5)
	div := 0

	if chConfig.ClickhouseDelay > 0 {
		chConfig.ClickhouseBatchSize = 1
		div = -1
		ticker = time.NewTicker(chConfig.ClickhouseDelay)
	} else {
		ticker.Stop()
	}

	for {
		select {
		case data := <-chConfig.outputChannel:
			for _, dnsQuery := range data.DNS.Question {
				c++
				if util.CheckIfWeSkip(chConfig.ClickhouseOutputType, dnsQuery.Name) {
					clickhouseSkipped.Inc(1)
					continue
				}
				clickhouseSentToOutput.Inc(1)

				fullQuery := ""
				if chConfig.ClickhouseSaveFullQuery {
					fullQuery = chConfig.outputMarshaller.Marshal(data)
				}

				QR := uint8(0)
				if data.DNS.Response {
					QR = 1
				}
				edns, doBit := uint8(0), uint8(0)
				if edns0 := data.DNS.IsEdns0(); edns0 != nil {
					edns = 1
					if edns0.Do() {
						doBit = 1
					}
				}
				err := batch.Append(
					data.Timestamp,
					now,
					util.GeneralFlags.ServerName,
					data.IPVersion,
					data.SrcIP.To16(),
					data.DstIP.To16(),
					data.Protocol,
					QR,
					uint8(data.DNS.Opcode),
					uint16(dnsQuery.Qclass),
					uint16(dnsQuery.Qtype),
					edns,
					doBit,
					fullQuery,
					uint8(data.DNS.Rcode),
					dnsQuery.Name,
					data.PacketLength,
				)
				if err != nil {
					log.Warnf("Error while executing batch: %v", err)
					clickhouseFailed.Inc(1)
				}
				if int(c%chConfig.ClickhouseBatchSize) == div {
					now = time.Now()
					err = batch.Send()
					if err != nil {
						log.Warnf("Error while executing batch: %v", err)
						clickhouseFailed.Inc(int64(c))
					}
					c = 0
					batch, _ = conn.PrepareBatch(ctx, "INSERT INTO DNS_LOG")
				}
			}
		case <-ticker.C:
			now = time.Now()
			err := batch.Send()
			if err != nil {
				log.Warnf("Error while executing batch: %v", err)
				clickhouseFailed.Inc(int64(c))
			}
			c = 0
			batch, _ = conn.PrepareBatch(ctx, "INSERT INTO DNS_LOG")
		}
	}
}

var _ = ClickhouseConfig{}.initializeFlags()
