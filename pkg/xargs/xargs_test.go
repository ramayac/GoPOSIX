package xargs

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestXargsJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "true"}, &out)
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

// BusyBox hardening: xargs -0 should split input on NUL bytes.
func TestXargsNullDelimited(t *testing.T) {
	// Create a temp file with NUL-delimited input
	tmp, err := os.CreateTemp("", "xargs-null-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("hello")
	tmp.Write([]byte{0})
	tmp.WriteString("world")
	tmp.Write([]byte{0})
	tmp.Close()

	// Replace stdin
	oldStdin := os.Stdin
	f, _ := os.Open(tmp.Name())
	os.Stdin = f
	defer func() { os.Stdin = oldStdin; f.Close() }()

	var out bytes.Buffer
	code := run([]string{"-0", "echo"}, &out)
	if code != 0 {
		t.Fatalf("xargs -0 exited with %d, want 0", code)
	}
	if out.String() != "hello world\n" {
		t.Errorf("got %q, want %q", out.String(), "hello world\n")
	}
}
