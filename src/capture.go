package main

import (
	"log"
	"sync"
	"time"

	"net"
	"os"
	"os/signal"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	mkdns "github.com/miekg/dns"
)

type Stats struct {
	PacketsGot        int
	PacketsLost       int
	PacketLossPercent float32
	sentToDB          int
	skippedDomains    int
}

var myStats Stats

type CaptureOptions struct {
	DevName                      string
	useAfpacket                  bool
	PcapFile                     string
	Filter                       string
	Port                         uint16
	GcTime                       time.Duration
	ResultChannel                chan<- DNSResult
	PacketHandlerCount           uint
	PacketChannelSize            uint
	TCPHandlerCount              uint
	TCPAssemblyChannelSize       uint
	TCPResultChannelSize         uint
	IPDefraggerChannelSize       uint
	IPDefraggerReturnChannelSize uint
	Done                         chan bool
}

type DNSCapturer struct {
	options    CaptureOptions
	processing chan gopacket.Packet
}

type DNSResult struct {
	Timestamp    time.Time
	DNS          mkdns.Msg
	IPVersion    uint8
	SrcIP        net.IP
	DstIP        net.IP
	Protocol     string
	PacketLength uint16
}

func initializeLivePcap(devName, filter string) *pcap.Handle {
	// Open device
	handle, err := pcap.OpenLive(devName, 65536, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}

	// Set Filter
	log.Printf("Using Device: %s\n", devName)
	log.Printf("Filter: %s\n", filter)
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}

	return handle
}

func initializeOfflinePcap(fileName, filter string) *pcap.Handle {
	handle, err := pcap.OpenOffline(fileName)
	if err != nil {
		log.Fatal(err)
	}

	// Set Filter
	log.Printf("Using File: %s\n", fileName)
	log.Printf("Filter: %s\n", filter)
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}
	return handle
}

func handleInterrupt(done chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Printf("SIGINT received")
			close(done)
			return
		}
	}()
}

func NewDNSCapturer(options CaptureOptions) DNSCapturer {
	if options.DevName != "" && options.PcapFile != "" {
		log.Fatal("You cant set DevName and PcapFile.")
	}
	var tcpChannels []chan tcpPacket

	tcpReturnChannel := make(chan tcpData, options.TCPResultChannelSize)
	processingChannel := make(chan gopacket.Packet, options.PacketChannelSize)
	ip4DefraggerChannel := make(chan ipv4ToDefrag, options.IPDefraggerChannelSize)
	ip6DefraggerChannel := make(chan ipv6FragmentInfo, options.IPDefraggerChannelSize)
	ip4DefraggerReturn := make(chan ipv4Defragged, options.IPDefraggerReturnChannelSize)
	ip6DefraggerReturn := make(chan ipv6Defragged, options.IPDefraggerReturnChannelSize)

	for i := uint(0); i < options.TCPHandlerCount; i++ {
		tcpChannels = append(tcpChannels, make(chan tcpPacket, options.TCPAssemblyChannelSize))
		go tcpAssembler(tcpChannels[i], tcpReturnChannel, options.GcTime, options.Done)
	}

	go ipv4Defragger(ip4DefraggerChannel, ip4DefraggerReturn, options.GcTime, options.Done)
	go ipv6Defragger(ip6DefraggerChannel, ip6DefraggerReturn, options.GcTime, options.Done)

	encoder := packetEncoder{
		options.Port,
		processingChannel,
		ip4DefraggerChannel,
		ip6DefraggerChannel,
		ip4DefraggerReturn,
		ip6DefraggerReturn,
		tcpChannels,
		tcpReturnChannel,
		options.ResultChannel,
		options.Done,
	}
	var wg sync.WaitGroup
	for i := uint(0); i < options.PacketHandlerCount; i++ {
		wg.Add(1)
		go encoder.run()
	}
	return DNSCapturer{options, processingChannel}
}

func (capturer *DNSCapturer) Start() {
	var handle *pcap.Handle
	var afhandle *afpacketHandle
	var packetSource *gopacket.PacketSource
	options := capturer.options
	if options.DevName != "" && !options.useAfpacket {
		handle = initializeLivePcap(options.DevName, options.Filter)
		defer handle.Close()
		packetSource = gopacket.NewPacketSource(handle, handle.LinkType())
		log.Println("Waiting for packets")
	} else if options.DevName != "" && options.useAfpacket {
		afhandle = initializeLiveAFpacket(options.DevName, options.Filter)
		defer afhandle.Close()
		packetSource = gopacket.NewPacketSource(afhandle, afhandle.LinkType())
		log.Println("Waiting for packets using AFpacket")
	} else {
		handle = initializeOfflinePcap(options.PcapFile, options.Filter)
		defer handle.Close()
		packetSource = gopacket.NewPacketSource(handle, handle.LinkType())
		log.Println("Reading off Pcap file")
	}
	packetSource.DecodeOptions.Lazy = true
	packetSource.NoCopy = true

	// Setup SIGINT handling
	handleInterrupt(options.Done)

	// Set up various tickers for different tasks
	captureStatsTicker := time.Tick(*captureStatsDelay)
	printStatsTicker := time.Tick(*printStatsDelay)
	skipDomainsFileTicker := time.NewTicker(*skipDomainsRefreshInterval)
	skipDomainsFileTickerChan := skipDomainsFileTicker.C
	if *skipDomainsFile == "" {
		skipDomainsFileTicker.Stop()
	}
	var cnt = 0
	for {
		cnt++
		select {
		case packet := <-packetSource.Packets():
			if packet == nil {
				log.Println("PacketSource returned nil, exiting (Possible end of pcap file?). Sleeping for 10 seconds waiting for processing to finish")
				time.Sleep(time.Second * 10)
				close(options.Done)
				return
			}

			if cnt%ratioB < ratioA {
				if cnt > ratioB*ratioA {
					cnt = 0
				}
				select {
				case capturer.processing <- packet:
				case <-options.Done:
					return
				}
			}
		case <-options.Done:
			return
		case <-captureStatsTicker:
			if handle != nil {
				mystats, err := handle.Stats()
				if err == nil {
					myStats.PacketsGot = mystats.PacketsReceived
					myStats.PacketsLost = mystats.PacketsDropped
				}
			} else {
				updateAfpacketStats(afhandle)
			}
			myStats.PacketLossPercent = (float32(myStats.PacketsLost) * 100.0 / float32(myStats.PacketsGot))

		case <-printStatsTicker:
			log.Printf("%+v\n", myStats)
		case <-skipDomainsFileTickerChan:
			SkipDomainList = loadSkipDomains()
		}

	}
}
