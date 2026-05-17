package fold

import (
	"bytes"
	"strings"
	"testing"
)

func TestFold_SpaceBreak(t *testing.T) {
	// fold -w 7 -s: input "123456\tasdf"
	//  6 cols + tab(→8) past width 7 → break before tab
	//  tab alone on new line (1 char → tab at col 0→8 past width 7)
	//  "asdf" on last line
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
	// fold -w1: each char on own line
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
	// fold -sw22 with NUL bytes
	input := "The NUL is here:>\x00< and another one is here:>\x00< - they must be preserved"
	r := strings.NewReader(input)
	out, err := Fold(r, 22, false, true)
	if err != nil {
		t.Fatal(err)
	}
	// NUL bytes must be preserved in output
	if !strings.Contains(out, "\x00") {
		t.Error("NUL bytes missing from output")
	}
	// Lines should be wrapped at ~22 cols
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Errorf("expected at least 3 wrapped lines, got %d: %q", len(lines), out)
	}
}

func TestFold_Default(t *testing.T) {
	// Long line with default 80 width
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
	// When no space exists in width and -s is set, break at width
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

// --- CLI layer ---

func TestFoldRun_Stdin(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	in := strings.NewReader("hello")
	rc := foldRun([]string{}, &outBuf, &errBuf, in)
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if outBuf.String() != "hello" {
		t.Errorf("got %q, want %q", outBuf.String(), "hello\n")
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
