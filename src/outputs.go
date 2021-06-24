package main

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

func setupOutputs(wg sync.WaitGroup, exiting chan bool) {
	generalConfig := generalConfig{
		exiting,
		&wg,
		*maskSize,
		*packetLimit,
		*saveFullQuery,
		*serverName,
		*printStatsDelay,
		*skipTlsVerification,
	}
	log.Info("Creating the dispatch Channel")
	go dispatchOutput(resultChannel, exiting, &wg)

	if *fileOutputType > 0 {
		log.Info("Creating File Output Channel")
		fConfig := fileConfig{
			fileResultChannel,
			*fileOutputPath,
			*fileOutputType,
			generalConfig,
		}
		go fileOutput(fConfig)
		// go fileOutput(fileResultChannel, exiting, &wg)
	}
	if *stdoutOutputType > 0 {
		log.Info("Creating stdout Output Channel")
		stdConfig := stdoutConfig{
			stdoutResultChannel,
			*stdoutOutputType,
			generalConfig,
		}
		go stdoutOutput(stdConfig)
		// go stdoutOutput(stdoutResultChannel, exiting, &wg)
	}
	if *syslogOutputType > 0 {
		log.Info("Creating syslog Output Channel")
		sysConfig := syslogConfig{
			syslogResultChannel,
			*syslogOutputEndpoint,
			*syslogOutputType,
			generalConfig,
		}
		go syslogOutput(sysConfig)
	}
	if *clickhouseOutputType > 0 {
		log.Info("Creating Clickhouse Output Channel")
		chConfig := clickHouseConfig{
			clickhouseResultChannel,
			*clickhouseAddress,
			*clickhouseBatchSize,
			*clickhouseOutputType,
			*clickhouseDebug,
			*clickhouseDelay,
			generalConfig,
		}
		go clickhouseOutput(chConfig)
	}
	if *kafkaOutputType > 0 {
		log.Info("Creating Kafka Output Channel")
		kafConfig := kafkaConfig{
			kafkaResultChannel,
			*kafkaOutputBroker,
			*kafkaOutputTopic,
			*kafkaOutputType,
			*kafkaBatchSize,
			*kafkaBatchDelay,
			generalConfig,
		}
		go kafkaOutput(kafConfig)
	}
	if *elasticOutputType > 0 {
		log.Info("Creating Elastic Output Channel")
		esConfig := elasticConfig{
			elasticResultChannel,
			*elasticOutputEndpoint,
			*elasticOutputIndex,
			*elasticOutputType,
			*elasticBatchSize,
			*elasticBatchDelay,
			generalConfig,
		}
		go elasticOutput(esConfig)
	}
	if *splunkOutputType > 0 {
		log.Info("Creating Splunk Output Channel")
		spConfig := splunkConfig{
			splunkResultChannel,
			splunkOutputEndpoints,
			*splunkOutputToken,
			*splunkOutputType,
			*splunkOutputIndex,
			*splunkOutputSource,
			*splunkOutputSourceType,
			*splunkBatchSize,
			*splunkBatchDelay,
			generalConfig,
		}
		go splunkOutput(spConfig)
	}
}
