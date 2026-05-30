package df

import (
	"bytes"
	"strings"
	"testing"
)

func TestDfRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"/"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "Filesystem") {
		t.Error("expected output header")
	}
}

func TestDfJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "/"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

func TestDFInvalidPath(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"/nonexistent/path/xyz"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("exit %d, want 1 for invalid path", code)
	}
}
