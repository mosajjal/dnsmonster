package main

import (
	"github.com/mosajjal/dnsmonster/types"
	log "github.com/sirupsen/logrus"
)

func setupOutputs() {
	generalConfig := generalConfig{
		generalOptions.MaskSize4,
		generalOptions.MaskSize6,
		generalOptions.PacketLimit,
		generalOptions.ServerName,
		generalOptions.PrintStatsDelay,
		generalOptions.SkipTLSVerification,
	}
	log.Info("Creating the dispatch Channel")
	go dispatchOutput(resultChannel)
	types.GlobalWaitingGroup.Add(1)
	defer types.GlobalWaitingGroup.Done()

	if outputOptions.FileOutputType > 0 {
		log.Info("Creating File Output Channel")
		fConfig := fileConfig{
			fileResultChannel,
			string(outputOptions.FileOutputPath),
			outputOptions.FileOutputType,
			generalConfig,
		}
		go fileOutput(fConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
		// go fileOutput(fileResultChannel, exiting, &wg)
	}
	if outputOptions.StdoutOutputType > 0 {
		log.Info("Creating stdout Output Channel")
		stdConfig := stdoutConfig{
			stdoutResultChannel,
			outputOptions.StdoutOutputType,
			generalConfig,
		}
		go stdoutOutput(stdConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
		// go stdoutOutput(stdoutResultChannel, exiting, &wg)
	}
	if outputOptions.SyslogOutputType > 0 {
		log.Info("Creating syslog Output Channel")
		sysConfig := syslogConfig{
			syslogResultChannel,
			outputOptions.SyslogOutputEndpoint,
			outputOptions.SyslogOutputType,
			generalConfig,
		}
		go syslogOutput(sysConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
	if outputOptions.ClickhouseOutputType > 0 {
		log.Info("Creating Clickhouse Output Channel")
		chConfig := clickHouseConfig{
			clickhouseResultChannel,
			outputOptions.ClickhouseAddress,
			outputOptions.ClickhouseBatchSize,
			outputOptions.ClickhouseOutputType,
			outputOptions.ClickhouseSaveFullQuery,
			outputOptions.ClickhouseDebug,
			outputOptions.ClickhouseDelay,
			outputOptions.ClickhouseWorkers,
			outputOptions.ClickhouseWorkerChannelSize,
			generalConfig,
		}
		go clickhouseOutput(chConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
	if outputOptions.KafkaOutputType > 0 {
		log.Info("Creating Kafka Output Channel")
		kafConfig := kafkaConfig{
			kafkaResultChannel,
			outputOptions.KafkaOutputBroker,
			outputOptions.KafkaOutputTopic,
			outputOptions.KafkaOutputType,
			outputOptions.KafkaBatchSize,
			outputOptions.KafkaBatchDelay,
			generalConfig,
		}
		go kafkaOutput(kafConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
	if outputOptions.ElasticOutputType > 0 {
		log.Info("Creating Elastic Output Channel")
		esConfig := elasticConfig{
			elasticResultChannel,
			outputOptions.ElasticOutputEndpoint,
			outputOptions.ElasticOutputIndex,
			outputOptions.ElasticOutputType,
			outputOptions.ElasticBatchSize,
			outputOptions.ElasticBatchDelay,
			generalConfig,
		}
		go elasticOutput(esConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
	if outputOptions.SplunkOutputType > 0 {
		log.Info("Creating Splunk Output Channel")
		spConfig := splunkConfig{
			splunkResultChannel,
			outputOptions.SplunkOutputEndpoints,
			outputOptions.SplunkOutputToken,
			outputOptions.SplunkOutputType,
			outputOptions.SplunkOutputIndex,
			outputOptions.SplunkOutputSource,
			outputOptions.SplunkOutputSourceType,
			outputOptions.SplunkBatchSize,
			outputOptions.SplunkBatchDelay,
			generalConfig,
		}
		go splunkOutput(spConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
}
