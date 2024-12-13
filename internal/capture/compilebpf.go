//go:build !nolibpcap && !nocgo
// +build !nolibpcap,!nocgo

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

// this file's sole purpose is to convert a tcpdump filter into bpf bytecode

package capture

import (
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"golang.org/x/net/bpf"
)

func tcpdumpToPcapgoBpf(filter string) []bpf.RawInstruction {
	returnByteCodes := []bpf.RawInstruction{}
	bytecodes, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, 1024, filter)
	for _, ins := range bytecodes {
		returnByteCodes = append(returnByteCodes, bpf.RawInstruction{Op: ins.Code, Jt: ins.Jt, Jf: ins.Jf, K: ins.K})
	}
	if err != nil {
		return nil
	}
	return returnByteCodes
}

// vim: foldmethod=marker
