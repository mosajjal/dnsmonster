package capture

import (
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcapgo"
	log "github.com/sirupsen/logrus"
)

type pcapFileHandle struct {
	reader   *pcapgo.Reader
	file     *os.File
	pktsRead uint
}

func initializeOfflinePcap(fileName, filter string) *pcapFileHandle {
	var f *os.File
	if fileName == "-" {
		f = os.Stdin
	} else {
		var err error
		f, err = os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
	}
	handle, err := pcapgo.NewReader(f)

	// Set Filter
	log.Infof("Using File: %s", fileName)
	log.Warnf("BPF Filter is not supported in offline mode.")
	if err != nil {
		log.Fatal(err)
	}
	return &pcapFileHandle{handle, f, 0}
}

func (h *pcapFileHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = h.reader.ReadPacketData()
	if err != nil {
		h.pktsRead++
	}
	return
}

func (h *pcapFileHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = h.reader.ZeroCopyReadPacketData()
	if err != nil {
		h.pktsRead++
	}
	return
}

func (h *pcapFileHandle) Close() {
	h.file.Close()
}

func (h *pcapFileHandle) Stat() (uint, uint, error) {
	// `pcapgo.Reader` doesn't have a Stats() method, so we track packets
	// captured by ourselves. There should be no loss for a PCAP file since
	// it's controlled by I/O and not network
	return h.pktsRead, 0, nil
}
