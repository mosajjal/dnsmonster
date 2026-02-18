//go:build !linux || android || nocgo

/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

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
	return nil, fmt.Errorf("Dnsmonster has been compiled without afpacket support for this platform")
}

func (h *afpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("Dnsmonster has been compiled without afpacket support for this platform")
}

func (h *afpacketHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("Dnsmonster has been compiled without afpacket support for this platform")
}

func (h *afpacketHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("Dnsmonster has been compiled without afpacket support for this platform")
}

func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *afpacketHandle) Close() {
}

func (afhandle *afpacketHandle) Stat() (uint, uint, error) {
	return 0, 0, fmt.Errorf("Dnsmonster has been compiled without afpacket support for this platform")
}

func (config captureConfig) initializeLiveAFpacket(devName, filter string) *afpacketHandle {
	return nil
}

// vim: foldmethod=marker
