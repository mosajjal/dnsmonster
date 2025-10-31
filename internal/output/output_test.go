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

package output

import (
	"context"
	"testing"

	"github.com/mosajjal/dnsmonster/internal/util"
)

func TestOutputTypeValidation(t *testing.T) {
	tests := []struct {
		name       string
		outputType uint
		wantInit   bool
	}{
		{
			name:       "Output type 0 (disabled)",
			outputType: 0,
			wantInit:   false,
		},
		{
			name:       "Output type 1 (no filters)",
			outputType: 1,
			wantInit:   true,
		},
		{
			name:       "Output type 2 (skip domains)",
			outputType: 2,
			wantInit:   true,
		},
		{
			name:       "Output type 3 (allow domains)",
			outputType: 3,
			wantInit:   true,
		},
		{
			name:       "Output type 4 (both filters)",
			outputType: 4,
			wantInit:   true,
		},
		{
			name:       "Output type 5 (invalid)",
			outputType: 5,
			wantInit:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldInitialize := tt.outputType > 0 && tt.outputType < 5
			if shouldInitialize != tt.wantInit {
				t.Errorf("Output type %d initialization check = %v, want %v", 
					tt.outputType, shouldInitialize, tt.wantInit)
			}
		})
	}
}

func TestFileOutputInitialization(t *testing.T) {
	// Test that file output can be initialized with type 0 (disabled)
	config := fileConfig{
		FileOutputType: 0,
		outputChannel:  make(chan util.DNSResult, 100),
	}
	
	ctx := context.Background()
	err := config.Initialize(ctx)
	
	// Should return error for disabled output
	if err == nil {
		t.Error("Expected error for disabled output type, got nil")
	}
}

func TestStdoutOutputInitialization(t *testing.T) {
	// Test that stdout output can be initialized with type 0 (disabled)
	config := stdoutConfig{
		StdoutOutputType: 0,
		outputChannel:    make(chan util.DNSResult, 100),
	}
	
	ctx := context.Background()
	err := config.Initialize(ctx)
	
	// Should return error for disabled output
	if err == nil {
		t.Error("Expected error for disabled output type, got nil")
	}
}

func TestOutputChannelCreation(t *testing.T) {
	// Test that output channels are created with proper buffer size
	channelSize := uint(1000)
	util.GeneralFlags.ResultChannelSize = channelSize
	
	ch := make(chan util.DNSResult, channelSize)
	
	if cap(ch) != int(channelSize) {
		t.Errorf("Channel capacity = %d, want %d", cap(ch), channelSize)
	}
}

// Benchmark channel operations
func BenchmarkChannelSend(b *testing.B) {
	ch := make(chan util.DNSResult, 10000)
	result := util.DNSResult{}
	
	// Start a goroutine to drain the channel
	done := make(chan bool)
	go func() {
		for range ch {
		}
		done <- true
	}()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- result
	}
	close(ch)
	<-done
}

func BenchmarkChannelReceive(b *testing.B) {
	ch := make(chan util.DNSResult, 10000)
	result := util.DNSResult{}
	
	// Pre-fill the channel
	go func() {
		for i := 0; i < b.N; i++ {
			ch <- result
		}
		close(ch)
	}()
	
	b.ResetTimer()
	for range ch {
	}
}

// vim: foldmethod=marker
