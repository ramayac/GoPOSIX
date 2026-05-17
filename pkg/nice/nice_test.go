package nice

import (
	"bytes"
	"io"
	"testing"
)

func TestNiceDefaultAdjustment(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"true"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestNiceCustomAdjustment(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-n", "5", "true"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
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
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"adjustment"`)) {
		t.Error("JSON output missing adjustment field")
	}
}
