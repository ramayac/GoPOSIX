package env

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunDefaultsToHostEnv(t *testing.T) {
	os.Setenv("COREGO_ENV_TEST", "yes")
	defer os.Unsetenv("COREGO_ENV_TEST")

	result := Run(false, nil)
	if result.Vars["COREGO_ENV_TEST"] != "yes" {
		t.Errorf("expected COREGO_ENV_TEST=yes in env output")
	}
}

func TestRunIgnoreEnvironment(t *testing.T) {
	os.Setenv("COREGO_ENV_TEST", "should_not_appear")
	defer os.Unsetenv("COREGO_ENV_TEST")

	result := Run(true, nil)
	if _, ok := result.Vars["COREGO_ENV_TEST"]; ok {
		t.Error("expected host env to be cleared when -i is set")
	}
}

func TestRunVarAssignment(t *testing.T) {
	result := Run(true, []string{"FOO=bar", "BAZ=qux"})
	if result.Vars["FOO"] != "bar" {
		t.Errorf("FOO: got %q, want bar", result.Vars["FOO"])
	}
	if result.Vars["BAZ"] != "qux" {
		t.Errorf("BAZ: got %q, want qux", result.Vars["BAZ"])
	}
}
func TestCLI_Default(t *testing.T) {
	os.Setenv("CLI_TEST_VAR", "hello")
	defer os.Unsetenv("CLI_TEST_VAR")
	var out bytes.Buffer
	code := run([]string{}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "CLI_TEST_VAR=hello") {
		t.Errorf("expected env var, got: %s", out.String())
	}
}
func TestCLI_IgnoreEnv(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-i"}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}
func TestCLI_VarAssignment(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-i", "FOO=bar", "BAZ=qux"}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "FOO=bar") {
		t.Errorf("expected FOO=bar, got: %s", out.String())
	}
}
func TestCLI_JSON(t *testing.T) {
	os.Setenv("J_VAR", "val")
	defer os.Unsetenv("J_VAR")
	var out bytes.Buffer
	code := run([]string{"--json", "J_VAR"}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "\"vars\"") {
		t.Errorf("expected JSON, got: %s", out.String())
	}
}
func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, &out)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}
