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

	"github.com/mosajjal/dnsmonster/util"
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
	dnstapMsg := &dnstap.Dnstap{}

	if err := proto.Unmarshal(msg, dnstapMsg); err != nil {
		return nil, err
	}

	var myDNSResult util.DNSResult
	myDNSResult.Identity = string(dnstapMsg.GetIdentity())
	myDNSResult.Version = string(dnstapMsg.GetVersion())
	myDNSResult.IPVersion = uint8(*dnstapMsg.GetMessage().SocketFamily)*2 + 2
	myDNSResult.SrcIP = dnstapMsg.Message.GetQueryAddress()
	myDNSResult.SrcPort = uint16(dnstapMsg.Message.GetQueryPort())
	myDNSResult.DstIP = dnstapMsg.Message.GetResponseAddress()
	myDNSResult.DstPort = uint16(dnstapMsg.Message.GetResponsePort())
	myDNSResult.Protocol = strings.ToLower(dnstapMsg.Message.GetSocketProtocol().String())

	message := dnstapMsg.Message.GetQueryMessage()
	if message != nil {
		// query
		myDNSResult.Timestamp = time.Unix(int64(dnstapMsg.Message.GetQueryTimeSec()), int64(dnstapMsg.Message.GetQueryTimeNsec()))
	} else {
		// response
		myDNSResult.Timestamp = time.Unix(int64(dnstapMsg.Message.GetResponseTimeSec()), int64(dnstapMsg.Message.GetResponseTimeNsec()))

		message = dnstapMsg.Message.GetResponseMessage()
	}

	myDNSResult.PacketLength = uint16(len(message))
	if err := myDNSResult.DNS.Unpack(message); err != nil {
		return nil, err
	}

	return &myDNSResult, nil
}

func (config captureConfig) StartDNSTap(ctx context.Context, path string) error {
	log.Info("Starting DNStap capture")

	packetsCaptured := metrics.GetOrRegisterGauge("packetsCaptured_"+path, metrics.DefaultRegistry)
	packetsDropped := metrics.GetOrRegisterGauge("packetsDropped_"+path, metrics.DefaultRegistry)
	packetsInvalid := metrics.GetOrRegisterGauge("packetsInvalid_"+path, metrics.DefaultRegistry)
	packetLossPercent := metrics.GetOrRegisterGaugeFloat64("packetLossPercent_"+path, metrics.DefaultRegistry)

	input := parseDnstapSocket(path, config.DnstapPermission)
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
