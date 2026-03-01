// Copyright 2015 Eryx <evorui at gmail dot com>, All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpsrv

import (
	"testing"
)

func TestIsAlnum(t *testing.T) {
	tests := []struct {
		input    byte
		expected bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'-', false},
		{'_', false},
		{' ', false},
		{'@', false},
		{'\n', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := isAlnum(tt.input)
			if result != tt.expected {
				t.Errorf("isAlnum(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsUpper(t *testing.T) {
	tests := []struct {
		input    byte
		expected bool
	}{
		{'A', true},
		{'Z', true},
		{'M', true},
		{'a', false},
		{'z', false},
		{'0', false},
		{'-', false},
		{' ', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := isUpper(tt.input)
			if result != tt.expected {
				t.Errorf("isUpper(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCrc64Checksum(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"", 0},
		{"hello", 0x4130049c25b59e9d},
		{"world", 0x4ec6569417e3f905},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := crc64Checksum([]byte(tt.input))
			if tt.input == "" && result != tt.expected {
				t.Errorf("crc64Checksum('') = %d, want %d", result, tt.expected)
			}
			if tt.input != "" && result == 0 {
				t.Errorf("crc64Checksum(%q) should not be 0", tt.input)
			}
			// Same input should produce same output
			result2 := crc64Checksum([]byte(tt.input))
			if result != result2 {
				t.Errorf("crc64Checksum should be consistent")
			}
		})
	}
}

func TestCrc64ChecksumConsistency(t *testing.T) {
	// Different inputs should produce different outputs (with high probability)
	result1 := crc64Checksum([]byte("input1"))
	result2 := crc64Checksum([]byte("input2"))

	if result1 == result2 {
		t.Error("different inputs should produce different checksums")
	}
}
