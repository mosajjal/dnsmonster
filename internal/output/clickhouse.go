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
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// ClickhouseConfig is the configuration and runtime struct for ClickHouse output.
type ClickhouseConfig struct {
	BaseConfig
	Address          []string
	Username         string
	Password         string
	Database         string
	Compress         int
	Debug            bool
	Secure           bool
	SaveFullQuery    bool
	Workers          int
	outputChannel    chan util.DNSResult
	closeChannel     chan bool
	Delay            time.Duration
	OutputType       uint
	outputMarshaller util.OutputMarshaller
}

// NewClickhouseConfig creates a new ClickhouseConfig with default values.
func NewClickhouseConfig() *ClickhouseConfig {
	return &ClickhouseConfig{
		outputChannel: nil,
		closeChannel:  nil,
	}
}

// WithAddress sets the Address and returns the config for chaining.
func (c *ClickhouseConfig) WithAddress(addr []string) *ClickhouseConfig {
	c.Address = addr
	return c
}
func (c *ClickhouseConfig) WithUsername(u string) *ClickhouseConfig {
	c.Username = u
	return c
}
func (c *ClickhouseConfig) WithPassword(p string) *ClickhouseConfig {
	c.Password = p
	return c
}
func (c *ClickhouseConfig) WithDatabase(db string) *ClickhouseConfig {
	c.Database = db
	return c
}
func (c *ClickhouseConfig) WithCompress(compr int) *ClickhouseConfig {
	c.Compress = compr
	return c
}
func (c *ClickhouseConfig) WithSecure(secure bool) *ClickhouseConfig {
	c.Secure = secure
	return c
}
func (c *ClickhouseConfig) WithSaveFullQuery(sfq bool) *ClickhouseConfig {
	c.SaveFullQuery = sfq
	return c
}
func (c *ClickhouseConfig) WithWorkers(w int) *ClickhouseConfig {
	c.Workers = w
	return c
}
func (c *ClickhouseConfig) WithChannelSize(size int) *ClickhouseConfig {
	c.outputChannel = make(chan util.DNSResult, size)
	c.closeChannel = make(chan bool)
	return c
}

func (c *ClickhouseConfig) IsEnabled() bool {
	return c.Enabled
}

// Initialize function should not block
func (chConfig *ClickhouseConfig) Initialize(ctx context.Context) error {
	if !chConfig.Enabled {
		return errors.New("output not enabled")
	}

	if chConfig.BatchSize == 0 {
		chConfig.BatchSize = 100000
	}

	if chConfig.Workers == 0 {
		chConfig.Workers = 1
	}

	if chConfig.Compress > 9 {
		log.Warn("invalid compression level provided")
	}

	log.Info("Creating Clickhouse Output Channel")
	go chConfig.Output(ctx)
	return nil
}

func (chConfig *ClickhouseConfig) Close() {
	// todo: implement this
	<-chConfig.closeChannel
}

func (chConfig *ClickhouseConfig) OutputChannel() chan util.DNSResult {
	return chConfig.outputChannel
}

func (chConfig *ClickhouseConfig) connectClickhouseRetry(ctx context.Context) (driver.Conn, driver.Batch) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		c, b, err := chConfig.connectClickhouse(ctx)
		if err == nil {
			return c, b
		}

		log.Errorf("Error connecting to Clickhouse: %s", err)
		select {
		case <-tick.C:
			continue
		case <-ctx.Done():
			return nil, nil
		}
	}
}

func (chConfig *ClickhouseConfig) connectClickhouse(ctx context.Context) (driver.Conn, driver.Batch, error) {
	compressOption := clickhouse.Compression{Method: clickhouse.CompressionNone, Level: 0}
	if chConfig.Compress > 0 {
		compressOption = clickhouse.Compression{Method: clickhouse.CompressionLZ4, Level: int(chConfig.Compress)}
	}

	tlsOption := &tls.Config{InsecureSkipVerify: util.GeneralFlags.SkipTLSVerification}
	if !chConfig.Secure {
		tlsOption = nil
	}

	connection, err := clickhouse.Open(&clickhouse.Options{
		Addr: chConfig.Address,
		Auth: clickhouse.Auth{
			Database: chConfig.Database,
			Username: chConfig.Username,
			Password: chConfig.Password,
		},
		DialTimeout:     time.Second * 2,
		MaxOpenConns:    32,
		MaxIdleConns:    16,
		ConnMaxLifetime: time.Hour,
		TLS:             tlsOption,
		Debug:           chConfig.Debug,
		Compression:     &compressOption,
	})
	// connection, err := clickhouse.Open(fmt.Sprintf("tcp://%v?debug=%v&skip_verify=%v&secure=%v&compress=%v&username=%s&password=%s&database=%s", chConfig.ClickhouseAddress, chConfig.ClickhouseDebug, util.GeneralFlags.SkipTLSVerification, chConfig.ClickhouseSecure, chConfig.ClickhouseCompress, chConfig.ClickhouseUsername, chConfig.ClickhousePassword, chConfig.ClickhouseDatabase))
	if err != nil {
		log.Error(err)
		return connection, nil, err
	}

	batch, err := connection.PrepareBatch(ctx, "INSERT INTO DNS_LOG")
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
func (chConfig *ClickhouseConfig) Output(ctx context.Context) {
	g, gCtx := errgroup.WithContext(ctx)
	for i := 0; i < int(chConfig.Workers); i++ {
		g.Go(func() error { return chConfig.clickhouseOutputWorker(gCtx) })
	}
}

func (chConfig *ClickhouseConfig) clickhouseOutputWorker(ctx context.Context) error {
	conn, batch := chConfig.connectClickhouseRetry(ctx)
	clickhouseSentToOutput := metrics.GetOrRegisterCounter("clickhouseSentToOutput", metrics.DefaultRegistry)
	clickhouseSkipped := metrics.GetOrRegisterCounter("clickhouseSkipped", metrics.DefaultRegistry)
	clickhouseFailed := metrics.GetOrRegisterCounter("clickhouseFailed", metrics.DefaultRegistry)

	c := uint(0)

	ticker := time.NewTicker(time.Second * 5)
	div := 0

	if chConfig.Delay > 0 {
		chConfig.BatchSize = 1
		div = -1
		ticker = time.NewTicker(chConfig.Delay)
	} else {
		ticker.Stop()
	}

	for {
		select {
		case data := <-chConfig.outputChannel:
			for _, dnsQuery := range data.DNS.Question {
				c++
				if util.CheckIfWeSkip(chConfig.OutputType, dnsQuery.Name) {
					clickhouseSkipped.Inc(1)
					continue
				}
				clickhouseSentToOutput.Inc(1)

				fullQuery := ""
				if chConfig.SaveFullQuery {
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
				if int(c%chConfig.BatchSize) == div {
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
			err := batch.Send()
			if err != nil {
				log.Warnf("Error while executing batch: %v", err)
				clickhouseFailed.Inc(int64(c))
			}
			c = 0
			batch, _ = conn.PrepareBatch(ctx, "INSERT INTO DNS_LOG")
		case <-ctx.Done():
			err := batch.Flush()
			if err != nil {
				log.Warnf("Errro while executing batch: %v", err)
				clickhouseFailed.Inc(int64(c))
			}
			conn.Close()
			log.Debug("exiting out of clickhouse output") //todo:remove
			return nil
		}
	}
}

// var _ = clickhouseConfig{}.initializeFlags()
// vim: foldmethod=marker
