// +build !linux

package main

import (
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type afpacketHandle struct {
}

func newAfpacketHandle(device string, snaplen int, blockSize int, numBlocks int,
	timeout time.Duration, enableAutoPromiscMode bool) (*afpacketHandle, error) {

	return nil, fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *afpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *afpacketHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *afpacketHandle) Close() {
}

func updateAfpacketStats(afhandle *afpacketHandle) {
	myStats.PacketsGot = 0
	myStats.PacketsLost = 0
}

func initializeLiveAFpacket(devName, filter string) *afpacketHandle {
	return nil
}
