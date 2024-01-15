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
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
	"github.com/mosajjal/dnsmonster/internal/util"
)

type splunkConfig struct {
	SplunkOutputType       uint          `long:"splunkoutputtype"            ini-name:"splunkoutputtype"            env:"DNSMONSTER_SPLUNKOUTPUTTYPE"            default:"0"                                                       description:"What should be written to HEC. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"           choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	SplunkOutputEndpoint   []string      `long:"splunkoutputendpoint"        ini-name:"splunkoutputendpoint"        env:"DNSMONSTER_SPLUNKOUTPUTENDPOINT"        default:""                                                        description:"splunk endpoint address, example: http://127.0.0.1:8088. Used if splunkOutputType is not none, can be specified multiple times for load balanace and HA"`
	SplunkOutputToken      string        `long:"splunkoutputtoken"           ini-name:"splunkoutputtoken"           env:"DNSMONSTER_SPLUNKOUTPUTTOKEN"           default:"00000000-0000-0000-0000-000000000000"                    description:"Splunk HEC Token"`
	SplunkOutputIndex      string        `long:"splunkoutputindex"           ini-name:"splunkoutputindex"           env:"DNSMONSTER_SPLUNKOUTPUTINDEX"           default:"temp"                                                    description:"Splunk Output Index"`
	SplunkOutputProxy      string        `long:"splunkoutputproxy"           ini-name:"splunkoutputproxy"           env:"DNSMONSTER_SPLUNKOUTPUTPROXY"           default:""                                                        description:"Splunk Output Proxy in URI format"`
	SplunkOutputSource     string        `long:"splunkoutputsource"          ini-name:"splunkoutputsource"          env:"DNSMONSTER_SPLUNKOUTPUTSOURCE"          default:"dnsmonster"                                              description:"Splunk Output Source"`
	SplunkOutputSourceType string        `long:"splunkoutputsourcetype"      ini-name:"splunkoutputsourcetype"      env:"DNSMONSTER_SPLUNKOUTPUTSOURCETYPE"      default:"json"                                                    description:"Splunk Output Sourcetype"`
	SplunkBatchSize        uint          `long:"splunkbatchsize"             ini-name:"splunkbatchsize"             env:"DNSMONSTER_SPLUNKBATCHSIZE"             default:"1000"                                                    description:"Send data to HEC in batch sizes"`
	SplunkBatchDelay       time.Duration `long:"splunkbatchdelay"            ini-name:"splunkbatchdelay"            env:"DNSMONSTER_SPLUNKBATCHDELAY"            default:"1s"                                                      description:"Interval between sending results to HEC if Batch size is not filled"`
	outputChannel          chan util.DNSResult
	outputMarshaller       util.OutputMarshaller
	closeChannel           chan bool
}

type splunkConnection struct {
	Client    *splunk.Client
	Unhealthy uint
	Err       error
}

func init() {
	c := splunkConfig{}
	if _, err := util.GlobalParser.AddGroup("splunk_output", "Splunk Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (spConfig splunkConfig) Initialize(ctx context.Context) error {
	var err error
	spConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if spConfig.SplunkOutputType > 0 && spConfig.SplunkOutputType < 5 {
		log.Info("Creating Splunk Output Channel")
		go spConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (spConfig splunkConfig) Close() {
	// todo: implement this
	<-spConfig.closeChannel
}

func (spConfig splunkConfig) OutputChannel() chan util.DNSResult {
	return spConfig.outputChannel
}

var splunkConnectionList = make(map[string]splunkConnection)

func (spConfig splunkConfig) connectMultiSplunkRetry() {
	for _, splunkEndpoint := range spConfig.SplunkOutputEndpoint {
		go spConfig.connectSplunkRetry(splunkEndpoint)
	}
}

func (spConfig splunkConfig) connectSplunkRetry(splunkEndpoint string) {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if spConfig.SplunkOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for range tick.C {
		// check to see if the connection exists
		if conn, ok := splunkConnectionList[splunkEndpoint]; ok {
			if conn.Unhealthy != 0 {
				log.Warnf("Connection is unhealthy")
				splunkConnectionList[splunkEndpoint] = spConfig.connectSplunk(splunkEndpoint)
			}
		} else {
			log.Warnf("new splunk endpoint %s", splunkEndpoint)
			splunkConnectionList[splunkEndpoint] = spConfig.connectSplunk(splunkEndpoint)
		}
	}
}

func (spConfig splunkConfig) connectSplunk(splunkEndpoint string) splunkConnection {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: util.GeneralFlags.SkipTLSVerification}}
	httpClient := &http.Client{Timeout: time.Second * 20, Transport: tr}

	if spConfig.SplunkOutputProxy != "" {
		proxyURL, err := url.Parse(spConfig.SplunkOutputProxy)
		if err != nil {
			panic(err)
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

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
		unhealthy++
	}
	myConn := splunkConnection{Client: client, Unhealthy: unhealthy, Err: err}
	log.Warnf("new splunk connection")
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

func (spConfig splunkConfig) Output(ctx context.Context) {
	splunkFailed := metrics.GetOrRegisterCounter("splunkFailed", metrics.DefaultRegistry)

	log.Infof("Connecting to Splunk endpoints")
	spConfig.connectMultiSplunkRetry()

	batch := make([]util.DNSResult, 0, spConfig.SplunkBatchSize)
	rand.Seed(time.Now().Unix())
	ticker := time.NewTicker(spConfig.SplunkBatchDelay)

	for {
		select {
		case data := <-spConfig.outputChannel:
			if util.GeneralFlags.PacketLimit == 0 || len(batch) < util.GeneralFlags.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			healthyID := selectHealthyConnection()
			if conn, ok := splunkConnectionList[healthyID]; ok {
				if err := spConfig.splunkSendData(conn.Client, batch); err != nil {
					log.Warn(err)
					log.Warnf("marking connection as unhealthy")
					conn.Unhealthy++
					splunkConnectionList[healthyID] = conn
					splunkFailed.Inc(int64(len(batch)))
				} else {
					batch = make([]util.DNSResult, 0, spConfig.SplunkBatchSize)
				}
			} else {
				log.Warn("Splunk Connection not found")
				splunkFailed.Inc(int64(len(batch)))
			}

		}
	}
}

func (spConfig splunkConfig) splunkSendData(client *splunk.Client, batch []util.DNSResult) error {
	splunkSentToOutput := metrics.GetOrRegisterCounter("splunkSentToOutput", metrics.DefaultRegistry)
	splunkSkipped := metrics.GetOrRegisterCounter("splunkSkipped", metrics.DefaultRegistry)
	var events []*splunk.Event
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if util.CheckIfWeSkip(spConfig.SplunkOutputType, dnsQuery.Name) {

				splunkSkipped.Inc(1)
				continue
			}
			splunkSentToOutput.Inc(1)
			events = append(
				events,
				client.NewEventWithTime(batch[i].Timestamp, string(spConfig.outputMarshaller.Marshal(batch[i])), spConfig.SplunkOutputSource, spConfig.SplunkOutputSourceType, spConfig.SplunkOutputIndex),
			)
		}
	}
	if len(events) > 0 {
		return client.LogEvents(events)
	}
	return nil
}

// This will allow an instance to be spawned at import time
// var _ = splunkConfig{}.initializeFlags()
// vim: foldmethod=marker
