package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	log "github.com/sirupsen/logrus"

	"github.com/rogpeppe/fastuuid"
	"github.com/segmentio/kafka-go"
)

var kafkaUuidGen = fastuuid.MustNewGenerator()
var kafkastats = outputStats{"Kafka", 0, 0}

func connectKafkaRetry(kafConfig kafkaConfig) *kafka.Conn {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if kafConfig.kafkaOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectKafka(kafConfig)
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

func connectKafka(kafConfig kafkaConfig) (*kafka.Conn, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", kafConfig.kafkaOutputBroker, kafConfig.kafkaOutputTopic, 0)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	return conn, err
}

func kafkaOutput(kafConfig kafkaConfig) {

	connect := connectKafkaRetry(kafConfig)
	batch := make([]types.DNSResult, 0, kafConfig.kafkaBatchSize)

	ticker := time.Tick(kafConfig.kafkaBatchDelay)
	printStatsTicker := time.Tick(kafConfig.general.printStatsDelay)

	for {
		select {
		case data := <-kafConfig.resultChannel:
			if kafConfig.general.packetLimit == 0 || len(batch) < kafConfig.general.packetLimit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := kafkaSendData(connect, batch, kafConfig); err != nil {
				log.Info(err)
				connect = connectKafkaRetry(kafConfig)
			} else {
				batch = make([]types.DNSResult, 0, kafConfig.kafkaBatchDelay)
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", kafkastats)
		}
	}
}

func kafkaSendData(connect *kafka.Conn, batch []types.DNSResult, kafConfig kafkaConfig) error {
	var msg []kafka.Message
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(kafConfig.kafkaOutputType, dnsQuery.Name) {
				kafkastats.Skipped++
				continue
			}
			kafkastats.SentToOutput++

			myUUID := kafkaUuidGen.Hex128()

			msg = append(msg, kafka.Message{
				Key:   []byte(myUUID),
				Value: []byte(fmt.Sprintf("%s\n", batch[i].String())),
			})

		}
	}
	_, err := connect.WriteMessages(msg...)
	return err

}
