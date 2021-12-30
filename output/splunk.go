package output

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
	"github.com/mosajjal/dnsmonster/util"
)

var splunkStats = types.OutputStats{Name: "splunk", SentToOutput: 0, Skipped: 0}
var splunkConnectionList = make(map[string]types.SplunkConnection)

func connectMultiSplunkRetry(spConfig types.SplunkConfig) {
	for _, splunkEndpoint := range spConfig.SplunkOutputEndpoints {
		go connectSplunkRetry(spConfig, splunkEndpoint)
	}
}

func connectSplunkRetry(spConfig types.SplunkConfig, splunkEndpoint string) types.SplunkConnection {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if spConfig.SplunkOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		// Error getting connection, wait the timer or check if we are exiting
		select {

		case <-tick.C:
			// check to see if the connection exists
			if conn, ok := splunkConnectionList[splunkEndpoint]; ok {
				if conn.Unhealthy != 0 {
					log.Warnf("Connection is unhealthy: %v", conn.Err)
					splunkConnectionList[splunkEndpoint] = connectSplunk(spConfig, splunkEndpoint)
				}
			} else {
				log.Warnf("new splunk endpoint %s", splunkEndpoint)
				splunkConnectionList[splunkEndpoint] = connectSplunk(spConfig, splunkEndpoint)
			}
		}
	}
}

func connectSplunk(spConfig types.SplunkConfig, splunkEndpoint string) types.SplunkConnection {

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: spConfig.General.SkipTlsVerification}}
	httpClient := &http.Client{Timeout: time.Second * 20, Transport: tr}

	splunkURL := splunkEndpoint
	if !strings.HasSuffix(splunkEndpoint, "/services/collector") {
		splunkURL = fmt.Sprintf("%s/services/collector", splunkEndpoint)
	}

	// we won't define sourcetype and index here, because we want to be able to do that per write
	client := splunk.NewClient(
		httpClient,
		splunkURL,
		spConfig.SplunkOutputToken,
		spConfig.SplunkOutputSource,
		spConfig.SplunkOutputSourceType,
		spConfig.SplunkOutputIndex,
	)
	err := client.CheckHealth()
	unhealthy := uint(0)
	if err != nil {
		unhealthy += 1
	}
	myConn := types.SplunkConnection{Client: client, Unhealthy: unhealthy, Err: err}
	log.Warnf("new splunk connection %v", myConn)
	return myConn
}

func selectHealthyConnection() string {
	// lastId is used where all the connections are unhealthy
	for id, connection := range splunkConnectionList {
		if connection.Unhealthy == 0 {
			return id
		}
	}
	log.Warn("No more healthy HEC connections left")
	return ""
}

func SplunkOutput(spConfig types.SplunkConfig) {
	log.Infof("Connecting to Splunk endpoints")
	connectMultiSplunkRetry(spConfig)

	batch := make([]types.DNSResult, 0, spConfig.SplunkBatchSize)
	rand.Seed(time.Now().Unix())
	ticker := time.NewTicker(spConfig.SplunkBatchDelay)
	printStatsTicker := time.NewTicker(spConfig.General.PrintStatsDelay)

	for {
		select {
		case data := <-spConfig.ResultChannel:
			if spConfig.General.PacketLimit == 0 || len(batch) < spConfig.General.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			healthyId := selectHealthyConnection()
			if conn, ok := splunkConnectionList[healthyId]; ok {

				if err := splunkSendData(conn.Client, batch, spConfig); err != nil {
					log.Info(err)
					log.Warnf("marking connection as unhealthy: %+v", conn)
					conn.Unhealthy += 1
					splunkConnectionList[healthyId] = conn
					splunkStats.Skipped += len(batch)
				} else {
					batch = make([]types.DNSResult, 0, spConfig.SplunkBatchSize)
				}
			} else {
				log.Warn("Splunk Connection not found")
				splunkStats.Skipped += len(batch)
			}

		case <-printStatsTicker.C:
			log.Infof("output: %+v", splunkStats)
		}
	}
}

func splunkSendData(client *splunk.Client, batch []types.DNSResult, spConfig types.SplunkConfig) error {
	var events []*splunk.Event
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if util.CheckIfWeSkip(spConfig.SplunkOutputType, dnsQuery.Name) {
				splunkStats.Skipped++
				continue
			}
			splunkStats.SentToOutput++
			events = append(
				events,
				client.NewEventWithTime(batch[i].Timestamp, batch[i].String(), spConfig.SplunkOutputSource, spConfig.SplunkOutputSourceType, spConfig.SplunkOutputIndex),
			)
		}
	}
	if len(events) > 0 {
		return client.LogEvents(events)
	} else {
		return nil
	}
}
