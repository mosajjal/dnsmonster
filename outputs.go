package main

import (
	"time"

	"github.com/mosajjal/dnsmonster/output"
	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

func setupOutputs() {
	generalConfig := types.GeneralConfig{
		MaskSize4:           util.GeneralFlags.MaskSize4,
		MaskSize6:           util.GeneralFlags.MaskSize6,
		PacketLimit:         util.GeneralFlags.PacketLimit,
		ServerName:          util.GeneralFlags.ServerName,
		SkipTlsVerification: util.GeneralFlags.SkipTLSVerification,
	}
	log.Info("Creating the dispatch Channel")

	go dispatchOutput(resultChannel)

	if util.OutputFlags.FileOutputType > 0 {
		log.Info("Creating File Output Channel")
		fConfig := types.FileConfig{
			ResultChannel:    fileResultChannel,
			FileOutputPath:   string(util.OutputFlags.FileOutputPath),
			FileOutputType:   util.OutputFlags.FileOutputType,
			FileOutputFormat: util.OutputFlags.FileOutputFormat,
			General:          generalConfig,
		}

		go output.FileOutput(fConfig)
		// go fileOutput(fileResultChannel, exiting, &wg)
	}
	if util.OutputFlags.StdoutOutputType > 0 {
		log.Info("Creating stdout Output Channel")
		stdConfig := types.StdoutConfig{
			ResultChannel:      stdoutResultChannel,
			StdoutOutputType:   util.OutputFlags.StdoutOutputType,
			StdoutOutputFormat: util.OutputFlags.StdoutOutputFormat,
			General:            generalConfig,
		}

		go output.StdoutOutput(stdConfig)
		// go stdoutOutput(stdoutResultChannel, exiting, &wg)
	}
	if util.OutputFlags.SyslogOutputType > 0 {
		log.Info("Creating syslog Output Channel")
		sysConfig := types.SyslogConfig{
			ResultChannel:        syslogResultChannel,
			SyslogOutputEndpoint: util.OutputFlags.SyslogOutputEndpoint,
			SyslogOutputType:     util.OutputFlags.SyslogOutputType,
			General:              generalConfig,
		}
		go output.SyslogOutput(sysConfig)
	}
	if util.OutputFlags.KafkaOutputType > 0 {
		log.Info("Creating Kafka Output Channel")
		kafConfig := types.KafkaConfig{
			ResultChannel:     kafkaResultChannel,
			KafkaOutputBroker: util.OutputFlags.KafkaOutputBroker,
			KafkaOutputTopic:  util.OutputFlags.KafkaOutputTopic,
			KafkaOutputType:   util.OutputFlags.KafkaOutputType,
			KafkaBatchSize:    util.OutputFlags.KafkaBatchSize,
			KafkaBatchDelay:   util.OutputFlags.KafkaBatchDelay,
			General:           generalConfig,
		}
		go output.KafkaOutput(kafConfig)
	}
	if util.OutputFlags.ElasticOutputType > 0 {
		log.Info("Creating Elastic Output Channel")
		esConfig := types.ElasticConfig{
			ResultChannel:         elasticResultChannel,
			ElasticOutputEndpoint: util.OutputFlags.ElasticOutputEndpoint,
			ElasticOutputIndex:    util.OutputFlags.ElasticOutputIndex,
			ElasticOutputType:     util.OutputFlags.ElasticOutputType,
			ElasticBatchSize:      util.OutputFlags.ElasticBatchSize,
			ElasticBatchDelay:     util.OutputFlags.ElasticBatchDelay,
			General:               generalConfig,
		}

		go output.ElasticOutput(esConfig)
	}
	if util.OutputFlags.SplunkOutputType > 0 {
		log.Info("Creating Splunk Output Channel")
		spConfig := types.SplunkConfig{
			ResultChannel:          splunkResultChannel,
			SplunkOutputEndpoints:  util.OutputFlags.SplunkOutputEndpoints,
			SplunkOutputToken:      util.OutputFlags.SplunkOutputToken,
			SplunkOutputType:       util.OutputFlags.SplunkOutputType,
			SplunkOutputIndex:      util.OutputFlags.SplunkOutputIndex,
			SplunkOutputSource:     util.OutputFlags.SplunkOutputSource,
			SplunkOutputSourceType: util.OutputFlags.SplunkOutputSourceType,
			SplunkBatchSize:        util.OutputFlags.SplunkBatchSize,
			SplunkBatchDelay:       util.OutputFlags.SplunkBatchDelay,
			General:                generalConfig,
		}

		go output.SplunkOutput(spConfig)
	}

}

func RemoveIndex(s []types.GenericOutput, index int) []types.GenericOutput {
	return append(s[:index], s[index+1:]...)
}

func dispatchOutput(resultChannel chan types.DNSResult) {

	// the new simplified output method
	for i, o := range types.GlobalDispatchList {
		err := o.Initialize()
		if err != nil {
			// the output does not exist, time to remove the item from our globaldispatcher
			log.Warnf("here") //todo:remove
			types.GlobalDispatchList = RemoveIndex(types.GlobalDispatchList, i)
		}
	}

	// Set up various tickers for different tasks
	skipDomainsFileTicker := time.NewTicker(util.GeneralFlags.SkipDomainsRefreshInterval)
	skipDomainsFileTickerChan := skipDomainsFileTicker.C
	if util.GeneralFlags.SkipDomainsFile == "" {
		skipDomainsFileTicker.Stop()
	}

	allowDomainsFileTicker := time.NewTicker(util.GeneralFlags.AllowDomainsRefreshInterval)
	allowDomainsFileTickerChan := allowDomainsFileTicker.C
	if util.GeneralFlags.AllowDomainsFile == "" {
		log.Infof("skipping allowDomains refresh since it's empty")
		allowDomainsFileTicker.Stop()
	} else {
		log.Infof("allowDomains refresh interval is %s", util.GeneralFlags.AllowDomainsRefreshInterval)
	}

	for {
		select {
		case data := <-resultChannel:
			if util.OutputFlags.StdoutOutputType > 0 {
				stdoutResultChannel <- data
			}
			if util.OutputFlags.FileOutputType > 0 {
				fileResultChannel <- data
			}
			if util.OutputFlags.SyslogOutputType > 0 {
				syslogResultChannel <- data
			}
			// if util.OutputFlags.ClickhouseOutputType > 0 {
			// 	clickhouseResultChannel <- data
			// }
			if util.OutputFlags.KafkaOutputType > 0 {
				kafkaResultChannel <- data
			}
			if util.OutputFlags.ElasticOutputType > 0 {
				elasticResultChannel <- data
			}
			if util.OutputFlags.SplunkOutputType > 0 {
				splunkResultChannel <- data
			}
			// new simplified output method. only works with Sentinel
			for _, o := range types.GlobalDispatchList {
				// todo: this blocks on type0 outputs
				o.OutputChannel() <- data
			}

		case <-skipDomainsFileTickerChan:
			log.Infof("reached skipDomains tick")
			if util.SkipDomainMapBool {
				util.SkipDomainMap = util.LoadDomainsToMap(util.GeneralFlags.SkipDomainsFile)
			} else {
				util.SkipDomainList = util.LoadDomainsToList(util.GeneralFlags.SkipDomainsFile)
			}
		case <-allowDomainsFileTickerChan:
			log.Infof("reached allowDomains tick")
			if util.AllowDomainMapBool {
				util.AllowDomainMap = util.LoadDomainsToMap(util.GeneralFlags.AllowDomainsFile)
			} else {
				util.AllowDomainList = util.LoadDomainsToList(util.GeneralFlags.AllowDomainsFile)
			}
		}
	}
}
