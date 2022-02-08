package capture

import (
	"os"
	"time"

	"github.com/mosajjal/dnsmonster/util"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

func (config CaptureConfig) StartNonDnsTap() {

	packetsCaptured := metrics.GetOrRegisterGauge("packetsCaptured", metrics.DefaultRegistry)
	packetsDropped := metrics.GetOrRegisterGauge("packetsDropped", metrics.DefaultRegistry)
	packetLossPercent := metrics.GetOrRegisterGaugeFloat64("packetLossPercent", metrics.DefaultRegistry)

	var myHandler genericPacketHandler

	packetBytesChannel := make(chan rawPacketBytes, config.PacketChannelSize)
	if config.DevName != "" && !config.UseAfpacket {
		myHandler = initializeLivePcap(config.DevName, config.Filter)
		log.Info("Waiting for packets")

	} else if config.DevName != "" && config.UseAfpacket {
		myHandler = config.initializeLiveAFpacket(config.DevName, config.Filter)
		log.Info("Waiting for packets using AFpacket")

	} else {
		myHandler = initializeOfflinePcap(config.PcapFile, config.Filter)
		log.Info("Reading off Pcap file")
	}

	defer myHandler.Close() // closes the packet handler and/or capture files. there might be a duplicate of this in pcapfile
	go func() {
		for {
			data, ci, err := myHandler.ReadPacketData()
			if err != nil && err.Error() == "EOF" {
				log.Warnf("EOF reached... exiting")
				os.Exit(0)
			}
			packetBytesChannel <- rawPacketBytes{data, ci}
		}
	}()

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
			if ratioCnt%config.ratioB < config.ratioA {
				if ratioCnt > config.ratioB*config.ratioA { //reset ratiocount before it goes to an absurdly high number
					ratioCnt = 0
				}
				totalCnt++
				config.processingChannel <- packetRawBytes
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
