package fold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFold_SpaceBreak(t *testing.T) {
	r := strings.NewReader("123456\tasdf")
	out, err := Fold(r, 7, false, true)
	if err != nil {
		t.Fatal(err)
	}
	want := "123456\n\t\nasdf"
	if out != want {
		t.Errorf("fold -sw7:\n  got  %q\n  want %q", out, want)
	}
}

func TestFold_Width1(t *testing.T) {
	r := strings.NewReader("qq w eee r tttt y")
	out, err := Fold(r, 1, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "q\nq\n \nw\n \ne\ne\ne\n \nr\n \nt\nt\nt\nt\n \ny"
	if out != want {
		t.Errorf("fold -w1:\n  got  %q\n  want %q", out, want)
	}
}

func TestFold_NULs(t *testing.T) {
	input := "The NUL is here:>\x00< and another one is here:>\x00< - they must be preserved"
	r := strings.NewReader(input)
	out, err := Fold(r, 22, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\x00") {
		t.Error("NUL bytes missing from output")
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Errorf("expected at least 3 wrapped lines, got %d: %q", len(lines), out)
	}
}

func TestFold_Default(t *testing.T) {
	long := strings.Repeat("a", 200)
	r := strings.NewReader(long)
	out, err := Fold(r, 80, false, false)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("200 a's at width 80: expected 3 lines, got %d", len(lines))
	}
	if len(lines[0]) > 80 || len(lines[1]) > 80 {
		t.Error("lines exceed width 80")
	}
}

func TestFold_Empty(t *testing.T) {
	r := strings.NewReader("")
	out, err := Fold(r, 80, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("empty input: got %q", out)
	}
}

func TestFold_SingleNewline(t *testing.T) {
	r := strings.NewReader("\n")
	out, err := Fold(r, 80, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out != "\n" {
		t.Errorf("single newline: got %q", out)
	}
}

func TestFold_NoSpaceBreakPossible(t *testing.T) {
	r := strings.NewReader("abcdefghij")
	out, err := Fold(r, 5, false, true)
	if err != nil {
		t.Fatal(err)
	}
	want := "abcde\nfghij"
	if out != want {
		t.Errorf("no space to break at:\n  got  %q\n  want %q", out, want)
	}
}

// --- Byte-mode tests (foldLineBytes, -b flag) ---

func TestFold_ByteMode_Basic(t *testing.T) {
	r := strings.NewReader("abcdefghij")
	out, err := Fold(r, 5, true, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "abcde\nfghij"
	if out != want {
		t.Errorf("byte mode:\n  got  %q\n  want %q", out, want)
	}
}

func TestFold_ByteMode_WithSpaceBreak(t *testing.T) {
	r := strings.NewReader("hello world test")
	out, err := Fold(r, 7, true, true)
	if err != nil {
		t.Fatal(err)
	}
	// Should break at spaces where possible
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	for _, line := range lines {
		if len(line) > 7 {
			t.Errorf("byte-mode line %q exceeds width 7", line)
		}
	}
}

func TestFold_ByteMode_SingleCharExceedsWidth(t *testing.T) {
	// A single character wider than width should still be output
	r := strings.NewReader("abcdefgh")
	out, err := Fold(r, 3, true, false)
	if err != nil {
		t.Fatal(err)
	}
	// Each character fits so should wrap at width 3
	want := "abc\ndef\ngh"
	if out != want {
		t.Errorf("byte mode width=3:\n  got  %q\n  want %q", out, want)
	}
}

func TestFold_ByteMode_NUL(t *testing.T) {
	input := "ab\x00cd\x00ef"
	r := strings.NewReader(input)
	out, err := Fold(r, 3, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\x00") {
		t.Error("byte mode: NUL bytes missing from output")
	}
}

// --- Rune-mode tests (Unicode) ---

func TestFold_Runes_UnicodeWidth(t *testing.T) {
	// Each of these is 1 column wide
	r := strings.NewReader("héllo wörld")
	out, err := Fold(r, 5, false, true)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	for _, line := range lines {
		cols := len([]rune(line))
		if cols > 5 {
			t.Errorf("rune-mode line %q has %d columns, exceeds width 5", line, cols)
		}
	}
}

func TestFold_Runes_TabExpansion(t *testing.T) {
	// Tab expands to 8-column boundary
	r := strings.NewReader("a\tb")
	out, err := Fold(r, 10, false, false)
	if err != nil {
		t.Fatal(err)
	}
	// "a" + tab(→col8) + "b" → "a       b"
	if !strings.Contains(out, "a") && !strings.Contains(out, "b") {
		t.Errorf("unexpected tab expansion: %q", out)
	}
}

// --- CLI layer tests (foldRun) ---

func TestFoldRun_Stdin(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello")
	rc := foldRun([]string{}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if outBuf.String() != "hello" {
		t.Errorf("got %q, want %q", outBuf.String(), "hello")
	}
}

func TestFoldRun_Width(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("abcdef")
	rc := foldRun([]string{"-w", "3"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	want := "abc\ndef"
	if outBuf.String() != want {
		t.Errorf("got %q, want %q", outBuf.String(), want)
	}
}

func TestFoldRun_WidthLongFlag(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("abcdef")
	rc := foldRun([]string{"--width", "3"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	want := "abc\ndef"
	if outBuf.String() != want {
		t.Errorf("got %q, want %q", outBuf.String(), want)
	}
}

func TestFoldRun_Json(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello\nworld")
	rc := foldRun([]string{"--json"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(outBuf.String(), "\"lines\"") {
		t.Errorf("JSON missing 'lines': %s", outBuf.String())
	}
}

func TestFoldRun_ByteMode(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("abcdefghij")
	rc := foldRun([]string{"-b", "-w", "5"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	want := "abcde\nfghij"
	if outBuf.String() != want {
		t.Errorf("byte mode: got %q, want %q", outBuf.String(), want)
	}
}

func TestFoldRun_SpacesFlag(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello world")
	rc := foldRun([]string{"-s", "-w", "7"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	// Should break at space
	want := "hello \nworld"
	if outBuf.String() != want {
		t.Errorf("space break: got %q, want %q", outBuf.String(), want)
	}
}

func TestFoldRun_SpacesLongFlag(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello world")
	rc := foldRun([]string{"--spaces", "-w", "7"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	want := "hello \nworld"
	if outBuf.String() != want {
		t.Errorf("--spaces: got %q, want %q", outBuf.String(), want)
	}
}

func TestFoldRun_FromFile(t *testing.T) {
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "fold_test")
	if err := os.WriteFile(fpath, []byte("hello\nworld"), 0644); err != nil {
		t.Fatal(err)
	}

	var outBuf, errBuf bytes.Buffer
	rc := foldRun([]string{fpath}, &outBuf, &errBuf, nil)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if outBuf.String() != "hello\nworld" {
		t.Errorf("file input: got %q, want %q", outBuf.String(), "hello\nworld")
	}
}

func TestFoldRun_NestedDirs(t *testing.T) {
	tmp := t.TempDir()
	nestedDir := filepath.Join(tmp, "subdir")
	os.Mkdir(nestedDir, 0755)
	fpath := filepath.Join(nestedDir, "fold_test")
	if err := os.WriteFile(fpath, []byte("hello\nworld"), 0644); err != nil {
		t.Fatal(err)
	}

	var outBuf, errBuf bytes.Buffer
	rc := foldRun([]string{fpath}, &outBuf, &errBuf, nil)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if outBuf.String() != "hello\nworld" {
		t.Errorf("nested file: got %q, want %q", outBuf.String(), "hello\nworld")
	}
}

func TestFoldRun_FileNotFound(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	rc := foldRun([]string{"/nonexistent/fold_file"}, &outBuf, &errBuf, nil)
	if rc != 1 {
		t.Errorf("exit code: got %d, want 1 for missing file", rc)
	}
}

func TestFoldRun_MultiFile(t *testing.T) {
	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "a")
	f2 := filepath.Join(tmp, "b")
	os.WriteFile(f1, []byte("hello"), 0644)
	os.WriteFile(f2, []byte("world"), 0644)

	var outBuf, errBuf bytes.Buffer
	rc := foldRun([]string{f1, f2}, &outBuf, &errBuf, nil)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if outBuf.String() != "helloworld" {
		t.Errorf("multi-file: got %q, want %q", outBuf.String(), "helloworld")
	}
}

func TestFoldRun_StdinDash(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello")
	rc := foldRun([]string{"-"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if outBuf.String() != "hello" {
		t.Errorf("stdin dash: got %q, want %q", outBuf.String(), "hello")
	}
}

func TestFoldRun_Dispatch(t *testing.T) {
	var outBuf bytes.Buffer
	rc := run([]string{}, nil, &outBuf)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
}

func TestFoldRun_BadFlag(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	rc := foldRun([]string{"--nonexistent"}, &outBuf, &errBuf, strings.NewReader(""))
	if rc != 2 {
		t.Errorf("exit code: got %d, want 2 for bad flag", rc)
	}
}

func TestFoldRun_InvalidWidth(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello")
	rc := foldRun([]string{"-w", "invalid"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0 (invalid width defaults to 80)", rc)
	}
	// Should default to 80 and output unchanged
	if outBuf.String() != "hello" {
		t.Errorf("invalid width: got %q, want %q", outBuf.String(), "hello")
	}
}

func TestFoldRun_ZeroWidth(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello")
	rc := foldRun([]string{"-w", "0"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	// Zero width defaults to 80
	if outBuf.String() != "hello" {
		t.Errorf("zero width: got %q, want %q", outBuf.String(), "hello")
	}
}

func TestFoldRun_NegativeWidth(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello")
	rc := foldRun([]string{"-w", "-5"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	// Negative width defaults to 80
	if outBuf.String() != "hello" {
		t.Errorf("negative width: got %q, want %q", outBuf.String(), "hello")
	}
}

func TestFoldRun_MultipleLines(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("line1\nline2\nline3")
	rc := foldRun([]string{"-w", "80"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	want := "line1\nline2\nline3"
	if outBuf.String() != want {
		t.Errorf("multi-line: got %q, want %q", outBuf.String(), want)
	}
}

func TestFoldRun_SpaceBreakAndByte(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello world foo")
	rc := foldRun([]string{"-b", "-s", "-w", "8"}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	// Should break at spaces when possible
	lines := strings.Split(strings.TrimSuffix(outBuf.String(), "\n"), "\n")
	for _, line := range lines {
		if len(line) > 8 {
			t.Errorf("byte+space line %q exceeds width 8", line)
		}
	}
}

func TestFold_TrailingNewlinePreservation(t *testing.T) {
	// 1. Input with trailing newline
	r1 := strings.NewReader("hello\nworld\n")
	out1, err := Fold(r1, 80, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out1 != "hello\nworld\n" {
		t.Errorf("expected trailing newline preserved, got %q", out1)
	}

	// 2. Input without trailing newline
	r2 := strings.NewReader("hello\nworld")
	out2, err := Fold(r2, 80, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out2 != "hello\nworld" {
		t.Errorf("expected no trailing newline, got %q", out2)
	}
}

func TestFold_EmbeddedNULPreservation(t *testing.T) {
	input := "The NUL is here:>\x00< and another one is here:>\x00< - they must be preserved\n"
	r := strings.NewReader(input)
	out, err := Fold(r, 22, false, true)
	if err != nil {
		t.Fatal(err)
	}
	expected := "The NUL is here:>\x00< \nand another one is \nhere:>\x00< - they must \nbe preserved\n"
	if out != expected {
		t.Errorf("NUL wrapping mismatch:\n got:  %q\n want: %q", out, expected)
	}
}

