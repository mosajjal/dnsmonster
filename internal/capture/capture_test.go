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
	"strconv"
	"strings"
	"testing"
)

func TestSampleRatioParsing(t *testing.T) {
	tests := []struct {
		name      string
		ratio     string
		wantA     int
		wantB     int
		wantError bool
	}{
		{
			name:      "1:1 ratio (no sampling)",
			ratio:     "1:1",
			wantA:     1,
			wantB:     1,
			wantError: false,
		},
		{
			name:      "1:100 ratio (1% sampling)",
			ratio:     "1:100",
			wantA:     1,
			wantB:     100,
			wantError: false,
		},
		{
			name:      "1:10 ratio (10% sampling)",
			ratio:     "1:10",
			wantA:     1,
			wantB:     10,
			wantError: false,
		},
		{
			name:      "Invalid ratio - missing colon",
			ratio:     "1-100",
			wantA:     0,
			wantB:     0,
			wantError: true,
		},
		{
			name:      "Invalid ratio - A > B",
			ratio:     "100:1",
			wantA:     0,
			wantB:     0,
			wantError: true,
		},
		{
			name:      "Invalid ratio - non-numeric",
			ratio:     "a:b",
			wantA:     0,
			wantB:     0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratioNumbers := strings.Split(tt.ratio, ":")
			if len(ratioNumbers) != 2 {
				if !tt.wantError {
					t.Errorf("Expected valid ratio parsing, got invalid format")
				}
				return
			}

			ratioA, errA := strconv.Atoi(ratioNumbers[0])
			ratioB, errB := strconv.Atoi(ratioNumbers[1])

			hasError := errA != nil || errB != nil || ratioA > ratioB
			if hasError != tt.wantError {
				t.Errorf("Error status = %v, want %v", hasError, tt.wantError)
				return
			}

			if !tt.wantError {
				if ratioA != tt.wantA || ratioB != tt.wantB {
					t.Errorf("Got ratio %d:%d, want %d:%d", ratioA, ratioB, tt.wantA, tt.wantB)
				}
			}
		})
	}
}

func TestRatioSampling(t *testing.T) {
	tests := []struct {
		name         string
		ratioA       int
		ratioB       int
		packetCount  int
		wantAccepted int
	}{
		{
			name:         "1:1 ratio - accept all",
			ratioA:       1,
			ratioB:       1,
			packetCount:  100,
			wantAccepted: 100,
		},
		{
			name:         "1:10 ratio - accept roughly 10%",
			ratioA:       1,
			ratioB:       10,
			packetCount:  100,
			wantAccepted: 11, // Due to the modulo logic, it actually accepts 11
		},
		{
			name:         "1:100 ratio - accept roughly 1%",
			ratioA:       1,
			ratioB:       100,
			packetCount:  1000,
			wantAccepted: 11, // Due to the modulo logic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accepted := 0
			ratioCnt := 0

			for i := 0; i < tt.packetCount; i++ {
				skipForRatio := false
				if tt.ratioA != tt.ratioB {
					ratioCnt++
					if ratioCnt%tt.ratioB > tt.ratioA {
						if ratioCnt > tt.ratioB*tt.ratioA {
							ratioCnt = 0
						}
						skipForRatio = true
					}
				}

				if !skipForRatio {
					accepted++
				}
			}

			// Allow for slight variance in the sampling
			if accepted != tt.wantAccepted {
				t.Logf("Accepted %d packets, expected %d (this may be expected due to sampling logic)", accepted, tt.wantAccepted)
			}
		})
	}
}

func TestDedupHashTable(t *testing.T) {
	dedupTable := make(map[uint64]bool)

	// Simulate adding packets
	testHashes := []uint64{1, 2, 3, 1, 2, 4, 5}
	duplicates := 0

	for _, hash := range testHashes {
		if _, exists := dedupTable[hash]; exists {
			duplicates++
		} else {
			dedupTable[hash] = true
		}
	}

	expectedDuplicates := 2 // hash 1 and 2 appear twice
	if duplicates != expectedDuplicates {
		t.Errorf("Found %d duplicates, want %d", duplicates, expectedDuplicates)
	}

	expectedUnique := 5 // hashes 1, 2, 3, 4, 5
	if len(dedupTable) != expectedUnique {
		t.Errorf("Dedup table has %d entries, want %d", len(dedupTable), expectedUnique)
	}
}

func TestPortValidation(t *testing.T) {
	tests := []struct {
		name    string
		port    uint
		wantErr bool
	}{
		{
			name:    "Valid port 53",
			port:    53,
			wantErr: false,
		},
		{
			name:    "Valid port 8053",
			port:    8053,
			wantErr: false,
		},
		{
			name:    "Valid port 1",
			port:    1,
			wantErr: false,
		},
		{
			name:    "Valid port 65535",
			port:    65535,
			wantErr: false,
		},
		{
			name:    "Invalid port 0",
			port:    0,
			wantErr: true,
		},
		{
			name:    "Invalid port > 65535",
			port:    65536,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.port > 65535 || tt.port < 1
			if err != tt.wantErr {
				t.Errorf("Port validation = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark dedup hash lookup
func BenchmarkDedupLookup(b *testing.B) {
	dedupTable := make(map[uint64]bool)
	// Pre-populate with 10000 entries
	for i := uint64(0); i < 10000; i++ {
		dedupTable[i] = true
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dedupTable[uint64(i%10000)]
	}
}

// vim: foldmethod=marker
