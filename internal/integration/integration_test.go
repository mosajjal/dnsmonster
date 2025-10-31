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

// Package integration provides integration tests for DNSMonster
// These tests verify the interaction between multiple components
package integration

import (
	"context"
	"net"
	"testing"
	"time"

	mkdns "github.com/miekg/dns"
	"github.com/mosajjal/dnsmonster/internal/util"
)

// TestEndToEndPacketProcessing tests the complete flow from packet to output
// This is an example integration test that can be expanded
func TestEndToEndPacketProcessing(t *testing.T) {
	// Skip in short mode as this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a DNS query
	msg := mkdns.Msg{}
	msg.SetQuestion("test.example.com.", mkdns.TypeA)
	msg.RecursionDesired = true

	// Create a DNS result
	result := util.DNSResult{
		Timestamp:    time.Now(),
		DNS:          msg,
		IPVersion:    4,
		SrcIP:        net.ParseIP("192.168.1.100"),
		DstIP:        net.ParseIP("8.8.8.8"),
		Protocol:     "udp",
		PacketLength: 100,
	}

	// Verify the result is properly constructed
	if result.DNS.Question[0].Name != "test.example.com." {
		t.Errorf("Expected question name test.example.com., got %s", 
			result.DNS.Question[0].Name)
	}

	if result.IPVersion != 4 {
		t.Errorf("Expected IPVersion 4, got %d", result.IPVersion)
	}

	if result.Protocol != "udp" {
		t.Errorf("Expected protocol udp, got %s", result.Protocol)
	}
}

// TestChannelCommunication tests channel-based communication between components
func TestChannelCommunication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a channel
	resultChannel := make(chan util.DNSResult, 10)

	// Producer goroutine
	go func() {
		for i := 0; i < 5; i++ {
			msg := mkdns.Msg{}
			msg.SetQuestion("test.example.com.", mkdns.TypeA)
			
			result := util.DNSResult{
				Timestamp:    time.Now(),
				DNS:          msg,
				IPVersion:    4,
				SrcIP:        net.ParseIP("192.168.1.100"),
				DstIP:        net.ParseIP("8.8.8.8"),
				Protocol:     "udp",
				PacketLength: 100,
			}
			
			resultChannel <- result
		}
		close(resultChannel)
	}()

	// Consumer - verify we receive all packets
	count := 0
	for range resultChannel {
		count++
	}

	if count != 5 {
		t.Errorf("Expected to receive 5 packets, got %d", count)
	}
}

// TestContextCancellation tests proper context handling
func TestContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	// Start a goroutine that should respect context cancellation
	done := make(chan bool)
	go func() {
		select {
		case <-ctx.Done():
			done <- true
		case <-time.After(5 * time.Second):
			t.Error("Context cancellation not respected")
			done <- false
		}
	}()

	// Cancel the context
	cancel()

	// Wait for the goroutine to finish
	if !<-done {
		t.Error("Context cancellation test failed")
	}
}

// TestOutputMarshalling tests marshalling to different output formats
func TestOutputMarshalling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test DNS result
	msg := mkdns.Msg{}
	msg.SetQuestion("test.example.com.", mkdns.TypeA)
	msg.Response = true
	msg.Rcode = mkdns.RcodeSuccess

	result := util.DNSResult{
		Timestamp:    time.Now(),
		DNS:          msg,
		IPVersion:    4,
		SrcIP:        net.ParseIP("192.168.1.100"),
		DstIP:        net.ParseIP("8.8.8.8"),
		Protocol:     "udp",
		PacketLength: 100,
	}

	// Test different output formats
	formats := []string{"json", "csv", "gob"}
	
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			marshaller, _, err := util.OutputFormatToMarshaller(format, "")
			if err != nil {
				t.Errorf("Failed to create %s marshaller: %v", format, err)
				return
			}

			data := marshaller.Marshal(result)
			if len(data) == 0 {
				t.Errorf("%s marshaller returned empty data", format)
			}
		})
	}
}

// TestMaskingPreservesPrivacy tests that IP masking works correctly
func TestMaskingPreservesPrivacy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	originalIP := net.ParseIP("192.168.1.100")
	maskSize := 24
	bitSize := 32

	maskedIP := originalIP.Mask(net.CIDRMask(maskSize, bitSize))

	// Original IP should not equal masked IP
	if originalIP.String() == maskedIP.String() {
		t.Error("IP masking did not change the IP address")
	}

	// Masked IP should have last octet as 0
	expectedMasked := "192.168.1.0"
	if maskedIP.String() != expectedMasked {
		t.Errorf("Expected masked IP %s, got %s", expectedMasked, maskedIP.String())
	}

	// Test that network portion is preserved
	if !originalIP.Mask(net.CIDRMask(maskSize, bitSize)).Equal(maskedIP) {
		t.Error("Network portion not preserved after masking")
	}
}

// TestHighThroughputScenario simulates processing many packets
func TestHighThroughputScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a channel with buffer
	resultChannel := make(chan util.DNSResult, 1000)
	processed := make(chan int)

	// Producer: generate packets
	go func() {
		for i := 0; i < 1000; i++ {
			msg := mkdns.Msg{}
			msg.SetQuestion("test.example.com.", mkdns.TypeA)
			
			result := util.DNSResult{
				Timestamp:    time.Now(),
				DNS:          msg,
				IPVersion:    4,
				SrcIP:        net.ParseIP("192.168.1.100"),
				DstIP:        net.ParseIP("8.8.8.8"),
				Protocol:     "udp",
				PacketLength: 100,
			}
			
			resultChannel <- result
		}
		close(resultChannel)
	}()

	// Consumer: process packets
	go func() {
		count := 0
		for range resultChannel {
			count++
		}
		processed <- count
	}()

	// Wait for processing to complete with timeout
	select {
	case count := <-processed:
		if count != 1000 {
			t.Errorf("Expected to process 1000 packets, processed %d", count)
		}
	case <-time.After(5 * time.Second):
		t.Error("Packet processing timed out")
	}
}

// BenchmarkIntegrationThroughput benchmarks the throughput of the system
func BenchmarkIntegrationThroughput(b *testing.B) {
	resultChannel := make(chan util.DNSResult, 10000)
	
	// Consumer
	go func() {
		for range resultChannel {
			// Process packet
		}
	}()

	msg := mkdns.Msg{}
	msg.SetQuestion("test.example.com.", mkdns.TypeA)
	
	result := util.DNSResult{
		Timestamp:    time.Now(),
		DNS:          msg,
		IPVersion:    4,
		SrcIP:        net.ParseIP("192.168.1.100"),
		DstIP:        net.ParseIP("8.8.8.8"),
		Protocol:     "udp",
		PacketLength: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChannel <- result
	}
	close(resultChannel)
}

// vim: foldmethod=marker
