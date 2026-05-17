package cksum

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPosixCRC_KnownValues(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
	}{
		{"", 4294967295},        // empty file
		{"hello", 3287646509},   // "hello" (5 bytes)
		{"1234567890", 1187747251}, // 10 bytes
	}
	for _, tt := range tests {
		got := posixCRC([]byte(tt.input))
		if got != tt.expected {
			t.Errorf("posixCRC(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestCksumStdin(t *testing.T) {
	// Test via Run with no files (but we can't easily test stdin in unit test)
	// So test the library function directly
	files := []string{}
	// This would read from os.Stdin, which isn't testable directly
	_ = files
}

func TestCksumFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	os.WriteFile(f, []byte("hello"), 0644)

	result, err := Run([]string{f})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	if result.Files[0].Checksum != 3287646509 {
		t.Errorf("expected checksum 3287646509, got %d", result.Files[0].Checksum)
	}
	if result.Files[0].Bytes != 5 {
		t.Errorf("expected 5 bytes, got %d", result.Files[0].Bytes)
	}
}

func TestCksumEmptyFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "empty")
	os.WriteFile(f, []byte{}, 0644)

	result, err := Run([]string{f})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Files[0].Checksum != 4294967295 {
		t.Errorf("empty file: expected 4294967295, got %d", result.Files[0].Checksum)
	}
	if result.Files[0].Bytes != 0 {
		t.Errorf("empty file: expected 0 bytes, got %d", result.Files[0].Bytes)
	}
}

func TestCksumNonexistent(t *testing.T) {
	_, err := Run([]string{"/tmp/nonexistent_goposix_cksum_test"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestCksumTextOutput(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	os.WriteFile(f, []byte("hello"), 0644)

	var buf bytes.Buffer
	code := run([]string{f}, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	expected := "3287646509 5 "
	if !strings.HasPrefix(buf.String(), expected) {
		t.Errorf("expected output to start with %q, got %q", expected, buf.String())
	}
}

func TestCksumJson(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	os.WriteFile(f, []byte("hello"), 0644)

	var buf bytes.Buffer
	code := run([]string{"--json", f}, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"checksum"`)) {
		t.Error("JSON output missing checksum field")
	}
}
