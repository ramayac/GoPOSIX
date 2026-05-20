package chmod

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestChmodMissingArgs(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, &out)
	if rc != 1 {
		t.Errorf("expected 1, got %d", rc)
	}
}

func TestChmodJSON(t *testing.T) {
	var out bytes.Buffer
	f, _ := os.CreateTemp("", "chmod")
	defer os.Remove(f.Name())

	rc := run([]string{"--json", "0755", f.Name()}, &out)
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

// --- CLI hardening ---

func TestCLI_OctalMode(t *testing.T) {
	f, _ := os.CreateTemp("", "chmod")
	defer os.Remove(f.Name())
	var out bytes.Buffer
	code := run([]string{"0644", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	info, _ := os.Stat(f.Name())
	if info.Mode().Perm() != 0644 {
		t.Errorf("expected 0644, got %o", info.Mode().Perm())
	}
}

func TestCLI_SymbolicMode(t *testing.T) {
	f, _ := os.CreateTemp("", "chmod")
	os.Chmod(f.Name(), 0600)
	defer os.Remove(f.Name())
	var out bytes.Buffer
	code := run([]string{"a+r", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	info, _ := os.Stat(f.Name())
	if info.Mode().Perm()&0444 == 0 {
		t.Errorf("expected read bits set, got %o", info.Mode().Perm())
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, &out)
	if code != 1 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestCLI_BadMode(t *testing.T) {
	f, _ := os.CreateTemp("", "chmod")
	defer os.Remove(f.Name())
	var out bytes.Buffer
	code := run([]string{"999", f.Name()}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for bad mode, got %d", code)
	}
}

func TestSymbolicMode_RemoveRead(t *testing.T) {
	f, _ := os.CreateTemp("", "chmod")
	os.Chmod(f.Name(), 0666)
	defer os.Remove(f.Name())
	var out bytes.Buffer
	code := run([]string{"a-r", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	info, _ := os.Stat(f.Name())
	if info.Mode().Perm()&0444 != 0 {
		t.Errorf("expected no read bits, got %o", info.Mode().Perm())
	}
}

func TestSymbolicMode_SetEquals(t *testing.T) {
	f, _ := os.CreateTemp("", "chmod")
	os.Chmod(f.Name(), 0777)
	defer os.Remove(f.Name())
	var out bytes.Buffer
	code := run([]string{"u=r", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	info, _ := os.Stat(f.Name())
	// u=r clears user bits then sets r: 0777 → 0477
	if info.Mode().Perm() != 0477 {
		t.Errorf("expected 0477, got %o", info.Mode().Perm())
	}
}

func TestSymbolicMode_GroupOnly(t *testing.T) {
	f, _ := os.CreateTemp("", "chmod")
	os.Chmod(f.Name(), 0000)
	defer os.Remove(f.Name())
	var out bytes.Buffer
	code := run([]string{"g+rw", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	info, _ := os.Stat(f.Name())
	if info.Mode().Perm() != 0060 {
		t.Errorf("expected 0060, got %o", info.Mode().Perm())
	}
}

func TestSymbolicMode_BadFormat(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"x", "/tmp"}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for bad symbolic mode, got %d", code)
	}
}

func TestCLI_JSON_Symbolic(t *testing.T) {
	f, _ := os.CreateTemp("", "chmod")
	os.Chmod(f.Name(), 0600)
	defer os.Remove(f.Name())
	var out bytes.Buffer
	code := run([]string{"--json", "a+r", f.Name()}, &out)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "\"changed\"") {
		t.Errorf("expected JSON changed field: %s", out.String())
	}
}

func TestCLI_NonexistentFile(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"0755", "/nonexistent_12345"}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for nonexistent file, got %d", code)
	}
}
