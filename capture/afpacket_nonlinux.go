//go:build !linux || android
// +build !linux android

package capture

// This entire file is a dummy one to make sure all our cross platform builds work even if the underlying OS doesn't suppot some of the functionality
// afpacket is a Linux-only feature, so we want the relevant function to technically "translate" to something here, which basically returns an error

import (
	"fmt"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

type afpacketHandle struct{}

func newAfpacketHandle(device string, snaplen int, blockSize int, numBlocks int,
	timeout time.Duration, enableAutoPromiscMode bool,
) (*afpacketHandle, error) {
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
func (h *afpacketHandle) Name() string {
	return ""
}
func (afhandle *afpacketHandle) Stat() (uint, uint, error) {
	return 0, 0, fmt.Errorf("Afpacket statistics are only available on Linux")
}

func (config captureConfig) initializeLiveAFpacket(devName, filter string) *afpacketHandle {
	return nil
}
