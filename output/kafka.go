package output

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"time"

	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/rogpeppe/fastuuid"
	"github.com/segmentio/kafka-go"
)

type KafkaConfig struct {
	KafkaOutputType         uint          `long:"kafkaOutputType"             env:"DNSMONSTER_KAFKAOUTPUTTYPE"             default:"0"                                                       description:"What should be written to kafka. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"         choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	KafkaOutputBroker       []string      `long:"kafkaOutputBroker"           env:"DNSMONSTER_KAFKAOUTPUTBROKER"           default:""                                                        description:"kafka broker address(es), example: 127.0.0.1:9092. Used if kafkaOutputType is not none"`
	KafkaOutputTopic        string        `long:"kafkaOutputTopic"            env:"DNSMONSTER_KAFKAOUTPUTTOPIC"            default:"dnsmonster"                                              description:"Kafka topic for logging"`
	KafkaBatchSize          uint          `long:"kafkaBatchSize"              env:"DNSMONSTER_KAFKABATCHSIZE"              default:"1000"                                                    description:"Minimun capacity of the cache array used to send data to Kafka"`
	KafkaTimeout            uint          `long:"kafkaTimeout"                env:"DNSMONSTER_KAFKATIMEOUT"                default:"3"                                                       description:"Kafka connection timeout in seconds"`
	KafkaBatchDelay         time.Duration `long:"kafkaBatchDelay"             env:"DNSMONSTER_KAFKABATCHDELAY"             default:"1s"                                                      description:"Interval between sending results to Kafka if Batch size is not filled"`
	KafkaCompress           bool          `long:"kafkaCompress"               env:"DNSMONSTER_KAFKACOMPRESS"                                                                                 description:"Compress Kafka connection"`
	KafkaSecure             bool          `long:"kafkaSecure"                 env:"DNSMONSTER_KAFKASECURE"                                                                                   description:"Use TLS for kafka connection"`
	KafkaCACertificatePath  string        `long:"kafkaCACertificatePath"      env:"DNSMONSTER_KAFKACACERTIFICATEPATH"      default:""                                                        description:"Path of CA certificate that signs Kafka broker certificate"`
	KafkaTLSCertificatePath string        `long:"kafkaTLSCertificatePath"     env:"DNSMONSTER_KAFKATLSCERTIFICATEPATH"     default:""                                                        description:"Path of TLS certificate to present to broker"`
	KafkaTLSKeyPath         string        `long:"kafkaTLSKeyPath"             env:"DNSMONSTER_KAFKATLSKEYPATH"             default:""                                                        description:"Path of TLS certificate key"`
	outputChannel           chan util.DNSResult
	closeChannel            chan bool
}

func (kafConfig KafkaConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("kafka_output", "Kafka Output", &kafConfig)

	kafConfig.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)

	util.GlobalDispatchList = append(util.GlobalDispatchList, &kafConfig)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (kafConfig KafkaConfig) Initialize() error {
	if kafConfig.KafkaOutputType > 0 && kafConfig.KafkaOutputType < 5 {
		log.Info("Creating Kafka Output Channel")
		go kafConfig.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (kafConfig KafkaConfig) Close() {
	close(kafConfig.closeChannel)
}

func (kafConfig KafkaConfig) OutputChannel() chan util.DNSResult {
	return kafConfig.outputChannel
}

func (kafConfig KafkaConfig) getWriter() *kafka.Writer {
	transport := &kafka.Transport{
		Dial: (&net.Dialer{
			Timeout:   time.Duration(kafConfig.KafkaTimeout) * time.Second,
			DualStack: true,
		}).DialContext,
	}

	if kafConfig.KafkaSecure {
		// setup TLS
		tlsConfig := &tls.Config{}

		if kafConfig.KafkaCACertificatePath != "" {
			caCert, err := ioutil.ReadFile(kafConfig.KafkaCACertificatePath)
			if err != nil {
				log.Fatalf("Could not read kafka CA certificate: %v", err)
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			tlsConfig.RootCAs = caCertPool
		}

		if kafConfig.KafkaTLSCertificatePath != "" && kafConfig.KafkaTLSKeyPath != "" {
			clientCert, err := tls.LoadX509KeyPair(kafConfig.KafkaTLSCertificatePath, kafConfig.KafkaTLSKeyPath)
			if err != nil {
				log.Fatalf("Could not read kafka client certificate: %v", err)
			}

			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}

		transport.TLS = tlsConfig
	}

	kWriter := &kafka.Writer{
		Addr:         kafka.TCP(kafConfig.KafkaOutputBroker...),
		Async:        true,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    int(kafConfig.KafkaBatchSize),
		BatchTimeout: kafConfig.KafkaBatchDelay,
		ErrorLogger:  log.New(),
		Topic:        kafConfig.KafkaOutputTopic,
		Transport:    transport,
	}

	if kafConfig.KafkaCompress {
		kWriter.Compression = kafka.Snappy
	}

	return kWriter
}

var kafkaUuidGen = fastuuid.MustNewGenerator()

func (kafConfig KafkaConfig) Output() {
	kWriter := kafConfig.getWriter()

	for {
		select {
		case data := <-kafConfig.outputChannel:
			if err := kafConfig.kafkaSendData(kWriter, data); err != nil {
				log.Errorf("Could not send kafka message: %v", err)
			}
		case <-kafConfig.closeChannel:
			log.Info("Closing kafka connection")
			kWriter.Close()
			return
		}
	}
}

func (kafConfig KafkaConfig) kafkaSendData(kWriter *kafka.Writer, dnsresult util.DNSResult) error {
	kafkaSentToOutput := metrics.GetOrRegisterCounter("kafkaSentToOutput", metrics.DefaultRegistry)
	kafkaSkipped := metrics.GetOrRegisterCounter("stdoutSkipped", metrics.DefaultRegistry)

	for _, dnsQuery := range dnsresult.DNS.Question {
		if util.CheckIfWeSkip(kafConfig.KafkaOutputType, dnsQuery.Name) {
			kafkaSkipped.Inc(1)
			return nil
		}
	}
	kafkaSentToOutput.Inc(1)

	myUUID := kafkaUuidGen.Hex128()

	return kWriter.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(myUUID),
		Value: []byte(dnsresult.GetJson()),
	})
}

// This will allow an instance to be spawned at import time
var _ = KafkaConfig{}.initializeFlags()
