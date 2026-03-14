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
	"os"

	"github.com/mosajjal/dnsmonster/internal/util"
	pb "github.com/mosajjal/dnsmonster/proto"
	"github.com/nats-io/nats.go"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type natsConfig struct {
	NatsOutputType    uint   `long:"natsoutputtype"    ini-name:"natsoutputtype"    env:"DNSMONSTER_NATSOUTPUTTYPE"    default:"0"    description:"What should be written to NATS. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	NatsOutputServer  string `long:"natsoutputserver"  ini-name:"natsoutputserver"  env:"DNSMONSTER_NATSOUTPUTSERVER"  default:"nats://localhost:4222" description:"NATS server address"`
	NatsOutputSubject string `long:"natsoutputsubject" ini-name:"natsoutputsubject" env:"DNSMONSTER_NATSOUTPUTSUBJECT" default:"dns.events"            description:"NATS subject to publish to"`
	NatsOutputTLSCert string `long:"natsoutputtlscert" ini-name:"natsoutputtlscert" env:"DNSMONSTER_NATSOUTPUTTLSCERT" default:""                     description:"Path to client TLS certificate for NATS mTLS"`
	NatsOutputTLSKey  string `long:"natsoutputtlskey"  ini-name:"natsoutputtlskey"  env:"DNSMONSTER_NATSOUTPUTTLSKEY"  default:""                     description:"Path to client TLS key for NATS mTLS"`
	NatsOutputTLSCA   string `long:"natsoutputtlsca"   ini-name:"natsoutputtlsca"   env:"DNSMONSTER_NATSOUTPUTTLSCA"   default:""                     description:"Path to CA certificate for NATS TLS verification"`
	outputChannel     chan util.DNSResult
	closeChannel      chan bool
}

func init() {
	c := natsConfig{}
	if _, err := util.GlobalParser.AddGroup("nats_output", "NATS Output", &c); err != nil {
		log.Fatalf("error adding NATS output module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

func (nc natsConfig) Initialize(ctx context.Context) error {
	if nc.NatsOutputType > 0 && nc.NatsOutputType < 5 {
		log.Info("Creating NATS Output Channel")
		go nc.Output(ctx)
	} else {
		return errors.New("no output")
	}
	return nil
}

func (nc natsConfig) Close() {
	close(nc.closeChannel)
}

func (nc natsConfig) OutputChannel() chan util.DNSResult {
	return nc.outputChannel
}

func (nc natsConfig) connect() (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("dnsmonster"),
		nats.MaxReconnects(-1),
	}

	if nc.NatsOutputTLSCert != "" && nc.NatsOutputTLSKey != "" {
		cert, err := tls.LoadX509KeyPair(nc.NatsOutputTLSCert, nc.NatsOutputTLSKey)
		if err != nil {
			return nil, err
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		if nc.NatsOutputTLSCA != "" {
			caCert, err := os.ReadFile(nc.NatsOutputTLSCA)
			if err != nil {
				return nil, err
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = pool
		}
		opts = append(opts, nats.Secure(tlsConfig))
	} else if nc.NatsOutputTLSCA != "" {
		caCert, err := os.ReadFile(nc.NatsOutputTLSCA)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)
		opts = append(opts, nats.Secure(&tls.Config{RootCAs: pool}))
	}

	return nats.Connect(nc.NatsOutputServer, opts...)
}

func dnsResultToProto(d util.DNSResult) ([]byte, error) {
	ev := &pb.DNSEvent{
		TimestampNs:  d.Timestamp.UnixNano(),
		SrcIp:        d.SrcIP,
		DstIp:        d.DstIP,
		SrcPort:      uint32(d.SrcPort),
		DstPort:      uint32(d.DstPort),
		IpVersion:    uint32(d.IPVersion),
		Protocol:     d.Protocol,
		PacketLength: uint32(d.PacketLength),
		Identity:     d.Identity,
		Version:      d.Version,
		IsResponse:   d.DNS.Response,
		Rcode:        uint32(d.DNS.Rcode),
	}

	if len(d.DNS.Question) > 0 {
		ev.Qname = d.DNS.Question[0].Name
		ev.Qtype = uint32(d.DNS.Question[0].Qtype)
	}

	wire, err := d.DNS.Pack()
	if err != nil {
		log.Debugf("failed to pack DNS message: %v", err)
	} else {
		ev.DnsWire = wire
	}

	return proto.Marshal(ev)
}

func (nc natsConfig) Output(ctx context.Context) {
	conn, err := nc.connect()
	if err != nil {
		log.Fatalf("could not connect to NATS: %v", err)
	}
	defer conn.Close()

	natsSent := metrics.GetOrRegisterCounter("natsSentToOutput", metrics.DefaultRegistry)
	natsSkipped := metrics.GetOrRegisterCounter("natsSkipped", metrics.DefaultRegistry)
	natsErrors := metrics.GetOrRegisterCounter("natsErrors", metrics.DefaultRegistry)

	for {
		select {
		case data := <-nc.outputChannel:
			for _, dnsQuery := range data.DNS.Question {
				if util.CheckIfWeSkip(nc.NatsOutputType, dnsQuery.Name) {
					natsSkipped.Inc(1)
					continue
				}
			}

			buf, err := dnsResultToProto(data)
			if err != nil {
				natsErrors.Inc(1)
				log.Errorf("failed to marshal DNSEvent: %v", err)
				continue
			}

			if err := conn.Publish(nc.NatsOutputSubject, buf); err != nil {
				natsErrors.Inc(1)
				log.Errorf("failed to publish to NATS: %v", err)
				continue
			}
			natsSent.Inc(1)

		case <-ctx.Done():
			log.Info("Context cancelled, closing NATS connection")
			conn.Flush()
			return
		case <-nc.closeChannel:
			log.Info("Closing NATS connection")
			conn.Flush()
			return
		}
	}
}

// vim: foldmethod=marker
