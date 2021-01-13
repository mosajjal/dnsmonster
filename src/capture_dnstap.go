package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/golang/protobuf/proto"
)

var done = make(chan bool)
var ln net.Listener

func parseDnstapSocket(socketString, socketChmod string) *dnstap.FrameStreamSockInput {
	connArray := strings.Split(socketString, "://")
	var err error
	ln, err = net.Listen(connArray[0], connArray[1])

	if err != nil {
		log.Fatalf("%v\n", err)
	}
	log.Printf("listening on DNStap socket %v\n", socketString)

	if strings.Contains(socketString, "unix://") {
		//Chmod is defined in 8 bits not 10 bits, needs to be converter then passed on to the program
		permission, err := strconv.ParseInt(socketChmod, 8, 0)
		if err != nil {
			log.Fatalf("%v\n", err)
		}
		err = os.Chmod(connArray[1], os.FileMode(permission))
		if err != nil {
			log.Fatalf("%v\n", err)
		}
	}
	return dnstap.NewFrameStreamSockInput(ln)

}

func handleDNSTapInterrupt(done chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Printf("SIGINT received.. Cleaning up")
			if strings.Contains(*dnstapSocket, "unix://") {
				os.Remove(strings.Split(*dnstapSocket, "://")[1])
			} else {
				ln.Close()
			}
			close(done)
			return
		}
	}()
}

func dnsTapMsgToDNSResult(msg []byte) DNSResult {
	dnstapObject := &dnstap.Dnstap{}

	proto.Unmarshal(msg, dnstapObject)

	// var myDNSrow DNSRow
	var myDNSResult DNSResult

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

func startDNSTap(resultChannel chan DNSResult) {
	log.Println("Starting DNStap capture")
	input := parseDnstapSocket(*dnstapSocket, *dnstapPermission)

	buf := make(chan []byte, 1024)

	ratioCnt := 0
	totalCnt := 0

	// Set up various tickers for different tasks
	captureStatsTicker := time.Tick(*captureStatsDelay)
	printStatsTicker := time.Tick(*printStatsDelay)
	skipDomainsFileTicker := time.NewTicker(*skipDomainsRefreshInterval)
	skipDomainsFileTickerChan := skipDomainsFileTicker.C
	if *skipDomainsFile == "" {
		skipDomainsFileTicker.Stop()
	}

	allowDomainsFileTicker := time.NewTicker(*allowDomainsRefreshInterval)
	allowDomainsFileTickerChan := allowDomainsFileTicker.C
	if *allowDomainsFile == "" {
		allowDomainsFileTicker.Stop()
	}

	// Setup SIGINT handling
	handleDNSTapInterrupt(done)

	for {

		go input.ReadInto(buf)
		select {
		case msg := <-buf:
			ratioCnt++
			totalCnt++

			if msg == nil {
				log.Println("dnstap socket is returning nil. exiting..")
				time.Sleep(time.Second * 2)
				close(done)
				return
			}
			if ratioCnt%ratioB < ratioA {
				if ratioCnt > ratioB*ratioA {
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
		case <-captureStatsTicker:
			myStats.PacketsGot = totalCnt
			myStats.PacketsLost = 0
			myStats.PacketLossPercent = (float32(myStats.PacketsLost) * 100.0 / float32(myStats.PacketsGot))
		case <-printStatsTicker:
			log.Printf("%+v\n", myStats)
		case <-skipDomainsFileTickerChan:
			skipDomainList = loadDomains(*skipDomainsFile)
		case <-allowDomainsFileTickerChan:
			allowDomainList = loadDomains(*allowDomainsFile)
		}
	}
}
