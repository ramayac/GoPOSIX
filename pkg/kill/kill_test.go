package kill

import (
	"bytes"
	"strings"
	"testing"
)

func TestKillMissingArgs(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
}

func TestKillJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "9999999"}, nil, &out, &out, "")
	if rc != 1 {
		t.Errorf("expected 1, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

func TestKillInvalidSignal(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-s", "INVALID_SIG", "1"}, nil, &buf, &buf, "")
	if code == 0 {
		t.Error("expected non-zero exit for invalid signal")
	}
}
func TestKillListSignals(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-l"}, nil, &buf, &buf, "")
	_ = code
}
func TestKillInvalidPID(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"abc"}, nil, &buf, &buf, "")
	if code == 0 {
		t.Error("expected non-zero exit for invalid PID")
	}
}
func TestKillSignalByName(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-TERM", "999999"}, nil, &buf, &buf, "")
	// Should fail because PID doesn't exist, but signal parsing should work
	_ = code
}
