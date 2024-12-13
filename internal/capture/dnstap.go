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

package capture

import (
	"context"
	b64 "encoding/base64"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mosajjal/dnsmonster/internal/util"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	dnstap "github.com/dnstap/golang-dnstap"
	"google.golang.org/protobuf/proto"
)

func parseDnstapSocket(socketString, socketChmod string) *dnstap.FrameStreamSockInput {
	var err error
	var ln net.Listener
	uri, err := url.ParseRequestURI(socketString)
	if err != nil {
		log.Fatal(err)
	}
	if uri.Scheme == "tcp4" || uri.Scheme == "tcp" || uri.Scheme == "tcp6" {
		ln, err = net.Listen(uri.Scheme, uri.Host)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Infof("listening on DNStap socket %v", socketString)
		// see if the socket exists
		if _, err := os.Stat(uri.Path); err == nil {
			log.Infof("socket exists, will try to overwrite the socket")
			os.Remove(uri.Path)
		}

		ln, err = net.Listen(uri.Scheme, uri.Path)
		if err != nil {
			log.Fatal(err)
		}

		if uri.Scheme == "unix" {
			permission := 0
			if len(socketChmod) > 3 {
				log.Fatal("Chmod is not in the correct format")
			}
			for _, c := range socketChmod {
				permBit, _ := strconv.Atoi(string(c))
				if permBit > 7 || permBit < 0 || err != nil {
					log.Fatal("Chmod string is not valid")
				}
				permission = permission*8 + permBit
			}
			err = os.Chmod(uri.Path, os.FileMode(permission))
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	dSocket := dnstap.NewFrameStreamSockInput(ln)
	dSocket.SetLogger(log.New())
	return dSocket
}

func dnsTapMsgToDNSResult(msg []byte) (*util.DNSResult, error) {
	dnstapObject := &dnstap.Dnstap{}

	if err := proto.Unmarshal(msg, dnstapObject); err != nil {
		return nil, err
	}

	var myDNSResult util.DNSResult
	myDNSResult.Identity = string(dnstapObject.GetIdentity())
	myDNSResult.Version = string(dnstapObject.GetVersion())
	myDNSResult.IPVersion = uint8(*dnstapObject.GetMessage().SocketFamily)*2 + 2
	myDNSResult.SrcIP = dnstapObject.Message.GetQueryAddress()
	myDNSResult.SrcPort = uint16(dnstapObject.Message.GetQueryPort())
	myDNSResult.DstIP = dnstapObject.Message.GetResponseAddress()
	myDNSResult.DstPort = uint16(dnstapObject.Message.GetResponsePort())
	myDNSResult.Protocol = strings.ToLower(dnstapObject.Message.GetSocketProtocol().String())

	message := dnstapObject.Message.GetQueryMessage()
	if message != nil {
		// query
		myDNSResult.Timestamp = time.Unix(int64(dnstapObject.Message.GetQueryTimeSec()), int64(dnstapObject.Message.GetQueryTimeNsec()))
	} else {
		// response
		myDNSResult.Timestamp = time.Unix(int64(dnstapObject.Message.GetResponseTimeSec()), int64(dnstapObject.Message.GetResponseTimeNsec()))

		message = dnstapObject.Message.GetResponseMessage()
	}

	myDNSResult.PacketLength = uint16(len(message))
	if err := myDNSResult.DNS.Unpack(message); err != nil {
		return nil, err
	}

	return &myDNSResult, nil
}

func (config captureConfig) StartDNSTap(ctx context.Context) error {
	log.Info("Starting DNStap capture")

	packetsCaptured := metrics.GetOrRegisterGauge("packetsCaptured", metrics.DefaultRegistry)
	packetsDropped := metrics.GetOrRegisterGauge("packetsDropped", metrics.DefaultRegistry)
	packetsInvalid := metrics.GetOrRegisterGauge("packetsInvalid", metrics.DefaultRegistry)
	packetLossPercent := metrics.GetOrRegisterGaugeFloat64("packetLossPercent", metrics.DefaultRegistry)

	input := parseDnstapSocket(config.DnstapSocket, config.DnstapPermission)
	buf := make(chan []byte, 1024)
	g, _ := errgroup.WithContext(ctx)
	//todo: can't pass on the context to this function.
	g.Go(func() error { input.ReadInto(buf); return nil })

	ratioCnt := 0
	totalCnt := int64(0)
	droppedCnt := int64(0)
	invalidCnt := int64(0)

	// Set up various tickers for different tasks
	captureStatsTicker := time.NewTicker(util.GeneralFlags.CaptureStatsDelay)

	// blocking loop
	for {
		select {
		case msg := <-buf:
			ratioCnt++
			totalCnt++

			if msg == nil {
				log.Info("dnstap socket is returning nil. exiting..")
				time.Sleep(time.Second * 2)
				//todo: commence clean exit
				config.cleanExit(ctx)
				return nil
			}
			if ratioCnt%config.ratioB < config.ratioA {
				if ratioCnt > config.ratioB*config.ratioA {
					ratioCnt = 0
				}
				res, err := dnsTapMsgToDNSResult(msg)
				if err != nil {
					log.Errorf("could not unpack message: %v, content: %s", err, b64.StdEncoding.EncodeToString(msg))
					invalidCnt++
					continue
				}

				config.resultChannel <- *res
			} else {
				droppedCnt++
			}
		case <-captureStatsTicker.C:
			packetsCaptured.Update(totalCnt)
			packetsDropped.Update(droppedCnt)
			packetsInvalid.Update(invalidCnt)
			packetLossPercent.Update(float64(packetsDropped.Value()) * 100.0 / float64(packetsCaptured.Value()))
		}
	}
}

// vim: foldmethod=marker
