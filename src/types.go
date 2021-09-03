package main

import (
	"container/list"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	mkdns "github.com/miekg/dns"
	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
)

type generalConfig struct {
	exiting             chan bool
	wg                  *sync.WaitGroup
	maskSize            int
	packetLimit         int
	saveFullQuery       bool
	serverName          string
	printStatsDelay     time.Duration
	skipTlsVerification bool
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

type kafkaConfig struct {
	resultChannel     chan DNSResult
	kafkaOutputBroker string
	kafkaOutputTopic  string
	kafkaOutputType   uint
	kafkaBatchSize    uint
	kafkaBatchDelay   time.Duration
	general           generalConfig
}

type splunkConfig struct {
	resultChannel          chan DNSResult
	splunkOutputEndpoints  []string
	splunkOutputToken      string
	splunkOutputType       uint
	splunkOutputIndex      string
	splunkOutputSource     string
	splunkOutputSourceType string
	splunkBatchSize        uint
	splunkBatchDelay       time.Duration
	general                generalConfig
}

type syslogConfig struct {
	resultChannel        chan DNSResult
	syslogOutputEndpoint string
	syslogOutputType     uint
	general              generalConfig
}

type fileConfig struct {
	resultChannel  chan DNSResult
	fileOutputPath string
	fileOutputType uint
	general        generalConfig
}

type stdoutConfig struct {
	resultChannel    chan DNSResult
	stdoutOutputType uint
	general          generalConfig
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
	NoEthernetframe   bool
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
	NoEthernetframe              bool
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

// DNSCapturer oobject is used to make our configuration portable within the entire code
type DNSCapturer struct {
	options    CaptureOptions
	processing chan gopacket.Packet
}

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

type outputStats struct {
	Name         string
	SentToOutput int
	Skipped      int
}

// captureStats is capturing statistics about our current live captures. At this point it's not accurate for PCAP files.
type captureStats struct {
	PacketsGot        int
	PacketsLost       int
	PacketLossPercent float32
}

// ipv6 is a struct to be used as a key.
type ipv6 struct {
	ip4 gopacket.Flow
	id  uint32
}

// fragmentList holds a container/list used to contains IP
// packets/fragments.  It stores internal counters to track the
// maximum total of byte, and the current length it has received.
// It also stores a flag to know if he has seen the last packet.
type fragmentList struct {
	List          list.List
	Highest       uint16
	Current       uint16
	FinalReceived bool
	LastSeen      time.Time
}

// IPv6Defragmenter is a struct which embedded a map of
// all fragment/packet.
type IPv6Defragmenter struct {
	sync.RWMutex
	ipFlows map[ipv6]*fragmentList
}

type splunkConnection struct {
	client    *splunk.Client
	unhealthy uint
	err       error
}

// Register a new Layer to detect IPv4 and IPv6 packets without an ethernet frame.
var LayerTypeDetectIP = gopacket.RegisterLayerType(250, gopacket.LayerTypeMetadata{Name: "DetectIP", Decoder: nil})

type DetectIP struct {
	layers.BaseLayer
	family layers.EthernetType
}
