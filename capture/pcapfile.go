package capture

import (
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcapgo"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type pcapFileHandle struct {
	reader *pcapgo.Reader
	file   *os.File
}

func initializeOfflinePcap(fileName, filter string) *pcapFileHandle {
	f, err := os.Open(fileName)
	// defer f.Close() //todo: find where to close the file. in here doesn't work
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
func (h *pcapFileHandle) Stats() (*unix.TpacketStats, error) {
	//todo: this needs to be implemented correctly
	tpacketStats := unix.TpacketStats{0, 0}
	return &tpacketStats, nil
}
