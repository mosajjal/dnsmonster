package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/olivere/elastic"
	"github.com/rogpeppe/fastuuid"
)

var elasticUuidGen = fastuuid.MustNewGenerator()
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
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
		// elastic.SetHeaders(http.Header{
		//   "X-Caller-Id": []string{"..."},
	)
	if err != nil {
		// Handle error
		panic(err)
	}

	// Ping the Elasticsearch server to get e.g. the version number
	info, code, err := client.Ping(elasticEndpoint).Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	fmt.Printf("Elasticsearch returned with code %d and version %s\n", code, info.Version.Number)

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
	if err != nil {
		// Handle error
		panic(err)
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(elasticIndex).Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	}

	for {
		select {
		case data := <-resultChannel:
			if limit == 0 || len(batch) < limit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := elasticSendData(connect, batch); err != nil {
				log.Println(err)
				connect = connectelasticRetry(exiting, elasticEndpoint, elasticEndpoint)
			} else {
				batch = make([]DNSResult, 0, elasticBatchSize)
			}
		case <-exiting:
			return
		case <-printStatsTicker:
			log.Printf("output: %+v\n", elasticstats)
		}
	}
}

func elasticSendData(connect *elastic.Conn, batch []DNSResult) error {
	var msg []elastic.Message
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(*elasticOutputType, dnsQuery.Name) {
				elasticstats.Skipped++
				continue
			}
			elasticstats.SentToOutput++

			myUUID := elasticUuidGen.Hex128()
			fullQuery, err := json.Marshal(batch[i])
			errorHandler(err)

			msg = append(msg, elastic.Message{
				Key:   []byte(myUUID),
				Value: []byte(fmt.Sprintf("%s\n", fullQuery)),
			})

		}
	}
	_, err := connect.WriteMessages(msg...)
	return err

}
