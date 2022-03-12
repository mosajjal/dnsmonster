//go:build nolibpcap
// +build nolibpcap

// this file's sole purpose is to convert a tcpdump filter into bpf bytecode

package capture

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/bpf"
)

func TcpdumpToPcapgoBpf(filter string) []bpf.RawInstruction {
	log.Warnf("dnsmonster has been compiled without libpcap support. BPF filters are not supported.")
	return nil
}
