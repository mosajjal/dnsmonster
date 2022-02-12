package output

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"
)

type ElasticConfig struct {
	ElasticOutputType     uint          `long:"elasticOutputType"           env:"DNSMONSTER_ELASTICOUTPUTTYPE"           default:"0"                                                       description:"What should be written to elastic. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"       choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ElasticOutputEndpoint string        `long:"elasticOutputEndpoint"       env:"DNSMONSTER_ELASTICOUTPUTENDPOINT"       default:""                                                        description:"elastic endpoint address, example: http://127.0.0.1:9200. Used if elasticOutputType is not none"`
	ElasticOutputIndex    string        `long:"elasticOutputIndex"          env:"DNSMONSTER_ELASTICOUTPUTINDEX"          default:"default"                                                 description:"elastic index"`
	ElasticBatchSize      uint          `long:"elasticBatchSize"            env:"DNSMONSTER_ELASTICBATCHSIZE"            default:"1000"                                                    description:"Send data to Elastic in batch sizes"`
	ElasticBatchDelay     time.Duration `long:"elasticBatchDelay"           env:"DNSMONSTER_ELASTICBATCHDELAY"           default:"1s"                                                      description:"Interval between sending results to Elastic if Batch size is not filled"`
	outputChannel         chan util.DNSResult
	closeChannel          chan bool
}

func (config ElasticConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("elastic_output", "Elastic Output", &config)

	config.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)

	util.GlobalDispatchList = append(util.GlobalDispatchList, &config)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (config ElasticConfig) Initialize() error {
	if config.ElasticOutputType > 0 && config.ElasticOutputType < 5 {
		log.Info("Creating Elastic Output Channel")
		go config.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (config ElasticConfig) Close() {
	//todo: implement this
	<-config.closeChannel
}

func (config ElasticConfig) OutputChannel() chan util.DNSResult {
	return config.outputChannel
}

// var elasticUuidGen = fastuuid.MustNewGenerator()
var ctx = context.Background()

func (esConfig ElasticConfig) connectelasticRetry() *elastic.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if esConfig.ElasticOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := esConfig.connectelastic()
		if err == nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue

	}
}

func (esConfig ElasticConfig) connectelastic() (*elastic.Client, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(esConfig.ElasticOutputEndpoint),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10*time.Second),
		// elastic.SetRetrier(connectelasticRetry(exiting, elasticEndpoint)),
		elastic.SetGzip(true),
		elastic.SetErrorLog(log.New()),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Ping the Elasticsearch server to get e.g. the version number
	info, code, err := client.Ping(esConfig.ElasticOutputEndpoint).Do(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	return client, err
}

func (esConfig ElasticConfig) Output() {
	client := esConfig.connectelasticRetry()
	batch := make([]util.DNSResult, 0, esConfig.ElasticBatchSize)

	ticker := time.NewTicker(esConfig.ElasticBatchDelay)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(esConfig.ElasticOutputIndex).Do(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(esConfig.ElasticOutputIndex).Do(ctx)
		if err != nil {
			log.Fatal(err)
		}

		if !createIndex.Acknowledged {
			log.Panicln("Could not create the Elastic index.. Exiting")
		}
	}

	for {
		select {
		case data := <-esConfig.outputChannel:
			if util.GeneralFlags.PacketLimit == 0 || len(batch) < util.GeneralFlags.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			if err := esConfig.elasticSendData(client, batch); err != nil {
				log.Info(err)
				client = esConfig.connectelasticRetry()
			} else {
				batch = make([]util.DNSResult, 0, esConfig.ElasticBatchSize)
			}

		}
	}
}

func (esConfig ElasticConfig) elasticSendData(client *elastic.Client, batch []util.DNSResult) error {
	elasticSentToOutput := metrics.GetOrRegisterCounter("elasticSentToOutput", metrics.DefaultRegistry)
	elasticSkipped := metrics.GetOrRegisterCounter("elasticSkipped", metrics.DefaultRegistry)

	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if util.CheckIfWeSkip(esConfig.ElasticOutputType, dnsQuery.Name) {
				elasticSkipped.Inc(1)
				continue
			}
			elasticSentToOutput.Inc(1)

			// batch[i].UUID = elasticUuidGen.Hex128()

			_, err := client.Index().
				Index(esConfig.ElasticOutputIndex).
				Type("_doc").
				BodyString(string(batch[i].GetJson())).
				Do(ctx)

			if err != nil {
				log.Fatal(err)
			}
		}
	}
	_, err := client.Flush().Index(esConfig.ElasticOutputIndex).Do(ctx)
	return err

}

// This will allow an instance to be spawned at import time
var _ = ElasticConfig{}.initializeFlags()
