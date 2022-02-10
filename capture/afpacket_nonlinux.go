//go:build !linux
// +build !linux

package capture

// This entire file is a dummy one to make sure all our cross platform builds work even if the underlying OS doesn't suppot some of the functionality
// afpacket is a Linux-only feature, so we want the relevant function to technically "translate" to something here, which basically returns an error

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

func (h *afpacketHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
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

func (afhandle *afpacketHandle) Stat() (uint, uint) {
	return 0, 0
}

func (config CaptureConfig) initializeLiveAFpacket(devName, filter string) *afpacketHandle {
	return nil
}
