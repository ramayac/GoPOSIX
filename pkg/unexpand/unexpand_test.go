package unexpand

import (
	"bytes"
	"strings"
	"testing"
)

// TestUnexpand_BusyBox mirrors the 11+ cases from test/busybox_testsuite/unexpand.tests.
// Tab width is 8 throughout (POSIX default).

func TestUnexpand_BusyBoxCase1(t *testing.T) {
	// 8 leading spaces -> 1 tab
	in := "        12345678\n"
	want := "\t12345678\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("case1:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_BusyBoxCase2(t *testing.T) {
	// 9 leading spaces -> tab + space
	in := "         12345678\n"
	want := "\t 12345678\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("case2:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_BusyBoxCase3(t *testing.T) {
	// 10 leading spaces -> tab + 2 spaces
	in := "          12345678\n"
	want := "\t  12345678\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("case3:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_BusyBoxCase4(t *testing.T) {
	// 7 spaces + tab -> tab
	in := "       \t12345678\n"
	want := "\t12345678\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("case4:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_BusyBoxCase5(t *testing.T) {
	// 6 spaces + tab -> tab
	in := "      \t12345678\n"
	want := "\t12345678\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("case5:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_BusyBoxCase6(t *testing.T) {
	// 5 spaces + tab -> tab
	in := "     \t12345678\n"
	want := "\t12345678\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("case6:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_BusyBoxCase7(t *testing.T) {
	// Space + tab in middle of line -> tab (BusyBox converts tab-containing runs)
	in := "123 \t 45678\n"
	want := "123\t 45678\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("case7:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_BusyBoxCase8(t *testing.T) {
	// Single space between words -> no change
	in := "a b\n"
	want := "a b\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("case8:\n  got  %q\n  want %q", got, want)
	}
}

// TestUnexpand_BusyBoxCase8_All verifies -a mode doesn't touch single spaces.
func TestUnexpand_BusyBoxCase8_All(t *testing.T) {
	in := "a b\n"
	want := "a b\n"
	got := unexpandLine(in, 8, true)
	if got != want {
		t.Errorf("case8 -a:\n  got  %q\n  want %q", got, want)
	}
}

// testCaseInput is the testcase() fixture from the BusyBox test.
const testCaseInput = "        a       b    c"

func TestUnexpand_FirstOnly_DefaultTab(t *testing.T) {
	// --first-only (default): only leading blanks converted.
	// 8 leading spaces -> tab. Rest unchanged.
	want := "\ta       b    c"
	got := unexpandLine(testCaseInput, 8, false)
	if got != want {
		t.Errorf("first-only default:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_All_DefaultTab(t *testing.T) {
	// -a: convert ALL blanks.
	want := "\ta\tb    c"
	got := unexpandLine(testCaseInput, 8, true)
	if got != want {
		t.Errorf("-a:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_All_Tab4(t *testing.T) {
	// -a -t4: all blanks, 4-char tab stops
	want := "\t\ta\t\tb\t c"
	got := unexpandLine(testCaseInput, 4, true)
	if got != want {
		t.Errorf("-a -t4:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_FirstOnly_Tab4(t *testing.T) {
	// --first-only -t4: only leading blanks, 4-char tabs
	want := "\t\ta       b    c"
	got := unexpandLine(testCaseInput, 4, false)
	if got != want {
		t.Errorf("--first-only -t4:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_MultiLine(t *testing.T) {
	in := "        a       b    c\n" +
		"         12345678\n" +
		"     \t12345678\n"
	want := "\ta       b    c\n" +
		"\t 12345678\n" +
		"\t12345678\n"
	got := Transform(in, 8, false)
	if got != want {
		t.Errorf("multiline:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_NoTrailingNewline(t *testing.T) {
	in := "        hello"
	want := "\thello"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("no trailing newline:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_EmptyLine(t *testing.T) {
	in := ""
	want := ""
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("empty:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_OnlySpaces(t *testing.T) {
	// 16 spaces -> 2 tabs
	in := "                "
	want := "\t\t"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("only spaces:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_MixedSpacesAndTabs_Leading(t *testing.T) {
	// Space + tab + space + tab -> minimum representation
	in := " \t \thello\n"
	// Space(1) + tab(→8) + space(9) + tab(→16) = col 16
	// Minimum: tab(→8) + tab(→16) = col 16
	want := "\t\thello\n"
	got := unexpandLine(in, 8, false)
	if got != want {
		t.Errorf("mixed blanks:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_TabWithNonDefaultWidth(t *testing.T) {
	// -t4: 8 spaces -> 2 tabs (each tab covers 4)
	in := "        12345678\n"
	want := "\t\t12345678\n"
	got := unexpandLine(in, 4, false)
	if got != want {
		t.Errorf("tab4:\n  got  %q\n  want %q", got, want)
	}
}

func TestUnexpand_AllBlanks_MultipleRuns(t *testing.T) {
	// -a -t8: multiple blank runs in one line
	// "hello" (cols 0-4) + 6 spaces (cols 5-10) + "world" + 2 spaces + "foo"
	// 6 spaces from col 5: tab stop at 8. Tab covers 3 (cols 5-7), then 3 spaces (cols 8-10).
	// Result: "hello\t   world  foo"
	in := "hello      world  foo\n"
	got := unexpandLine(in, 8, true)
	t.Logf("got: %q", got)
	if !strings.Contains(got, "\t") {
		t.Error("expected at least one tab in output")
	}
	// Verify column alignment is preserved: after 'hello' + tab + 3 spaces = same visual pos
	if !strings.HasPrefix(got, "hello\t   ") {
		t.Errorf("unexpected prefix: %q", got)
	}
}

// --- CLI layer tests (via injectable unexpandRun) ---

func TestUnexpandRun_Stdin(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("        12345678\n")
	rc := unexpandRun([]string{}, &out, &errOut, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	want := "\t12345678\n"
	if out.String() != want {
		t.Errorf("output:\n  got  %q\n  want %q", out.String(), want)
	}
}

func TestUnexpandRun_JsonFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("        hello\n")
	rc := unexpandRun([]string{"--json"}, &out, &errOut, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if !strings.Contains(out.String(), "\"lines\"") {
		t.Errorf("JSON output missing 'lines': %s", out.String())
	}
}

func TestUnexpandRun_AllFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("123     456\n")
	rc := unexpandRun([]string{"-a"}, &out, &errOut, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	// Should contain a tab between 123 and 456
	if !strings.Contains(out.String(), "\t") {
		t.Errorf("-a output should contain tab: %q", out.String())
	}
}

func TestUnexpandRun_TabWidth(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("        12345678\n")
	rc := unexpandRun([]string{"-t", "4"}, &out, &errOut, in, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	want := "\t\t12345678\n"
	if out.String() != want {
		t.Errorf("-t4 output:\n  got  %q\n  want %q", out.String(), want)
	}
}

func TestUnexpandRun_MultipleFiles(t *testing.T) {
	t.Skip("requires temp files — tested via integration")
}
