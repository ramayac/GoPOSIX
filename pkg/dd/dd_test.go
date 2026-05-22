package dd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDd_StdinToStdout verifies default stdin→stdout copy.
func TestDd_StdinToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("I WANT\n")
	code := ddRun(nil, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if stdout.String() != "I WANT\n" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "I WANT\n")
	}
	if !strings.Contains(stderr.String(), "records in") {
		t.Errorf("stderr should contain 'records in', got: %q", stderr.String())
	}
}

// TestDd_IfFlag verifies reading from a named file via if=.
func TestDd_IfFlag(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "foo")
	os.WriteFile(fpath, []byte("I WANT"), 0644)

	var stdout, stderr bytes.Buffer
	code := ddRun([]string{"if=" + fpath}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if stdout.String() != "I WANT" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "I WANT")
	}
}

// TestDd_OfFlag verifies writing to a named file via of=.
func TestDd_OfFlag(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "of_test")
	stdin := strings.NewReader("I WANT\n")
	var stderr bytes.Buffer
	code := ddRun([]string{"of=" + fpath}, stdin, io.Discard, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	data, _ := os.ReadFile(fpath)
	if string(data) != "I WANT\n" {
		t.Errorf("file content = %q, want %q", string(data), "I WANT\n")
	}
}

// TestDd_CountBytes verifies byte-level truncation via count=N iflag=count_bytes.
func TestDd_CountBytes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("I WANT\n")
	code := ddRun([]string{"count=3", "iflag=count_bytes"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if stdout.String() != "I W" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "I W")
	}
}

// TestDd_CountBlocks verifies block-level truncation via count=N.
func TestDd_CountBlocks(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// 10 chars with bs=5 → 2 blocks. count=1 should give first 5 chars.
	stdin := strings.NewReader("abcdefghij")
	code := ddRun([]string{"bs=5", "count=1"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if stdout.String() != "abcde" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "abcde")
	}
}

// TestDd_Skip verifies skip=N input offset.
func TestDd_Skip(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("abcdefghijklmno") // 15 chars
	code := ddRun([]string{"bs=5", "skip=1", "count=1"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if stdout.String() != "fghij" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "fghij")
	}
}

// TestDd_Seek verifies seek=N output offset.
func TestDd_Seek(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "seek_test")
	stdin := strings.NewReader("WORLD")
	var stderr bytes.Buffer
	// Write 5 NULs at offset 0, then write "WORLD" after 5-byte seek
	// Pre-create file with 5 bytes to seek past
	os.WriteFile(fpath, []byte("XXXXX"), 0644)
	code := ddRun([]string{"of=" + fpath, "bs=5", "seek=1", "conv=notrunc"}, stdin, io.Discard, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	data, _ := os.ReadFile(fpath)
	if string(data) != "XXXXXWORLD" {
		t.Errorf("file content = %q, want %q", string(data), "XXXXXWORLD")
	}
}

// TestDd_StatusNone verifies that status=none suppresses the stderr status line.
func TestDd_StatusNone(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("hello")
	code := ddRun([]string{"status=none"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty with status=none, got: %q", stderr.String())
	}
}

// TestDd_ConvSync verifies conv=sync NUL-pads short blocks.
func TestDd_ConvSync(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("short") // 5 chars
	code := ddRun([]string{"bs=10", "conv=sync"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if len(stdout.String()) != 10 {
		t.Errorf("expected 10 bytes with conv=sync, got %d", len(stdout.String()))
	}
}

// TestDd_WriteError verifies exit code 1 on write error.
func TestDd_WriteError(t *testing.T) {
	var stderr bytes.Buffer
	stdin := strings.NewReader("data")
	// /dev/full is a Linux special device that always returns ENOSPC on write.
	// Skip on non-Linux or if /dev/full doesn't exist.
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full not available")
	}
	code := ddRun([]string{"of=/dev/full"}, stdin, io.Discard, &stderr, "")
	if code != 1 {
		t.Errorf("exit code %d, want 1 (write error)", code)
	}
}
