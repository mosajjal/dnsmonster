package output

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"
)

type elasticConfig struct {
	ElasticOutputType     uint          `long:"elasticoutputtype"           ini-name:"elasticoutputtype"           env:"DNSMONSTER_ELASTICOUTPUTTYPE"           default:"0"                                                       description:"What should be written to elastic. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"       choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ElasticOutputEndpoint string        `long:"elasticoutputendpoint"       ini-name:"elasticoutputendpoint"       env:"DNSMONSTER_ELASTICOUTPUTENDPOINT"       default:""                                                        description:"elastic endpoint address, example: http://127.0.0.1:9200. Used if elasticOutputType is not none"`
	ElasticOutputIndex    string        `long:"elasticoutputindex"          ini-name:"elasticoutputindex"          env:"DNSMONSTER_ELASTICOUTPUTINDEX"          default:"default"                                                 description:"elastic index"`
	ElasticBatchSize      uint          `long:"elasticbatchsize"            ini-name:"elasticbatchsize"            env:"DNSMONSTER_ELASTICBATCHSIZE"            default:"1000"                                                    description:"Send data to Elastic in batch sizes"`
	ElasticBatchDelay     time.Duration `long:"elasticbatchdelay"           ini-name:"elasticbatchdelay"           env:"DNSMONSTER_ELASTICBATCHDELAY"           default:"1s"                                                      description:"Interval between sending results to Elastic if Batch size is not filled"`
	outputChannel         chan util.DNSResult
	outputMarshaller      util.OutputMarshaller
	closeChannel          chan bool
}

func init() {
	c := elasticConfig{}
	if _, err := util.GlobalParser.AddGroup("elastic_output", "Elastic Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (esConfig elasticConfig) Initialize(ctx context.Context) error {
	var err error
	esConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if esConfig.ElasticOutputType > 0 && esConfig.ElasticOutputType < 5 {
		log.Info("Creating Elastic Output Channel")
		go esConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (esConfig elasticConfig) Close() {
	// todo: implement this
	<-esConfig.closeChannel
}

func (esConfig elasticConfig) OutputChannel() chan util.DNSResult {
	return esConfig.outputChannel
}

// var elasticUuidGen = fastuuid.MustNewGenerator()
// var ctx = context.Background()

func (esConfig elasticConfig) connectelasticRetry(ctx context.Context) *elastic.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if esConfig.ElasticOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := esConfig.connectelastic(ctx)
		if err == nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue

	}
}

func (esConfig elasticConfig) connectelastic(ctx context.Context) (*elastic.Client, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: util.GeneralFlags.SkipTLSVerification},
	}
	httpClient := &http.Client{Transport: tr}

	client, err := elastic.NewClient(
		elastic.SetHttpClient(httpClient),
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

func (esConfig elasticConfig) Output(ctx context.Context) {
	client := esConfig.connectelasticRetry(ctx)
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
			log.Fatalln("Could not create the Elastic index.. Exiting")
		}
	}

	for {
		select {
		case data := <-esConfig.outputChannel:
			if util.GeneralFlags.PacketLimit == 0 || len(batch) < util.GeneralFlags.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			if err := esConfig.elasticSendData(ctx, client, batch); err != nil {
				log.Info(err)
				client = esConfig.connectelasticRetry(ctx)
			} else {
				batch = make([]util.DNSResult, 0, esConfig.ElasticBatchSize)
			}

		}
	}
}

func (esConfig elasticConfig) elasticSendData(ctx context.Context, client *elastic.Client, batch []util.DNSResult) error {
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
				BodyString(esConfig.outputMarshaller.Marshal(batch[i])).
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
// var _ = elasticConfig{}.initializeFlags()
