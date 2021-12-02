package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"
)

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
		case <-types.GlobalExitChannel:
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
	client := connectelasticRetry(esConfig)
	batch := make([]types.DNSResult, 0, esConfig.elasticBatchSize)

	ticker := time.Tick(esConfig.elasticBatchDelay)
	printStatsTicker := time.Tick(esConfig.general.printStatsDelay)

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
			if esConfig.general.packetLimit == 0 || len(batch) < esConfig.general.packetLimit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := elasticSendData(client, batch, esConfig); err != nil {
				log.Info(err)
				client = connectelasticRetry(esConfig)
			} else {
				batch = make([]types.DNSResult, 0, esConfig.elasticBatchSize)
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", elasticstats)
		}
	}
}

func elasticSendData(client *elastic.Client, batch []types.DNSResult, esConfig elasticConfig) error {
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(esConfig.elasticOutputType, dnsQuery.Name) {
				elasticstats.Skipped++
				continue
			}
			elasticstats.SentToOutput++

			// batch[i].UUID = elasticUuidGen.Hex128()

			_, err := client.Index().
				Index(esConfig.elasticOutputIndex).
				Type("_doc").
				BodyString(string(batch[i].String())).
				Do(ctx)

			errorHandler(err)
		}
	}
	_, err := client.Flush().Index(esConfig.elasticOutputIndex).Do(ctx)
	return err

}
