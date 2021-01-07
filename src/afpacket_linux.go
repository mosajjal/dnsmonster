package main

import (
	"log"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/bpf"
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
	pcapBPF, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, snaplen, filter)
	if err != nil {
		return err
	}
	bpfIns := []bpf.RawInstruction{}
	for _, ins := range pcapBPF {
		bpfIns2 := bpf.RawInstruction{
			Op: ins.Code,
			Jt: ins.Jt,
			Jf: ins.Jf,
			K:  ins.K,
		}
		bpfIns = append(bpfIns, bpfIns2)
	}
	if h.TPacket.SetBPF(bpfIns); err != nil {
		return err
	}
	return h.TPacket.SetBPF(bpfIns)
}

func (h *afpacketHandle) Close() {

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
		log.Println("Interface buffersize is too small")
		return 0, 0, 0, err
	}

	return frameSize, blockSize, numBlocks, nil
}

func initializeLiveAFpacket(devName, filter string) *afpacketHandle {
	// Open device
	// var tPacket *afpacket.TPacket
	var err error
	handle := &afpacketHandle{}

	frameSize, blockSize, numBlocks, err := afpacketComputeSize(
		*afPacketBuffersizeMb,
		65536,
		uint(os.Getpagesize()))
	if err != nil {
		log.Fatalf("Error calculating afpacket size: %s", err)
	}

	handle.TPacket, err = afpacket.NewTPacket(
		afpacket.OptInterface(devName),
		afpacket.OptFrameSize(frameSize),
		afpacket.OptBlockSize(blockSize),
		afpacket.OptNumBlocks(numBlocks),
		afpacket.OptPollTimeout(pcap.BlockForever),
		afpacket.SocketRaw,
		afpacket.TPacketVersion3)
	if err != nil {
		log.Fatalf("Error opening afpacket interface: %s", err)
	}

	handle.SetBPFFilter(filter, 1024)

	return handle
}

func updateAfpacketStats(afhandle *afpacketHandle) {
	mystats, statsv3, _ := afhandle.TPacket.SocketStats()
	myStats.PacketsGot = int(mystats.Packets() + statsv3.Packets())
	myStats.PacketsLost = int(mystats.Drops() + statsv3.Drops())
}
