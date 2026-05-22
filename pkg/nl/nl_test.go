package nl

import (
	"bytes"
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
