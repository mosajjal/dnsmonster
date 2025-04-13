/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

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

// PostgresConfig is the configuration and runtime struct for PostgreSQL output
type psqlConfig struct {
	// Configuration options
	OutputType    uint          `long:"psqloutputtype" ini-name:"psqloutputtype" env:"DNSMONSTER_PSQLOUTPUTTYPE" default:"0" description:"What should be written to Microsoft Psql. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	Endpoint      string        `long:"psqlendpoint" ini-name:"psqlendpoint" env:"DNSMONSTER_PSQLOUTPUTENDPOINT" default:"" description:"Psql endpoint used. must be in uri format. example: postgres://username:password@hostname:port/database?sslmode=disable"`
	Workers       uint          `long:"psqlworkers" ini-name:"psqlworkers" env:"DNSMONSTER_PSQLWORKERS" default:"1" description:"Number of PSQL workers"`
	BatchSize     uint          `long:"psqlbatchsize" ini-name:"psqlbatchsize" env:"DNSMONSTER_PSQLBATCHSIZE" default:"1" description:"Psql Batch Size"`
	BatchDelay    time.Duration `long:"psqlbatchdelay" ini-name:"psqlbatchdelay" env:"DNSMONSTER_PSQLBATCHDELAY" default:"0s" description:"Interval between sending results to Psql if Batch size is not filled. Any value larger than zero takes precedence over Batch Size"`
	BatchTimeout  time.Duration `long:"psqlbatchtimeout" ini-name:"psqlbatchtimeout" env:"DNSMONSTER_PSQLBATCHTIMEOUT" default:"5s" description:"Timeout for any INSERT operation before we consider them failed"`
	SaveFullQuery bool          `long:"psqlsavefullquery" ini-name:"psqlsavefullquery" env:"DNSMONSTER_PSQLSAVEFULLQUERY" description:"Save full packet query and response in JSON format."`

	// Runtime resources
	outputChannel    chan util.DNSResult
	outputMarshaller util.OutputMarshaller
	closeChannel     chan bool
}

// NewPostgresConfig creates a new PostgresConfig with default values
func NewPostgresConfig() *psqlConfig {
	return &psqlConfig{
		outputChannel: nil,
		closeChannel:  nil,
	}
}

// WithOutputType sets the OutputType and returns the config for chaining
func (c *psqlConfig) WithOutputType(t uint) *psqlConfig {
	c.OutputType = t
	return c
}

// WithEndpoint sets the Endpoint and returns the config for chaining
func (c *psqlConfig) WithEndpoint(endpoint string) *psqlConfig {
	c.Endpoint = endpoint
	return c
}

// WithWorkers sets the Workers and returns the config for chaining
func (c *psqlConfig) WithWorkers(workers uint) *psqlConfig {
	c.Workers = workers
	return c
}

// WithBatchSize sets the BatchSize and returns the config for chaining
func (c *psqlConfig) WithBatchSize(size uint) *psqlConfig {
	c.BatchSize = size
	return c
}

// WithBatchDelay sets the BatchDelay and returns the config for chaining
func (c *psqlConfig) WithBatchDelay(delay time.Duration) *psqlConfig {
	c.BatchDelay = delay
	return c
}

// WithBatchTimeout sets the BatchTimeout and returns the config for chaining
func (c *psqlConfig) WithBatchTimeout(timeout time.Duration) *psqlConfig {
	c.BatchTimeout = timeout
	return c
}

// WithSaveFullQuery sets the SaveFullQuery and returns the config for chaining
func (c *psqlConfig) WithSaveFullQuery(save bool) *psqlConfig {
	c.SaveFullQuery = save
	return c
}

// WithChannelSize initializes the output and close channels and returns the config for chaining
func (c *psqlConfig) WithChannelSize(channelSize int) *psqlConfig {
	c.outputChannel = make(chan util.DNSResult, channelSize)
	c.closeChannel = make(chan bool)
	return c
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

	if psqConf.OutputType > 0 && psqConf.OutputType < 5 {
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
	c, err := pgxpool.Connect(context.Background(), psqConf.Endpoint)
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
	psqlSentToOutput := metrics.GetOrRegisterCounter("psqlSentToOutput", metrics.DefaultRegistry)
	psqlSkipped := metrics.GetOrRegisterCounter("psqlSkipped", metrics.DefaultRegistry)

	conn := psqConf.connectPsql()
	batch := new(pgx.Batch)
	ticker := time.NewTicker(time.Second * 5)

	for {
		select {
		case data := <-psqConf.outputChannel:
			for _, dnsQuery := range data.DNS.Question {
				if util.CheckIfWeSkip(psqConf.OutputType, dnsQuery.Name) {
					psqlSkipped.Inc(1)
					continue
				}
				psqlSentToOutput.Inc(1)
				batch.Queue(
					`INSERT INTO DNS_LOG (PacketTime, Question) VALUES ($1, $2)`,
					data.Timestamp, dnsQuery.Name,
				)
			}
		case <-ticker.C:
			br := conn.SendBatch(ctx, batch)
			if _, err := br.Exec(); err != nil {
				log.Warnf("Error executing batch: %v", err)
			}
			batch = new(pgx.Batch)
		case <-ctx.Done():
			conn.Close()
			log.Debug("Exiting Postgres output")
			return
		}
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
	if psqConf.BatchDelay > 0 {
		psqConf.BatchSize = 1
		div = -1
		ticker = time.NewTicker(psqConf.BatchDelay)
	} else {
		ticker.Stop()
	}

	batch := new(pgx.Batch)
	insertQuery := `INSERT INTO DNS_LOG(
		PacketTime, IndexTime, Server, IPVersion, SrcIP ,DstIP, Protocol, QR, OpCode,
		Class, Type, Edns0Present, DoBit, FullQuery, ResponseCode, Question, Size)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17);`

	timeoutContext, cancel := context.WithTimeout(context.Background(), psqConf.BatchTimeout)
	defer cancel()

	for {
		select {
		case data := <-psqConf.outputChannel:
			for _, dnsQuery := range data.DNS.Question {

				c++
				if util.CheckIfWeSkip(psqConf.OutputType, dnsQuery.Name) {
					psqlSkipped.Inc(1)
					continue
				}

				fullQuery := ""
				if psqConf.SaveFullQuery {
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

				if int(c%psqConf.BatchSize) == div { // this block will never reach if batch delay is enabled
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
// vim: foldmethod=marker
