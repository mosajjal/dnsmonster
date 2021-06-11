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

// var elasticUuidGen = fastuuid.MustNewGenerator()
var elasticstats = outputStats{"elastic", 0, 0}
var ctx = context.Background()

func connectelasticRetry(exiting chan bool, elasticEndpoint string) *elastic.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if *elasticOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectelastic(exiting, elasticEndpoint)
		if err == nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-exiting:
			// When exiting, return immediately
			return nil
		case <-tick.C:
			continue
		}
	}
}

func connectelastic(exiting chan bool, elasticEndpoint string) (*elastic.Client, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(elasticEndpoint),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10*time.Second),
		// elastic.SetRetrier(connectelasticRetry(exiting, elasticEndpoint)),
		elastic.SetGzip(true),
		elastic.SetErrorLog(log.New()),
	)
	errorHandler(err)

	// Ping the Elasticsearch server to get e.g. the version number
	info, code, err := client.Ping(elasticEndpoint).Do(ctx)
	errorHandler(err)
	fmt.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	return client, err
}

func elasticOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup, elasticEndpoint string, elasticIndex string, elasticBatchSize uint, batchDelay time.Duration, limit int) {
	wg.Add(1)
	defer wg.Done()

	client := connectelasticRetry(exiting, elasticEndpoint)
	batch := make([]DNSResult, 0, elasticBatchSize)

	ticker := time.Tick(batchDelay)
	printStatsTicker := time.Tick(*printStatsDelay)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(elasticIndex).Do(ctx)
	errorHandler(err)

	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(elasticIndex).Do(ctx)
		errorHandler(err)

		if !createIndex.Acknowledged {
			log.Panicln("Could not create the Elastic index.. Exiting")
		}
	}

	for {
		select {
		case data := <-resultChannel:
			if limit == 0 || len(batch) < limit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := elasticSendData(client, elasticIndex, batch); err != nil {
				log.Info(err)
				client = connectelasticRetry(exiting, elasticEndpoint)
			} else {
				batch = make([]DNSResult, 0, elasticBatchSize)
			}
		case <-exiting:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", elasticstats)
		}
	}
}

func elasticSendData(client *elastic.Client, elasticIndex string, batch []DNSResult) error {
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(*elasticOutputType, dnsQuery.Name) {
				elasticstats.Skipped++
				continue
			}
			elasticstats.SentToOutput++

			// batch[i].UUID = elasticUuidGen.Hex128()
			fullQuery, err := json.Marshal(batch[i])
			errorHandler(err)

			_, err = client.Index().
				Index(elasticIndex).
				Type("_doc").
				BodyString(string(fullQuery)).
				Do(ctx)

			errorHandler(err)
		}
	}
	_, err := client.Flush().Index(elasticIndex).Do(ctx)
	return err

}
