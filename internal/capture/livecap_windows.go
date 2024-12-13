//go:build windows
// +build windows

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

import (
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/pcap"
)

type livePcapHandle struct {
	handle *pcap.Handle
}

func initializeLivePcap(devName, filter string) *livePcapHandle {
	handle, err := pcap.OpenLive(devName, 1600, true, pcap.BlockForever)
	if err != nil {
		panic(err)
	} else if err := handle.SetBPFFilter(filter); err != nil { // optional
		panic(err)
	}
	h := livePcapHandle{handle}
	return &h
}

func (h *livePcapHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.handle.ReadPacketData()
}

func (h *livePcapHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.handle.ZeroCopyReadPacketData()
}

func (h *livePcapHandle) Close() {
	h.handle.Close()
}

func (h *livePcapHandle) Stat() (uint, uint, error) {
	stats, err := h.handle.Stats()
	if err != nil {
		return 0, 0, err
	}
	return uint(stats.PacketsReceived), uint(stats.PacketsDropped), nil
}

// vim: foldmethod=marker
