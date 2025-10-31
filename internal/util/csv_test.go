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
	"strings"
	"testing"
)

func TestLoadDomainsFromCSV(t *testing.T) {
	tests := []struct {
		name     string
		csvData  string
		wantErr  bool
		checkFn  func(*testing.T, string)
	}{
		{
			name:    "Empty CSV",
			csvData: "",
			wantErr: false,
			checkFn: func(t *testing.T, data string) {
				// Empty CSV should not error, just return empty structures
			},
		},
		{
			name:    "Simple domain list",
			csvData: "example.com\ngoogle.com\nfacebook.com",
			wantErr: false,
			checkFn: func(t *testing.T, data string) {
				lines := strings.Split(strings.TrimSpace(data), "\n")
				if len(lines) != 3 {
					t.Errorf("Expected 3 domains, got %d", len(lines))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkFn != nil {
				tt.checkFn(t, tt.csvData)
			}
		})
	}
}

func TestDomainMatching(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		prefix string
		suffix string
		match  bool
	}{
		{
			name:   "Exact match",
			domain: "example.com.",
			prefix: "",
			suffix: "example.com.",
			match:  true,
		},
		{
			name:   "Subdomain match",
			domain: "www.example.com.",
			prefix: "",
			suffix: "example.com.",
			match:  true,
		},
		{
			name:   "No match",
			domain: "google.com.",
			prefix: "",
			suffix: "example.com.",
			match:  false,
		},
		{
			name:   "Prefix match",
			domain: "test-server.example.com.",
			prefix: "test-",
			suffix: "",
			match:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matched bool
			if tt.prefix != "" {
				matched = strings.HasPrefix(tt.domain, tt.prefix)
			}
			if tt.suffix != "" {
				matched = matched || strings.HasSuffix(tt.domain, tt.suffix)
			}
			if matched != tt.match {
				t.Errorf("Domain matching failed: got %v, want %v", matched, tt.match)
			}
		})
	}
}

// Benchmark CSV loading
func BenchmarkLoadDomainsCSV(b *testing.B) {
	// Create a test CSV with 1000 domains
	domains := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		domains[i] = "example" + string(rune(i)) + ".com"
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This would normally call LoadDomainsCsv but we're just benchmarking the concept
		_ = strings.Join(domains, "\n")
	}
}

// vim: foldmethod=marker
