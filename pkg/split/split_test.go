package split

import (
	"bytes"
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

// --- parseSize tests ---

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"100", 100},
		{"1k", 1024},
		{"2k", 2048},
		{"1kb", 1000},
		{"1m", 1024 * 1024},
		{"1mb", 1000 * 1000},
		{"1g", 1024 * 1024 * 1024},
		{"0", 0},
	}
	for _, tt := range tests {
		got, err := parseSize(tt.input)
		if err != nil {
			t.Errorf("parseSize(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestParseSize_Invalid(t *testing.T) {
	_, err := parseSize("notanumber")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestParseSize_UpperK(t *testing.T) {
	n, err := parseSize("2K")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2048 {
		t.Errorf("parseSize(2K) = %d, want 2048", n)
	}
}

func TestParseSize_UpperM(t *testing.T) {
	n, err := parseSize("1M")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1024*1024 {
		t.Errorf("parseSize(1M) = %d, want %d", n, 1024*1024)
	}
}

func TestParseSize_UpperG(t *testing.T) {
	n, err := parseSize("1G")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1024*1024*1024 {
		t.Errorf("parseSize(1G) = %d, want %d", n, 1024*1024*1024)
	}
}

// --- Run edge cases ---

func TestSplitExactChunkBoundary(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "x")

	// Exactly 10 lines, chunk size 5 → 2 chunks
	input := strings.NewReader("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n")
	result, err := Run(input, prefix, 5, 0, 2, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Chunks != 2 {
		t.Errorf("expected 2 chunks, got %d", result.Chunks)
	}
}

func TestSplitSingleLine(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "x")

	input := strings.NewReader("hello")
	result, err := Run(input, prefix, 5, 0, 2, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Chunks != 1 {
		t.Errorf("expected 1 chunk, got %d", result.Chunks)
	}
}

func TestSplitDefaultLineCount(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "x")

	// Passing 0,0 should default to 1000 lines per chunk
	input := strings.NewReader("a\nb\n")
	result, err := Run(input, prefix, 0, 0, 2, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Chunks != 1 {
		t.Errorf("expected 1 chunk with default 1000 lines, got %d", result.Chunks)
	}
}

// --- CLI layer (run) tests ---

func TestSplitCLI_DefaultStdin(t *testing.T) {
	// Use a temp dir as working directory for file output
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	var buf bytes.Buffer
	code := run([]string{"-l", "2"}, &buf)
	// Empty stdin — should produce one chunk with no content
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestSplitCLI_LineMode(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	var buf bytes.Buffer
	code := run([]string{"-l", "3"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestSplitCLI_ByteMode(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	var buf bytes.Buffer
	code := run([]string{"-b", "10"}, &buf)
	// Empty stdin in byte mode: should produce 1 chunk
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestSplitCLI_JsonMode(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	var buf bytes.Buffer
	code := run([]string{"--json", "-l", "2"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	// Should produce JSON output
	if !bytes.Contains(buf.Bytes(), []byte(`"files"`)) {
		t.Error("JSON output missing files field")
	}
}

func TestSplitCLI_NumericSuffix(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	var buf bytes.Buffer
	code := run([]string{"-d", "-l", "2"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestSplitCLI_SuffixLength(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	var buf bytes.Buffer
	code := run([]string{"-a", "3", "-l", "2"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestSplitCLI_FileArg(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Create a temporary input file
	inputFile := filepath.Join(dir, "input.txt")
	os.WriteFile(inputFile, []byte("a\nb\nc\nd\ne\n"), 0644)

	var buf bytes.Buffer
	code := run([]string{"-l", "2", inputFile, "pfx"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	// Verify output files were created
	if _, err := os.Stat("pfxaa"); os.IsNotExist(err) {
		t.Error("expected output file pfxaa to exist")
	}
}

func TestSplitCLI_InvalidFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--nonexistent"}, &buf)
	if code != 2 {
		t.Errorf("expected exit 2 for invalid flag, got %d", code)
	}
}
