package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/rogpeppe/fastuuid"
	"github.com/segmentio/kafka-go"
)

var kafkaUuidGen = fastuuid.MustNewGenerator()
var kafkastats = outputStats{"Kafka", 0, 0}

func connectKafkaRetry(exiting chan bool, kafkaBroker string, kafkaTopic string) *kafka.Conn {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if *kafkaOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectKafka(exiting, kafkaBroker, kafkaTopic)
		if err == nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-exiting:
			// When exiting, return immediately
			return nil
		case <-tick.C:
			continue
		}
	}
}

func connectKafka(exiting chan bool, kafkaBroker string, kafkaTopic string) (*kafka.Conn, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", kafkaBroker, kafkaTopic, 0)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	return conn, err
}

func kafkaOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup, kafkaBroker string, kafkaTopic string, kafkaBatchSize uint, batchDelay time.Duration, limit int) {
	wg.Add(1)
	defer wg.Done()

	connect := connectKafkaRetry(exiting, kafkaBroker, kafkaTopic)
	batch := make([]DNSResult, 0, kafkaBatchSize)

	ticker := time.Tick(batchDelay)
	printStatsTicker := time.Tick(*printStatsDelay)

	for {
		select {
		case data := <-resultChannel:
			if limit == 0 || len(batch) < limit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := kafkaSendData(connect, batch); err != nil {
				log.Info(err)
				connect = connectKafkaRetry(exiting, kafkaBroker, kafkaBroker)
			} else {
				batch = make([]DNSResult, 0, kafkaBatchSize)
			}
		case <-exiting:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", kafkastats)
		}
	}
}

func kafkaSendData(connect *kafka.Conn, batch []DNSResult) error {
	var msg []kafka.Message
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if checkIfWeSkip(*kafkaOutputType, dnsQuery.Name) {
				kafkastats.Skipped++
				continue
			}
			kafkastats.SentToOutput++

			myUUID := kafkaUuidGen.Hex128()
			fullQuery, err := json.Marshal(batch[i])
			errorHandler(err)

			msg = append(msg, kafka.Message{
				Key:   []byte(myUUID),
				Value: []byte(fmt.Sprintf("%s\n", fullQuery)),
			})

		}
	}
	_, err := connect.WriteMessages(msg...)
	return err

}
