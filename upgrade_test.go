package goposix

import (
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// a > b
		{"1.0.0", "0.9.0", true},
		{"0.2.0", "0.1.0", true},
		{"0.1.1", "0.1.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.0", "0.1.0", true},
		// a == b
		{"1.0.0", "1.0.0", false},
		{"0.1.0", "0.1.0", false},
		// a < b
		{"0.9.0", "1.0.0", false},
		{"0.1.0", "0.2.0", false},
		{"0.1.0", "0.1.1", false},
		// different segment count
		{"1.0", "1.0.0", false},
		{"1.0.0", "1.0", true},
		{"1.0.0.0", "1.0.0", true},
		// with v prefix (should be stripped by caller, but test anyway)
		{"v1.0.0", "v0.9.0", false}, // parseVersion treats 'v' as non-numeric → segment 0
		// git-derived versions (non-numeric) sort before numeric
		{"abc1234", "0.1.0", false},
		{"0.1.0", "abc1234", true},
		{"abc1234", "abc1234", false},
		// release candidates sort before stable
		{"0.1.0", "0.1.0-rc1", true},
		{"0.1.0-rc1", "0.1.0", false},
	}

	for _, tt := range tests {
		got := isNewer(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in   string
		want []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"0.1.0", []int{0, 1, 0}},
		{"v1.0.0", []int{0, 0, 0}}, // 'v' is non-numeric → 0
		{"abc1234", []int{0}},
		{"0.1.0-rc1", []int{0, 1, 0}}, // suffix stripped
		{"1", []int{1}},
		{"0", []int{0}},
	}

	for _, tt := range tests {
		got := parseVersion(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("parseVersion(%q) len = %d, want %d", tt.in, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}
