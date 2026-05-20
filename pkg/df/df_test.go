package df

import (
	"bytes"
	"strings"
	"testing"
)

func TestDfRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"/"}, nil, &out)
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "Filesystem") {
		t.Error("expected output header")
	}
}

func TestDfJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "/"}, nil, &out)
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}
