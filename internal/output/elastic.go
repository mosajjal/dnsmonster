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
	"net/http"
	"time"

	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"
)

// (OutputConfig interface now defined in output.go)

// ElasticConfig is the configuration and runtime struct for Elastic output.
type ElasticConfig struct {
	OutputType       uint
	Address          []string
	OutputIndex      string
	BatchSize        uint
	BatchDelay       time.Duration
	outputChannel    chan util.DNSResult
	outputMarshaller util.OutputMarshaller
	closeChannel     chan bool
}

// NewElasticConfig creates a new ElasticConfig with default values.
func NewElasticConfig() *ElasticConfig {
	return &ElasticConfig{
		outputChannel: nil,
		closeChannel:  nil,
	}
}

// WithOutputType sets the OutputType and returns the config for chaining.
func (c *ElasticConfig) WithOutputType(t uint) *ElasticConfig {
	c.OutputType = t
	return c
}

// WithAddress sets the Address and returns the config for chaining.
func (c *ElasticConfig) WithAddress(addr []string) *ElasticConfig {
	c.Address = addr
	return c
}

// WithOutputIndex sets the OutputIndex and returns the config for chaining.
func (c *ElasticConfig) WithOutputIndex(index string) *ElasticConfig {
	c.OutputIndex = index
	return c
}

// WithBatchSize sets the BatchSize and returns the config for chaining.
func (c *ElasticConfig) WithBatchSize(size uint) *ElasticConfig {
	c.BatchSize = size
	return c
}

// WithBatchDelay sets the BatchDelay and returns the config for chaining.
func (c *ElasticConfig) WithBatchDelay(delay time.Duration) *ElasticConfig {
	c.BatchDelay = delay
	return c
}

// WithChannelSize initializes the output and close channels and returns the config for chaining.
func (c *ElasticConfig) WithChannelSize(channelSize int) *ElasticConfig {
	c.outputChannel = make(chan util.DNSResult, channelSize)
	c.closeChannel = make(chan bool)
	return c
}

// Configuration for Elastic output is now provided via the main TOML config and passed in at runtime.

// initialize function should not block. otherwise the dispatcher will get stuck
func (esConfig *ElasticConfig) Initialize(ctx context.Context) error {
	var err error
	esConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if esConfig.OutputType > 0 && esConfig.OutputType < 5 {
		log.Info("Creating Elastic Output Channel")
		go esConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (esConfig *ElasticConfig) Close() {
	// todo: implement this
	<-esConfig.closeChannel
}

func (esConfig *ElasticConfig) OutputChannel() chan util.DNSResult {
	return esConfig.outputChannel
}

// var elasticUuidGen = fastuuid.MustNewGenerator()
// var ctx = context.Background()

func (esConfig *ElasticConfig) connectelasticRetry(ctx context.Context) *elastic.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if esConfig.OutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := esConfig.connectelastic(ctx)
		if err == nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue

	}
}

func (esConfig *ElasticConfig) connectelastic(ctx context.Context) (*elastic.Client, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: util.GeneralFlags.SkipTLSVerification},
	}
	httpClient := &http.Client{Transport: tr}

	client, err := elastic.NewClient(
		elastic.SetHttpClient(httpClient),
		elastic.SetURL(esConfig.Address...),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10*time.Second),
		// elastic.SetRetrier(connectelasticRetry(exiting, elasticEndpoint)),
		elastic.SetGzip(true),
		elastic.SetErrorLog(log.New()),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Ping the Elasticsearch server to get e.g. the version number
	info, code, err := client.Ping(esConfig.Address[0]).Do(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	return client, err
}

func (esConfig *ElasticConfig) Output(ctx context.Context) {
	client := esConfig.connectelasticRetry(ctx)
	batch := make([]util.DNSResult, 0, esConfig.BatchSize)

	ticker := time.NewTicker(esConfig.BatchDelay)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(esConfig.OutputIndex).Do(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(esConfig.OutputIndex).Do(ctx)
		if err != nil {
			log.Fatal(err)
		}

		if !createIndex.Acknowledged {
			log.Fatalln("Could not create the Elastic index.. Exiting")
		}
	}

	for {
		select {
		case data := <-esConfig.outputChannel:
			if util.GeneralFlags.PacketLimit == 0 || len(batch) < util.GeneralFlags.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			if err := esConfig.elasticSendData(ctx, client, batch); err != nil {
				log.Info(err)
				client = esConfig.connectelasticRetry(ctx)
			} else {
				batch = make([]util.DNSResult, 0, esConfig.BatchSize)
			}

		}
	}
}

func (esConfig *ElasticConfig) elasticSendData(ctx context.Context, client *elastic.Client, batch []util.DNSResult) error {
	elasticSentToOutput := metrics.GetOrRegisterCounter("elasticSentToOutput", metrics.DefaultRegistry)
	elasticSkipped := metrics.GetOrRegisterCounter("elasticSkipped", metrics.DefaultRegistry)

	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if util.CheckIfWeSkip(esConfig.OutputType, dnsQuery.Name) {
				elasticSkipped.Inc(1)
				continue
			}
			elasticSentToOutput.Inc(1)

			// batch[i].UUID = elasticUuidGen.Hex128()

			_, err := client.Index().
				Index(esConfig.OutputIndex).
				Type("_doc").
				BodyString(string(esConfig.outputMarshaller.Marshal(batch[i]))).
				Do(ctx)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	_, err := client.Flush().Index(esConfig.OutputIndex).Do(ctx)
	return err
}

// This will allow an instance to be spawned at import time
// var _ = elasticConfig{}.initializeFlags()
// vim: foldmethod=marker
