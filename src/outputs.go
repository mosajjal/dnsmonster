package main

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

func setupOutputs(wg sync.WaitGroup, exiting chan bool) {
	generalConfig := generalConfig{
		exiting,
		&wg,
		generalOptions.MaskSize,
		generalOptions.PacketLimit,
		generalOptions.SaveFullQuery,
		generalOptions.ServerName,
		generalOptions.PrintStatsDelay,
		generalOptions.SkipTLSVerification,
	}
	log.Info("Creating the dispatch Channel")
	go dispatchOutput(resultChannel, exiting, &wg)

	if outputOptions.FileOutputType > 0 {
		log.Info("Creating File Output Channel")
		fConfig := fileConfig{
			fileResultChannel,
			string(outputOptions.FileOutputPath),
			outputOptions.FileOutputType,
			generalConfig,
		}
		go fileOutput(fConfig)
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
	}
	if outputOptions.ClickhouseOutputType > 0 {
		log.Info("Creating Clickhouse Output Channel")
		chConfig := clickHouseConfig{
			clickhouseResultChannel,
			outputOptions.ClickhouseAddress,
			outputOptions.ClickhouseBatchSize,
			outputOptions.ClickhouseOutputType,
			outputOptions.ClickhouseDebug,
			outputOptions.ClickhouseDelay,
			generalConfig,
		}
		go clickhouseOutput(chConfig)
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
	}
}
