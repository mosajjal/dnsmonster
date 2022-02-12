package capture

import (
	"container/list"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"github.com/mosajjal/dnsmonster/util"
)

type CaptureConfig struct {
	DevName                    string `long:"devName"                    env:"DNSMONSTER_DEVNAME"                    default:""                                                                                                  description:"Device used to capture"`
	PcapFile                   string `long:"pcapFile"                   env:"DNSMONSTER_PCAPFILE"                   default:""                                                                                                  description:"Pcap filename to run"`
	DnstapSocket               string `long:"dnstapSocket"               env:"DNSMONSTER_DNSTAPSOCKET"               default:""                                                                                                  description:"dnstrap socket path. Example: unix:///tmp/dnstap.sock, tcp://127.0.0.1:8080"`
	Port                       uint   `long:"port"                       env:"DNSMONSTER_PORT"                       default:"53"                                                                                                description:"Port selected to filter packets"`
	SampleRatio                string `long:"sampleRatio"                env:"DNSMONSTER_SAMPLERATIO"                default:"1:1"                                                                                               description:"Capture Sampling by a:b. eg sampleRatio of 1:100 will process 1 percent of the incoming packets"`
	DnstapPermission           string `long:"dnstapPermission"           env:"DNSMONSTER_DNSTAPPERMISSION"           default:"755"                                                                                               description:"Set the dnstap socket permission, only applicable when unix:// is used"`
	PacketHandlerCount         uint   `long:"packetHandlerCount"         env:"DNSMONSTER_PACKETHANDLERCOUNT"         default:"2"                                                                                                 description:"Number of routines used to handle received packets"`
	TcpAssemblyChannelSize     uint   `long:"tcpAssemblyChannelSize"     env:"DNSMONSTER_TCPASSEMBLYCHANNELSIZE"     default:"10000"                                                                                             description:"Size of the tcp assembler"`
	TcpResultChannelSize       uint   `long:"tcpResultChannelSize"       env:"DNSMONSTER_TCPRESULTCHANNELSIZE"       default:"10000"                                                                                             description:"Size of the tcp result channel"`
	TcpHandlerCount            uint   `long:"tcpHandlerCount"            env:"DNSMONSTER_TCPHANDLERCOUNT"            default:"1"                                                                                                 description:"Number of routines used to handle tcp assembly"`
	DefraggerChannelSize       uint   `long:"defraggerChannelSize"       env:"DNSMONSTER_DEFRAGGERCHANNELSIZE"       default:"10000"                                                                                             description:"Size of the channel to send packets to be defragged"`
	DefraggerChannelReturnSize uint   `long:"defraggerChannelReturnSize" env:"DNSMONSTER_DEFRAGGERCHANNELRETURNSIZE" default:"10000"                                                                                             description:"Size of the channel where the defragged packets are returned"`
	PacketChannelSize          uint   `long:"packetChannelSize"          env:"DNSMONSTER_PACKETCHANNELSIZE"          default:"1000"                                                                                              description:"Size of the packet handler channel"`
	AfpacketBuffersizeMb       uint   `long:"afpacketBuffersizeMb"       env:"DNSMONSTER_AFPACKETBUFFERSIZEMB"       default:"64"                                                                                                description:"Afpacket Buffersize in MB"`
	Filter                     string `long:"filter"                     env:"DNSMONSTER_FILTER"                     default:"((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))" description:"BPF filter applied to the packet stream. If port is selected, the packets will not be defragged."`
	UseAfpacket                bool   `long:"useAfpacket"                env:"DNSMONSTER_USEAFPACKET"                description:"Use AFPacket for live captures. Supported on Linux 3.0+ only"`
	NoEthernetframe            bool   `long:"noEtherframe"               env:"DNSMONSTER_NOETHERFRAME"               description:"The PCAP capture does not contain ethernet frames"`
	processingChannel          chan *rawPacketBytes
	ip4Defrgger                chan ipv4ToDefrag
	ip6Defrgger                chan ipv6FragmentInfo
	ip4DefrggerReturn          chan ipv4Defragged
	ip6DefrggerReturn          chan ipv6Defragged
	tcpAssembly                chan tcpPacket
	tcpReturnChannel           chan tcpData
	resultChannel              chan util.DNSResult
	ratioA                     int
	ratioB                     int
	// input                <-chan rawPacketBytes
}

// this function will run at import time, before parsing the flags
func (config CaptureConfig) initializeFlags() error {
	_, err := util.GlobalParser.AddGroup("capture", "Options specific to capture side", &config)
	GlobalCaptureConfig = &config
	config.resultChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	config.tcpAssembly = make(chan tcpPacket, config.TcpAssemblyChannelSize)
	config.tcpReturnChannel = make(chan tcpData, config.TcpResultChannelSize)
	config.processingChannel = make(chan *rawPacketBytes, config.PacketChannelSize)
	config.ip4Defrgger = make(chan ipv4ToDefrag, config.DefraggerChannelSize)
	config.ip6Defrgger = make(chan ipv6FragmentInfo, config.DefraggerChannelSize)
	config.ip4DefrggerReturn = make(chan ipv4Defragged, config.DefraggerChannelReturnSize)
	config.ip6DefrggerReturn = make(chan ipv6Defragged, config.DefraggerChannelReturnSize)

	return err
}

func (config CaptureConfig) GetResultChannel() *chan util.DNSResult {
	return &config.resultChannel
}

func (config CaptureConfig) CheckFlagsAndStart() {
	if config.Port > 65535 {
		log.Fatal("--port must be between 1 and 65535")
	}
	if config.DevName == "" && config.PcapFile == "" && config.DnstapSocket == "" {
		log.Fatal("one of --devName, --pcapFile or --dnstapSocket is required")
	}

	if config.DevName != "" {
		if config.PcapFile != "" || config.DnstapSocket != "" {
			log.Fatal("You must set only --devName, --pcapFile or --dnstapSocket")
		}
	} else {
		if config.PcapFile != "" && config.DnstapSocket != "" {
			log.Fatal("You must set only --devName, --pcapFile or --dnstapSocket")
		}
	}

	if config.DnstapSocket != "" {
		if !strings.HasPrefix(config.DnstapSocket, "unix://") && !strings.HasPrefix(config.DnstapSocket, "tcp://") {
			log.Fatal("You must provide a unix:// or tcp:// socket for dnstap")
		}
	}
	ratioNumbers := strings.Split(config.SampleRatio, ":")
	if len(ratioNumbers) != 2 {
		log.Fatal("wrong --sampleRatio syntax")
	}
	var errA error
	var errB error
	config.ratioA, errA = strconv.Atoi(ratioNumbers[0])
	config.ratioB, errB = strconv.Atoi(ratioNumbers[1])
	if errA != nil || errB != nil || config.ratioA > config.ratioB {
		log.Fatal("wrong --sampleRatio syntax")
	}

	// start the defrag goroutines
	for i := uint(0); i < config.TcpHandlerCount; i++ {
		go tcpAssembler(config.tcpAssembly, config.tcpReturnChannel, util.GeneralFlags.GcTime)
	}
	go ipv4Defragger(config.ip4Defrgger, config.ip4DefrggerReturn, util.GeneralFlags.GcTime)
	go ipv6Defragger(config.ip6Defrgger, config.ip6DefrggerReturn, util.GeneralFlags.GcTime)

	// start the packet decoder goroutines
	util.GeneralFlags.GetWg().Add(1)
	go config.StartPacketDecoder()

	// Start listening if we're not using DNSTap
	if config.DnstapSocket == "" {
		go config.StartNonDnsTap()
	} else {
		// dnstap is totally different, hence only the result channel is being pushed to it
		go config.StartDnsTap()
	}
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

// an interface to unify different types of packet capture.
// right now, most functionality of afpacket, pcap file and libpcap
// are captured in this interface
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

// This will allow an instance to be spawned at import time
var GlobalCaptureConfig *CaptureConfig
var _ = CaptureConfig{}.initializeFlags()
