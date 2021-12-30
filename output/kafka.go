package output

import (
	"context"
	"fmt"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"

	"github.com/rogpeppe/fastuuid"
	"github.com/segmentio/kafka-go"
)

var kafkaUuidGen = fastuuid.MustNewGenerator()
var kafkastats = types.OutputStats{Name: "Kafka", SentToOutput: 0, Skipped: 0}

func connectKafkaRetry(kafConfig types.KafkaConfig) *kafka.Conn {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if kafConfig.KafkaOutputType == 0 {
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

func connectKafka(kafConfig types.KafkaConfig) (*kafka.Conn, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", kafConfig.KafkaOutputBroker, kafConfig.KafkaOutputTopic, 0)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	return conn, err
}

func KafkaOutput(kafConfig types.KafkaConfig) {
	defer types.GlobalWaitingGroup.Done()
	connect := connectKafkaRetry(kafConfig)
	batch := make([]types.DNSResult, 0, kafConfig.KafkaBatchSize)

	ticker := time.NewTicker(kafConfig.KafkaBatchDelay)
	printStatsTicker := time.NewTicker(kafConfig.General.PrintStatsDelay)

	for {
		select {
		case data := <-kafConfig.ResultChannel:
			if kafConfig.General.PacketLimit == 0 || len(batch) < kafConfig.General.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			if err := kafkaSendData(connect, batch, kafConfig); err != nil {
				log.Info(err)
				connect = connectKafkaRetry(kafConfig)
			} else {
				batch = make([]types.DNSResult, 0, kafConfig.KafkaBatchDelay)
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker.C:
			log.Infof("output: %+v", kafkastats)
		}
	}
}

func kafkaSendData(connect *kafka.Conn, batch []types.DNSResult, kafConfig types.KafkaConfig) error {
	var msg []kafka.Message
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if util.CheckIfWeSkip(kafConfig.KafkaOutputType, dnsQuery.Name) {
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
