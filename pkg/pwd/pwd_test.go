package pwd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdirThroughSymlink creates a real dir and a symlink to it, chdirs to the
// symlink, and returns the real dir plus a restore function.
func chdirThroughSymlink(t *testing.T) (realDir, linkDir string) {
	t.Helper()
	dir := t.TempDir()
	realDir = filepath.Join(dir, "real")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	linkDir = filepath.Join(dir, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(linkDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return realDir, linkDir
}


func TestRunPhysicalDefault(t *testing.T) {
	realDir, _ := chdirThroughSymlink(t)
	result, err := Run(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != realDir {
		t.Errorf("default (physical) = %q, want %q", result.Path, realDir)
	}
}

func TestRunLogicalValidPWD(t *testing.T) {
	_, linkDir := chdirThroughSymlink(t)
	t.Setenv("PWD", linkDir)
	result, err := Run(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != filepath.Clean(linkDir) {
		t.Errorf("logical = %q, want %q", result.Path, filepath.Clean(linkDir))
	}
}

func TestRunLogicalInvalidPWD(t *testing.T) {
	realDir, _ := chdirThroughSymlink(t)
	t.Setenv("PWD", "/definitely/not/the/cwd")
	result, err := Run(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != realDir {
		t.Errorf("logical with bogus PWD = %q, want physical %q", result.Path, realDir)
	}
}

func TestRunLogicalRelativePWD(t *testing.T) {
	realDir, _ := chdirThroughSymlink(t)
	t.Setenv("PWD", "relative/path")
	result, err := Run(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != realDir {
		t.Errorf("logical with relative PWD = %q, want physical %q", result.Path, realDir)
	}
}

func TestRunLogicalUnsetPWD(t *testing.T) {
	realDir, _ := chdirThroughSymlink(t)
	orig, hadPWD := os.LookupEnv("PWD")
	os.Unsetenv("PWD")
	defer func() {
		if hadPWD {
			os.Setenv("PWD", orig)
		}
	}()
	result, err := Run(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != realDir {
		t.Errorf("logical with unset PWD = %q, want physical %q", result.Path, realDir)
	}
}

func TestCLI_DefaultPhysical(t *testing.T) {
	realDir, _ := chdirThroughSymlink(t)
	var out bytes.Buffer
	if code := run([]string{}, nil, &out, &out, ""); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out.String()) != realDir {
		t.Errorf("default output = %q, want %q", strings.TrimSpace(out.String()), realDir)
	}
}

func TestCLI_PhysicalFlag(t *testing.T) {
	realDir, _ := chdirThroughSymlink(t)
	var out bytes.Buffer
	if code := run([]string{"-P"}, nil, &out, &out, ""); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out.String()) != realDir {
		t.Errorf("-P output = %q, want %q", strings.TrimSpace(out.String()), realDir)
	}
}

func TestCLI_LogicalFlag(t *testing.T) {
	_, linkDir := chdirThroughSymlink(t)
	t.Setenv("PWD", linkDir)
	var out bytes.Buffer
	if code := run([]string{"-L"}, nil, &out, &out, ""); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out.String()) != filepath.Clean(linkDir) {
		t.Errorf("-L output = %q, want %q", strings.TrimSpace(out.String()), filepath.Clean(linkDir))
	}
}

func TestCLI_LongFlags(t *testing.T) {
	realDir, linkDir := chdirThroughSymlink(t)
	t.Setenv("PWD", linkDir)
	var out bytes.Buffer
	if code := run([]string{"--logical"}, nil, &out, &out, ""); code != 0 {
		t.Fatalf("--logical exit %d", code)
	}
	if strings.TrimSpace(out.String()) != filepath.Clean(linkDir) {
		t.Errorf("--logical output = %q, want %q", strings.TrimSpace(out.String()), filepath.Clean(linkDir))
	}
	out.Reset()
	if code := run([]string{"--physical"}, nil, &out, &out, ""); code != 0 {
		t.Fatalf("--physical exit %d", code)
	}
	if strings.TrimSpace(out.String()) != realDir {
		t.Errorf("--physical output = %q, want %q", strings.TrimSpace(out.String()), realDir)
	}
}

func TestCLI_JSON(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--json"}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), `"path"`) {
		t.Errorf("no JSON: %s", out.String())
	}
	if !strings.Contains(out.String(), `"command":"pwd"`) {
		t.Errorf("missing command envelope: %s", out.String())
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}
