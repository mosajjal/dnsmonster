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

package util

import (
	"bytes"
	"encoding/gob"
	"net"
	"time"
)

type gobOutput struct{}
type DNSResultBinary struct {
	Timestamp    time.Time
	Server	     string
	DNS          []byte //packed version of dns.msg (dns.Msg.Pack())
	IPVersion    uint8
	SrcIP        net.IP
	SrcPort      uint16 `json:",omitempty"`
	DstIP        net.IP
	DstPort      uint16 `json:",omitempty"`
	Protocol     string
	PacketLength uint16
	Identity     string `json:",omitempty"`
	Version      string `json:",omitempty"`
}

func (g gobOutput) Marshal(d DNSResult) []byte {
	d.DNS.Compress = true
	bMsg, _ := d.DNS.Pack()
	dnsBin := DNSResultBinary{
		Timestamp:    d.Timestamp,
		Server:	      d.Server,
		DNS:          bMsg,
		IPVersion:    d.IPVersion,
		SrcIP:        d.SrcIP,
		SrcPort:      d.SrcPort,
		DstIP:        d.DstIP,
		DstPort:      d.DstPort,
		Protocol:     d.Protocol,
		PacketLength: d.PacketLength,
		Identity:     d.Identity,
		Version:      d.Version,
	}
	// convert to gob
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(dnsBin); err != nil {
		return nil
	}
	return b.Bytes()
}

func (g gobOutput) Init() (string, error) {
	gob.Register(DNSResultBinary{})
	return "", nil
}
// vim: foldmethod=marker
