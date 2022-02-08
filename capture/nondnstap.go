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

	// Set up various tickers for different tasks
	captureStatsTicker := time.NewTicker(util.GeneralFlags.CaptureStatsDelay)

	var ratioCnt = 0
	var totalCnt = int64(0)

	// updating the metrics in a separate goroutine
	go func() {
		for range captureStatsTicker.C {

			packets, drop := myHandler.Stat()
			if packets == 0 { // to make up for pcap not being able to get stats
				packetsCaptured.Update(totalCnt)
			} else {
				packetsCaptured.Update(int64(packets))
				packetsCaptured.Update(int64(drop))
			}
			packetLossPercent.Update(float64(packetsDropped.Value()) * 100.0 / float64(packetsCaptured.Value()))
		}
	}()

	// blocking loop to capture packets and send them to processing channel
	for {
		data, ci, err := myHandler.ReadPacketData() //todo: ZeroCopyReadPacketData is slower than ReadPacketData. need to investigate why
		if data == nil || err != nil {
			log.Info("PacketSource returned nil, exiting (Possible end of pcap file?). Sleeping for 10 seconds waiting for processing to finish")
			time.Sleep(time.Second * 10)
			os.Exit(0)
		}

		if config.ratioA != config.ratioB { // this confirms the ratio is in use
			ratioCnt++
			if ratioCnt%config.ratioB < config.ratioA {
				if ratioCnt > config.ratioB*config.ratioA { //reset ratiocount before it goes to an absurdly high number
					ratioCnt = 0
				}
				totalCnt++
				config.processingChannel <- &rawPacketBytes{data, ci}
			}
		} else { // always pass the data through if there's no ratio logic
			config.processingChannel <- &rawPacketBytes{data, ci}
		}
	}

}
