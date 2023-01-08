package output

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/mosajjal/dnsmonster/internal/util"
	log "github.com/sirupsen/logrus"
)

type psqlConfig struct {
	PsqlOutputType    uint          `long:"psqloutputtype"          ini-name:"psqloutputtype"          env:"DNSMONSTER_PSQLOUTPUTTYPE"          default:"0"                                                       description:"What should be written to Microsoft Psql. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	PsqlEndpoint      string        `long:"psqlendpoint"            ini-name:"psqlendpoint"            env:"DNSMONSTER_PSQLOUTPUTENDPOINT"      default:""                                                        description:"Psql endpoint used. must be in uri format. example: postgres://username:password@hostname:port/database?sslmode=disable"`
	PsqlWorkers       uint          `long:"psqlworkers"             ini-name:"psqlworkers"             env:"DNSMONSTER_PSQLWORKERS"             default:"1"                                                       description:"Number of PSQL workers"`
	PsqlBatchSize     uint          `long:"psqlbatchsize"           ini-name:"psqlbatchsize"           env:"DNSMONSTER_PSQLBATCHSIZE"           default:"1"                                                       description:"Psql Batch Size"`
	PsqlBatchDelay    time.Duration `long:"psqlbatchdelay"          ini-name:"psqlbatchdelay"          env:"DNSMONSTER_PSQLBATCHDELAY"          default:"0s"                                                      description:"Interval between sending results to Psql if Batch size is not filled. Any value larger than zero takes precedence over Batch Size"`
	PsqlBatchTimeout  time.Duration `long:"psqlbatchtimeout"        ini-name:"psqlbatchtimeout"        env:"DNSMONSTER_PSQLBATCHTIMEOUT"        default:"5s"                                                      description:"Timeout for any INSERT operation before we consider them failed"`
	PsqlSaveFullQuery bool          `long:"psqlsavefullquery"       ini-name:"psqlsavefullquery"       env:"DNSMONSTER_PSQLSAVEFULLQUERY"       description:"Save full packet query and response in JSON format."`
	outputChannel     chan util.DNSResult
	outputMarshaller  util.OutputMarshaller
	closeChannel      chan bool
}

func init() {
	c := psqlConfig{}
	if _, err := util.GlobalParser.AddGroup("psql_output", "PSQL Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (psqConf psqlConfig) Initialize(ctx context.Context) error {
	var err error
	psqConf.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if psqConf.PsqlOutputType > 0 && psqConf.PsqlOutputType < 5 {
		log.Info("Creating Psql Output Channel")
		go psqConf.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (psqConf psqlConfig) Close() {
	// todo: implement this
	<-psqConf.closeChannel
}

func (psqConf psqlConfig) OutputChannel() chan util.DNSResult {
	return psqConf.outputChannel
}

func (psqConf psqlConfig) connectPsql() *pgxpool.Pool {
	c, err := pgxpool.Connect(context.Background(), psqConf.PsqlEndpoint)
	if err != nil {
		// This will not be a connection error, but a DSN parse error or
		// another initialization error.
		log.Fatal(err)
	}
	// defer c.Close() // todo: move to a close channel
	err = c.Ping(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS DNS_LOG (PacketTime timestamp, IndexTime timestamp,
				Server text, IPVersion integer, SrcIP inet, DstIP inet, Protocol char(3),
				QR smallint, OpCode smallint, Class smallint, Type integer, Edns0Present smallint,
				DoBit smallint, FullQuery text, ResponseCode smallint, Question text, Size smallint);`,
	)
	if err != nil {
		log.Error(err.Error())
	}

	return c
}

func (psqConf psqlConfig) Output(ctx context.Context) {
	for i := 0; i < int(psqConf.PsqlWorkers); i++ {
		go psqConf.OutputWorker()
	}
}

func (psqConf psqlConfig) OutputWorker() {
	psqlSkipped := metrics.GetOrRegisterCounter("psqlSkipped", metrics.DefaultRegistry)
	psqlSentToOutput := metrics.GetOrRegisterCounter("psqlSentToOutput", metrics.DefaultRegistry)
	psqlFailed := metrics.GetOrRegisterCounter("psqlFailed", metrics.DefaultRegistry)

	c := uint(0)

	conn := psqConf.connectPsql()

	ticker := time.NewTicker(time.Second * 5)
	div := 0
	if psqConf.PsqlBatchDelay > 0 {
		psqConf.PsqlBatchSize = 1
		div = -1
		ticker = time.NewTicker(psqConf.PsqlBatchDelay)
	} else {
		ticker.Stop()
	}

	batch := new(pgx.Batch)
	insertQuery := `INSERT INTO DNS_LOG(
		PacketTime, IndexTime, Server, IPVersion, SrcIP ,DstIP, Protocol, QR, OpCode,
		Class, Type, Edns0Present, DoBit, FullQuery, ResponseCode, Question, Size)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17);`

	timeoutContext, cancel := context.WithTimeout(context.Background(), psqConf.PsqlBatchTimeout)
	defer cancel()

	for {
		select {
		case data := <-psqConf.outputChannel:
			for _, dnsQuery := range data.DNS.Question {

				c++
				if util.CheckIfWeSkip(psqConf.PsqlOutputType, dnsQuery.Name) {
					psqlSkipped.Inc(1)
					continue
				}

				fullQuery := ""
				if psqConf.PsqlSaveFullQuery {
					fullQuery = string(psqConf.outputMarshaller.Marshal(data))
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

				batch.Queue(insertQuery,
					data.Timestamp,
					time.Now(),
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

				if int(c%psqConf.PsqlBatchSize) == div { // this block will never reach if batch delay is enabled
					log.Warnf("here %d", c) //todo:remove
					br := conn.SendBatch(timeoutContext, batch)
					_, err := br.Exec()
					if err != nil {
						log.Errorf("Error while executing batch: %v", err)
						psqlFailed.Inc(int64(c))
					} else {
						psqlSentToOutput.Inc(int64(c))
					}
					c = 0
					batch = new(pgx.Batch)
				}

			}
		case <-ticker.C:
			br := conn.SendBatch(timeoutContext, batch)
			_, err := br.Exec()
			if err != nil {
				log.Errorf("Error while executing batch: %v", err)
				psqlFailed.Inc(int64(c))
			} else {
				psqlSentToOutput.Inc(int64(c))
			}
			c = 0
			batch = new(pgx.Batch)
		}
	}
}

// This will allow an instance to be spawned at import time
// var _ = psqlConfig{}.initializeFlags()
