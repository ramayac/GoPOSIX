package patch

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Unit tests (library layer)
// =============================================================================

func TestParseUnifiedDiff(t *testing.T) {
	patchData := `--- input	Jan 01 01:01:01 2000
+++ input	Jan 01 01:01:01 2000
@@ -1,2 +1,3 @@
 qwe
+asd
 zxc
`
	patches, err := ParsePatch(strings.NewReader(patchData))
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	p := patches[0]
	if p.OldFile != "input\tJan 01 01:01:01 2000" {
		t.Errorf("OldFile = %q", p.OldFile)
	}
	if len(p.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(p.Hunks))
	}
	h := p.Hunks[0]
	if h.OldStart != 1 || h.OldCount != 2 {
		t.Errorf("old range = %d,%d, want 1,2", h.OldStart, h.OldCount)
	}
	if h.NewStart != 1 || h.NewCount != 3 {
		t.Errorf("new range = %d,%d, want 1,3", h.NewStart, h.NewCount)
	}
}

func TestParseMultiHunk(t *testing.T) {
	patchData := `--- a	2020-01-01
+++ b	2020-01-01
@@ -1,3 +1,4 @@
 a
+b
 c
@@ -5,2 +6,3 @@
 x
+y
 z
`
	patches, err := ParsePatch(strings.NewReader(patchData))
	if err != nil {
		t.Fatal(err)
	}
	if len(patches[0].Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(patches[0].Hunks))
	}
}

func TestApplyHunkAddLine(t *testing.T) {
	// Add a line between qwe and zxc
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("qwe\nzxc\n"), 0644)

	patchData := `--- input
+++ input
@@ -1,2 +1,3 @@
 qwe
+asd
 zxc
`
	result, err := Run([]byte(patchData), fpath, 0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected 1 applied, got %d", result.Applied)
	}
	data, _ := os.ReadFile(fpath)
	if string(data) != "qwe\nasd\nzxc\n" {
		t.Errorf("file = %q, want %q", string(data), "qwe\nasd\nzxc\n")
	}
}

func TestApplyHunkDeleteLines(t *testing.T) {
	// Delete lines 1-3, add one changed line
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("111\n222\n333\n444\n555\n666\n777\n888\n999\n"), 0644)

	patchData := `--- input
+++ input
@@ -1,6 +1,4 @@
-111
-222
-333
+111changed
 444
 555
 666
`
	result, err := Run([]byte(patchData), fpath, 0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected 1 applied, got %d", result.Applied)
	}
	data, _ := os.ReadFile(fpath)
	if !strings.Contains(string(data), "111changed") {
		t.Errorf("expected 111changed in file, got: %q", string(data))
	}
	if strings.Contains(string(data), "222") {
		t.Errorf("222 should have been deleted")
	}
}

func TestApplyNewFile(t *testing.T) {
	// --- /dev/null creates a new file
	dir := t.TempDir()
	fpath := filepath.Join(dir, "testfile")

	patchData := `--- /dev/null
+++ testfile
@@ -0,0 +1 @@
+qwerty
`
	result, err := Run([]byte(patchData), fpath, 0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected 1 applied, got %d", result.Applied)
	}
	data, _ := os.ReadFile(fpath)
	if string(data) != "qwerty\n" {
		t.Errorf("file = %q, want %q", string(data), "qwerty\n")
	}
}

func TestReverse(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("qwe\nasd\nzxc\n"), 0644)

	patchData := `--- input
+++ input
@@ -1,2 +1,3 @@
 qwe
+asd
 zxc
`
	result, err := Run([]byte(patchData), fpath, 0, true, false) // reverse=true
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected 1 applied, got %d", result.Applied)
	}
	data, _ := os.ReadFile(fpath)
	if string(data) != "qwe\nzxc\n" {
		t.Errorf("file = %q, want %q", string(data), "qwe\nzxc\n")
	}
}

func TestStripPrefix(t *testing.T) {
	// -p1 strips first directory component
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	os.Mkdir(subdir, 0755)
	fpath := filepath.Join(subdir, "file")
	os.WriteFile(fpath, []byte("qwe\nzxc\n"), 0644)

	patchData := `--- a/file
+++ a/file
@@ -1,2 +1,3 @@
 qwe
+asd
 zxc
`
	// With -p1, a/file → file, but we need the full path
	// Actually, Run uses targetPath as-is, strip only affects the patch header filename
	// when targetPath is empty. Let's test the stripPath function directly.
	result, err := Run([]byte(patchData), fpath, 0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected 1 applied, got %d", result.Applied)
	}
}

func TestAlreadyApplied(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("abc\ndef\n123\n"), 0644)

	patchData := `--- input
+++ input
@@ -1,2 +1,3 @@
 abc
+def
 123
`
	result, err := Run([]byte(patchData), fpath, 0, false, false)
	if err == nil {
		t.Error("expected error for already applied hunk")
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
}

func TestIgnoreApplied(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("abc\ndef\n123\n"), 0644)

	patchData := `--- input
+++ input
@@ -1,2 +1,3 @@
 abc
+def
 123
`
	result, err := Run([]byte(patchData), fpath, 0, false, true) // ignoreApplied=true
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected 1 applied (ignored), got %d", result.Applied)
	}
}

func TestStripPath(t *testing.T) {
	tests := []struct {
		path   string
		level  int
		expect string
	}{
		{"a/b/file", 0, "a/b/file"},
		{"a/b/file", 1, "b/file"},
		{"a/b/file", 2, "file"},
		{"a/b/file", 3, "file"},
		{"a/b/file\t2020-01-01", 1, "b/file"},
		{"dir2///file", 1, "file"},
		{"bogus_dir///dir2///file", 1, "dir2///file"},
	}
	for _, tc := range tests {
		got := stripPath(tc.path, tc.level)
		if got != tc.expect {
			t.Errorf("stripPath(%q, %d) = %q, want %q", tc.path, tc.level, got, tc.expect)
		}
	}
}

func TestNonexistentFile(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("qwe\nzxc\n"), 0644)

	patchData := `--- input.doesnt_exist	Jan 01 01:01:01 2000
+++ input	Jan 01 01:01:01 2000
@@ -1,2 +1,3 @@
 qwe
+asd
 zxc
`
	result, err := Run([]byte(patchData), fpath, 0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected 1 applied, got %d", result.Applied)
	}
	data, _ := os.ReadFile(fpath)
	if string(data) != "qwe\nasd\nzxc\n" {
		t.Errorf("file = %q, want %q", string(data), "qwe\nasd\nzxc\n")
	}
}

func TestMultiHunkPatch(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("foo\n\n\n\n\n\nbar\n"), 0644)

	patchData := `--- a/input.orig
+++ b/input
@@ -5,5 +5,8 @@ foo
 
 
 
+1
+2
+3
 
 bar
`
	result, err := Run([]byte(patchData), fpath, 0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 1 {
		t.Fatalf("expected 1 applied, got %d", result.Applied)
	}
	data, _ := os.ReadFile(fpath)
	if !strings.Contains(string(data), "1\n2\n3") {
		t.Errorf("expected 1\\n2\\n3 in file, got: %q", string(data))
	}
}

// =============================================================================
// CLI tests (patchRun)
// =============================================================================

func TestPatchRun_Basic(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("qwe\nzxc\n"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader(`--- input
+++ input
@@ -1,2 +1,3 @@
 qwe
+asd
 zxc
`)
	code := patchRun([]string{fpath}, &stdout, &stderr, stdin, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "patching file") {
		t.Errorf("expected 'patching file' in stderr, got: %q", stderr.String())
	}
	data, _ := os.ReadFile(fpath)
	if string(data) != "qwe\nasd\nzxc\n" {
		t.Errorf("file = %q, want %q", string(data), "qwe\nasd\nzxc\n")
	}
}

func TestPatchRun_AlreadyApplied(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input")
	os.WriteFile(fpath, []byte("abc\ndef\n123\n"), 0644)

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader(`--- input
+++ input
@@ -1,2 +1,3 @@
 abc
+def
 123
`)
	code := patchRun([]string{fpath}, &stdout, &stderr, stdin, "")
	if code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "FAILED") {
		t.Errorf("expected FAILED in stderr, got: %q", stderr.String())
	}
}
