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
	"crypto/tls"
	"database/sql"
	"errors"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type clickhouseConfig struct {
	ClickhouseAddress           []string      `long:"clickhouseaddress"           ini-name:"clickhouseaddress"           env:"DNSMONSTER_CLICKHOUSEADDRESS"           default:"localhost:9000"                                          description:"Address of the clickhouse database to save the results. multiple values can be provided."`
	ClickhouseProtocol          string        `long:"clickhouseprotocol"          ini-name:"clickhouseprotocol"          env:"DNSMONSTER_CLICKHOUSEPROTOCOL"          default:"native"                                                  description:"clickhouse connection protocol. options: native, http" choice:"native" choice:"http"`
	ClickhouseUsername          string        `long:"clickhouseusername"          ini-name:"clickhouseusername"          env:"DNSMONSTER_CLICKHOUSEUSERNAME"          default:""                                                        description:"Username to connect to the clickhouse database"`
	ClickhousePassword          string        `long:"clickhousepassword"          ini-name:"clickhousepassword"          env:"DNSMONSTER_CLICKHOUSEPASSWORD"          default:""                                                        description:"Password to connect to the clickhouse database"`
	ClickhouseDatabase          string        `long:"clickhousedatabase"          ini-name:"clickhousedatabase"          env:"DNSMONSTER_CLICKHOUSEDATABASE"          default:"default"                                                 description:"Database to connect to the clickhouse database"`
	ClickhouseDelay             time.Duration `long:"clickhousedelay"             ini-name:"clickhousedelay"             env:"DNSMONSTER_CLICKHOUSEDELAY"             default:"0s"                                                      description:"Interval between sending results to ClickHouse. If non-0, Batch size is ignored and batch delay is used"`
	ClickhouseCompress          uint8         `long:"clickhousecompress"          ini-name:"clickhousecompress"          env:"DNSMONSTER_CLICKHOUSECOMPRESS"          description:"Clickhouse connection LZ4 compression level, 0 means no compression"`
	ClickhouseDebug             bool          `long:"clickhousedebug"             ini-name:"clickhousedebug"             env:"DNSMONSTER_CLICKHOUSEDEBUG"             description:"Debug Clickhouse connection"`
	ClickhouseSecure            bool          `long:"clickhousesecure"            ini-name:"clickhousesecure"            env:"DNSMONSTER_CLICKHOUSESECURE"            description:"Use TLS for Clickhouse connection"`
	ClickhouseSaveFullQuery     bool          `long:"clickhousesavefullquery"     ini-name:"clickhousesavefullquery"     env:"DNSMONSTER_CLICKHOUSESAVEFULLQUERY"     description:"Save full packet query and response in JSON format."`
	ClickhouseOutputType        uint          `long:"clickhouseoutputtype"        ini-name:"clickhouseoutputtype"        env:"DNSMONSTER_CLICKHOUSEOUTPUTTYPE"        default:"0"                                                       description:"What should be written to clickhouse. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"    choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ClickhouseBatchSize         uint          `long:"clickhousebatchsize"         ini-name:"clickhousebatchsize"         env:"DNSMONSTER_CLICKHOUSEBATCHSIZE"         default:"100000"                                                  description:"Minimum capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations"`
	ClickhouseWorkers           uint          `long:"clickhouseworkers"           ini-name:"clickhouseworkers"           env:"DNSMONSTER_CLICKHOUSEWORKERS"           default:"1"                                                       description:"Number of Clickhouse output Workers"`
	ClickhouseWorkerChannelSize uint          `long:"clickhouseworkerchannelsize" ini-name:"clickhouseworkerchannelsize" env:"DNSMONSTER_CLICKHOUSEWORKERCHANNELSIZE" default:"100000"                                                  description:"Channel Size for each Clickhouse Worker"`
	outputChannel               chan util.DNSResult
	outputMarshaller            util.OutputMarshaller
	closeChannel                chan bool
}

// init function runs at import time
func init() {
	c := clickhouseConfig{}
	if _, err := util.GlobalParser.AddGroup("clickhouse_output", "ClickHouse Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// Initialize function should not block. otherwise the dispatcher will get stuck
func (chConfig clickhouseConfig) Initialize(ctx context.Context) error {
	var err error
	chConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if chConfig.ClickhouseOutputType > 0 && chConfig.ClickhouseOutputType < 5 {
		log.Info("Creating Clickhouse Output Channel")
		go chConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}

	if chConfig.ClickhouseCompress > 9 {
		log.Warnf("invalid compression level provided. Things might break")
	}

	return nil
}

func (chConfig clickhouseConfig) Close() {
	// todo: implement this
	<-chConfig.closeChannel
}

func (chConfig clickhouseConfig) OutputChannel() chan util.DNSResult {
	return chConfig.outputChannel
}

func (chConfig clickhouseConfig) connectClickhouseRetry(ctx context.Context) (*sql.Conn, error) {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if chConfig.ClickhouseOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		c, err := chConfig.connectClickhouse(ctx)
		if err == nil {
			return c, nil
		}

		log.Errorf("Error connecting to Clickhouse: %s", err)
		// todo: try and create table if it doesn't exist

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue
	}
}

func (chConfig clickhouseConfig) connectClickhouse(ctx context.Context) (*sql.Conn, error) {
	compressOption := clickhouse.Compression{Method: clickhouse.CompressionNone, Level: 0}
	if chConfig.ClickhouseCompress > 0 {
		compressOption = clickhouse.Compression{Method: clickhouse.CompressionLZ4, Level: int(chConfig.ClickhouseCompress)}
	}

	tlsOption := &tls.Config{InsecureSkipVerify: util.GeneralFlags.SkipTLSVerification}
	if !chConfig.ClickhouseSecure {
		tlsOption = nil
	}

	// determine protocol
	protocol := clickhouse.HTTP
	if chConfig.ClickhouseProtocol == "native" {
		log.Debug("Using native protocol for Clickhouse")
		protocol = clickhouse.Native
	} else {
		log.Debug("Using HTTP protocol for Clickhouse")
	}

	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr:     chConfig.ClickhouseAddress,
		Protocol: protocol,
		Auth: clickhouse.Auth{
			Database: chConfig.ClickhouseDatabase,
			Username: chConfig.ClickhouseUsername,
			Password: chConfig.ClickhousePassword,
		},
		DialTimeout: time.Second * 5,
		TLS:         tlsOption,
		Debug:       chConfig.ClickhouseDebug,
		Compression: &compressOption,
	})
	db.SetMaxIdleConns(16)
	db.SetMaxOpenConns(32)
	db.SetConnMaxLifetime(time.Hour)

	connection, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	if connection.PingContext(ctx) != nil {
		return nil, err
	}

	return connection, err
}

/*
Output function brings up the workers. the data from the dispatched output channel will reach this function
Essentially, the function is responsible to hold an available connection ready by calling another goroutine,
maintain the incoming data batch and try to INSERT them as quick as possible into the Clickhouse table
the table structure of Clickhouse is hardcoded into the code so before outputting to Clickhouse, the user
needs to make sure that there is proper Database connection and table are present. Refer to the project's
clickhouse folder for the file tables.sql
*/
func (chConfig clickhouseConfig) Output(ctx context.Context) {
	g, gCtx := errgroup.WithContext(ctx)
	for i := 0; i < int(chConfig.ClickhouseWorkers); i++ {
		g.Go(func() error { return chConfig.clickhouseOutputWorker(gCtx) })
	}
}

func refreshBatchRetry(ctx context.Context, conn *sql.Conn) (tx *sql.Tx, batch *sql.Stmt) {
	tx, batch, err := refreshBatch(ctx, conn)
	if err != nil {
		time.Sleep(5 * time.Second)
		return refreshBatchRetry(ctx, conn)
	}
	return tx, batch
}

func refreshBatch(ctx context.Context, conn *sql.Conn) (tx *sql.Tx, batch *sql.Stmt, err error) {
	tx, err = conn.BeginTx(ctx, nil)
	if err != nil {
		return tx, nil, err
	}
	batch, err = tx.PrepareContext(ctx, "INSERT INTO DNS_LOG")
	return tx, batch, err
}

func (chConfig clickhouseConfig) clickhouseOutputWorker(ctx context.Context) error {
	conn, err := chConfig.connectClickhouseRetry(ctx)
	if err != nil {
		log.Errorf("Error connecting to Clickhouse: %s", err)
	}
	tx, batch, err := refreshBatch(ctx, conn)
	if err != nil {
		log.Errorf("Error preparing batch: %s", err)
	}

	clickhouseSentToOutput := metrics.GetOrRegisterCounter("clickhouseSentToOutput", metrics.DefaultRegistry)
	clickhouseSkipped := metrics.GetOrRegisterCounter("clickhouseSkipped", metrics.DefaultRegistry)
	clickhouseFailed := metrics.GetOrRegisterCounter("clickhouseFailed", metrics.DefaultRegistry)

	c := uint(0)

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
					fullQuery = string(chConfig.outputMarshaller.Marshal(data))
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
				if batch != nil {
					_, err := batch.Exec(
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
					if err != nil {
						log.Warnf("Error while executing batch: %v", err)
						clickhouseFailed.Inc(1)
					}
				} else {
					log.Warnf("Batch is nil")
					clickhouseFailed.Inc(1)
				}
				if int(c%chConfig.ClickhouseBatchSize) == div {
					err = tx.Commit()
					if err != nil {
						log.Warnf("Error while executing batch: %v", err)
						clickhouseFailed.Inc(int64(c))
					}
					c = 0
					tx, batch = refreshBatchRetry(ctx, conn)
				}
			}
		case <-ticker.C:
			err := tx.Commit()
			if err != nil {
				log.Warnf("Error while executing batch: %v", err)
				clickhouseFailed.Inc(int64(c))
			}
			c = 0
			tx, batch = refreshBatchRetry(ctx, conn)
		case <-ctx.Done():
			err := tx.Commit()
			if err != nil {
				log.Warnf("Errro while executing batch: %v", err)
				clickhouseFailed.Inc(int64(c))
			}
			conn.Close()
			log.Debug("exitting out of clickhouse output") //todo:remove
			return nil
		}
	}
}

// var _ = clickhouseConfig{}.initializeFlags()
// vim: foldmethod=marker
