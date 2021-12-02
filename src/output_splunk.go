package main

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
	"github.com/mosajjal/dnsmonster/types"
)

var splunkStats = outputStats{"splunk", 0, 0}
var splunkConnectionList = make(map[string]splunkConnection)
var splunkKickOff bool = false

func connectMultiSplunkRetry(spConfig splunkConfig) {
	for _, splunkEndpoint := range spConfig.splunkOutputEndpoints {
		go connectSplunkRetry(spConfig, splunkEndpoint)
	}
}

func connectSplunkRetry(spConfig splunkConfig, splunkEndpoint string) splunkConnection {
	tick := time.NewTicker(5 * time.Second)
	conn := splunkConnection{}
	// don't retry connection if we're doing dry run
	if spConfig.splunkOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-types.GlobalExitChannel:
			// When exiting, return immediately
			return conn
		case <-tick.C:
			// check to see if the connection exists
			if conn, ok := splunkConnectionList[splunkEndpoint]; ok {
				if conn.unhealthy != 0 {
					log.Warnf("Connection is unhealthy: %v", conn.err)
					splunkConnectionList[splunkEndpoint] = connectSplunk(spConfig, splunkEndpoint)
				}
			} else {
				log.Warnf("new splunk endpoint %s", splunkEndpoint)
				splunkConnectionList[splunkEndpoint] = connectSplunk(spConfig, splunkEndpoint)
			}
		}
	}
}

func connectSplunk(spConfig splunkConfig, splunkEndpoint string) splunkConnection {

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: spConfig.general.skipTlsVerification}}
	httpClient := &http.Client{Timeout: time.Second * 20, Transport: tr}

	splunkURL := splunkEndpoint
	if !strings.HasSuffix(splunkEndpoint, "/services/collector") {
		splunkURL = fmt.Sprintf("%s/services/collector", splunkEndpoint)
	}

	// we won't define sourcetype and index here, because we want to be able to do that per write
	client := splunk.NewClient(
		httpClient,
		splunkURL,
		spConfig.splunkOutputToken,
		spConfig.splunkOutputSource,
		spConfig.splunkOutputSourceType,
		spConfig.splunkOutputIndex,
	)
	err := client.CheckHealth()
	unhealthy := uint(0)
	if err != nil {
		unhealthy += 1
	}
	myConn := splunkConnection{client, unhealthy, err}
	log.Warnf("new splunk connection %v", myConn)
	return myConn
}

func selectHealthyConnection() string {
	// lastId is used where all the connections are unhealthy
	for id, connection := range splunkConnectionList {
		if connection.unhealthy == 0 {
			return id
		}
	}
	log.Warn("No more healthy HEC connections left")
	splunkKickOff = false
	return ""
}

func splunkOutput(spConfig splunkConfig) {

	log.Infof("Connecting to Splunk endpoints")
	connectMultiSplunkRetry(spConfig)

	batch := make([]types.DNSResult, 0, spConfig.splunkBatchSize)
	rand.Seed(time.Now().Unix())
	ticker := time.Tick(spConfig.splunkBatchDelay)
	printStatsTicker := time.Tick(spConfig.general.printStatsDelay)

	for {
		select {
		case data := <-spConfig.resultChannel:
			if spConfig.general.packetLimit == 0 || len(batch) < spConfig.general.packetLimit {
				batch = append(batch, data)
			}
		case <-ticker:
			healthyId := selectHealthyConnection()
			if conn, ok := splunkConnectionList[healthyId]; ok {

				if err := splunkSendData(conn.client, batch, spConfig); err != nil {
					log.Info(err)
					log.Warnf("marking connection as unhealthy: %+v", conn)
					conn.unhealthy += 1
					splunkConnectionList[healthyId] = conn
					splunkStats.Skipped += len(batch)
				} else {
					batch = make([]types.DNSResult, 0, spConfig.splunkBatchSize)
				}
			} else {
				log.Warn("Splunk Connection not found")
				splunkStats.Skipped += len(batch)
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", splunkStats)
		}
	}
}

func splunkSendData(client *splunk.Client, batch []types.DNSResult, spConfig splunkConfig) error {
	var events []*splunk.Event
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(spConfig.splunkOutputType, dnsQuery.Name) {
				splunkStats.Skipped++
				continue
			}
			splunkStats.SentToOutput++
			events = append(
				events,
				client.NewEventWithTime(batch[i].Timestamp, batch[i].String(), spConfig.splunkOutputSource, spConfig.splunkOutputSourceType, spConfig.splunkOutputIndex),
			)
		}
	}
	if len(events) > 0 {
		return client.LogEvents(events)
	} else {
		return nil
	}
}
