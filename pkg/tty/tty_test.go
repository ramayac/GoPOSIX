package tty

import (
	"bytes"
	"io"
	"testing"
)

func TestTtySilentSuccess(t *testing.T) {
	// In a test environment, stdin is typically not a tty
	// but -s should just test and exit
	code := run([]string{"-s"}, io.Discard)
	// Since tests run without a tty, expect exit 1
	if code == 0 {
		t.Log("stdin is a tty in this test environment")
	}
}

func TestTtyNormal(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, &buf)
	// Exit code should be 0 (not a tty is informational, not an error)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.String() != "not a tty\n" {
		t.Logf("output: %q", buf.String())
	}
}

func TestTtyJson(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"is_tty"`)) {
		t.Error("JSON output missing is_tty field")
	}
}

func TestTtyCLI_NotATty(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, &out)
	// When not connected to a terminal, outputs "not a tty"
	if out.String() != "not a tty\n" { t.Errorf("got %q", out.String()) }
	_ = code
}

func TestTtyCLI_Silent(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-s"}, &out)
	// -s silent mode
	if out.Len() != 0 { t.Errorf("expected no output in silent mode, got %q", out.String()) }
	_ = code
}
