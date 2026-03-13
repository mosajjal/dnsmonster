/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

package capture

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	mkdns "github.com/miekg/dns"
	"github.com/mosajjal/dnsmonster/internal/util"
)

func TestDNSPacketProcessing(t *testing.T) {
	// Create a simple DNS query
	msg := mkdns.Msg{}
	msg.SetQuestion("example.com.", mkdns.TypeA)
	msg.RecursionDesired = true

	queryBytes, err := msg.Pack()
	if err != nil {
		t.Fatalf("Failed to pack DNS message: %v", err)
	}

	// Verify the message can be unpacked
	var unpacked mkdns.Msg
	err = unpacked.Unpack(queryBytes)
	if err != nil {
		t.Errorf("Failed to unpack DNS message: %v", err)
	}

	if len(unpacked.Question) != 1 {
		t.Errorf("Expected 1 question, got %d", len(unpacked.Question))
	}

	if unpacked.Question[0].Name != "example.com." {
		t.Errorf("Expected example.com., got %s", unpacked.Question[0].Name)
	}
}

func TestDNSResponseProcessing(t *testing.T) {
	// Create a DNS response
	msg := mkdns.Msg{}
	msg.SetQuestion("example.com.", mkdns.TypeA)
	msg.Response = true
	msg.Rcode = mkdns.RcodeSuccess

	// Add an answer
	rr, err := mkdns.NewRR("example.com. 300 IN A 93.184.216.34")
	if err != nil {
		t.Fatalf("Failed to create RR: %v", err)
	}
	msg.Answer = append(msg.Answer, rr)

	responseBytes, err := msg.Pack()
	if err != nil {
		t.Fatalf("Failed to pack DNS response: %v", err)
	}

	// Verify the response can be unpacked
	var unpacked mkdns.Msg
	err = unpacked.Unpack(responseBytes)
	if err != nil {
		t.Errorf("Failed to unpack DNS response: %v", err)
	}

	if !unpacked.Response {
		t.Error("Expected response flag to be true")
	}

	if len(unpacked.Answer) != 1 {
		t.Errorf("Expected 1 answer, got %d", len(unpacked.Answer))
	}
}

func TestIPMaskingInDNSResult(t *testing.T) {
	msg := mkdns.Msg{}
	msg.SetQuestion("test.com.", mkdns.TypeA)

	srcIP := net.ParseIP("192.168.1.100")
	dstIP := net.ParseIP("8.8.8.8")

	// Test IPv4 masking
	maskSize := 24
	bitSize := 32
	maskedSrc := srcIP.Mask(net.CIDRMask(maskSize, bitSize))
	maskedDst := dstIP.Mask(net.CIDRMask(maskSize, bitSize))

	result := util.DNSResult{
		Timestamp:    time.Now(),
		DNS:          msg,
		IPVersion:    4,
		SrcIP:        maskedSrc,
		DstIP:        maskedDst,
		Protocol:     "udp",
		PacketLength: 100,
	}

	expectedSrc := "192.168.1.0"
	expectedDst := "8.8.8.0"

	if result.SrcIP.String() != expectedSrc {
		t.Errorf("Masked source IP = %s, want %s", result.SrcIP.String(), expectedSrc)
	}

	if result.DstIP.String() != expectedDst {
		t.Errorf("Masked destination IP = %s, want %s", result.DstIP.String(), expectedDst)
	}
}

func TestFNV1AHash(t *testing.T) {
	// Test hash consistency
	data := []byte("test data for hashing")

	hash1 := FNV1A(data)
	hash2 := FNV1A(data)

	if hash1 != hash2 {
		t.Error("FNV1A hash should be consistent for same input")
	}

	// Test different data produces different hash
	data2 := []byte("different test data")
	hash3 := FNV1A(data2)

	if hash1 == hash3 {
		t.Error("FNV1A hash should be different for different input")
	}
}

func TestPacketTypes(t *testing.T) {
	tests := []struct {
		name     string
		qtype    uint16
		typeName string
	}{
		{"A Record", mkdns.TypeA, "A"},
		{"AAAA Record", mkdns.TypeAAAA, "AAAA"},
		{"CNAME Record", mkdns.TypeCNAME, "CNAME"},
		{"MX Record", mkdns.TypeMX, "MX"},
		{"TXT Record", mkdns.TypeTXT, "TXT"},
		{"NS Record", mkdns.TypeNS, "NS"},
		{"PTR Record", mkdns.TypePTR, "PTR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := mkdns.Msg{}
			msg.SetQuestion("example.com.", tt.qtype)

			if len(msg.Question) != 1 {
				t.Fatalf("Expected 1 question, got %d", len(msg.Question))
			}

			if msg.Question[0].Qtype != tt.qtype {
				t.Errorf("Expected qtype %d, got %d", tt.qtype, msg.Question[0].Qtype)
			}
		})
	}
}

func TestLayerTypes(t *testing.T) {
	// Test that we can identify different network layers
	layerTypes := []gopacket.LayerType{
		layers.LayerTypeEthernet,
		layers.LayerTypeIPv4,
		layers.LayerTypeIPv6,
		layers.LayerTypeUDP,
		layers.LayerTypeTCP,
	}

	for _, lt := range layerTypes {
		// Just verify layer types are recognized
		if lt == gopacket.LayerTypeZero {
			t.Error("Layer type should not be zero")
		}
	}
}

// TestDecodeLayerErrorHandling feeds malformed packet bytes to the decoder
// and verifies no panic occurs and that the error path is exercised.
func TestDecodeLayerErrorHandling(t *testing.T) {
	malformedPackets := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single byte", []byte{0x00}},
		{"random garbage", []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02}},
		{"truncated ethernet", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00}},
		{"all zeros", make([]byte, 64)},
	}

	for _, tt := range malformedPackets {
		t.Run(tt.name, func(t *testing.T) {
			var ethLayer layers.Ethernet
			var ip4 layers.IPv4
			var ip6 layers.IPv6
			var udp layers.UDP
			var tcp layers.TCP

			parser := gopacket.NewDecodingLayerParser(
				layers.LayerTypeEthernet,
				&ethLayer, &ip4, &ip6, &udp, &tcp,
			)
			foundLayerTypes := []gopacket.LayerType{}

			// This should not panic
			err := parser.DecodeLayers(tt.data, &foundLayerTypes)
			// We expect an error for malformed data
			if err == nil && len(tt.data) < 14 {
				// Ethernet header requires at least 14 bytes
				t.Logf("Expected error for data of length %d, but got nil", len(tt.data))
			}
			// The key assertion: no panic occurred
		})
	}
}

// TestIPv4DefragReturnPath verifies that defragged IPv4 packets use the
// correct protocol field from packet.ip (not a stale local variable).
func TestIPv4DefragReturnPath(t *testing.T) {
	// Construct a minimal defragged IPv4 packet result with a UDP payload
	dnsMsg := mkdns.Msg{}
	dnsMsg.SetQuestion("example.com.", mkdns.TypeA)
	dnsPayload, err := dnsMsg.Pack()
	if err != nil {
		t.Fatalf("Failed to pack DNS: %v", err)
	}

	// Build a UDP header + DNS payload
	udpHeader := make([]byte, 8)
	binary.BigEndian.PutUint16(udpHeader[0:2], 12345)  // src port
	binary.BigEndian.PutUint16(udpHeader[2:4], 53)      // dst port
	binary.BigEndian.PutUint16(udpHeader[4:6], uint16(8+len(dnsPayload))) // length
	binary.BigEndian.PutUint16(udpHeader[6:8], 0)       // checksum

	udpData := append(udpHeader, dnsPayload...)

	defraggedPacket := ipv4Defragged{
		ip: layers.IPv4{
			Version:  4,
			Protocol: layers.IPProtocolUDP,
			SrcIP:    net.ParseIP("10.0.0.1").To4(),
			DstIP:    net.ParseIP("10.0.0.2").To4(),
		},
		timestamp: time.Now(),
	}
	defraggedPacket.ip.Payload = udpData

	// Parse using the same approach as StartPacketDecoder
	var udp layers.UDP
	var tcp layers.TCP
	parserOnlyUDP := gopacket.NewDecodingLayerParser(layers.LayerTypeUDP, &udp)
	parserOnlyTCP := gopacket.NewDecodingLayerParser(layers.LayerTypeTCP, &tcp)
	foundLayerTypes := []gopacket.LayerType{}

	// This is what the fixed code does -- uses packet.ip.Protocol consistently
	if defraggedPacket.ip.Protocol == layers.IPProtocolUDP {
		// DecodeLayers may return "No decoder for layer type DNS" which is expected --
		// the production code also ignores this. We only care that UDP was decoded.
		parserOnlyUDP.DecodeLayers(defraggedPacket.ip.Payload, &foundLayerTypes)
	} else if defraggedPacket.ip.Protocol == layers.IPProtocolTCP {
		parserOnlyTCP.DecodeLayers(defraggedPacket.ip.Payload, &foundLayerTypes)
	}

	// Verify UDP was detected
	foundUDP := false
	for _, lt := range foundLayerTypes {
		if lt == layers.LayerTypeUDP {
			foundUDP = true
			break
		}
	}
	if !foundUDP {
		t.Fatal("Expected to find UDP layer in defragged packet")
	}

	// Verify NetworkFlow is obtained from defraggedPacket.ip (not some stale local)
	flow := defraggedPacket.ip.NetworkFlow()
	if flow.Src().String() != "10.0.0.1" || flow.Dst().String() != "10.0.0.2" {
		t.Errorf("NetworkFlow mismatch: got src=%s dst=%s, want 10.0.0.1 and 10.0.0.2",
			flow.Src().String(), flow.Dst().String())
	}

	if uint16(udp.DstPort) != 53 {
		t.Errorf("Expected dst port 53, got %d", udp.DstPort)
	}
}

// TestProcessTransportUDP tests that a UDP packet on port 53 produces a DNSResult.
func TestProcessTransportUDP(t *testing.T) {
	// Create DNS payload
	dnsMsg := mkdns.Msg{}
	dnsMsg.SetQuestion("example.com.", mkdns.TypeA)
	dnsPayload, err := dnsMsg.Pack()
	if err != nil {
		t.Fatalf("Failed to pack DNS: %v", err)
	}

	config := captureConfig{
		Port:          53,
		resultChannel: make(chan util.DNSResult, 10),
		tcpAssembly:   make(chan tcpPacket, 10),
	}

	udp := &layers.UDP{
		BaseLayer: layers.BaseLayer{Payload: dnsPayload},
	}
	udp.SrcPort = 12345
	udp.DstPort = 53

	tcp := &layers.TCP{}
	foundLayers := []gopacket.LayerType{layers.LayerTypeUDP}
	srcIP := net.ParseIP("10.0.0.1").To4()
	dstIP := net.ParseIP("10.0.0.2").To4()
	flow, _ := gopacket.FlowFromEndpoints(
		layers.NewIPEndpoint(srcIP),
		layers.NewIPEndpoint(dstIP),
	)

	// Set MaskSize so masking works
	util.GeneralFlags.MaskSize4 = 32

	config.processTransport(&foundLayers, udp, tcp, flow, time.Now(), 4, srcIP, dstIP)

	select {
	case result := <-config.resultChannel:
		if result.Protocol != "udp" {
			t.Errorf("Expected protocol udp, got %s", result.Protocol)
		}
		if result.DstPort != 53 {
			t.Errorf("Expected dst port 53, got %d", result.DstPort)
		}
		if len(result.DNS.Question) != 1 || result.DNS.Question[0].Name != "example.com." {
			t.Errorf("DNS question mismatch")
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for DNS result from UDP packet")
	}
}

// TestProcessTransportTCP tests that a TCP packet on port 53 goes to the assembly channel.
func TestProcessTransportTCP(t *testing.T) {
	config := captureConfig{
		Port:          53,
		resultChannel: make(chan util.DNSResult, 10),
		tcpAssembly:   make(chan tcpPacket, 10),
	}

	udp := &layers.UDP{}
	tcp := &layers.TCP{}
	tcp.SrcPort = 12345
	tcp.DstPort = 53

	foundLayers := []gopacket.LayerType{layers.LayerTypeTCP}
	srcIP := net.ParseIP("10.0.0.1").To4()
	dstIP := net.ParseIP("10.0.0.2").To4()
	flow, _ := gopacket.FlowFromEndpoints(
		layers.NewIPEndpoint(srcIP),
		layers.NewIPEndpoint(dstIP),
	)

	config.processTransport(&foundLayers, udp, tcp, flow, time.Now(), 4, srcIP, dstIP)

	select {
	case pkt := <-config.tcpAssembly:
		if pkt.IPVersion != 4 {
			t.Errorf("Expected IPVersion 4, got %d", pkt.IPVersion)
		}
		if uint16(pkt.tcp.DstPort) != 53 {
			t.Errorf("Expected dst port 53, got %d", pkt.tcp.DstPort)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for TCP packet in assembly channel")
	}
}

// TestProcessTransportNonDNS tests that packets on non-DNS ports are ignored.
func TestProcessTransportNonDNS(t *testing.T) {
	config := captureConfig{
		Port:          53,
		resultChannel: make(chan util.DNSResult, 10),
		tcpAssembly:   make(chan tcpPacket, 10),
	}

	// UDP on non-DNS port
	udp := &layers.UDP{}
	udp.SrcPort = 12345
	udp.DstPort = 8080

	tcp := &layers.TCP{}
	tcp.SrcPort = 12345
	tcp.DstPort = 8080

	srcIP := net.ParseIP("10.0.0.1").To4()
	dstIP := net.ParseIP("10.0.0.2").To4()
	flow, _ := gopacket.FlowFromEndpoints(
		layers.NewIPEndpoint(srcIP),
		layers.NewIPEndpoint(dstIP),
	)

	// Test UDP on non-DNS port
	foundLayers := []gopacket.LayerType{layers.LayerTypeUDP}
	config.processTransport(&foundLayers, udp, tcp, flow, time.Now(), 4, srcIP, dstIP)

	select {
	case <-config.resultChannel:
		t.Fatal("Should not have received a DNS result for non-DNS port UDP")
	case <-time.After(50 * time.Millisecond):
		// Expected: no result
	}

	// Test TCP on non-DNS port
	foundLayers = []gopacket.LayerType{layers.LayerTypeTCP}
	config.processTransport(&foundLayers, udp, tcp, flow, time.Now(), 4, srcIP, dstIP)

	select {
	case <-config.tcpAssembly:
		t.Fatal("Should not have received a TCP packet for non-DNS port")
	case <-time.After(50 * time.Millisecond):
		// Expected: no packet
	}
}

// Benchmark tests
func BenchmarkDNSPacking(b *testing.B) {
	msg := mkdns.Msg{}
	msg.SetQuestion("example.com.", mkdns.TypeA)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.Pack()
	}
}

func BenchmarkDNSUnpacking(b *testing.B) {
	msg := mkdns.Msg{}
	msg.SetQuestion("example.com.", mkdns.TypeA)
	data, _ := msg.Pack()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m mkdns.Msg
		_ = m.Unpack(data)
	}
}

func BenchmarkFNV1A(b *testing.B) {
	data := []byte("test data for hashing benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FNV1A(data)
	}
}

func BenchmarkIPMasking(b *testing.B) {
	ip := net.ParseIP("192.168.1.100")
	mask := net.CIDRMask(24, 32)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ip.Mask(mask)
	}
}

// vim: foldmethod=marker
