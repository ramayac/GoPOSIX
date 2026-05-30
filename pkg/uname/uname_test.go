package uname

import (
	"bytes"
	"testing"
)

func TestRunReturnsFields(t *testing.T) {
	result, err := Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Sysname == "" {
		t.Error("expected non-empty Sysname")
	}
	if result.Machine == "" {
		t.Error("expected non-empty Machine")
	}
}

func TestRunCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
}

func TestUnameAllFlags(t *testing.T) {
	flags := []string{"-s", "-n", "-r", "-m", "-a"}
	for _, f := range flags {
		t.Run(f, func(t *testing.T) {
			var buf bytes.Buffer
			code := run([]string{f}, nil, &buf, &buf, "")
			if code != 0 {
				t.Errorf("uname %s: exit %d", f, code)
			}
		})
	}
}
