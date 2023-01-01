//go:build !nolibpcap && !nocgo
// +build !nolibpcap,!nocgo

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
