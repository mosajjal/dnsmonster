package types

import (
	"net"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	mkdns "github.com/miekg/dns"
)

// DNSResult is the middleware that connects the packet encoder to Clickhouse.
// For DNStap, this is probably going to be replaced with something else.
type DNSResult struct {
	Timestamp    time.Time
	DNS          mkdns.Msg
	IPVersion    uint8
	SrcIP        net.IP
	DstIP        net.IP
	Protocol     string
	PacketLength uint16
}

func (d *DNSResult) String() string {
	res, _ := sonic.Marshal(d)
	return string(res)
}

// Setup output routine
var GlobalExitChannel = make(chan bool)
var GlobalWaitingGroup sync.WaitGroup
