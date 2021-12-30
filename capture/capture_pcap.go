package capture

import (
	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"

	"os"
	"os/signal"

	"github.com/google/gopacket/pcapgo"
)

var pcapStats captureStats

func initializeLivePcap(devName, filter string) *pcapgo.EthernetHandle {
	// Open device
	handle, err := pcapgo.NewEthernetHandle(devName)
	// handle, err := pcap.OpenLive(devName, 65536, true, pcap.BlockForever)
	util.ErrorHandler(err)

	// Set Filter
	log.Infof("Using Device: %s", devName)
	log.Infof("Filter: %s", filter)
	err = handle.SetBPF(TcpdumpToPcapgoBpf(filter))
	util.ErrorHandler(err)

	return handle
}

func handleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Infof("SIGINT received")
			close(types.GlobalExitChannel)
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
	types.GlobalWaitingGroup.Add(1)
	go encoder.run()

	return DNSCapturer{options, processingChannel}
}

func (capturer *DNSCapturer) Start() {

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

	defer myHandler.Close() // closes the packet handler and/or capture files
	types.GlobalWaitingGroup.Add(1)
	go func() {
		defer types.GlobalWaitingGroup.Done()
		for {
			//todo: the capture info has the timestamps, need to find a way to push this to handler
			data, ci, err := myHandler.ReadPacketData()
			if err != nil {
				log.Error(err)
				return
			}
			packetBytesChannel <- rawPacketBytes{data, ci}
		}
	}()

	// Setup SIGINT handling
	handleInterrupt()

	// Set up various tickers for different tasks
	captureStatsTicker := time.NewTicker(util.GeneralFlags.CaptureStatsDelay)
	printStatsTicker := time.NewTicker(util.GeneralFlags.PrintStatsDelay)

	var ratioCnt = 0
	var totalCnt = 0
	for {
		ratioCnt++

		select {
		case packetRawBytes := <-packetBytesChannel:
			if packetRawBytes.bytes == nil {
				log.Info("PacketSource returned nil, exiting (Possible end of pcap file?). Sleeping for 10 seconds waiting for processing to finish")
				time.Sleep(time.Second * 10)
				// close(types.GlobalExitChannel)  //todo: is this needed here?
				return
			}
			if ratioCnt%util.RatioB < util.RatioA {
				if ratioCnt > util.RatioB*util.RatioA { //reset ratiocount before it goes to an absurdly high number
					ratioCnt = 0
				}
				capturer.processing <- packetRawBytes
			}
		case <-types.GlobalExitChannel:
			log.Warn("Exiting capture loop")
			return
		case <-captureStatsTicker.C:

			mystats, err := myHandler.Stats()
			if err == nil {
				pcapStats.PacketsGot = int(mystats.Packets)
				pcapStats.PacketsLost = int(mystats.Drops)
			}
			if err != nil || mystats.Packets == 0 { // to make up for pcap not being able to get stats
				pcapStats.PacketsGot = totalCnt
			}

			pcapStats.PacketLossPercent = (float32(pcapStats.PacketsLost) * 100.0 / float32(pcapStats.PacketsGot))

		case <-printStatsTicker.C:
			log.Infof("%+v", pcapStats)

		}

	}
}
