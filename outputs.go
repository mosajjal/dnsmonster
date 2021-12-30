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
		PrintStatsDelay:     util.GeneralFlags.PrintStatsDelay,
		SkipTlsVerification: util.GeneralFlags.SkipTLSVerification,
	}
	log.Info("Creating the dispatch Channel")
	types.GlobalWaitingGroup.Add(1)
	go dispatchOutput(resultChannel)

	if util.OutputFlags.FileOutputType > 0 {
		log.Info("Creating File Output Channel")
		fConfig := types.FileConfig{
			ResultChannel:  fileResultChannel,
			FileOutputPath: string(util.OutputFlags.FileOutputPath),
			FileOutputType: util.OutputFlags.FileOutputType,
			General:        generalConfig,
		}
		types.GlobalWaitingGroup.Add(1)
		go output.FileOutput(fConfig)
		// go fileOutput(fileResultChannel, exiting, &wg)
	}
	if util.OutputFlags.StdoutOutputType > 0 {
		log.Info("Creating stdout Output Channel")
		stdConfig := types.StdoutConfig{
			ResultChannel:    stdoutResultChannel,
			StdoutOutputType: util.OutputFlags.StdoutOutputType,
			General:          generalConfig,
		}
		types.GlobalWaitingGroup.Add(1)
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
		types.GlobalWaitingGroup.Add(1)
		go output.SyslogOutput(sysConfig)
	}
	if util.OutputFlags.ClickhouseOutputType > 0 {
		log.Info("Creating Clickhouse Output Channel")
		chConfig := types.ClickHouseConfig{
			ResultChannel:               clickhouseResultChannel,
			ClickhouseAddress:           util.OutputFlags.ClickhouseAddress,
			ClickhouseBatchSize:         util.OutputFlags.ClickhouseBatchSize,
			ClickhouseOutputType:        util.OutputFlags.ClickhouseOutputType,
			ClickhouseSaveFullQuery:     util.OutputFlags.ClickhouseSaveFullQuery,
			ClickhouseDebug:             util.OutputFlags.ClickhouseDebug,
			ClickhouseDelay:             util.OutputFlags.ClickhouseDelay,
			ClickhouseWorkers:           util.OutputFlags.ClickhouseWorkers,
			ClickhouseWorkerChannelSize: util.OutputFlags.ClickhouseWorkerChannelSize,
			General:                     generalConfig,
		}
		types.GlobalWaitingGroup.Add(1)
		go output.ClickhouseOutput(chConfig)
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
		types.GlobalWaitingGroup.Add(1)
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
		types.GlobalWaitingGroup.Add(1)
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
		types.GlobalWaitingGroup.Add(1)
		go output.SplunkOutput(spConfig)
	}
}

func dispatchOutput(resultChannel chan types.DNSResult) {
	defer types.GlobalWaitingGroup.Done()
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
			if util.OutputFlags.ClickhouseOutputType > 0 {
				clickhouseResultChannel <- data
			}
			if util.OutputFlags.KafkaOutputType > 0 {
				kafkaResultChannel <- data
			}
			if util.OutputFlags.ElasticOutputType > 0 {
				elasticResultChannel <- data
			}
			if util.OutputFlags.SplunkOutputType > 0 {
				splunkResultChannel <- data
			}
		case <-types.GlobalExitChannel:
			return
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
