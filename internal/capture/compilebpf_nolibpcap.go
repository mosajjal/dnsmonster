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

//go:build nolibpcap || nocgo
// +build nolibpcap nocgo

// this file's sole purpose is to convert a tcpdump filter into bpf bytecode

package capture

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/bpf"
)

func tcpdumpToPcapgoBpf(filter string) []bpf.RawInstruction {
	log.Warnf("dnsmonster has been compiled without libpcap support. tcpdump-style BPF filters are not directly supported.")
	log.Warnf("to generate a filter, use tcpdump and unix ulitities like so:")
	log.Warnf(`tcpdump -ddd "port 53 and not vlan 1024" | gzip -9 | base64 -w0`)
	// H4sIAAAAAAAAA3WO0Q0AIQhD/5nCEaRW9PZf7EDU3M9FE+HZFkBhLXEUAvV3lsaOLpwLowZGCNpShMZSqsPv8XeuX0bZLlxKhZuDpgseynkHtP8B85Pvi9hTLKg+KjpGrk0ZONUO8kmHnU2DWeYYlNxNlRfV0U3mAQEAAA==
	log.Warnf("then provide the output base64 as a filter to dnsmonster")
	returnByteCodes := []bpf.RawInstruction{}
	z, err := base64.StdEncoding.DecodeString(filter)
	if err != nil {
		log.Warnf("invalid base64 input, ignoring. error: %s", err)
		return nil
	}
	r, err := gzip.NewReader(bytes.NewReader(z))
	if err != nil {
		log.Warnf("invalid gzip input, ignoring. error: %s", err)
		return nil
	}
	input, _ := ioutil.ReadAll(r)
	for _, line := range strings.Split(string(input), "\n") {
		// skip empty line
		if line == "" {
			continue
		}
		instruction := bpf.RawInstruction{}
		instructs := strings.Split(line, " ")
		// should be at least 1 per line. first line has one, the others have 4 each
		if t, err := strconv.ParseUint(instructs[0], 10, 16); err != nil {
			log.Warnf("invalid instructions %d, ignoring. err: %s", t, err)
			return nil
		} else {
			instruction.Op = uint16(t)
		}
		if len(instructs) == 4 {
			if t, err := strconv.ParseUint(instructs[1], 10, 8); err != nil {
				log.Warnf("invalid instructions, ignoring")
				return nil
			} else {
				instruction.Jt = uint8(t)
			}
			if t, err := strconv.ParseUint(instructs[2], 10, 8); err != nil {
				log.Warnf("invalid instructions, ignoring")
				return nil
			} else {
				instruction.Jf = uint8(t)
			}
			if t, err := strconv.ParseUint(instructs[3], 10, 32); err != nil {
				log.Warnf("invalid instructions, ignoring")
				return nil
			} else {
				instruction.K = uint32(t)
			}
			returnByteCodes = append(returnByteCodes, instruction)
		}

	}

	return returnByteCodes
}
// vim: foldmethod=marker
