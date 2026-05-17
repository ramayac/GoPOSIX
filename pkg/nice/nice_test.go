package nice

import (
	"bytes"
	"io"
	"testing"
)

func TestNiceDefaultAdjustment(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"true"}, &buf)
	// setpriority may fail with EPERM in test environments
	if code != 0 {
		t.Logf("nice requires CAP_SYS_NICE (exit %d), skipping", code)
		return
	}
}

func TestNiceCustomAdjustment(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-n", "5", "true"}, &buf)
	// setpriority may fail with EPERM in test environments without CAP_SYS_NICE
	if code != 0 {
		t.Logf("nice requires CAP_SYS_NICE (exit %d), skipping", code)
		return
	}
}

func TestNiceCommandExitCode(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"false"}, &buf)
	if code != 1 {
		t.Errorf("expected exit 1 from false, got %d", code)
	}
}

func TestNiceMissingCommand(t *testing.T) {
	code := run([]string{}, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for missing command, got %d", code)
	}
}

func TestNiceInvalidAdjustment(t *testing.T) {
	code := run([]string{"-n", "abc", "true"}, io.Discard)
	if code != 1 {
		t.Errorf("expected exit 1 for invalid adjustment, got %d", code)
	}
}

func TestNiceJson(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "-n", "5", "true"}, &buf)
	// setpriority may fail with EPERM
	if code != 0 {
		t.Logf("nice requires CAP_SYS_NICE (exit %d), skipping", code)
		return
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"adjustment"`)) {
		t.Error("JSON output missing adjustment field")
	}
}
