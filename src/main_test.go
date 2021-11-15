package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"runtime/trace"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mosajjal/dnsmonster/types"
)

var packetSampleHex = "48df377637b8e48184e3747986dd600cbde600f611362600900053045b0000000000000000012403980000110040000000000000000100359b5800f6cc57f91384000001000200040001096b656570616c69766508676f74696e64657203636f6d0000010001c00c0005000100000001000c096170692d756531617ac016c0340005000100000001000e03617069016305756531617ac016c016000200010002a3000017076e732d3131313509617773646e732d3131036f726700c016000200010002a3000019076e732d3139393009617773646e732d353602636f02756b00c016000200010002a3000013066e732d33333509617773646e732d3431c01fc016000200010002a3000016066e732d37353109617773646e732d3239036e6574000000291000000000000000"
var packetSampleBytes, _ = hex.DecodeString(packetSampleHex)
var packetSample = gopacket.NewPacket(packetSampleBytes, layers.LayerTypeEthernet, gopacket.Default)

const (
	TCPResultChannelSize         = 60000
	PacketChannelSize            = 60000
	IPDefraggerChannelSize       = 60000
	IPDefraggerReturnChannelSize = 60000
	TCPAssemblyChannelSize       = 60000
	ResultChannelSize            = 60000
	NoEthernetframe              = false
)

var testReturnChannel = make(chan tcpData, TCPResultChannelSize)
var testInputChannel = make(chan gopacket.Packet, PacketChannelSize)
var testIp4DefraggerChannel = make(chan ipv4ToDefrag, IPDefraggerChannelSize)
var testIp6DefraggerChannel = make(chan ipv6FragmentInfo, IPDefraggerChannelSize)
var testIp4DefraggerReturn = make(chan ipv4Defragged, IPDefraggerReturnChannelSize)
var testIp6DefraggerReturn = make(chan ipv6Defragged, IPDefraggerReturnChannelSize)
var testResultChannel = make(chan types.DNSResult, ResultChannelSize)
var testDoneChannel = make(chan bool)
var testTcpChannels []chan tcpPacket

var cnt = 0

func DummySink() {
	for {
		select {
		case <-testResultChannel:
			cnt++
		}
	}
}

func benchmarkUdpPacketProcessingWorker(workers uint, b *testing.B) {
	fmt.Println("Benchmarking UDP Packet Processing Worker")
	file, _ := os.Create("trace.out")
	trace.Start(file)
	defer trace.Stop()
	e := packetEncoder{
		53,
		testInputChannel,
		testIp4DefraggerChannel,
		testIp6DefraggerChannel,
		testIp4DefraggerReturn,
		testIp6DefraggerReturn,
		testTcpChannels,
		testReturnChannel,
		testResultChannel,
		workers,
		false,
	}
	go e.run()
	testTcpChannels = append(testTcpChannels, make(chan tcpPacket, TCPAssemblyChannelSize))
	go DummySink()

	for n := 0; n < b.N; n++ {
		testInputChannel <- packetSample
	}
}

func BenchmarkUdpPacketProcessingWorker1(b *testing.B) { benchmarkUdpPacketProcessingWorker(1, b) }

// func BenchmarkUdpPacketProcessingWorker2(b *testing.B)  { benchmarkUdpPacketProcessingWorker(2, b) }
// func BenchmarkUdpPacketProcessingWorker4(b *testing.B)  { benchmarkUdpPacketProcessingWorker(4, b) }
// func BenchmarkUdpPacketProcessingWorker6(b *testing.B)  { benchmarkUdpPacketProcessingWorker(6, b) }
// func BenchmarkUdpPacketProcessingWorker8(b *testing.B)  { benchmarkUdpPacketProcessingWorker(8, b) }
// func BenchmarkUdpPacketProcessingWorker16(b *testing.B) { benchmarkUdpPacketProcessingWorker(16, b) }
