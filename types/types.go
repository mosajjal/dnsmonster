package types

import (
	"net"
	"time"

	mkdns "github.com/miekg/dns"
	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
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

type GeneralConfig struct {
	MaskSize4           int
	MaskSize6           int
	PacketLimit         int
	ServerName          string
	SkipTlsVerification bool
}

type SplunkConnection struct {
	Client    *splunk.Client
	Unhealthy uint
	Err       error
}
type GenericOutput interface {
	Initialize() error             // try to initialize the output by checking flags and connections
	Output()                       // the output is a goroutine that fetches data from the registered channel and pushes it to output, possibly in multiple workers
	OutputChannel() chan DNSResult //returns the output channel associated with the output
	Close()                        // close down the connections and exit cleanly
}

var GlobalDispatchList = make([]GenericOutput, 0, 1024) // 1024 outputs is an absurdly high number
