//go:build linux && !android
// +build linux,!android

package capture

import (
	"os"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

type afpacketHandle struct {
	TPacket *afpacket.TPacket
}

func (h *afpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ReadPacketData()
}

func (h *afpacketHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ZeroCopyReadPacketData()
}

func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *afpacketHandle) SetBPFFilter(filter string, snaplen int) (err error) {
	pcapBPF := tcpdumpToPcapgoBpf(filter)
	// nil means the binary is compiled w/o bpf support
	if pcapBPF != nil {
		log.Infof("Filter: %s", filter)
		err = h.TPacket.SetBPF(pcapBPF)
		if err != nil {
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return err
}

func (h *afpacketHandle) Close() {
	h.TPacket.Close()
}

func afpacketComputeSize(targetSizeMb uint, snaplen uint, pageSize uint) (
	frameSize uint, blockSize uint, numBlocks uint, err error,
) {
	if snaplen < pageSize {
		frameSize = pageSize / (pageSize / snaplen)
	} else {
		frameSize = (snaplen/pageSize + 1) * pageSize
	}

	// 128 is the default from the gopacket library so just use that
	blockSize = frameSize * 128
	numBlocks = (targetSizeMb * 1024 * 1024) / blockSize

	if numBlocks == 0 {
		log.Info("Interface buffersize is too small")
		return 0, 0, 0, err
	}

	return frameSize, blockSize, numBlocks, nil
}

func (config captureConfig) setPromiscuous() error {
	var err error
	if !config.NoPromiscuous {
		// TODO: replace with x/net/bpf or pcap
		err = syscall.SetLsfPromisc(config.DevName, !config.NoPromiscuous)
		log.Infof("Promiscuous mode: %v", !config.NoPromiscuous)
	}
	return err
}

func (config captureConfig) initializeLiveAFpacket(devName, filter string) *afpacketHandle {
	// Open device
	// var tPacket *afpacket.TPacket
	var err error
	handle := &afpacketHandle{}
	frameSize, blockSize, numBlocks, err := afpacketComputeSize(
		config.AfpacketBuffersizeMb,
		65536,
		uint(os.Getpagesize()))
	if err != nil {
		log.Fatal(err)
	}
	handle.TPacket, err = afpacket.NewTPacket(
		afpacket.OptInterface(devName),
		afpacket.OptFrameSize(frameSize),
		afpacket.OptBlockSize(blockSize),
		afpacket.OptNumBlocks(numBlocks),
		afpacket.OptPollTimeout(-10*time.Millisecond),
		afpacket.SocketRaw,
		afpacket.TPacketVersion3)
	if err != nil {
		log.Fatal(err)
	}
	err = handle.SetBPFFilter(filter, 1024)
	if err != nil {
		log.Fatal("Error setting BPF filter.. exiting")
	}
	// set up promisc mode. first we need to get the fd for the interface we just opened. using a hacky mode
	// v := reflect.ValueOf(handle.TPacket)
	// fd := v.FieldByName("fd").Int()
	err = config.setPromiscuous()
	if err != nil {
		log.Fatal("Error setting the interface to promiscuous.. exiting")
	}
	log.Infof("Opened: %s", devName)
	return handle
}

func (h *afpacketHandle) Stat() (uint, uint) {
	mystats, statsv3, err := h.TPacket.SocketStats()
	if err != nil {
		return uint(mystats.Packets() + statsv3.Packets()), uint(mystats.Drops() + statsv3.Drops())
	}
	return 0, 0
}
