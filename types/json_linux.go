//go:build linux && amd64
// +build linux,amd64

package types

import (
	"fmt"

	"github.com/bytedance/sonic"
)

func (d *DNSResult) String() string {
	res, _ := sonic.Marshal(d)
	return string(res)
}

func (d *DNSResult) Values() string {

	edns, doBit := uint8(0), uint8(0)
	if edns0 := d.DNS.IsEdns0(); edns0 != nil {
		edns = 1
		if edns0.Do() {
			doBit = 1
		}
	}

	// timestamp, IPVersion, SrcIP, DstIP, Protocol, OpCode, Class, Type, ResponseCode, Question, Size, Edns0Present, DoBit, ID
	return fmt.Sprintf("%s,%d,%s,%d,%s,%d,%d,%d,%d,%s,%d,%d,%d,%d\n", d.Timestamp, d.IPVersion, d.SrcIP, d.DstIP, d.Protocol, d.DNS.Opcode, d.DNS.Question[0].Qclass, d.DNS.Question[0].Qtype, d.DNS.Rcode, d.DNS.Question[0].String(), d.PacketLength, edns, doBit, d.DNS.Id)
}
