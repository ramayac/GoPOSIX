package id

import (
	"bytes"
	"strings"
	"testing"
)

func TestIdRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "uid=") {
		t.Error("expected output uid=")
	}
}

func TestIdJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}
