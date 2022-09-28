package capture

import (
	"context"
	"time"

	"github.com/mosajjal/dnsmonster/util"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (config captureConfig) StartNonDNSTap(ctx context.Context) error {
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

	ratioCnt := 0

	// updating the metrics in a separate goroutine

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			select {
			case <-captureStatsTicker.C:
				packets, drop, err := myHandler.Stat()
				if err != nil {
					log.Warnf("Error reading stats: %s", err)
					continue
				}

				packetsCaptured.Update(int64(packets))
				packetsDropped.Update(int64(drop))

				if packetsCaptured.Value() > 0 {
					packetLossPercent.Update(float64(packetsDropped.Value()) * 100.0 / float64(packetsCaptured.Value()))
				}
			case <-gCtx.Done():
				log.Debug("exitting out of metric update goroutine") //todo:remove
				return nil
			}
		}
	})

	// blocking loop to capture packets and send them to processing channel
	// todo: should this be blocking or have a gCtx.Done() listener somewhere
	for {
		data, ci, err := myHandler.ReadPacketData() // todo: ZeroCopyReadPacketData is slower than ReadPacketData. need to investigate why
		if data == nil || err != nil {
			log.Info("PacketSource returned nil, exiting (Possible end of pcap file?). Sleeping for 2 seconds waiting for processing to finish")
			time.Sleep(time.Second * 2)
			//todo: commence clean exit from errorgroup
			util.GlobalCancel()
			config.cleanExit(ctx)
			return nil
		}

		// ratio checks
		skipForRatio := false
		if config.ratioA != config.ratioB { // this confirms the ratio is in use
			ratioCnt++
			if ratioCnt%config.ratioB > config.ratioA {
				if ratioCnt > config.ratioB*config.ratioA { // reset ratiocount before it goes to an absurdly high number
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
