package capture

import (
	"container/list"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"github.com/mosajjal/dnsmonster/types"
)

type packetEncoder struct {
	port              uint16
	input             <-chan rawPacketBytes
	ip4Defrgger       chan<- ipv4ToDefrag
	ip6Defrgger       chan<- ipv6FragmentInfo
	ip4DefrggerReturn <-chan ipv4Defragged
	ip6DefrggerReturn <-chan ipv6Defragged
	tcpAssembly       []chan tcpPacket
	tcpReturnChannel  <-chan tcpData
	resultChannel     chan<- types.DNSResult
	handlerCount      uint
	NoEthernetframe   bool
}

// CaptureOptions is a set of generated options variables to use within our capture routine
type CaptureOptions struct {
	DevName                      string
	UseAfpacket                  bool
	PcapFile                     string
	Filter                       string
	Port                         uint16
	GcTime                       time.Duration
	ResultChannel                chan<- types.DNSResult
	PacketHandlerCount           uint
	PacketChannelSize            uint
	TCPHandlerCount              uint
	TCPAssemblyChannelSize       uint
	TCPResultChannelSize         uint
	IPDefraggerChannelSize       uint
	IPDefraggerReturnChannelSize uint
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

// DNSCapturer object is used to make our configuration portable within the entire code
type DNSCapturer struct {
	options    CaptureOptions
	processing chan rawPacketBytes
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

// Register a new Layer to detect IPv4 and IPv6 packets without an ethernet frame.
var LayerTypeDetectIP = gopacket.RegisterLayerType(250, gopacket.LayerTypeMetadata{Name: "DetectIP", Decoder: nil})

type DetectIP struct {
	layers.BaseLayer
	family layers.EthernetType
}

type genericPacketHandler interface {
	ReadPacketData() ([]byte, gopacket.CaptureInfo, error)
	ZeroCopyReadPacketData() ([]byte, gopacket.CaptureInfo, error)
	Close()
	Stat() (uint, uint)
}

type rawPacketBytes struct {
	bytes []byte
	info  gopacket.CaptureInfo
}
