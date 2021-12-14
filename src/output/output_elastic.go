package output

import (
	"context"
	"fmt"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic"
)

// var elasticUuidGen = fastuuid.MustNewGenerator()
var elasticstats = types.OutputStats{"elastic", 0, 0}
var ctx = context.Background()

func connectelasticRetry(esConfig types.ElasticConfig) *elastic.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if esConfig.ElasticOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectelastic(esConfig)
		if err == nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-types.GlobalExitChannel:
			// When exiting, return immediately
			return nil
		case <-tick.C:
			continue
		}
	}
}

func connectelastic(esConfig types.ElasticConfig) (*elastic.Client, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(esConfig.ElasticOutputEndpoint),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10*time.Second),
		// elastic.SetRetrier(connectelasticRetry(exiting, elasticEndpoint)),
		elastic.SetGzip(true),
		elastic.SetErrorLog(log.New()),
	)
	util.ErrorHandler(err)

	//TODO: are we retrying without exiting out of dnsmonster?
	// Ping the Elasticsearch server to get e.g. the version number
	info, code, err := client.Ping(esConfig.ElasticOutputEndpoint).Do(ctx)
	util.ErrorHandler(err)
	fmt.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	return client, err
}

func ElasticOutput(esConfig types.ElasticConfig) {
	client := connectelasticRetry(esConfig)
	batch := make([]types.DNSResult, 0, esConfig.ElasticBatchSize)

	ticker := time.Tick(esConfig.ElasticBatchDelay)
	printStatsTicker := time.Tick(esConfig.General.PrintStatsDelay)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(esConfig.ElasticOutputIndex).Do(ctx)
	util.ErrorHandler(err)

	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(esConfig.ElasticOutputIndex).Do(ctx)
		util.ErrorHandler(err)

		if !createIndex.Acknowledged {
			log.Panicln("Could not create the Elastic index.. Exiting")
		}
	}

	for {
		select {
		case data := <-esConfig.ResultChannel:
			if esConfig.General.PacketLimit == 0 || len(batch) < esConfig.General.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := elasticSendData(client, batch, esConfig); err != nil {
				log.Info(err)
				client = connectelasticRetry(esConfig)
			} else {
				batch = make([]types.DNSResult, 0, esConfig.ElasticBatchSize)
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", elasticstats)
		}
	}
}

func elasticSendData(client *elastic.Client, batch []types.DNSResult, esConfig types.ElasticConfig) error {
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if util.CheckIfWeSkip(esConfig.ElasticOutputType, dnsQuery.Name) {
				elasticstats.Skipped++
				continue
			}
			elasticstats.SentToOutput++

			// batch[i].UUID = elasticUuidGen.Hex128()

			_, err := client.Index().
				Index(esConfig.ElasticOutputIndex).
				Type("_doc").
				BodyString(string(batch[i].String())).
				Do(ctx)

			util.ErrorHandler(err)
		}
	}
	_, err := client.Flush().Index(esConfig.ElasticOutputIndex).Do(ctx)
	return err

}
