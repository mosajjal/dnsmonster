//go:build linux
// +build linux

package capture

import (
	"os"
	"time"

	"github.com/mosajjal/dnsmonster/util"
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
	pcapBPF := TcpdumpToPcapgoBpf(filter)
	log.Infof("Filter: %s", filter)
	err = h.TPacket.SetBPF(pcapBPF)
	if err != nil {
		util.ErrorHandler(err)
	}
	return err
}

func (h *afpacketHandle) Close() {
	h.TPacket.Close()
}

func afpacketComputeSize(targetSizeMb uint, snaplen uint, pageSize uint) (
	frameSize uint, blockSize uint, numBlocks uint, err error) {

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

func (config CaptureConfig) initializeLiveAFpacket(devName, filter string) *afpacketHandle {
	// Open device
	// var tPacket *afpacket.TPacket
	var err error
	handle := &afpacketHandle{}

	frameSize, blockSize, numBlocks, err := afpacketComputeSize(
		config.AfpacketBuffersizeMb,
		65536,
		uint(os.Getpagesize()))
	util.ErrorHandler(err)
	handle.TPacket, err = afpacket.NewTPacket(
		afpacket.OptInterface(devName),
		afpacket.OptFrameSize(frameSize),
		afpacket.OptBlockSize(blockSize),
		afpacket.OptNumBlocks(numBlocks),
		afpacket.OptPollTimeout(-10*time.Millisecond),
		afpacket.SocketRaw,
		afpacket.TPacketVersion3)
	util.ErrorHandler(err)

	handle.SetBPFFilter(filter, 1024)
	log.Infof("Opened: %s", devName)
	return handle
}

func (afhandle *afpacketHandle) Stat() (uint, uint) {
	mystats, statsv3, err := afhandle.TPacket.SocketStats()
	if err != nil {
		return uint(mystats.Packets() + statsv3.Packets()), uint(mystats.Drops() + statsv3.Drops())
	}
	return 0, 0
}
