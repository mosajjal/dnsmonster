//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

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
	"github.com/gopacket/gopacket/bsdbpf"
	log "github.com/sirupsen/logrus"
)

type BsdHandle struct {
	sniffer    bsdbpf.BPFSniffer
	readCnt    uint
	droppedCnt uint
}

func initializeLivePcap(devName, filter string) *BsdHandle {
	// Open device

	var options = bsdbpf.Options{
		BPFDeviceName:    "",
		ReadBufLen:       32767,
		Timeout:          nil,
		Promisc:          !GlobalCaptureConfig.NoPromiscuous,
		Immediate:        true,
		PreserveLinkAddr: true,
	}

	handle, err := bsdbpf.NewBPFSniffer(devName, &options)
	// handle, err := pcap.OpenLive(devName, 65536, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}

	// Set Filter
	log.Infof("Using Device: %s", devName)
	log.Warnf("dnsmonster doesn't support BPF in BSD (yet)")
	// bpf := tcpdumpToPcapgoBpf(filter)
	// if bpf != nil {
	// 	log.Infof("Filter: %s", filter)
	// 	err = handle.SetBPF(bpf)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
	// h := livePcapHandle{handle}
	return &BsdHandle{*handle, 0, 0}
}

func (h *BsdHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = h.sniffer.ReadPacketData()
	// end of packet capture doesn't make sense for live interface
	// and our logic in the main for loop of nondnstap doesn't work
	// with the default bsd capture setup. have to do this instead
	if data == nil {
		data = []byte{1}
		h.droppedCnt++
	}
	if err != nil {
		h.droppedCnt++
	} else {
		h.readCnt++
	}
	return data, ci, nil
}

func (h *BsdHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.sniffer.ReadPacketData()
}

func (h *BsdHandle) Close() {
	h.sniffer.Close()
}

func (h *BsdHandle) Stat() (uint, uint, error) {
	return h.readCnt, h.droppedCnt, nil
}

// vim: foldmethod=marker
