//go:build windows
// +build windows

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
	h.handle.Close()
}

func (h *livePcapHandle) Stat() (uint, uint, error) {
	stats, err := h.handle.Stats()
	if err != nil {
		return uint(stats.PacketsReceived), uint(stats.PacketsDropped), nil
	}
	return 0, 0, err
}
