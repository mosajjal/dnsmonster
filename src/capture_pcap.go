package main

import (
	"time"

	"github.com/mosajjal/dnsmonster/types"
	log "github.com/sirupsen/logrus"

	"os"
	"os/signal"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

var pcapStats captureStats

func initializeLivePcap(devName, filter string) *pcap.Handle {
	// Open device
	handle, err := pcap.OpenLive(devName, 65536, true, pcap.BlockForever)
	errorHandler(err)

	// Set Filter
	log.Infof("Using Device: %s", devName)
	log.Infof("Filter: %s", filter)
	err = handle.SetBPFFilter(filter)
	errorHandler(err)

	return handle
}

func initializeOfflinePcap(fileName, filter string) *pcap.Handle {
	handle, err := pcap.OpenOffline(fileName)
	errorHandler(err)

	// Set Filter
	log.Infof("Using File: %s", fileName)
	log.Infof("Filter: %s", filter)
	err = handle.SetBPFFilter(filter)
	errorHandler(err)
	return handle
}

func handleInterrupt(done chan bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Infof("SIGINT received")
			close(done)
			return
		}
	}()
}

func newDNSCapturer(options CaptureOptions) DNSCapturer {
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
		go tcpAssembler(tcpChannels[i], tcpReturnChannel, options.GcTime)
	}

	go ipv4Defragger(ip4DefraggerChannel, ip4DefraggerReturn, options.GcTime)
	go ipv6Defragger(ip6DefraggerChannel, ip6DefraggerReturn, options.GcTime)
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
		options.PacketHandlerCount,
		options.NoEthernetframe,
	}
	go encoder.run()
	types.GlobalWaitingGroup.Add(1)
	defer types.GlobalWaitingGroup.Done()
	// todo: use the global wg for this

	return DNSCapturer{options, processingChannel}
}

func (capturer *DNSCapturer) start() {
	var handle *pcap.Handle
	var afhandle *afpacketHandle
	var packetSource *gopacket.PacketSource
	options := capturer.options
	if options.DevName != "" && !options.useAfpacket {
		handle = initializeLivePcap(options.DevName, options.Filter)
		defer handle.Close()
		packetSource = gopacket.NewPacketSource(handle, handle.LinkType())
		log.Info("Waiting for packets")
	} else if options.DevName != "" && options.useAfpacket {
		afhandle = initializeLiveAFpacket(options.DevName, options.Filter)
		defer afhandle.Close()
		packetSource = gopacket.NewPacketSource(afhandle, afhandle.LinkType())
		log.Info("Waiting for packets using AFpacket")
	} else {
		handle = initializeOfflinePcap(options.PcapFile, options.Filter)
		defer handle.Close()
		packetSource = gopacket.NewPacketSource(handle, handle.LinkType())
		log.Info("Reading off Pcap file")
	}
	packetSource.DecodeOptions.Lazy = true
	packetSource.NoCopy = true

	// Setup SIGINT handling
	handleInterrupt(types.GlobalExitChannel)

	// Set up various tickers for different tasks
	captureStatsTicker := time.Tick(generalOptions.CaptureStatsDelay)
	printStatsTicker := time.Tick(generalOptions.PrintStatsDelay)

	var ratioCnt = 0
	var totalCnt = 0
	for {
		ratioCnt++
		select {
		case packet := <-packetSource.Packets():
			if packet == nil {
				log.Info("PacketSource returned nil, exiting (Possible end of pcap file?). Sleeping for 10 seconds waiting for processing to finish")
				time.Sleep(time.Second * 10)
				close(types.GlobalExitChannel)
				return
			}
			if ratioCnt%ratioB < ratioA {
				if ratioCnt > ratioB*ratioA {
					ratioCnt = 0
				}
				select {
				case capturer.processing <- packet:
					totalCnt++
				case <-types.GlobalExitChannel:
					return
				}
			}
		case <-types.GlobalExitChannel:
			return
		case <-captureStatsTicker:
			if handle != nil {
				mystats, err := handle.Stats()
				if err == nil {
					pcapStats.PacketsGot = mystats.PacketsReceived
					pcapStats.PacketsLost = mystats.PacketsDropped
				} else {
					pcapStats.PacketsGot = totalCnt
				}
			} else {
				updateAfpacketStats(afhandle)
			}
			pcapStats.PacketLossPercent = (float32(pcapStats.PacketsLost) * 100.0 / float32(pcapStats.PacketsGot))

		case <-printStatsTicker:
			log.Infof("%+v", pcapStats)

		}

	}
}
