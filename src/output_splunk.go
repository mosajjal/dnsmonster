package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
)

var splunkStats = outputStats{"splunk", 0, 0}

func connectMultiSplunkRetry(spConfig splunkConfig) []*splunk.Client {
	var outputs []*splunk.Client
	for _, splunkEndpoint := range spConfig.splunkOutputEndpoints {
		outputs = append(outputs, connectSplunkRetry(spConfig, splunkEndpoint))
	}
	return outputs
}

func connectSplunkRetry(spConfig splunkConfig, splunkEndpoint string) *splunk.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if spConfig.splunkOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectSplunk(spConfig, splunkEndpoint)
		if err == nil {
			return conn
		}
		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-spConfig.general.exiting:
			// When exiting, return immediately
			return nil
		case <-tick.C:
			continue
		}
	}
}

func connectSplunk(spConfig splunkConfig, splunkEndpoint string) (*splunk.Client, error) {

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: spConfig.general.skipTlsVerification}}
	httpClient := &http.Client{Timeout: time.Second * 20, Transport: tr}

	splunkURL := splunkEndpoint
	if !strings.HasSuffix(splunkEndpoint, "/services/collector") {
		splunkURL = fmt.Sprintf("%s/services/collector", splunkEndpoint)
	}

	client := splunk.NewClient(
		httpClient,
		splunkURL,
		spConfig.splunkOutputToken,
		"",
		"",
		"",
	)
	err := client.CheckHealth()
	return client, err
}

func splunkOutput(spConfig splunkConfig) {
	spConfig.general.wg.Add(1)
	defer spConfig.general.wg.Done()

	clients := connectMultiSplunkRetry(spConfig)

	batch := make([]DNSResult, 0, spConfig.splunkBatchSize)
	rand.Seed(time.Now().Unix())
	ticker := time.Tick(spConfig.splunkBatchDelay)
	printStatsTicker := time.Tick(spConfig.general.printStatsDelay)

	for {
		select {
		case data := <-resultChannel:
			if spConfig.general.packetLimit == 0 || len(batch) < spConfig.general.packetLimit {
				batch = append(batch, data)
			}
		case <-ticker:
			client := clients[rand.Intn(len(clients))]
			if err := splunkSendData(client, batch, spConfig); err != nil {
				log.Info(err)
				client = connectSplunkRetry(spConfig, client.URL)
			} else {
				batch = make([]DNSResult, 0, spConfig.splunkBatchSize)
			}
		case <-spConfig.general.exiting:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", splunkStats)
		}
	}
}

func splunkSendData(client *splunk.Client, batch []DNSResult, spConfig splunkConfig) error {
	var events []*splunk.Event
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(spConfig.splunkOutputType, dnsQuery.Name) {
				splunkStats.Skipped++
				continue
			}
			splunkStats.SentToOutput++
			fullQuery, err := json.Marshal(batch[i])
			errorHandler(err)
			events = append(
				events,
				client.NewEventWithTime(batch[i].Timestamp, string(fullQuery), spConfig.splunkOutputSource, spConfig.splunkOutputSourceType, spConfig.splunkOutputIndex),
			)
		}
	}
	if len(events) > 0 {
		return client.LogEvents(events)
	} else {
		return nil
	}
}
