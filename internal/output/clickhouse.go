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
	"errors"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type clickhouseConfig struct {
	ClickhouseAddress           []string      `long:"clickhouseaddress"           ini-name:"clickhouseaddress"           env:"DNSMONSTER_CLICKHOUSEADDRESS"           default:"localhost:9000"                                          description:"Address of the clickhouse database to save the results. multiple values can be provided."`
	ClickhouseUsername          string        `long:"clickhouseusername"          ini-name:"clickhouseusername"          env:"DNSMONSTER_CLICKHOUSEUSERNAME"          default:""                                                        description:"Username to connect to the clickhouse database"`
	ClickhousePassword          string        `long:"clickhousepassword"          ini-name:"clickhousepassword"          env:"DNSMONSTER_CLICKHOUSEPASSWORD"          default:""                                                        description:"Password to connect to the clickhouse database"`
	ClickhouseDatabase          string        `long:"clickhousedatabase"          ini-name:"clickhousedatabase"          env:"DNSMONSTER_CLICKHOUSEDATABASE"          default:"default"                                                 description:"Database to connect to the clickhouse database"`
	ClickhouseTable             string        `long:"clickhousetable"             ini-name:"clickhousetable"             env:"DNSMONSTER_CLICKHOUSETABLE"             default:"DNS_LOG"                                                 description:"Table which data will be stored on clickhouse database"`
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

func (chConfig clickhouseConfig) connectClickhouseRetry(ctx context.Context) (driver.Conn, driver.Batch) {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if chConfig.ClickhouseOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		c, b, err := chConfig.connectClickhouse(ctx)
		if err == nil {
			return c, b
		}

		log.Errorf("Error connecting to Clickhouse: %s", err)
		
		// Try to create table if connection succeeded but batch preparation failed
		if c != nil {
			if createErr := chConfig.createTableIfNotExists(ctx, c); createErr != nil {
				log.Errorf("Failed to create table: %v", createErr)
			} else {
				log.Infof("Table creation attempted, retrying connection")
				// Close the old connection and try again immediately
				c.Close()
				continue
			}
		}

		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-tick.C:
			continue
		case <-ctx.Done():
			log.Info("Context cancelled, stopping ClickHouse connection retry")
			return nil, nil
		}
	}
}

func (chConfig clickhouseConfig) createTableIfNotExists(ctx context.Context, conn driver.Conn) error {
	// Default table schema for DNS monitoring
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			DnsDate Date,
			timestamp DateTime,
			Server String,
			IPVersion UInt8,
			SrcIP String,
			DstIP String,
			Protocol String,
			QR UInt8,
			OpCode UInt8,
			Class UInt8,
			Type UInt8,
			Edns0Present UInt8,
			DoBit UInt8,
			FullQuery String,
			ResponseCode UInt8,
			Question String,
			Size UInt16,
			Rcode String
		) ENGINE = MergeTree()
		PARTITION BY DnsDate
		ORDER BY (timestamp, Server)
		TTL DnsDate + INTERVAL 30 DAY
	`, chConfig.ClickhouseTable)

	err := conn.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	log.Infof("Successfully created or verified table %s", chConfig.ClickhouseTable)
	return nil
}

func (chConfig clickhouseConfig) connectClickhouse(ctx context.Context) (driver.Conn, driver.Batch, error) {
	compressOption := clickhouse.Compression{Method: clickhouse.CompressionNone, Level: 0}
	if chConfig.ClickhouseCompress > 0 {
		compressOption = clickhouse.Compression{Method: clickhouse.CompressionLZ4, Level: int(chConfig.ClickhouseCompress)}
	}

	tlsOption := &tls.Config{InsecureSkipVerify: util.GeneralFlags.SkipTLSVerification}
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
		Compression:     &compressOption,
	})
	// connection, err := clickhouse.Open(fmt.Sprintf("tcp://%v?debug=%v&skip_verify=%v&secure=%v&compress=%v&username=%s&password=%s&database=%s", chConfig.ClickhouseAddress, chConfig.ClickhouseDebug, util.GeneralFlags.SkipTLSVerification, chConfig.ClickhouseSecure, chConfig.ClickhouseCompress, chConfig.ClickhouseUsername, chConfig.ClickhousePassword, chConfig.ClickhouseDatabase))
	if err != nil {
		log.Error(err)
		return connection, nil, err
	}

	batch, err := connection.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %v", chConfig.ClickhouseTable))
	return connection, batch, err
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

func (chConfig clickhouseConfig) clickhouseOutputWorker(ctx context.Context) error {
	conn, batch := chConfig.connectClickhouseRetry(ctx)
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
				err := batch.Append(
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
				if int(c%chConfig.ClickhouseBatchSize) == div {
					err = batch.Send()
					if err != nil {
						log.Warnf("Error while executing batch: %v", err)
						clickhouseFailed.Inc(int64(c))
					}
					c = 0
					batch, _ = conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %v", chConfig.ClickhouseTable))
				}
			}
		case <-ticker.C:
			err := batch.Send()
			if err != nil {
				log.Warnf("Error while executing batch: %v", err)
				clickhouseFailed.Inc(int64(c))
			}
			c = 0
			batch, _ = conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %v", chConfig.ClickhouseTable))
		case <-ctx.Done():
			err := batch.Flush()
			if err != nil {
				log.Warnf("Errro while executing batch: %v", err)
				clickhouseFailed.Inc(int64(c))
			}
			conn.Close()
			return nil
		}
	}
}

// var _ = clickhouseConfig{}.initializeFlags()
// vim: foldmethod=marker
