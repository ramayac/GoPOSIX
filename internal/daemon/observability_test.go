package daemon

import (
	"testing"
)

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"back\\slash", "back\\\\slash"},
		{"quote\"mark", "quote\\\"mark"},
		{"new\nline", "new\\nline"},
		{"multi\\\n\"line", "multi\\\\\\n\\\"line"},
		{"", ""},
		{"\\", "\\\\"},
		{"\n", "\\n"},
		{"\"", "\\\""},
	}
	for _, tt := range tests {
		got := sanitizeLabel(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
