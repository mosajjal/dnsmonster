//go:build !linux
// +build !linux

package capture

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

type livePcapHandle struct {
	handle *pcap.Handle
}

func initializeLivePcap(devName, filter string) *livePcapHandle {
	handle, err := pcap.OpenLive(devName, 1600, true, pcap.BlockForever)
	if err != nil {
		panic(err)
	} else if err := handle.SetBPFFilter(filter); err != nil { // optional
		panic(err)
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
	h.Close()
}

func (h *livePcapHandle) Stat() (uint, uint) {
	// in printstats, we check if this is 0, and we add the total counter to this to make sure we have a better number
	// in essence, there should be 0 packet loss for a pcap file since the rate of the packet is controlled by i/o not network
	stats, err := h.handle.Stats()
	if err != nil {
		return uint(stats.PacketsReceived), uint(stats.PacketsDropped)
	}
	return 0, 0
}
