package ps

import (
	"bytes"
	"strings"
	"testing"
)

func TestPsRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-e"}, nil, &out)
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "PID") {
		t.Error("expected output header")
	}
}

func TestPsJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json"}, nil, &out)
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}
