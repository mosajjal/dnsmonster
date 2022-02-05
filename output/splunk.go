package output

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
)

type SplunkConfig struct {
	SplunkOutputType       uint          `long:"splunkOutputType"            env:"DNSMONSTER_SPLUNKOUTPUTTYPE"            default:"0"                                                       description:"What should be written to HEC. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"           choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	SplunkOutputEndpoints  []string      `long:"splunkOutputEndpoints"       env:"DNSMONSTER_SPLUNKOUTPUTENDPOINTS"       default:""                                                        description:"splunk endpoint address, example: http://127.0.0.1:8088. Used if splunkOutputType is not none"`
	SplunkOutputToken      string        `long:"splunkOutputToken"           env:"DNSMONSTER_SPLUNKOUTPUTTOKEN"           default:"00000000-0000-0000-0000-000000000000"                    description:"Splunk HEC Token"`
	SplunkOutputIndex      string        `long:"splunkOutputIndex"           env:"DNSMONSTER_SPLUNKOUTPUTINDEX"           default:"temp"                                                    description:"Splunk Output Index"`
	SplunkOutputSource     string        `long:"splunkOutputSource"          env:"DNSMONSTER_SPLUNKOUTPUTSOURCE"          default:"dnsmonster"                                              description:"Splunk Output Source"`
	SplunkOutputSourceType string        `long:"splunkOutputSourceType"      env:"DNSMONSTER_SPLUNKOUTPUTSOURCETYPE"      default:"json"                                                    description:"Splunk Output Sourcetype"`
	SplunkBatchSize        uint          `long:"splunkBatchSize"             env:"DNSMONSTER_SPLUNKBATCHSIZE"             default:"1000"                                                    description:"Send data to HEC in batch sizes"`
	SplunkBatchDelay       time.Duration `long:"splunkBatchDelay"            env:"DNSMONSTER_SPLUNKBATCHDELAY"            default:"1s"                                                      description:"Interval between sending results to HEC if Batch size is not filled"`
	outputChannel          chan types.DNSResult
	closeChannel           chan bool
}

type SplunkConnection struct {
	Client    *splunk.Client
	Unhealthy uint
	Err       error
}

func (config SplunkConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("splunk_output", "Splunk Output", &config)

	config.outputChannel = make(chan types.DNSResult, util.GeneralFlags.ResultChannelSize)

	types.GlobalDispatchList = append(types.GlobalDispatchList, &config)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (config SplunkConfig) Initialize() error {
	if config.SplunkOutputType > 0 && config.SplunkOutputType < 5 {
		log.Info("Creating Splunk Output Channel")
		go config.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (config SplunkConfig) Close() {
	//todo: implement this
	<-config.closeChannel
}

func (config SplunkConfig) OutputChannel() chan types.DNSResult {
	return config.outputChannel
}

var splunkConnectionList = make(map[string]SplunkConnection)

func (spConfig SplunkConfig) connectMultiSplunkRetry() {
	for _, splunkEndpoint := range spConfig.SplunkOutputEndpoints {
		go spConfig.connectSplunkRetry(splunkEndpoint)
	}
}

func (spConfig SplunkConfig) connectSplunkRetry(splunkEndpoint string) {
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
				log.Warnf("Connection is unhealthy: %v", conn.Err)
				splunkConnectionList[splunkEndpoint] = spConfig.connectSplunk(splunkEndpoint)
			}
		} else {
			log.Warnf("new splunk endpoint %s", splunkEndpoint)
			splunkConnectionList[splunkEndpoint] = spConfig.connectSplunk(splunkEndpoint)
		}
	}

}

func (spConfig SplunkConfig) connectSplunk(splunkEndpoint string) SplunkConnection {

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: util.GeneralFlags.SkipTLSVerification}}
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
	myConn := SplunkConnection{Client: client, Unhealthy: unhealthy, Err: err}
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

func (spConfig SplunkConfig) Output() {
	splunkFailed := metrics.GetOrRegisterCounter("splunkFailed", metrics.DefaultRegistry)

	log.Infof("Connecting to Splunk endpoints")
	spConfig.connectMultiSplunkRetry()

	batch := make([]types.DNSResult, 0, spConfig.SplunkBatchSize)
	rand.Seed(time.Now().Unix())
	ticker := time.NewTicker(spConfig.SplunkBatchDelay)

	for {
		select {
		case data := <-spConfig.outputChannel:
			if util.GeneralFlags.PacketLimit == 0 || len(batch) < util.GeneralFlags.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			healthyId := selectHealthyConnection()
			if conn, ok := splunkConnectionList[healthyId]; ok {

				if err := spConfig.splunkSendData(conn.Client, batch); err != nil {
					log.Info(err)
					log.Warnf("marking connection as unhealthy: %+v", conn)
					conn.Unhealthy += 1
					splunkConnectionList[healthyId] = conn
					splunkFailed.Inc(int64(len(batch)))
				} else {
					batch = make([]types.DNSResult, 0, spConfig.SplunkBatchSize)
				}
			} else {
				log.Warn("Splunk Connection not found")
				splunkFailed.Inc(int64(len(batch)))
			}

		}
	}
}

func (spConfig SplunkConfig) splunkSendData(client *splunk.Client, batch []types.DNSResult) error {
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

// actually run this as a goroutine
var _ = SplunkConfig{}.initializeFlags()
