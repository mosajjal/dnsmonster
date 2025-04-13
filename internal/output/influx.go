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

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

// InfluxConfig is the configuration and runtime struct for InfluxDB output
type influxConfig struct {
	// Configuration options
	OutputType    uint   `long:"influxoutputtype" ini-name:"influxoutputtype" env:"DNSMONSTER_INFLUXOUTPUTTYPE" default:"0" description:"What should be written to influx. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	OutputServer  string `long:"influxoutputserver" ini-name:"influxoutputserver" env:"DNSMONSTER_INFLUXOUTPUTSERVER" default:"" description:"influx Server address, example: http://localhost:8086. Used if influxOutputType is not none"`
	OutputToken   string `long:"influxoutputtoken" ini-name:"influxoutputtoken" env:"DNSMONSTER_INFLUXOUTPUTTOKEN" default:"dnsmonster" description:"Influx Server Auth Token"`
	OutputBucket  string `long:"influxoutputbucket" ini-name:"influxoutputbucket" env:"DNSMONSTER_INFLUXOUTPUTBUCKET" default:"dnsmonster" description:"Influx Server Bucket"`
	OutputOrg     string `long:"influxoutputorg" ini-name:"influxoutputorg" env:"DNSMONSTER_INFLUXOUTPUTORG" default:"dnsmonster" description:"Influx Server Org"`
	OutputWorkers uint   `long:"influxoutputworkers" ini-name:"influxoutputworkers" env:"DNSMONSTER_INFLUXOUTPUTWORKERS" default:"8" description:"Minimum capacity of the cache array used to send data to Influx"`
	BatchSize     uint   `long:"influxbatchsize" ini-name:"influxbatchsize" env:"DNSMONSTER_INFLUXBATCHSIZE" default:"1000" description:"Minimum capacity of the cache array used to send data to Influx"`

	// Runtime resources
	outputChannel chan util.DNSResult
	closeChannel  chan bool
}

// NewInfluxConfig creates a new InfluxConfig with default values
func NewInfluxConfig() *influxConfig {
	return &influxConfig{
		outputChannel: nil,
		closeChannel:  nil,
	}
}

// WithOutputType sets the OutputType and returns the config for chaining
func (c *influxConfig) WithOutputType(t uint) *influxConfig {
	c.OutputType = t
	return c
}

// WithOutputServer sets the OutputServer and returns the config for chaining
func (c *influxConfig) WithOutputServer(server string) *influxConfig {
	c.OutputServer = server
	return c
}

// WithOutputToken sets the OutputToken and returns the config for chaining
func (c *influxConfig) WithOutputToken(token string) *influxConfig {
	c.OutputToken = token
	return c
}

// WithOutputBucket sets the OutputBucket and returns the config for chaining
func (c *influxConfig) WithOutputBucket(bucket string) *influxConfig {
	c.OutputBucket = bucket
	return c
}

// WithOutputOrg sets the OutputOrg and returns the config for chaining
func (c *influxConfig) WithOutputOrg(org string) *influxConfig {
	c.OutputOrg = org
	return c
}

// WithOutputWorkers sets the OutputWorkers and returns the config for chaining
func (c *influxConfig) WithOutputWorkers(workers uint) *influxConfig {
	c.OutputWorkers = workers
	return c
}

// WithBatchSize sets the BatchSize and returns the config for chaining
func (c *influxConfig) WithBatchSize(size uint) *influxConfig {
	c.BatchSize = size
	return c
}

// WithChannelSize initializes the output and close channels and returns the config for chaining
func (c *influxConfig) WithChannelSize(channelSize int) *influxConfig {
	c.outputChannel = make(chan util.DNSResult, channelSize)
	c.closeChannel = make(chan bool)
	return c
}

func init() {
	c := influxConfig{}
	if _, err := util.GlobalParser.AddGroup("influx_output", "Influx Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// Initialize function should not block. otherwise the dispatcher will get stuck
func (c influxConfig) Initialize(ctx context.Context) error {
	if c.OutputType > 0 && c.OutputType < 5 {
		log.Info("Creating Influx Output Channel")
		go c.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (c influxConfig) Close() {
	// todo: implement this
	<-c.closeChannel
}

func (c influxConfig) OutputChannel() chan util.DNSResult {
	return c.outputChannel
}

func (c influxConfig) connectInfluxRetry() influxdb2.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if c.OutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn := c.connectInflux()
		if conn != nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue

	}
}

func (c influxConfig) connectInflux() influxdb2.Client {
	client := influxdb2.NewClientWithOptions(c.OutputServer, c.OutputToken, influxdb2.DefaultOptions().SetBatchSize(c.BatchSize))
	return client
}

func (c influxConfig) Output(ctx context.Context) {
	influxSentToOutput := metrics.GetOrRegisterCounter("influxSentToOutput", metrics.DefaultRegistry)
	influxSkipped := metrics.GetOrRegisterCounter("influxSkipped", metrics.DefaultRegistry)

	client := c.connectInfluxRetry()
	writeAPI := client.WriteAPI(c.OutputOrg, c.OutputBucket)
	batch := make([]util.DNSResult, 0, c.BatchSize)
	ticker := time.NewTicker(time.Second * 5)

	for {
		select {
		case data := <-c.outputChannel:
			batch = append(batch, data)
		case <-ticker.C:
			for _, item := range batch {
				for _, dnsQuery := range item.DNS.Question {
					if util.CheckIfWeSkip(c.OutputType, dnsQuery.Name) {
						influxSkipped.Inc(1)
						continue
					}
					influxSentToOutput.Inc(1)
					row := map[string]interface{}{
						"ipversion": item.IPVersion,
						"SrcIP":     item.SrcIP,
						"DstIP":     item.DstIP,
						"protocol":  item.Protocol,
						"qr":        item.DNS.Response,
						"question":  dnsQuery.Name,
					}
					p := influxdb2.NewPoint("dns", nil, row, item.Timestamp)
					writeAPI.WritePoint(p)
				}
			}
			writeAPI.Flush()
			batch = batch[:0]
		case <-ctx.Done():
			writeAPI.Flush()
			client.Close()
			log.Debug("Exiting Influx output")
			return
		}
	}
}

func (c influxConfig) InfluxWorker() {
	influxSentToOutput := metrics.GetOrRegisterCounter("influxSentToOutput", metrics.DefaultRegistry)
	influxSkipped := metrics.GetOrRegisterCounter("stdoutSkipped", metrics.DefaultRegistry)
	client := c.connectInfluxRetry()
	writeAPI := client.WriteAPI(c.OutputOrg, c.OutputBucket)

	for data := range c.outputChannel {
		for _, dnsQuery := range data.DNS.Question {
			if util.CheckIfWeSkip(c.OutputType, dnsQuery.Name) {
				influxSkipped.Inc(1)
				continue
			}
			influxSentToOutput.Inc(1)

			edns, dobit := uint8(0), uint8(0)
			if edns0 := data.DNS.IsEdns0(); edns0 != nil {
				edns = 1
				if edns0.Do() {
					dobit = 1
				}
			}
			row := map[string]interface{}{
				"ipversion":    data.IPVersion,
				"SrcIP":        data.SrcIP,
				"DstIP":        data.DstIP,
				"protocol":     data.Protocol,
				"qr":           data.DNS.Response,
				"opCode":       data.DNS.Opcode,
				"class":        dnsQuery.Qclass,
				"qtype":        dnsQuery.Qtype,
				"responseCode": data.DNS.Rcode,
				"question":     dnsQuery.Name,
				"size":         data.PacketLength,
				"edns":         edns,
				"dobit":        dobit,
				"id":           data.DNS.Id,
			}

			p := influxdb2.NewPoint("system", map[string]string{
				"hostname": util.GeneralFlags.ServerName,
			}, row, data.Timestamp)
			writeAPI.WritePoint(p)
		}
	}

	// Force all unwritten data to be sent
	writeAPI.Flush()
	// Ensures background processes finishes
	client.Close()
}

// This will allow an instance to be spawned at import time
// var _ = influxConfig{}.initializeFlags()
// vim: foldmethod=marker
