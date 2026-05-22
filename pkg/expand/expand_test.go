package expand

import (
	"strings"
	"testing"
)

func TestExpand_BusyBoxCase1(t *testing.T) {
	// Tab at pos 0 → 8 spaces; tab at pos 16 → 8 spaces
	in := "\t12345678\t12345678\n"
	want := "        12345678        12345678\n"
	got := expandLine(in, 8, false)
	if got != want {
		t.Errorf("case1:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_BusyBoxCase1_Unicode(t *testing.T) {
	// Δ at col 0 → col 1. Tab at col 1 → 7 spaces to col 8.
	// "12345" at col 8-12. ΔΔΔ at col 13-15.
	// Tab at col 16 → 8 spaces to col 24. "12345678" at col 24-31.
	in := "Δ\t12345ΔΔΔ\t12345678\n"
	want := "Δ       12345ΔΔΔ        12345678\n"
	got := expandLine(in, 8, false)
	if got != want {
		t.Errorf("unicode:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_InitialOnly(t *testing.T) {
	// -i: only expand leading tabs
	in := "\thello\tworld\n"
	want := "        hello\tworld\n"
	got := expandLine(in, 8, true)
	if got != want {
		t.Errorf("-i:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_TabWidth4(t *testing.T) {
	in := "\t1234\t5678\n"
	want := "    1234    5678\n" // 4 spaces each
	got := expandLine(in, 4, false)
	if got != want {
		t.Errorf("-t4:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_MultiLine(t *testing.T) {
	in := "\thello\n\tworld\n"
	want := "        hello\n        world\n"
	got := Transform(in, 8, false)
	if got != want {
		t.Errorf("multiline:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_NoTabs(t *testing.T) {
	in := "hello world\n"
	want := "hello world\n"
	got := expandLine(in, 8, false)
	if got != want {
		t.Errorf("no tabs:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_InitialOnly_InternalTabsPreserved(t *testing.T) {
	in := "hello\tworld\ttest\n"
	want := "hello\tworld\ttest\n"
	got := expandLine(in, 8, true)
	if got != want {
		t.Errorf("-i internal:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_InitialOnly_LeadingAndInternal(t *testing.T) {
	in := "\thello\tworld\n"
	want := "        hello\tworld\n"
	got := expandLine(in, 8, true)
	if got != want {
		t.Errorf("-i mixed:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_TrailingNewline(t *testing.T) {
	in := "\thello\n\tworld\n"
	want := "        hello\n        world\n"
	got := Transform(in, 8, false)
	if got != want {
		t.Errorf("trailing newline:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_NoTrailingNewline(t *testing.T) {
	in := "\thello"
	want := "        hello"
	got := Transform(in, 8, false)
	if got != want {
		t.Errorf("no trailing newline:\n  got  %q\n  want %q", got, want)
	}
}

func TestExpand_Empty(t *testing.T) {
	got := Transform("", 8, false)
	if got != "" {
		t.Errorf("empty: got %q", got)
	}
}

// --- CLI layer tests ---

func TestExpandRun_Stdin(t *testing.T) {
	var out, errOut strings.Builder
	stdin := strings.NewReader("\thello\n")
	rc := expandRun([]string{}, &out, &errOut, stdin, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	want := "        hello\n"
	if out.String() != want {
		t.Errorf("output:\n  got  %q\n  want %q", out.String(), want)
	}
}

func TestExpandRun_JsonFlag(t *testing.T) {
	var out, errOut strings.Builder
	stdin := strings.NewReader("\thello\n")
	rc := expandRun([]string{"--json"}, &out, &errOut, stdin, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(out.String(), "\"lines\"") {
		t.Errorf("JSON output missing 'lines': %s", out.String())
	}
}
