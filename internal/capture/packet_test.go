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
