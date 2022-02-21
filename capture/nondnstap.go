package capture

import (
	"time"

	"github.com/mosajjal/dnsmonster/util"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

func (config CaptureConfig) StartNonDnsTap() {

	packetsCaptured := metrics.GetOrRegisterGauge("packetsCaptured", metrics.DefaultRegistry)
	packetsDropped := metrics.GetOrRegisterGauge("packetsDropped", metrics.DefaultRegistry)
	packetsDuplicate := metrics.GetOrRegisterCounter("packetsDuplicate", metrics.DefaultRegistry)
	packetsOverRatio := metrics.GetOrRegisterCounter("packetsOverRatio", metrics.DefaultRegistry)
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
			log.Info("PacketSource returned nil, exiting (Possible end of pcap file?). Sleeping for 2 seconds waiting for processing to finish")
			time.Sleep(time.Second * 2)
			util.GeneralFlags.GetWg().Done()
			return
		}

		totalCnt++ // * there is a race condition between this and the metrics being captured at line 48 (packetsCaptured.Update(totalCnt))
		// ratio checks
		skipForRatio := false
		if config.ratioA != config.ratioB { // this confirms the ratio is in use
			ratioCnt++
			if ratioCnt%config.ratioB < config.ratioA {
				if ratioCnt > config.ratioB*config.ratioA { //reset ratiocount before it goes to an absurdly high number
					ratioCnt = 0
				}
				packetsOverRatio.Inc(1)
				skipForRatio = true
			}
		}

		// dedup checks
		skipForDudup := false
		if config.Dedup {
			hash := FNV1A(data)
			_, ok := config.dedupHashTable[hash] // check for existence
			if !ok {
				config.dedupHashTable[hash] = true
			} else {
				skipForDudup = true
				packetsDuplicate.Inc(1)
			}
		}

		if !skipForRatio && !skipForDudup {
			config.processingChannel <- &rawPacketBytes{data, ci}
		}

	}
}
