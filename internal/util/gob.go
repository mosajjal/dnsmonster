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

func (g gobOutput) Marshal(d DNSResult) string {
	d.DNS.Compress = true
	bMsg, _ := d.DNS.Pack()
	dnsBin := DNSResultBinary{
		Timestamp:    d.Timestamp,
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
		return ""
	}
	return b.String()
}

func (g gobOutput) Init() (string, error) {
	gob.Register(DNSResultBinary{})
	return "", nil
}
