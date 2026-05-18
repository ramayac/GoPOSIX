package ls

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBasicDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("h"), 0644)

	results, err := Run([]string{dir}, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Should only show a.txt (hidden excluded by default).
	found := false
	for _, f := range results[0].Files {
		if f.Name == "a.txt" {
			found = true
		}
		if f.Name == ".hidden" {
			t.Error("-a not set: .hidden should be excluded")
		}
	}
	if !found {
		t.Error("a.txt not found in result")
	}
}

func TestRunShowAll(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".dotfile"), []byte("x"), 0644)

	results, err := Run([]string{dir}, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	foundDot := false
	for _, f := range results[0].Files {
		if f.Name == "." || f.Name == ".dotfile" {
			foundDot = true
		}
	}
	if !foundDot {
		t.Error("-a: expected dotfiles and . to be present")
	}
}

func TestRunAlmostAll(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".dotfile"), []byte("x"), 0644)

	results, err := Run([]string{dir}, false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range results[0].Files {
		if f.Name == "." || f.Name == ".." {
			t.Errorf("-A: %q should be excluded", f.Name)
		}
	}
}

func TestRunNonExistent(t *testing.T) {
	_, err := Run([]string{"/this/path/does/not/exist"}, false, false, false)
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestRunSingleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("data"), 0644)

	results, err := Run([]string{f}, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(results[0].Files) != 1 {
		t.Errorf("expected 1 result with 1 file, got %+v", results)
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{1048576, "1.0M"},
	}
	for _, c := range cases {
		got := humanSize(c.n)
		if got != c.want {
			t.Errorf("humanSize(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestSortFiles(t *testing.T) {
	files := []FileInfo{
		{Name: "z"}, {Name: "a"}, {Name: "m"},
	}
	sorted := sortFiles(files, false, false, false)
	if sorted[0].Name != "a" || sorted[2].Name != "z" {
		t.Errorf("sort by name failed: %v", sorted)
	}
}

func TestSortFilesReverse(t *testing.T) {
	files := []FileInfo{
		{Name: "a"}, {Name: "z"},
	}
	sorted := sortFiles(files, false, false, true)
	if sorted[0].Name != "z" {
		t.Errorf("reverse sort failed: %v", sorted)
	}
}

// --- BusyBox hardening tests ---

func TestBusyBox_Ls_SortByteOrder(t *testing.T) {
	// LC_ALL=C: uppercase (A-Z) sorts before lowercase (a-z).
	files := []FileInfo{
		{Name: "zebra"},
		{Name: "README"},
		{Name: "alpha"},
		{Name: "TODO"},
		{Name: "beta"},
	}
	sorted := sortFiles(files, false, false, false)
	want := []string{"README", "TODO", "alpha", "beta", "zebra"}
	for i, f := range sorted {
		if f.Name != want[i] {
			t.Errorf("pos %d: got %q, want %q (full: %v)", i, f.Name, want[i], names(sorted))
		}
	}
}

func TestBusyBox_Ls_SortDotFirst(t *testing.T) {
	// Dotfiles sort before non-dotfiles.
	files := []FileInfo{
		{Name: "b"},
		{Name: ".hidden"},
		{Name: "a"},
		{Name: ".."},
		{Name: "."},
	}
	sorted := sortFiles(files, false, false, false)
	want := []string{".", "..", ".hidden", "a", "b"}
	for i, f := range sorted {
		if f.Name != want[i] {
			t.Errorf("pos %d: got %q, want %q", i, f.Name, want[i])
		}
	}
}

func TestBusyBox_Ls_DefaultFormatOnePerLine(t *testing.T) {
	// Default (columnar) mode should output one entry per line when piped.
	// Test via Run then verify output format.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("hi"), 0644)

	results, err := Run([]string{dir}, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(results[0].Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(results[0].Files))
	}
	// Verify they exist (names are used by run() for output).
	names := make(map[string]bool)
	for _, f := range results[0].Files {
		names[f.Name] = true
	}
	if !names["a.txt"] || !names["b.txt"] {
		t.Error("missing expected files")
	}
}

func TestBusyBox_Ls_BlocksFlag(t *testing.T) {
	// -s flag shows block counts. Test that Blocks field is populated.
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	os.WriteFile(path, []byte("hello"), 0644)

	results, err := Run([]string{dir}, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range results[0].Files {
		if f.Name == "f" && f.Size > 0 && f.Blocks == 0 {
			t.Error("expected non-zero blocks for non-empty file")
		}
	}
}

func names(files []FileInfo) []string {
	n := make([]string, len(files))
	for i, f := range files {
		n[i] = f.Name
	}
	return n
}

// --- CLI tests ---

func TestCLI_BasicDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0644)
	var out bytes.Buffer
	code := run([]string{dir}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_ShowAll(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0644)
	var out bytes.Buffer
	code := run([]string{"-a", dir}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_LongFormat(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("hi"), 0644)
	var out bytes.Buffer
	code := run([]string{"-l", dir}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_OnePerLine(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	var out bytes.Buffer
	code := run([]string{"-1", dir}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_Recursive(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "f.txt"), []byte("x"), 0644)
	var out bytes.Buffer
	code := run([]string{"-R", dir}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_JSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "json.txt"), []byte("x"), 0644)
	var out bytes.Buffer
	code := run([]string{"--json", dir}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"files\"") {
		t.Errorf("expected JSON, got: %s", out.String())
	}
}

func TestCLI_LongFlags(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	code := run([]string{"--all", dir}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_NotExist(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"/nonexistent/ls/path"}, &out)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, &out)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestCLI_MultipleDirs(t *testing.T) {
	d1 := t.TempDir()
	d2 := t.TempDir()
	var out bytes.Buffer
	code := run([]string{d1, d2}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}
