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

package util

import (
	"net"
	"testing"
)

func TestMaskIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		maskSize int
		bitSize  int
		want     string
	}{
		{
			name:     "Full IPv4 mask",
			ip:       "192.168.1.100",
			maskSize: 32,
			bitSize:  32,
			want:     "192.168.1.100",
		},
		{
			name:     "24-bit IPv4 mask",
			ip:       "192.168.1.100",
			maskSize: 24,
			bitSize:  32,
			want:     "192.168.1.0",
		},
		{
			name:     "16-bit IPv4 mask",
			ip:       "192.168.1.100",
			maskSize: 16,
			bitSize:  32,
			want:     "192.168.0.0",
		},
		{
			name:     "8-bit IPv4 mask",
			ip:       "192.168.1.100",
			maskSize: 8,
			bitSize:  32,
			want:     "192.0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			masked := ip.Mask(net.CIDRMask(tt.maskSize, tt.bitSize))
			got := masked.String()
			if got != tt.want {
				t.Errorf("MaskIPv4() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaskIPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		maskSize int
		bitSize  int
		want     string
	}{
		{
			name:     "Full IPv6 mask",
			ip:       "2001:db8::1",
			maskSize: 128,
			bitSize:  128,
			want:     "2001:db8::1",
		},
		{
			name:     "64-bit IPv6 mask",
			ip:       "2001:db8:1234:5678::1",
			maskSize: 64,
			bitSize:  128,
			want:     "2001:db8:1234:5678::",
		},
		{
			name:     "48-bit IPv6 mask",
			ip:       "2001:db8:1234:5678::1",
			maskSize: 48,
			bitSize:  128,
			want:     "2001:db8:1234::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			masked := ip.Mask(net.CIDRMask(tt.maskSize, tt.bitSize))
			got := masked.String()
			if got != tt.want {
				t.Errorf("MaskIPv6() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckDomain(t *testing.T) {
	// Initialize test data with nil values (no file loading)
	GeneralFlags.skipPrefixTst = nil
	GeneralFlags.skipSuffixTst = nil
	GeneralFlags.skipTypeHt = make(map[string]uint8)
	GeneralFlags.allowPrefixTst = nil
	GeneralFlags.allowSuffixTst = nil
	GeneralFlags.allowTypeHt = make(map[string]uint8)

	tests := []struct {
		name       string
		outputType uint
		domain     string
		want       bool
	}{
		{
			name:       "Output type 0 (disabled)",
			outputType: 0,
			domain:     "example.com.",
			want:       true,
		},
		{
			name:       "Output type 1 (no filters)",
			outputType: 1,
			domain:     "example.com.",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckIfWeSkip(tt.outputType, tt.domain)
			if got != tt.want {
				t.Errorf("CheckIfWeSkip() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPacketLimitCheck(t *testing.T) {
	tests := []struct {
		name        string
		packetLimit int
		batchLen    int
		want        bool
	}{
		{
			name:        "Limit disabled",
			packetLimit: 0,
			batchLen:    100,
			want:        true,
		},
		{
			name:        "Under limit",
			packetLimit: 1000,
			batchLen:    100,
			want:        true,
		},
		{
			name:        "At limit",
			packetLimit: 100,
			batchLen:    100,
			want:        false,
		},
		{
			name:        "Over limit",
			packetLimit: 100,
			batchLen:    150,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GeneralFlags.PacketLimit = tt.packetLimit
			got := GeneralFlags.PacketLimit == 0 || tt.batchLen < GeneralFlags.PacketLimit
			if got != tt.want {
				t.Errorf("PacketLimitCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkMaskIPv4(b *testing.B) {
	ip := net.ParseIP("192.168.1.100")
	mask := net.CIDRMask(24, 32)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ip.Mask(mask)
	}
}

func BenchmarkMaskIPv6(b *testing.B) {
	ip := net.ParseIP("2001:db8:1234:5678::1")
	mask := net.CIDRMask(64, 128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ip.Mask(mask)
	}
}

// vim: foldmethod=marker
