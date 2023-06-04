/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

package output

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"time"

	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"github.com/rogpeppe/fastuuid"
	"github.com/segmentio/kafka-go"
)

type kafkaConfig struct {
	KafkaOutputType         uint          `long:"kafkaoutputtype"             ini-name:"kafkaoutputtype"             env:"DNSMONSTER_KAFKAOUTPUTTYPE"             default:"0"                                                       description:"What should be written to kafka. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"         choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	KafkaOutputBroker       []string      `long:"kafkaoutputbroker"           ini-name:"kafkaoutputbroker"           env:"DNSMONSTER_KAFKAOUTPUTBROKER"           default:""                                                        description:"kafka broker address(es), example: 127.0.0.1:9092. Used if kafkaOutputType is not none"`
	KafkaOutputTopic        string        `long:"kafkaoutputtopic"            ini-name:"kafkaoutputtopic"            env:"DNSMONSTER_KAFKAOUTPUTTOPIC"            default:"dnsmonster"                                              description:"Kafka topic for logging"`
	KafkaBatchSize          uint          `long:"kafkabatchsize"              ini-name:"kafkabatchsize"              env:"DNSMONSTER_KAFKABATCHSIZE"              default:"1000"                                                    description:"Minimum capacity of the cache array used to send data to Kafka"`
	KafkaOutputFormat       string        `long:"kafkaoutputformat"           ini-name:"kafkaoutputformat"           env:"DNSMONSTER_KAFKAOUTPUTFORMAT"           default:"json"                                                    description:"Output format. options:json, gob. "                                                                                                                                               choice:"json" choice:"gob"`
	KafkaTimeout            uint          `long:"kafkatimeout"                ini-name:"kafkatimeout"                env:"DNSMONSTER_KAFKATIMEOUT"                default:"3"                                                       description:"Kafka connection timeout in seconds"`
	KafkaBatchDelay         time.Duration `long:"kafkabatchdelay"             ini-name:"kafkabatchdelay"             env:"DNSMONSTER_KAFKABATCHDELAY"             default:"1s"                                                      description:"Interval between sending results to Kafka if Batch size is not filled"`
	KafkaCompress           bool          `long:"kafkacompress"               ini-name:"kafkacompress"               env:"DNSMONSTER_KAFKACOMPRESS"                                                                                 description:"Compress Kafka connection"`
	KafkaSecure             bool          `long:"kafkasecure"                 ini-name:"kafkasecure"                 env:"DNSMONSTER_KAFKASECURE"                                                                                   description:"Use TLS for kafka connection"`
	KafkaCACertificatePath  string        `long:"kafkacacertificatepath"      ini-name:"kafkacacertificatepath"      env:"DNSMONSTER_KAFKACACERTIFICATEPATH"      default:""                                                        description:"Path of CA certificate that signs Kafka broker certificate"`
	KafkaTLSCertificatePath string        `long:"kafkatlscertificatepath"     ini-name:"kafkatlscertificatepath"     env:"DNSMONSTER_KAFKATLSCERTIFICATEPATH"     default:""                                                        description:"Path of TLS certificate to present to broker"`
	KafkaTLSKeyPath         string        `long:"kafkatlskeypath"             ini-name:"kafkatlskeypath"             env:"DNSMONSTER_KAFKATLSKEYPATH"             default:""                                                        description:"Path of TLS certificate key"`
	outputChannel           chan util.DNSResult
	outputMarshaller        util.OutputMarshaller
	closeChannel            chan bool
}

func init() {
	c := kafkaConfig{}
	if _, err := util.GlobalParser.AddGroup("kafka_output", "Kafka Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (kafConfig kafkaConfig) Initialize(ctx context.Context) error {
	var err error
	kafConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller(kafConfig.KafkaOutputFormat, "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if kafConfig.KafkaOutputType > 0 && kafConfig.KafkaOutputType < 5 {
		log.Info("Creating Kafka Output Channel")
		go kafConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (kafConfig kafkaConfig) Close() {
	close(kafConfig.closeChannel)
}

func (kafConfig kafkaConfig) OutputChannel() chan util.DNSResult {
	return kafConfig.outputChannel
}

func (kafConfig kafkaConfig) getWriter() *kafka.Writer {
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

var kafkaUUIDGen = fastuuid.MustNewGenerator()

func (kafConfig kafkaConfig) Output(ctx context.Context) {
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

func (kafConfig kafkaConfig) kafkaSendData(kWriter *kafka.Writer, dnsresult util.DNSResult) error {
	kafkaSentToOutput := metrics.GetOrRegisterCounter("kafkaSentToOutput", metrics.DefaultRegistry)
	kafkaSkipped := metrics.GetOrRegisterCounter("kafkaSkipped", metrics.DefaultRegistry)

	for _, dnsQuery := range dnsresult.DNS.Question {
		if util.CheckIfWeSkip(kafConfig.KafkaOutputType, dnsQuery.Name) {
			kafkaSkipped.Inc(1)
			return nil
		}
	}
	kafkaSentToOutput.Inc(1)

	myUUID := kafkaUUIDGen.Hex128()

	return kWriter.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(myUUID),
		Value: []byte(kafConfig.outputMarshaller.Marshal(dnsresult)),
	})
}

// This will allow an instance to be spawned at import time
// var _ = kafkaConfig{}.initializeFlags()
// vim: foldmethod=marker
