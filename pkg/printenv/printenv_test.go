package printenv

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunAllVars(t *testing.T) {
	result := Run(nil)
	if len(result.Vars) == 0 {
		t.Error("expected at least one environment variable")
	}
}

func TestRunSpecificVar(t *testing.T) {
	os.Setenv("COREGO_TEST", "hello")
	defer os.Unsetenv("COREGO_TEST")

	result := Run([]string{"COREGO_TEST"})
	if result.Vars["COREGO_TEST"] != "hello" {
		t.Errorf("got %q, want hello", result.Vars["COREGO_TEST"])
	}
}

func TestRunMissingVar(t *testing.T) {
	result := Run([]string{"COREGO_DEFINITELY_NOT_SET_XYZ"})
	if _, ok := result.Vars["COREGO_DEFINITELY_NOT_SET_XYZ"]; ok {
		t.Error("expected missing var to be absent from results")
	}
}
func TestCLI_Basic(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.Len() == 0 {
		t.Error("expected some output")
	}
}
func TestCLI_SingleVar(t *testing.T) {
	os.Setenv("GOPOSIX_TEST_VAR", "hello")
	var out bytes.Buffer
	code := run([]string{"GOPOSIX_TEST_VAR"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Errorf("expected 'hello', got: %s", out.String())
	}
}
func TestCLI_MissingVar(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"NONEXISTENT_VAR_ZZZ"}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
}
func TestCLI_JSON(t *testing.T) {
	os.Setenv("JSON_VAR", "val")
	var out bytes.Buffer
	code := run([]string{"--json", "JSON_VAR"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "\"vars\"") {
		t.Errorf("no JSON: %s", out.String())
	}
}
func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}
