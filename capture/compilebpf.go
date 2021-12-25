//go:build !nolibpcap
// +build !nolibpcap

// this file's sole purpose is to convert a tcpdump filter into bpf bytecode

package capture

import (
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/bpf"
)

func TcpdumpToPcapgoBpf(filter string) []bpf.RawInstruction {
	returnByteCodes := []bpf.RawInstruction{}
	bytecodes, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, 1024, filter)
	for _, ins := range bytecodes {
		returnByteCodes = append(returnByteCodes, bpf.RawInstruction{ins.Code, ins.Jt, ins.Jf, ins.K})
	}
	if err != nil {
		return nil
	}
	return returnByteCodes
}
