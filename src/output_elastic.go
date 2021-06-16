package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"
)

type elasticConfig struct {
	exiting               chan bool
	wg                    *sync.WaitGroup
	resultChannel         chan DNSResult
	elasticOutputEndpoint string
	elasticOutputIndex    string
	elasticOutputType     uint
	elasticBatchSize      uint
	elasticBatchDelay     time.Duration
	maskSize              int
	packetLimit           int
	saveFullQuery         bool
	serverName            string
	printStatsDelay       time.Duration
}

// var elasticUuidGen = fastuuid.MustNewGenerator()
var elasticstats = outputStats{"elastic", 0, 0}
var ctx = context.Background()

func connectelasticRetry(esConfig elasticConfig) *elastic.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if esConfig.elasticOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectelastic(esConfig)
		if err == nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-esConfig.exiting:
			// When exiting, return immediately
			return nil
		case <-tick.C:
			continue
		}
	}
}

func connectelastic(esConfig elasticConfig) (*elastic.Client, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(esConfig.elasticOutputEndpoint),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10*time.Second),
		// elastic.SetRetrier(connectelasticRetry(exiting, elasticEndpoint)),
		elastic.SetGzip(true),
		elastic.SetErrorLog(log.New()),
	)
	errorHandler(err)

	//TODO: are we retrying without exiting out of dnsmonster?
	// Ping the Elasticsearch server to get e.g. the version number
	info, code, err := client.Ping(esConfig.elasticOutputEndpoint).Do(ctx)
	errorHandler(err)
	fmt.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	return client, err
}

func elasticOutput(esConfig elasticConfig) {
	esConfig.wg.Add(1)
	defer esConfig.wg.Done()

	client := connectelasticRetry(esConfig)
	batch := make([]DNSResult, 0, esConfig.elasticBatchSize)

	ticker := time.Tick(esConfig.elasticBatchDelay)
	printStatsTicker := time.Tick(esConfig.printStatsDelay)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(esConfig.elasticOutputIndex).Do(ctx)
	errorHandler(err)

	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(esConfig.elasticOutputIndex).Do(ctx)
		errorHandler(err)

		if !createIndex.Acknowledged {
			log.Panicln("Could not create the Elastic index.. Exiting")
		}
	}

	for {
		select {
		case data := <-esConfig.resultChannel:
			if esConfig.packetLimit == 0 || len(batch) < esConfig.packetLimit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := elasticSendData(client, batch, esConfig); err != nil {
				log.Info(err)
				client = connectelasticRetry(esConfig)
			} else {
				batch = make([]DNSResult, 0, esConfig.elasticBatchSize)
			}
		case <-esConfig.exiting:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", elasticstats)
		}
	}
}

func elasticSendData(client *elastic.Client, batch []DNSResult, esConfig elasticConfig) error {
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(esConfig.elasticOutputType, dnsQuery.Name) {
				elasticstats.Skipped++
				continue
			}
			elasticstats.SentToOutput++

			// batch[i].UUID = elasticUuidGen.Hex128()
			fullQuery, err := json.Marshal(batch[i])
			errorHandler(err)

			_, err = client.Index().
				Index(esConfig.elasticOutputIndex).
				Type("_doc").
				BodyString(string(fullQuery)).
				Do(ctx)

			errorHandler(err)
		}
	}
	_, err := client.Flush().Index(esConfig.elasticOutputIndex).Do(ctx)
	return err

}
