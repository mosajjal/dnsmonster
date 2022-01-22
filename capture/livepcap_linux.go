//go:build linux
// +build linux

package capture

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcapgo"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

type livePcapHandle struct {
	handle *pcapgo.EthernetHandle
}

func initializeLivePcap(devName, filter string) *livePcapHandle {
	// Open device
	handle, err := pcapgo.NewEthernetHandle(devName)
	// handle, err := pcap.OpenLive(devName, 65536, true, pcap.BlockForever)
	util.ErrorHandler(err)

	// Set Filter
	log.Infof("Using Device: %s", devName)
	log.Infof("Filter: %s", filter)
	err = handle.SetBPF(TcpdumpToPcapgoBpf(filter))
	util.ErrorHandler(err)
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
