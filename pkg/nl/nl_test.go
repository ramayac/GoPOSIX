package nl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNl_AllLines(t *testing.T) {
	// -b a: number all lines
	input := "line 1\n\nline 3\n"
	r := strings.NewReader(input)
	lines, _ := NumberLines(r, "a", 1, 6)
	got := strings.Join(lines, "\n")
	want := "     1\tline 1\n     2\t\n     3\tline 3"
	if got != want {
		t.Errorf("-b a:\n  got  %q\n  want %q", got, want)
	}
}

func TestNl_NonEmptyLines(t *testing.T) {
	// -b t: number non-empty lines only (default)
	input := "line 1\n\nline 3\n"
	r := strings.NewReader(input)
	lines, _ := NumberLines(r, "t", 1, 6)
	got := strings.Join(lines, "\n")
	want := "     1\tline 1\n       \n     2\tline 3"
	if got != want {
		t.Errorf("-b t:\n  got  %q\n  want %q", got, want)
	}
}

func TestNl_NoLines(t *testing.T) {
	// -b n: no numbering
	input := "line 1\n\nline 3\n"
	r := strings.NewReader(input)
	lines, _ := NumberLines(r, "n", 1, 6)
	got := strings.Join(lines, "\n")
	want := "       line 1\n       \n       line 3"
	if got != want {
		t.Errorf("-b n:\n  got  %q\n  want %q", got, want)
	}
}

func TestNl_CustomStartNumber(t *testing.T) {
	// -v 10: start numbering at 10
	input := "a\nb\nc\n"
	r := strings.NewReader(input)
	lines, _ := NumberLines(r, "a", 10, 6)
	got := strings.Join(lines, "\n")
	want := "    10\ta\n    11\tb\n    12\tc"
	if got != want {
		t.Errorf("-v 10:\n  got  %q\n  want %q", got, want)
	}
}

func TestNl_CustomWidth(t *testing.T) {
	// -w 3: number width of 3
	input := "a\nb\n"
	r := strings.NewReader(input)
	lines, _ := NumberLines(r, "a", 1, 3)
	got := strings.Join(lines, "\n")
	want := "  1\ta\n  2\tb"
	if got != want {
		t.Errorf("-w 3:\n  got  %q\n  want %q", got, want)
	}
}

func TestNl_EmptyInput(t *testing.T) {
	// Empty input produces no lines (bufio.Scanner behavior)
	input := ""
	r := strings.NewReader(input)
	lines, _ := NumberLines(r, "a", 1, 6)
	if len(lines) != 0 {
		t.Errorf("empty input: expected 0 lines, got %d: %q", len(lines), lines)
	}
}

func TestNl_JsonOutput(t *testing.T) {
	input := "line 1\nline 2\n"
	r := strings.NewReader(input)
	_, result := NumberLines(r, "a", 1, 6)
	if len(result.Lines) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Lines))
	}
	if result.Lines[0].Number != 1 || result.Lines[0].Text != "line 1" {
		t.Errorf("line 0: got %+v", result.Lines[0])
	}
	if result.Lines[1].Number != 2 || result.Lines[1].Text != "line 2" {
		t.Errorf("line 1: got %+v", result.Lines[1])
	}
}

func TestNlRun_Stdin(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	r := strings.NewReader("hello\nworld\n")
	rc := nlRun([]string{"-b", "a"}, &outBuf, &errBuf, r, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0. stderr: %s", rc, errBuf.String())
	}
	got := strings.TrimRight(outBuf.String(), "\n")
	want := "     1\thello\n     2\tworld"
	if got != want {
		t.Errorf("run:\n  got  %q\n  want %q", got, want)
	}
}

func TestNlRun_Json(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	r := strings.NewReader("test\n")
	rc := nlRun([]string{"-b", "a", "--json"}, &outBuf, &errBuf, r, "")
	if rc != 0 {
		t.Errorf("exit code: got %d", rc)
	}
	if !strings.Contains(outBuf.String(), `"number":1`) {
		t.Errorf("JSON missing number: %s", outBuf.String())
	}
}

func TestNl_CLIRun(t *testing.T) {
	// Test the CLI glue run() function.
	var outBuf, errBuf bytes.Buffer
	stdin := bytes.NewBufferString("line1\n")
	rc := run([]string{}, stdin, &outBuf, &errBuf, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(outBuf.String(), "1") {
		t.Errorf("expected numbered output, got %q", outBuf.String())
	}
}

func TestNl_NoNumbering(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := nlRun([]string{"-b", "n"}, &out, &errOut, strings.NewReader("a\nb\n"), ""); code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if strings.Contains(out.String(), "1") {
		t.Errorf("expected no line numbers, got %q", out.String())
	}
}

func TestNl_FileInput(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(f, []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := nlRun([]string{f}, &out, &errOut, nil, ""); code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "1") {
		t.Errorf("expected numbered lines, got %q", out.String())
	}
}

func TestNl_FileNotFound(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := nlRun([]string{"/nonexistent-xyz"}, &out, &errOut, nil, ""); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
}

func TestNl_BadFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := nlRun([]string{"--badflag"}, &out, &errOut, nil, ""); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestNl_InvalidVAndW(t *testing.T) {
	var out, errOut bytes.Buffer
	// Invalid values fall back to defaults (start 1, width 6).
	if code := nlRun([]string{"-v", "abc", "-w", "-1"}, &out, &errOut, strings.NewReader("x\n"), ""); code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "1") {
		t.Errorf("expected default numbering, got %q", out.String())
	}
}
