//go:build linux
// +build linux

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
	"fmt"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/pcapgo"
	log "github.com/sirupsen/logrus"
)

type livePcapHandle struct {
	handle *pcapgo.EthernetHandle
}

func initializeLivePcap(devName, filter string) (*livePcapHandle, error) {
	// Open device
	handle, err := pcapgo.NewEthernetHandle(devName)
	// handle, err := pcap.OpenLive(devName, 65536, true, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("failed to open ethernet handle on %s: %w", devName, err)
	}
	err = handle.SetPromiscuous(!GlobalCaptureConfig.NoPromiscuous)
	if err != nil {
		handle.Close()
		return nil, fmt.Errorf("failed to set promiscuous mode: %w", err)
	}
	// Set Filter
	log.Infof("Using Device: %s", devName)
	bpf := tcpdumpToPcapgoBpf(filter)
	if bpf != nil {
		log.Infof("Filter: %s", filter)
		err = handle.SetBPF(bpf)
		if err != nil {
			handle.Close()
			return nil, fmt.Errorf("failed to set BPF filter: %w", err)
		}
	}
	h := livePcapHandle{handle}
	return &h, nil
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
	return uint(stats.Packets), uint(stats.Drops), nil
}

// vim: foldmethod=marker
