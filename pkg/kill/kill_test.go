package kill

import (
	"bytes"
	"strings"
	"testing"
)

func TestKillMissingArgs(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, &out)
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
}

func TestKillJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "9999999"}, &out)
	if rc != 1 {
		t.Errorf("expected 1, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}
