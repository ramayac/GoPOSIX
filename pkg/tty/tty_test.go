package tty

import (
	"bytes"
	"io"
	"testing"
)

func TestTtySilentSuccess(t *testing.T) {
	// In a test environment, stdin is typically not a tty
	// but -s should just test and exit
	code := run([]string{"-s"}, nil, io.Discard, io.Discard, "")
	// Since tests run without a tty, expect exit 1
	if code == 0 {
		t.Log("stdin is a tty in this test environment")
	}
}

func TestTtyNormal(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, nil, &buf, &buf, "")
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
	code := run([]string{"--json"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"is_tty"`)) {
		t.Error("JSON output missing is_tty field")
	}
}

func TestTtyCLI_NotATty(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, nil, &out, &out, "")
	// When not connected to a terminal, outputs "not a tty"
	if out.String() != "not a tty\n" {
		t.Errorf("got %q", out.String())
	}
	_ = code
}

func TestTtyCLI_Silent(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-s"}, nil, &out, &out, "")
	// -s silent mode
	if out.Len() != 0 {
		t.Errorf("expected no output in silent mode, got %q", out.String())
	}
	_ = code
}

func TestTtyRun_NotATty(t *testing.T) {
	result, err := Run()
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if result.IsTTY {
		t.Log("stdin is a tty in this environment — skipping non-tty assertions")
	} else {
		// In CI / non-interactive: IsTTY=false, Path should be empty
		if result.Path != "" {
			t.Errorf("expected empty path for non-tty, got %q", result.Path)
		}
	}
}

func TestTtyRun_ReturnsStructuredResult(t *testing.T) {
	result, err := Run()
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	// Either IsTTY is true (with a path) or false
	if result.IsTTY && result.Path == "" {
		t.Error("IsTTY=true but Path is empty")
	}
}

func TestTtyCLI_JsonSilent(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--json", "-s"}, nil, &out, &out, "")
	// In silent mode with --json, still returns structured output
	if code != 0 && code != 1 {
		t.Errorf("expected exit 0 or 1, got %d", code)
	}
}

func TestTtyCLI_InvalidFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("expected exit 2 for invalid flag, got %d", code)
	}
}

type mockFdReader struct {
	io.Reader
}

func (mockFdReader) Fd() uintptr {
	return 42
}

func TestTtyMockedTerminal(t *testing.T) {
	// Backup original functions
	origIsTerminal := isTerminalFn
	origTtyname := ttynameFn
	defer func() {
		isTerminalFn = origIsTerminal
		ttynameFn = origTtyname
	}()

	// 1. Test success terminal case
	isTerminalFn = func(fd int) bool { return true }
	ttynameFn = func(fd int) (string, error) { return "/dev/pts/9", nil }

	var out bytes.Buffer
	code := run([]string{}, mockFdReader{}, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if out.String() != "/dev/pts/9\n" {
		t.Errorf("expected '/dev/pts/9\\n', got %q", out.String())
	}

	// 2. Test ttyname error fallback
	ttynameFn = func(fd int) (string, error) { return "", io.EOF }
	out.Reset()
	code = run([]string{}, mockFdReader{}, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if out.String() != "unknown\n" {
		t.Errorf("expected 'unknown\\n', got %q", out.String())
	}

	// 3. Test --json with mocked tty success
	ttynameFn = func(fd int) (string, error) { return "/dev/tty2", nil }
	out.Reset()
	code = run([]string{"--json"}, mockFdReader{}, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"/dev/tty2"`)) {
		t.Errorf("expected JSON to contain '/dev/tty2', got: %s", out.String())
	}

	// 4. Test silent mode with terminal
	out.Reset()
	code = run([]string{"-s"}, mockFdReader{}, &out, &out, "")
	if code != 0 {
		t.Errorf("expected exit 0 for silent with terminal, got %d", code)
	}
}

func TestTtynameDirect(t *testing.T) {
	_, err := ttyname(-1)
	if err == nil {
		t.Error("expected error for invalid fd -1")
	}
	_, _ = ttyname(0)
}

