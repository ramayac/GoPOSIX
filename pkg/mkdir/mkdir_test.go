package mkdir

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBasic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "newdir")
	result, err := Run([]string{target}, false, 0755)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 1 || result.Created[0] != target {
		t.Errorf("unexpected Created: %v", result.Created)
	}
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("directory was not created")
	}
}

func TestRunParents(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")
	_, err := Run([]string{target}, true, 0755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("parent directories not created")
	}
}

func TestRunExistsFails(t *testing.T) {
	dir := t.TempDir()
	_, err := Run([]string{dir}, false, 0755)
	if err == nil {
		t.Error("expected error for existing directory without -p")
	}
}

func TestRunMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "modedir")
	_, err := Run([]string{target}, false, 0700)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm() != 0700 {
		t.Errorf("expected mode 0700, got %v", info.Mode().Perm())
	}
}
func TestCLI_Basic(t *testing.T) { d := filepath.Join(t.TempDir(), "newdir"); var out bytes.Buffer; code := run([]string{d}, &out); if code != 0 { t.Fatalf("exit %d", code) }; if _, err := os.Stat(d); os.IsNotExist(err) { t.Error("dir not created") } }
func TestCLI_Parents(t *testing.T) { d := filepath.Join(t.TempDir(), "a", "b", "c"); var out bytes.Buffer; code := run([]string{"-p", d}, &out); if code != 0 { t.Fatalf("exit %d", code) }; if _, err := os.Stat(d); os.IsNotExist(err) { t.Error("nested dir not created") } }
func TestCLI_JSON(t *testing.T) { d := filepath.Join(t.TempDir(), "jd"); var out bytes.Buffer; code := run([]string{"--json", d}, &out); if code != 0 { t.Fatalf("exit %d", code) }; if !strings.Contains(out.String(), "\"created\"") { t.Errorf("no JSON: %s", out.String()) } }
func TestCLI_BadFlag(t *testing.T) { var out bytes.Buffer; code := run([]string{"--nonexistent"}, &out); if code != 2 { t.Errorf("exit %d, want 2", code) } }
func TestCLI_LongFlag(t *testing.T) { d := filepath.Join(t.TempDir(), "long"); var out bytes.Buffer; code := run([]string{"--parents", d}, &out); if code != 0 { t.Fatalf("exit %d", code) } }
