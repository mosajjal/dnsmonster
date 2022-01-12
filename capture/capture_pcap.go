package capture

import (
	"time"

	"github.com/mosajjal/dnsmonster/util"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	"os"
	"os/signal"
)

func handleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			for {
				log.Infof("SIGINT Received. Stopping capture...")

				<-time.After(10 * time.Second)
				log.Fatal("emergency exit")
				return
			}
		}
	}()
}

func NewDNSCapturer(options CaptureOptions) DNSCapturer {
	if options.DevName != "" && options.PcapFile != "" {
		log.Fatal("You cant set DevName and PcapFile.")
	}
	var tcpChannels []chan tcpPacket

	tcpReturnChannel := make(chan tcpData, options.TCPResultChannelSize)
	processingChannel := make(chan rawPacketBytes, options.PacketChannelSize)
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

	return DNSCapturer{options, processingChannel}
}

func (capturer *DNSCapturer) Start() {
	packetsCaptured := metrics.GetOrRegisterGauge("packetsCaptured", metrics.DefaultRegistry)
	packetsDropped := metrics.GetOrRegisterGauge("packetsDropped", metrics.DefaultRegistry)
	packetLossPercent := metrics.GetOrRegisterGaugeFloat64("packetLossPercent", metrics.DefaultRegistry)

	var myHandler genericHandler

	options := capturer.options
	packetBytesChannel := make(chan rawPacketBytes, options.PacketChannelSize)
	if options.DevName != "" && !options.UseAfpacket {
		myHandler = initializeLivePcap(options.DevName, options.Filter)
		log.Info("Waiting for packets")

	} else if options.DevName != "" && options.UseAfpacket {
		myHandler = initializeLiveAFpacket(options.DevName, options.Filter)
		log.Info("Waiting for packets using AFpacket")

	} else {
		myHandler = initializeOfflinePcap(options.PcapFile, options.Filter)
		log.Info("Reading off Pcap file")
	}

	defer myHandler.Close() // closes the packet handler and/or capture files. there might be a duplicate of this in pcapfile
	go func() {
		for {
			data, ci, err := myHandler.ReadPacketData()
			util.ErrorHandler(err)
			packetBytesChannel <- rawPacketBytes{data, ci}
		}
	}()

	// Setup SIGINT handling
	handleInterrupt()

	// Set up various tickers for different tasks
	captureStatsTicker := time.NewTicker(util.GeneralFlags.CaptureStatsDelay)

	var ratioCnt = 0
	var totalCnt = int64(0)
	for {
		ratioCnt++

		select {
		case packetRawBytes := <-packetBytesChannel:
			if packetRawBytes.bytes == nil {
				log.Info("PacketSource returned nil, exiting (Possible end of pcap file?). Sleeping for 10 seconds waiting for processing to finish")
				time.Sleep(time.Second * 10)
				return
			}
			if ratioCnt%util.RatioB < util.RatioA {
				if ratioCnt > util.RatioB*util.RatioA { //reset ratiocount before it goes to an absurdly high number
					ratioCnt = 0
				}
				totalCnt++
				capturer.processing <- packetRawBytes
			}

		case <-captureStatsTicker.C:

			packets, drop := myHandler.Stat()
			if packets == 0 { // to make up for pcap not being able to get stats
				packetsCaptured.Update(totalCnt)
			} else {
				packetsCaptured.Update(int64(packets))
				packetsCaptured.Update(int64(drop))
			}
			packetLossPercent.Update(float64(packetsDropped.Value()) * 100.0 / float64(packetsCaptured.Value()))

		}

	}
}
