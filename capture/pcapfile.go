package capture

import (
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcapgo"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

type pcapFileHandle struct {
	reader *pcapgo.Reader
	file   *os.File
}

func initializeOfflinePcap(fileName, filter string) *pcapFileHandle {
	f, err := os.Open(fileName)
	util.ErrorHandler(err)
	handle, err := pcapgo.NewReader(f)

	// Set Filter
	log.Infof("Using File: %s", fileName)
	log.Warnf("BPF Filter is not supported in offline mode.")
	util.ErrorHandler(err)
	return &pcapFileHandle{handle, f}
}
func (h *pcapFileHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.reader.ReadPacketData()
}
func (h *pcapFileHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.reader.ZeroCopyReadPacketData()
}

func (h *pcapFileHandle) Close() {
	h.file.Close()
}

func (h *pcapFileHandle) Stat() (uint, uint) {
	// in printstats, we check if this is 0, and we add the total counter to this to make sure we have a better number
	// in essence, there should be 0 packet loss for a pcap file since the rate of the packet is controlled by i/o not network
	return 0, 0
}
