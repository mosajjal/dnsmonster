package capture

import (
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"

	dnstap "github.com/dnstap/golang-dnstap"
	"google.golang.org/protobuf/proto"
)

var done = make(chan bool)
var ln net.Listener

func parseDnstapSocket(socketString, socketChmod string) *dnstap.FrameStreamSockInput {
	var err error
	uri, err := url.ParseRequestURI(socketString)
	util.ErrorHandler(err)
	if uri.Scheme == "tcp4" || uri.Scheme == "tcp" || uri.Scheme == "tcp6" {
		ln, err = net.Listen(uri.Scheme, uri.Host)
		util.ErrorHandler(err)
	} else {
		ln, err = net.Listen(uri.Scheme, uri.Path)
		util.ErrorHandler(err)
	}
	log.Infof("listening on DNStap socket %v", socketString)

	if uri.Scheme == "unix" {
		//Chmod is defined in 8 bits not 10 bits, needs to be converter then passed on to the program
		permission, err := strconv.ParseInt(socketChmod, 8, 0)
		util.ErrorHandler(err)
		err = os.Chmod(uri.Path, os.FileMode(permission))
		util.ErrorHandler(err)
	}
	return dnstap.NewFrameStreamSockInput(ln)

}

func handleDNSTapInterrupt(done chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Infof("SIGINT received.. Cleaning up")
			if strings.Contains(util.CaptureFlags.DnstapSocket, "unix://") {
				os.Remove(strings.Split(util.CaptureFlags.DnstapSocket, "://")[1])
			} else {
				ln.Close()
			}
			close(done)
		}
	}()
}

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

func StartDNSTap(resultChannel chan types.DNSResult) {
	log.Info("Starting DNStap capture")
	input := parseDnstapSocket(util.CaptureFlags.DnstapSocket, util.CaptureFlags.DnstapPermission)

	buf := make(chan []byte, 1024)

	ratioCnt := 0
	totalCnt := uint(0)

	// Setup SIGINT handling
	handleDNSTapInterrupt(done)

	// Set up various tickers for different tasks
	captureStatsTicker := time.NewTicker(util.GeneralFlags.CaptureStatsDelay)
	printStatsTicker := time.NewTicker(util.GeneralFlags.PrintStatsDelay)

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
			if ratioCnt%util.RatioB < util.RatioA {
				if ratioCnt > util.RatioB*util.RatioA {
					ratioCnt = 0
				}
				select {
				case resultChannel <- dnsTapMsgToDNSResult(msg):

				case <-done:
					return
				}
			}
		case <-done:
			return
		case <-captureStatsTicker.C:
			pcapStats.PacketsGot = totalCnt
			pcapStats.PacketsLost = 0
			pcapStats.PacketLossPercent = (float32(pcapStats.PacketsLost) * 100.0 / float32(pcapStats.PacketsGot))
		case <-printStatsTicker.C:
			log.Infof("%+v", pcapStats)
		}
	}
}
