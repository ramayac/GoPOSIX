package split

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitByLines(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "x")

	input := strings.NewReader("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n")
	result, err := Run(input, prefix, 5, 0, 2, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Chunks != 3 {
		t.Errorf("expected 3 chunks, got %d", result.Chunks)
	}
	if len(result.Files) != 3 {
		t.Errorf("expected 3 files, got %d", len(result.Files))
	}

	// Check first file content
	data, _ := os.ReadFile(prefix + "aa")
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines in first chunk, got %d", len(lines))
	}
}

func TestSplitByBytes(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "x")

	input := strings.NewReader("1234567890ABCDEF")
	result, err := Run(input, prefix, 0, 5, 2, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Chunks != 4 {
		t.Errorf("expected 4 chunks, got %d", result.Chunks)
	}

	// Check content
	data, _ := os.ReadFile(prefix + "aa")
	if string(data) != "12345" {
		t.Errorf("expected '12345', got %q", string(data))
	}
}

func TestSplitNumericSuffix(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "x")

	input := strings.NewReader("1\n2\n3\n4\n")
	result, err := Run(input, prefix, 3, 0, 2, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have files x00, x01
	if _, err := os.Stat(prefix + "00"); os.IsNotExist(err) {
		t.Error("expected x00 to exist with numeric suffixes")
	}
	if _, err := os.Stat(prefix + "01"); os.IsNotExist(err) {
		t.Error("expected x01 to exist with numeric suffixes")
	}
	_ = result
}

func TestSplitSuffixLength(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "x")

	input := strings.NewReader("a\nb\nc\n")
	result, err := Run(input, prefix, 2, 0, 3, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(prefix + "aaa"); os.IsNotExist(err) {
		t.Error("expected xaaa to exist with suffix length 3")
	}
	_ = result
}

func TestSplitEmpty(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "x")

	input := strings.NewReader("")
	result, err := Run(input, prefix, 5, 0, 2, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Chunks != 1 {
		t.Errorf("expected 1 chunk for empty input, got %d", result.Chunks)
	}
}

func TestGenerateSuffix(t *testing.T) {
	tests := []struct {
		n        int
		len      int
		numeric  bool
		expected string
	}{
		{0, 2, false, "aa"},
		{1, 2, false, "ab"},
		{25, 2, false, "az"},
		{26, 2, false, "ba"},
		{0, 2, true, "00"},
		{5, 2, true, "05"},
		{99, 2, true, "99"},
		{0, 3, false, "aaa"},
		{26, 3, false, "aba"},
	}
	for _, tt := range tests {
		got := generateSuffix(tt.n, tt.len, tt.numeric)
		if got != tt.expected {
			t.Errorf("generateSuffix(%d, %d, %v) = %q, want %q",
				tt.n, tt.len, tt.numeric, got, tt.expected)
		}
	}
}


