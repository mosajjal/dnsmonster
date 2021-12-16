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
	go dispatchOutput(resultChannel)
	types.GlobalWaitingGroup.Add(1)
	defer types.GlobalWaitingGroup.Done()

	if util.OutputFlags.FileOutputType > 0 {
		log.Info("Creating File Output Channel")
		fConfig := types.FileConfig{
			ResultChannel:  fileResultChannel,
			FileOutputPath: string(util.OutputFlags.FileOutputPath),
			FileOutputType: util.OutputFlags.FileOutputType,
			General:        generalConfig,
		}
		go output.FileOutput(fConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
		// go fileOutput(fileResultChannel, exiting, &wg)
	}
	if util.OutputFlags.StdoutOutputType > 0 {
		log.Info("Creating stdout Output Channel")
		stdConfig := types.StdoutConfig{
			ResultChannel:    stdoutResultChannel,
			StdoutOutputType: util.OutputFlags.StdoutOutputType,
			General:          generalConfig,
		}
		go output.StdoutOutput(stdConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
		// go stdoutOutput(stdoutResultChannel, exiting, &wg)
	}
	if util.OutputFlags.SyslogOutputType > 0 {
		log.Info("Creating syslog Output Channel")
		sysConfig := types.SyslogConfig{
			syslogResultChannel,
			util.OutputFlags.SyslogOutputEndpoint,
			util.OutputFlags.SyslogOutputType,
			generalConfig,
		}
		go output.SyslogOutput(sysConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
	if util.OutputFlags.ClickhouseOutputType > 0 {
		log.Info("Creating Clickhouse Output Channel")
		chConfig := types.ClickHouseConfig{
			clickhouseResultChannel,
			util.OutputFlags.ClickhouseAddress,
			util.OutputFlags.ClickhouseBatchSize,
			util.OutputFlags.ClickhouseOutputType,
			util.OutputFlags.ClickhouseSaveFullQuery,
			util.OutputFlags.ClickhouseDebug,
			util.OutputFlags.ClickhouseDelay,
			util.OutputFlags.ClickhouseWorkers,
			util.OutputFlags.ClickhouseWorkerChannelSize,
			generalConfig,
		}
		go output.ClickhouseOutput(chConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
	if util.OutputFlags.KafkaOutputType > 0 {
		log.Info("Creating Kafka Output Channel")
		kafConfig := types.KafkaConfig{
			kafkaResultChannel,
			util.OutputFlags.KafkaOutputBroker,
			util.OutputFlags.KafkaOutputTopic,
			util.OutputFlags.KafkaOutputType,
			util.OutputFlags.KafkaBatchSize,
			util.OutputFlags.KafkaBatchDelay,
			generalConfig,
		}
		go output.KafkaOutput(kafConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
	if util.OutputFlags.ElasticOutputType > 0 {
		log.Info("Creating Elastic Output Channel")
		esConfig := types.ElasticConfig{
			elasticResultChannel,
			util.OutputFlags.ElasticOutputEndpoint,
			util.OutputFlags.ElasticOutputIndex,
			util.OutputFlags.ElasticOutputType,
			util.OutputFlags.ElasticBatchSize,
			util.OutputFlags.ElasticBatchDelay,
			generalConfig,
		}
		go output.ElasticOutput(esConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
	if util.OutputFlags.SplunkOutputType > 0 {
		log.Info("Creating Splunk Output Channel")
		spConfig := types.SplunkConfig{
			splunkResultChannel,
			util.OutputFlags.SplunkOutputEndpoints,
			util.OutputFlags.SplunkOutputToken,
			util.OutputFlags.SplunkOutputType,
			util.OutputFlags.SplunkOutputIndex,
			util.OutputFlags.SplunkOutputSource,
			util.OutputFlags.SplunkOutputSourceType,
			util.OutputFlags.SplunkBatchSize,
			util.OutputFlags.SplunkBatchDelay,
			generalConfig,
		}
		go output.SplunkOutput(spConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
}

func dispatchOutput(resultChannel chan types.DNSResult) {

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
