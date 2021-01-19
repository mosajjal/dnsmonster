package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

func connectKafkaRetry(exiting chan bool, kafkaHost string) *kafka.Conn {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if *kafkaOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectKafka(exiting, kafkaHost)
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

func connectKafka(exiting chan bool, kafkaHost string) (*kafka.Conn, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", "topic", 0)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return conn, err
}

func kafkaOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup, clickhouseHost string, kafkaBatchSize uint, batchDelay time.Duration, limit int, server string) {
	wg.Add(1)
	defer wg.Done()

	connect := connectKafkaRetry(exiting, clickhouseHost)
	batch := make([]DNSResult, 0, kafkaBatchSize)

	ticker := time.Tick(batchDelay)
	for {
		select {
		case data := <-resultChannel:
			if limit == 0 || len(batch) < limit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := kafkaSendData(connect, batch); err != nil {
				log.Println(err)
				connect = connectKafkaRetry(exiting, clickhouseHost)
			} else {
				batch = make([]DNSResult, 0, kafkaBatchSize)
			}
		case <-exiting:
			return
		}
	}
}

func kafkaSendData(connect *kafka.Conn, batch []DNSResult) error {
	return nil
	// TODO
}
