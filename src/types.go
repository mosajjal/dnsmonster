package main

import (
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/tcpassembly/tcpreader"
)

type generalConfig struct {
	exiting         chan bool
	wg              *sync.WaitGroup
	maskSize        int
	packetLimit     int
	saveFullQuery   bool
	serverName      string
	printStatsDelay time.Duration
}

type clickHouseConfig struct {
	resultChannel        chan DNSResult
	clickhouseAddress    string
	clickhouseBatchSize  uint
	clickhouseOutputType uint
	clickhouseDebug      bool
	clickhouseDelay      time.Duration
	general              generalConfig
}

type elasticConfig struct {
	resultChannel         chan DNSResult
	elasticOutputEndpoint string
	elasticOutputIndex    string
	elasticOutputType     uint
	elasticBatchSize      uint
	elasticBatchDelay     time.Duration
	general               generalConfig
}

type packetEncoder struct {
	port              uint16
	input             <-chan gopacket.Packet
	ip4Defrgger       chan<- ipv4ToDefrag
	ip6Defrgger       chan<- ipv6FragmentInfo
	ip4DefrggerReturn <-chan ipv4Defragged
	ip6DefrggerReturn <-chan ipv6Defragged
	tcpAssembly       []chan tcpPacket
	tcpReturnChannel  <-chan tcpData
	resultChannel     chan<- DNSResult
	done              chan bool
}

// CaptureOptions is a set of generated options variables to use within our capture routine
type CaptureOptions struct {
	DevName                      string
	useAfpacket                  bool
	PcapFile                     string
	Filter                       string
	Port                         uint16
	GcTime                       time.Duration
	ResultChannel                chan<- DNSResult
	PacketHandlerCount           uint
	PacketChannelSize            uint
	TCPHandlerCount              uint
	TCPAssemblyChannelSize       uint
	TCPResultChannelSize         uint
	IPDefraggerChannelSize       uint
	IPDefraggerReturnChannelSize uint
	Done                         chan bool
}

type ipv4ToDefrag struct {
	ip        layers.IPv4
	timestamp time.Time
}

type ipv4Defragged struct {
	ip        layers.IPv4
	timestamp time.Time
}

type ipv6FragmentInfo struct {
	ip         layers.IPv6
	ipFragment layers.IPv6Fragment
	timestamp  time.Time
}

type ipv6Defragged struct {
	ip        layers.IPv6
	timestamp time.Time
}

type tcpPacket struct {
	IPVersion uint8
	tcp       layers.TCP
	timestamp time.Time
	flow      gopacket.Flow
}

type tcpData struct {
	IPVersion uint8
	data      []byte
	SrcIP     net.IP
	DstIP     net.IP
	timestamp time.Time
}

type dnsStreamFactory struct {
	tcpReturnChannel chan tcpData
	IPVersion        uint8
	currentTimestamp time.Time
}

type dnsStream struct {
	Net              gopacket.Flow
	reader           tcpreader.ReaderStream
	tcpReturnChannel chan tcpData
	IPVersion        uint8
	timestamp        time.Time
}
