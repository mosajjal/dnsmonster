//go:build nolibpcap || nocgo

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
	gopcap "github.com/packetcap/go-pcap/filter"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/bpf"
)

func tcpdumpToPcapgoBpf(filter string) []bpf.RawInstruction {
	log.Warnf("dnsmonster has been compiled without libpcap support. some advance bpf filters maybe unsupported.")

	instructions, err := gopcap.NewExpression(filter).Compile().Compile()
	if err != nil {
		log.Errorf("failed to compile filter: %s", err)
		return nil
	}
	rawInstructions := make([]bpf.RawInstruction, len(instructions))
	for i, inst := range instructions {
		rawInstructions[i], _ = inst.Assemble()
	}
	return rawInstructions
}

// vim: foldmethod=marker
