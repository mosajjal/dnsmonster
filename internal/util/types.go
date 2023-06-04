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
	"context"
	"net"
	"time"

	mkdns "github.com/miekg/dns"
)

// DNSResult is the middleware that connects the packet encoder to Any output.
// For DNStap, this is probably going to be replaced with something else.
type DNSResult struct {
	Timestamp    time.Time
	DNS          mkdns.Msg
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

// GenericOutput is an interface to speficy the behaviour of output modules
// and make it extendable
type GenericOutput interface {
	Initialize(context.Context) error // try to initialize the output by checking flags and connections
	Output(context.Context)           // the output is a goroutine that fetches data from the registered channel and pushes it to output, possibly in multiple workers
	OutputChannel() chan DNSResult    // returns the output channel associated with the output
	Close()                           // close down the connections and exit cleanly
}

// OutputMarshaller is an interface to make it easier to build
// output formats regardless of the output.
type OutputMarshaller interface {
	Marshal(d DNSResult) []byte // marshal the DNSResult into the output format
	Init() (string, error)      // initialize the output format
}

// GlobalDispatchList acts as a fanout mechanism, sending the dnsresult channel to all the outputs
var GlobalDispatchList = make([]GenericOutput, 0, 1024) // 1024 outputs is an absurdly high number
// vim: foldmethod=marker
