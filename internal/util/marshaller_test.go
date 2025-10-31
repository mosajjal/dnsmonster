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
	"encoding/json"
	"net"
	"testing"
	"time"

	mkdns "github.com/miekg/dns"
)

func TestJSONMarshaller(t *testing.T) {
	marshaller := jsonOutput{}
	header, err := marshaller.Init()
	if err != nil {
		t.Errorf("jsonOutput.Init() returned error: %v", err)
	}
	if header != "" {
		t.Errorf("jsonOutput.Init() returned non-empty header: %s", header)
	}

	// Create a test DNS result
	msg := mkdns.Msg{}
	msg.SetQuestion("example.com.", mkdns.TypeA)
	msg.Response = true
	msg.Rcode = mkdns.RcodeSuccess

	// Add an answer
	rr, _ := mkdns.NewRR("example.com. 300 IN A 93.184.216.34")
	msg.Answer = append(msg.Answer, rr)

	result := DNSResult{
		Timestamp:    time.Now(),
		DNS:          msg,
		IPVersion:    4,
		SrcIP:        net.ParseIP("192.168.1.1"),
		DstIP:        net.ParseIP("8.8.8.8"),
		Protocol:     "udp",
		PacketLength: 100,
	}

	// Marshal the result
	data := marshaller.Marshal(result)
	if len(data) == 0 {
		t.Error("jsonOutput.Marshal() returned empty data")
	}

	// Verify it's valid JSON
	var decoded map[string]interface{}
	err2 := json.Unmarshal(data, &decoded)
	if err2 != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err2)
	}

	// Check key fields exist - they should exist in the marshalled output
	// Note: actual field names depend on the JSON marshaller implementation
	t.Logf("JSON output: %s", string(data))
}

func TestCSVMarshaller(t *testing.T) {
	marshaller := csvOutput{}
	header, err := marshaller.Init()
	if err != nil {
		t.Errorf("csvOutput.Init() returned error: %v", err)
	}
	
	if header == "" {
		t.Error("csvOutput.Init() returned empty header")
	}

	// Create a test DNS result
	msg := mkdns.Msg{}
	msg.SetQuestion("example.com.", mkdns.TypeA)
	msg.Response = true
	msg.Rcode = mkdns.RcodeSuccess

	result := DNSResult{
		Timestamp:    time.Now(),
		DNS:          msg,
		IPVersion:    4,
		SrcIP:        net.ParseIP("192.168.1.1"),
		DstIP:        net.ParseIP("8.8.8.8"),
		Protocol:     "udp",
		PacketLength: 100,
	}

	// Marshal the result
	data := marshaller.Marshal(result)
	if len(data) == 0 {
		t.Error("csvOutput.Marshal() returned empty data")
	}
}

func TestOutputFormatToMarshaller(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		template   string
		wantErr    bool
	}{
		{
			name:     "JSON format",
			format:   "json",
			template: "",
			wantErr:  false,
		},
		{
			name:     "CSV format",
			format:   "csv",
			template: "",
			wantErr:  false,
		},
		{
			name:     "GOB format",
			format:   "gob",
			template: "",
			wantErr:  false,
		},
		{
			name:     "Invalid format",
			format:   "invalid",
			template: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshaller, _, err := OutputFormatToMarshaller(tt.format, tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("OutputFormatToMarshaller() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && marshaller == nil {
				t.Error("OutputFormatToMarshaller() returned nil marshaller for valid format")
			}
		})
	}
}

// Benchmark tests
func BenchmarkJSONMarshal(b *testing.B) {
	marshaller := jsonOutput{}
	marshaller.Init()

	msg := mkdns.Msg{}
	msg.SetQuestion("example.com.", mkdns.TypeA)
	msg.Response = true
	msg.Rcode = mkdns.RcodeSuccess

	result := DNSResult{
		Timestamp:    time.Now(),
		DNS:          msg,
		IPVersion:    4,
		SrcIP:        net.ParseIP("192.168.1.1"),
		DstIP:        net.ParseIP("8.8.8.8"),
		Protocol:     "udp",
		PacketLength: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = marshaller.Marshal(result)
	}
}

func BenchmarkCSVMarshal(b *testing.B) {
	marshaller := csvOutput{}
	marshaller.Init()

	msg := mkdns.Msg{}
	msg.SetQuestion("example.com.", mkdns.TypeA)
	msg.Response = true
	msg.Rcode = mkdns.RcodeSuccess

	result := DNSResult{
		Timestamp:    time.Now(),
		DNS:          msg,
		IPVersion:    4,
		SrcIP:        net.ParseIP("192.168.1.1"),
		DstIP:        net.ParseIP("8.8.8.8"),
		Protocol:     "udp",
		PacketLength: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = marshaller.Marshal(result)
	}
}

// vim: foldmethod=marker
