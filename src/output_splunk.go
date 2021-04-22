package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
)

var splunkStats = outputStats{"splunk", 0, 0}

func connectMultiSplunkRetry(exiting chan bool, splunkEndpoints []string, splunkHecToken string, skipTlsVerification bool) []*splunk.Client {
	var outputs []*splunk.Client
	for _, splunkEndpoint := range splunkEndpoints {
		outputs = append(outputs, connectSplunkRetry(exiting, splunkEndpoint, splunkHecToken, skipTlsVerification))
	}
	return outputs
}

func connectSplunkRetry(exiting chan bool, splunkEndpoint string, splunkHecToken string, skipTlsVerification bool) *splunk.Client {
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

	splunkURL := splunkEndpoint
	if !strings.HasSuffix(splunkEndpoint, "/services/collector") {
		splunkURL = fmt.Sprintf("%s/services/collector", splunkEndpoint)
	}

	client := splunk.NewClient(
		httpClient,
		splunkURL,
		splunkHecToken,
		"",
		"",
		"",
	)
	err := client.CheckHealth()
	return client, err
}

func splunkOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup, splunkEndpoints []string, splunkHecToken string, splunkIndex string, splunkBatchSize uint, batchDelay time.Duration, limit int) {
	wg.Add(1)
	defer wg.Done()

	clients := connectMultiSplunkRetry(exiting, splunkEndpoints, splunkHecToken, *skipTlsVerification)

	batch := make([]DNSResult, 0, splunkBatchSize)
	rand.Seed(time.Now().Unix())
	ticker := time.Tick(batchDelay)
	printStatsTicker := time.Tick(*printStatsDelay)

	for {
		select {
		case data := <-resultChannel:
			if limit == 0 || len(batch) < limit {
				batch = append(batch, data)
			}
		case <-ticker:
			client := clients[rand.Intn(len(clients))]
			if err := splunkSendData(client, splunkIndex, *splunkOutputSource, *splunkOutputSourceType, batch); err != nil {
				log.Info(err)
				client = connectSplunkRetry(exiting, client.URL, splunkHecToken, *skipTlsVerification)
			} else {
				batch = make([]DNSResult, 0, splunkBatchSize)
			}
		case <-exiting:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", splunkStats)
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
