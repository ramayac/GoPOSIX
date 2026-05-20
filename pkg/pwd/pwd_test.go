package pwd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunReturnsCwd(t *testing.T) {
	expected, _ := os.Getwd()
	result, err := Run(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != expected {
		t.Errorf("got %q, want %q", result.Path, expected)
	}
}

func TestRunPhysical(t *testing.T) {
	result, err := Run(true)
	if err != nil {
		t.Fatalf("unexpected error: %v (physical mode)", err)
	}
	if result.Path == "" {
		t.Error("expected non-empty path in physical mode")
	}
}
func TestCLI_Basic(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, nil, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "/") {
		t.Error("expected path")
	}
}
func TestCLI_JSON(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--json"}, nil, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "\"path\"") {
		t.Errorf("no JSON: %s", out.String())
	}
}
func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}
