package rm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBasic(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "todelete.txt")
	os.WriteFile(f, []byte("x"), 0644)

	result, err := Run([]string{f}, false, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Removed) != 1 {
		t.Errorf("expected 1 removed, got %v", result.Removed)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("file still exists after rm")
	}
}

func TestRunRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "f"), []byte("x"), 0644)

	_, err := Run([]string{sub}, true, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sub); !os.IsNotExist(err) {
		t.Error("directory still exists after rm -r")
	}
}

func TestRunRefusesRoot(t *testing.T) {
	_, err := Run([]string{"/"}, true, false, false, false)
	if err == nil {
		t.Error("expected error when trying to rm /")
	}
}

func TestIsSafeToRemove(t *testing.T) {
	if isSafeToRemove("/", false) {
		t.Error("/ should not be safe to remove")
	}
	if !isSafeToRemove("/tmp", false) {
		t.Error("/tmp should be safe to remove")
	}
	if !isSafeToRemove("/", true) {
		t.Error("/ should be safe to remove with no-preserve-root")
	}
}

// --- CLI tests via run() ---

func TestCLI_BasicFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "x.txt")
	os.WriteFile(f, []byte("data"), 0644)
	var out bytes.Buffer
	code := run([]string{f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("file still exists")
	}
}

func TestCLI_Recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "d")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "f.txt"), []byte("x"), 0644)
	var out bytes.Buffer
	code := run([]string{"-r", sub}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if _, err := os.Stat(sub); !os.IsNotExist(err) {
		t.Error("dir still exists")
	}
}

func TestCLI_Force(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "z.txt")
	// File doesn't exist; -f should suppress error
	var out bytes.Buffer
	code := run([]string{"-f", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_RefuseRoot(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"/"}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for rm /, got %d", code)
	}
}

func TestCLI_JSON(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "j.json")
	os.WriteFile(f, []byte("x"), 0644)
	var out bytes.Buffer
	code := run([]string{"--json", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"removed\"") {
		t.Errorf("expected JSON, got: %s", out.String())
	}
}

func TestCLI_MissingOperand(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for missing operand, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, &out)
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestCLI_LongFlags(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "longtest")
	os.MkdirAll(sub, 0755)
	var out bytes.Buffer
	code := run([]string{"--recursive", sub}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if _, err := os.Stat(sub); !os.IsNotExist(err) {
		t.Error("dir still exists after --recursive")
	}
}

func TestCLI_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a")
	f2 := filepath.Join(dir, "b")
	os.WriteFile(f1, []byte("a"), 0644)
	os.WriteFile(f2, []byte("b"), 0644)
	var out bytes.Buffer
	code := run([]string{f1, f2}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Error("f1 still exists")
	}
	if _, err := os.Stat(f2); !os.IsNotExist(err) {
		t.Error("f2 still exists")
	}
}
