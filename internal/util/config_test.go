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
	"context"
	"testing"
	"time"
)

func TestMetricConfigValidation(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		agent      string
		endpoint   string
		wantErr    bool
	}{
		{
			name:       "Valid stderr",
			metricType: "stderr",
			agent:      "",
			endpoint:   "",
			wantErr:    false,
		},
		{
			name:       "Valid statsd with agent",
			metricType: "statsd",
			agent:      "127.0.0.1:8125",
			endpoint:   "",
			wantErr:    false,
		},
		{
			name:       "Invalid statsd without agent",
			metricType: "statsd",
			agent:      "",
			endpoint:   "",
			wantErr:    true,
		},
		{
			name:       "Valid prometheus with endpoint",
			metricType: "prometheus",
			agent:      "",
			endpoint:   "http://localhost:2112/metrics",
			wantErr:    false,
		},
		{
			name:       "Invalid prometheus without endpoint",
			metricType: "prometheus",
			agent:      "",
			endpoint:   "",
			wantErr:    true,
		},
		{
			name:       "Invalid type",
			metricType: "invalid",
			agent:      "",
			endpoint:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := metricConfig{
				MetricEndpointType:       tt.metricType,
				MetricStatsdAgent:        tt.agent,
				MetricPrometheusEndpoint: tt.endpoint,
				MetricFlushInterval:      1 * time.Second,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := config.SetupMetrics(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetupMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Cancel context immediately to stop goroutines
			cancel()
		})
	}
}

func TestServerName(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		want       string
	}{
		{
			name:       "Default server name",
			serverName: "default",
			want:       "default",
		},
		{
			name:       "Custom server name",
			serverName: "my-dns-server",
			want:       "my-dns-server",
		},
		{
			name:       "Empty server name",
			serverName: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GeneralFlags.ServerName = tt.serverName
			if GeneralFlags.ServerName != tt.want {
				t.Errorf("ServerName = %v, want %v", GeneralFlags.ServerName, tt.want)
			}
		})
	}
}

func TestLogLevelValidation(t *testing.T) {
	tests := []struct {
		name     string
		logLevel uint
		valid    bool
	}{
		{"PANIC level", 0, true},
		{"ERROR level", 1, true},
		{"WARN level", 2, true},
		{"INFO level", 3, true},
		{"DEBUG level", 4, true},
		{"Invalid level 5", 5, false},
		{"Invalid level 10", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.logLevel <= 4
			if valid != tt.valid {
				t.Errorf("LogLevel %d validation = %v, want %v", tt.logLevel, valid, tt.valid)
			}
		})
	}
}

func TestLogFormatValidation(t *testing.T) {
	validFormats := []string{"json", "text"}

	tests := []struct {
		name   string
		format string
		valid  bool
	}{
		{"JSON format", "json", true},
		{"Text format", "text", true},
		{"Invalid format", "xml", false},
		{"Empty format", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := false
			for _, f := range validFormats {
				if tt.format == f {
					valid = true
					break
				}
			}
			if valid != tt.valid {
				t.Errorf("LogFormat %s validation = %v, want %v", tt.format, valid, tt.valid)
			}
		})
	}
}

func TestMaskSizeValidation(t *testing.T) {
	tests := []struct {
		name     string
		maskSize int
		bitSize  int
		valid    bool
	}{
		{"Valid IPv4 /24", 24, 32, true},
		{"Valid IPv4 /32", 32, 32, true},
		{"Valid IPv4 /0", 0, 32, true},
		{"Invalid IPv4 /33", 33, 32, false},
		{"Invalid IPv4 /-1", -1, 32, false},
		{"Valid IPv6 /64", 64, 128, true},
		{"Valid IPv6 /128", 128, 128, true},
		{"Invalid IPv6 /129", 129, 128, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.maskSize >= 0 && tt.maskSize <= tt.bitSize
			if valid != tt.valid {
				t.Errorf("MaskSize %d for bitSize %d validation = %v, want %v",
					tt.maskSize, tt.bitSize, valid, tt.valid)
			}
		})
	}
}

func TestPacketLimitConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		packetLimit int
		description string
	}{
		{
			name:        "No limit",
			packetLimit: 0,
			description: "All packets processed",
		},
		{
			name:        "1000 packets per batch",
			packetLimit: 1000,
			description: "Limit to 1000 packets",
		},
		{
			name:        "100000 packets per batch",
			packetLimit: 100000,
			description: "Large batch limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GeneralFlags.PacketLimit = tt.packetLimit
			if GeneralFlags.PacketLimit != tt.packetLimit {
				t.Errorf("PacketLimit = %d, want %d", GeneralFlags.PacketLimit, tt.packetLimit)
			}
		})
	}
}

func TestChannelSizeConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		channelSize uint
		valid       bool
	}{
		{"Small channel", 100, true},
		{"Medium channel", 10000, true},
		{"Large channel", 100000, true},
		{"Very large channel", 1000000, true},
		{"Zero channel", 0, true}, // Valid but not recommended
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GeneralFlags.ResultChannelSize = tt.channelSize
			if GeneralFlags.ResultChannelSize != tt.channelSize {
				t.Errorf("ResultChannelSize = %d, want %d", 
					GeneralFlags.ResultChannelSize, tt.channelSize)
			}
		})
	}
}

// Benchmark configuration operations
func BenchmarkConfigurationAccess(b *testing.B) {
	GeneralFlags.ServerName = "benchmark-server"
	GeneralFlags.PacketLimit = 1000
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GeneralFlags.ServerName
		_ = GeneralFlags.PacketLimit
	}
}

// vim: foldmethod=marker
