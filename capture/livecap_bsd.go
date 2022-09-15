//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

package capture

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/bsdbpf"
	log "github.com/sirupsen/logrus"
)

type BsdHandle struct {
	sniffer bsdbpf.BPFSniffer
}

func initializeLivePcap(devName, filter string) *BsdHandle {
	// Open device

	var options = bsdbpf.Options{
		BPFDeviceName:    "",
		ReadBufLen:       32767,
		Timeout:          nil,
		Promisc:          !GlobalCaptureConfig.NoPromiscuous,
		Immediate:        true,
		PreserveLinkAddr: true,
	}

	handle, err := bsdbpf.NewBPFSniffer(devName, &options)
	// handle, err := pcap.OpenLive(devName, 65536, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}

	// Set Filter
	log.Infof("Using Device: %s", devName)
	log.Warnf("dnsmonster doesn't support BPF in BSD (yet)")
	// bpf := tcpdumpToPcapgoBpf(filter)
	// if bpf != nil {
	// 	log.Infof("Filter: %s", filter)
	// 	err = handle.SetBPF(bpf)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
	// h := livePcapHandle{handle}
	return &BsdHandle{*handle}
}

func (h *BsdHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = h.sniffer.ReadPacketData()
	// end of packet capture doesn't make sense for live interface
	// and our logic in the main for loop of nondnstap doesn't work
	// with the default bsd capture setup. have to do this instead
	if data == nil {
		data = []byte{1}
	}
	return data, ci, nil
}

func (h *BsdHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.sniffer.ReadPacketData()
}

func (h *BsdHandle) Close() {
	h.sniffer.Close()
}

func (h *BsdHandle) Stat() (uint, uint, error) {
	stats, err := h.handle.Stats()
	if err != nil {
		return uint(stats.Packets), uint(stats.Drops), nil
	}
	return 0, 0, err
}
