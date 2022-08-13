//go:build linux
// +build linux

package capture

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcapgo"
	log "github.com/sirupsen/logrus"
)

type livePcapHandle struct {
	handle *pcapgo.EthernetHandle
}

func initializeLivePcap(devName, filter string) *livePcapHandle {
	// Open device
	handle, err := pcapgo.NewEthernetHandle(devName)
	// handle, err := pcap.OpenLive(devName, 65536, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	err = handle.SetPromiscuous(!GlobalCaptureConfig.NoPromiscuous)
	if err != nil {
		log.Fatal("Error setting interface to promiscuous.. Exiting")
	}
	// Set Filter
	log.Infof("Using Device: %s", devName)
	bpf := tcpdumpToPcapgoBpf(filter)
	if bpf != nil {
		log.Infof("Filter: %s", filter)
		err = handle.SetBPF(bpf)
		if err != nil {
			log.Fatal(err)
		}
	}
	h := livePcapHandle{handle}
	return &h
}

func (h *livePcapHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.handle.ReadPacketData()
}

func (h *livePcapHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.handle.ZeroCopyReadPacketData()
}

func (h *livePcapHandle) Close() {
	h.handle.Close()
}

func (h *livePcapHandle) Stat() (uint, uint) {
	// in printstats, we check if this is 0, and we add the total counter to this to make sure we have a better number
	// in essence, there should be 0 packet loss for a pcap file since the rate of the packet is controlled by i/o not network
	stats, err := h.handle.Stats()
	if err != nil {
		return uint(stats.Packets), uint(stats.Drops)
	}
	return 0, 0
}
