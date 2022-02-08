package capture

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	dnstap "github.com/dnstap/golang-dnstap"
	"google.golang.org/protobuf/proto"
)

var done = make(chan bool)
var ln net.Listener

func parseDnstapSocket(socketString, socketChmod string) *dnstap.FrameStreamSockInput {
	var err error
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
		if err != nil {
			log.Fatal(err)
		}
	}

	return dnstap.NewFrameStreamSockInput(ln)

}

// func handleDNSTapInterrupt(done chan bool) {
// 	c := make(chan os.Signal, 1)
// 	signal.Notify(c, os.Interrupt)
// 	go func() {
// 		for range c {
// 			log.Infof("SIGINT received.. Cleaning up")
// 			if strings.Contains(util.CaptureFlags.DnstapSocket, "unix://") {
// 				os.Remove(strings.Split(util.CaptureFlags.DnstapSocket, "://")[1])
// 			} else {
// 				ln.Close()
// 			}
// 			close(done)
// 		}
// 	}()
// }

func dnsTapMsgToDNSResult(msg []byte) types.DNSResult {
	dnstapObject := &dnstap.Dnstap{}

	proto.Unmarshal(msg, dnstapObject)

	// var myDNSrow DNSRow
	var myDNSResult types.DNSResult

	if dnstapObject.Message.GetQueryMessage() != nil {
		myDNSResult.DNS.Unpack(dnstapObject.Message.GetQueryMessage())
	} else {
		myDNSResult.DNS.Unpack(dnstapObject.Message.GetResponseMessage())
	}

	myDNSResult.Timestamp = time.Unix(int64(dnstapObject.Message.GetQueryTimeSec()), int64(dnstapObject.Message.GetQueryTimeNsec()))
	myDNSResult.IPVersion = uint8(*dnstapObject.GetMessage().SocketFamily)*2 + 2
	myDNSResult.SrcIP = dnstapObject.Message.GetQueryAddress()
	myDNSResult.DstIP = dnstapObject.Message.GetQueryAddress()
	myDNSResult.Protocol = strings.ToLower(dnstapObject.Message.GetSocketProtocol().String())
	myDNSResult.PacketLength = uint16(len(dnstapObject.Message.GetResponseMessage()) + len(dnstapObject.Message.GetQueryMessage()))

	return myDNSResult
}

func (config CaptureConfig) StartDnsTap() {
	log.Info("Starting DNStap capture")

	packetsCaptured := metrics.GetOrRegisterGauge("packetsCaptured", metrics.DefaultRegistry)
	packetsDropped := metrics.GetOrRegisterGauge("packetsDropped", metrics.DefaultRegistry)
	packetLossPercent := metrics.GetOrRegisterGaugeFloat64("packetLossPercent", metrics.DefaultRegistry)

	input := parseDnstapSocket(config.DnstapSocket, config.DnstapPermission)

	buf := make(chan []byte, 1024)

	ratioCnt := 0
	totalCnt := int64(0)

	// Setup SIGINT handling
	// handleDNSTapInterrupt(done)

	// Set up various tickers for different tasks
	captureStatsTicker := time.NewTicker(util.GeneralFlags.CaptureStatsDelay)

	for {

		go input.ReadInto(buf)
		select {
		case msg := <-buf:
			ratioCnt++
			totalCnt++

			if msg == nil {
				log.Info("dnstap socket is returning nil. exiting..")
				time.Sleep(time.Second * 2)
				close(done)
				return
			}
			if ratioCnt%config.ratioB < config.ratioA {
				if ratioCnt > config.ratioB*config.ratioA {
					ratioCnt = 0
				}
				select {
				case config.resultChannel <- dnsTapMsgToDNSResult(msg):

				case <-done:
					return
				}
			}
		case <-done:
			return
		case <-captureStatsTicker.C:
			packetsCaptured.Update(totalCnt)
			packetsDropped.Update(0) //todo: this is not correct, need to fix
			packetLossPercent.Update(float64(packetsDropped.Value()) * 100.0 / float64(packetsCaptured.Value()))

		}
	}
}
