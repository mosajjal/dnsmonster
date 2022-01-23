package output

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/rogpeppe/fastuuid"
	"github.com/segmentio/kafka-go"
)

type KafkaConfig struct {
	KafkaOutputType   uint          `long:"kafkaOutputType"             env:"DNSMONSTER_KAFKAOUTPUTTYPE"             default:"0"                                                       description:"What should be written to kafka. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"         choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	KafkaOutputBroker string        `long:"kafkaOutputBroker"           env:"DNSMONSTER_KAFKAOUTPUTBROKER"           default:""                                                        description:"kafka broker address, example: 127.0.0.1:9092. Used if kafkaOutputType is not none"`
	KafkaOutputTopic  string        `long:"kafkaOutputTopic"            env:"DNSMONSTER_KAFKAOUTPUTTOPIC"            default:"dnsmonster"                                              description:"Kafka topic for logging"`
	KafkaBatchSize    uint          `long:"kafkaBatchSize"              env:"DNSMONSTER_KAFKABATCHSIZE"              default:"1000"                                                    description:"Minimun capacity of the cache array used to send data to Kafka"`
	KafkaBatchDelay   time.Duration `long:"kafkaBatchDelay"             env:"DNSMONSTER_KAFKABATCHDELAY"             default:"1s"                                                      description:"Interval between sending results to Kafka if Batch size is not filled"`
	outputChannel     chan types.DNSResult
	closeChannel      chan bool
}

func (config KafkaConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("kafka_output", "Kafka Output", &config)

	config.outputChannel = make(chan types.DNSResult, util.GeneralFlags.ResultChannelSize)

	types.GlobalDispatchList = append(types.GlobalDispatchList, &config)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (config KafkaConfig) Initialize() error {
	if config.KafkaOutputType > 0 && config.KafkaOutputType < 5 {
		log.Info("Creating Kafka Output Channel")
		go config.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (config KafkaConfig) Close() {
	//todo: implement this
	<-config.closeChannel
}

func (config KafkaConfig) OutputChannel() chan types.DNSResult {
	return config.outputChannel
}

var kafkaUuidGen = fastuuid.MustNewGenerator()

func (kafConfig KafkaConfig) connectKafkaRetry() *kafka.Conn {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if kafConfig.KafkaOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := kafConfig.connectKafka()
		if err == nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue

	}
}

func (kafConfig KafkaConfig) connectKafka() (*kafka.Conn, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", kafConfig.KafkaOutputBroker, kafConfig.KafkaOutputTopic, 0)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	return conn, err
}

func (kafConfig KafkaConfig) Output() {
	connect := kafConfig.connectKafkaRetry()
	batch := make([]types.DNSResult, 0, kafConfig.KafkaBatchSize)

	ticker := time.NewTicker(kafConfig.KafkaBatchDelay)

	for {
		select {
		case data := <-kafConfig.outputChannel:
			if util.GeneralFlags.PacketLimit == 0 || len(batch) < util.GeneralFlags.PacketLimit {
				batch = append(batch, data)
			}
		case <-ticker.C:
			if err := kafConfig.kafkaSendData(connect, batch); err != nil {
				log.Info(err)
				connect = kafConfig.connectKafkaRetry()
			} else {
				batch = make([]types.DNSResult, 0, kafConfig.KafkaBatchSize)
			}

		}
	}
}

func (kafConfig KafkaConfig) kafkaSendData(connect *kafka.Conn, batch []types.DNSResult) error {
	kafkaSentToOutput := metrics.GetOrRegisterCounter("kafkaSentToOutput", metrics.DefaultRegistry)
	kafkaSkipped := metrics.GetOrRegisterCounter("stdoutSkipped", metrics.DefaultRegistry)
	var msg []kafka.Message
	for i := range batch {
		for _, dnsQuery := range batch[i].DNS.Question {
			if util.CheckIfWeSkip(kafConfig.KafkaOutputType, dnsQuery.Name) {
				kafkaSkipped.Inc(1)
				continue
			}
			kafkaSentToOutput.Inc(1)

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

// actually run this as a goroutine
var _ = KafkaConfig{}.initializeFlags()
