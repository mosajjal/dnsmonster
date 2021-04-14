package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
)

var splunkStats = outputStats{"splunk", 0, 0}

func connecSplunkRetry(exiting chan bool, splunkEndpoint string, splunkHecToken string, skipTlsVerification bool) *splunk.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if *splunkOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectSplunk(exiting, splunkEndpoint, splunkHecToken, skipTlsVerification)
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

func connectSplunk(exiting chan bool, splunkEndpoint string, splunkHecToken string, skipTlsVerification bool) (*splunk.Client, error) {

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTlsVerification}}
	httpClient := &http.Client{Timeout: time.Second * 20, Transport: tr}
	client := splunk.NewClient(
		httpClient,
		fmt.Sprintf("%s/services/collector", splunkEndpoint),
		splunkHecToken,
		"",
		"",
		"",
	)
	err := client.CheckHealth()
	return client, err
}

func splunkOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup, splunkEndpoint string, splunkHecToken string, splunkIndex string, splunkBatchSize uint, batchDelay time.Duration, limit int) {
	wg.Add(1)
	defer wg.Done()

	client := connecSplunkRetry(exiting, splunkEndpoint, splunkHecToken, *skipTlsVerification)

	batch := make([]DNSResult, 0, splunkBatchSize)

	ticker := time.Tick(batchDelay)
	printStatsTicker := time.Tick(*printStatsDelay)

	for {
		select {
		case data := <-resultChannel:
			if limit == 0 || len(batch) < limit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := splunkSendData(client, splunkIndex, *splunkOutputSource, *splunkOutputSourceType, batch); err != nil {
				log.Println(err)
				client = connecSplunkRetry(exiting, splunkEndpoint, splunkHecToken, *skipTlsVerification)
			} else {
				batch = make([]DNSResult, 0, splunkBatchSize)
			}
		case <-exiting:
			return
		case <-printStatsTicker:
			log.Printf("output: %+v\n", splunkStats)
		}
	}
}

func splunkSendData(client *splunk.Client, splunkIndex string, splunkSource string, splunkSourceType string, batch []DNSResult) error {
	var events []*splunk.Event
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(*splunkOutputType, dnsQuery.Name) {
				splunkStats.Skipped++
				continue
			}
			splunkStats.SentToOutput++
			fullQuery, err := json.Marshal(batch[i])
			errorHandler(err)
			events = append(
				events,
				client.NewEventWithTime(batch[i].Timestamp, string(fullQuery), splunkSource, splunkSourceType, splunkIndex),
			)
		}
	}
	if len(events) > 0 {
		return client.LogEvents(events)
	} else {
		return nil
	}
}
