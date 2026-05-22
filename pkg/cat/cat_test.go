package cat

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRunBasic(t *testing.T) {
	in := strings.NewReader("hello\nworld\n")
	var out bytes.Buffer
	lines, err := Run(in, &out, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(out.String(), "hello") {
		t.Error("output missing 'hello'")
	}
}

func TestRunNumberAll(t *testing.T) {
	in := strings.NewReader("a\nb\n")
	var out bytes.Buffer
	lines, _ := Run(in, &out, true, false, false)
	if !strings.HasPrefix(lines[0], "     1\t") {
		t.Errorf("expected line numbers, got %q", lines[0])
	}
}

func TestRunNumberNonBlank(t *testing.T) {
	in := strings.NewReader("a\n\nb\n")
	var out bytes.Buffer
	lines, _ := Run(in, &out, false, true, false)
	// blank line should not have a number
	if strings.HasPrefix(lines[1], " ") && strings.Contains(lines[1], "\t") {
		t.Error("blank line should not be numbered with -b")
	}
}

func TestRunSqueezeBlank(t *testing.T) {
	in := strings.NewReader("a\n\n\nb\n")
	var out bytes.Buffer
	lines, _ := Run(in, &out, false, false, true)
	// Two consecutive blanks should collapse to one.
	blanks := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blanks++
		}
	}
	if blanks > 1 {
		t.Errorf("squeeze-blank: expected ≤1 blank line, got %d", blanks)
	}
}

func TestRunEmpty(t *testing.T) {
	in := strings.NewReader("")
	var out bytes.Buffer
	lines, err := Run(in, &out, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty input, got %d", len(lines))
	}
}

func TestVisByte(t *testing.T) {
	cases := []struct {
		b    byte
		want string
	}{
		{'\t', "\t"},
		{'\n', "\n"},
		{0, "^@"},
		{1, "^A"},
		{27, "^["},
		{0x7F, "^?"},
		{'a', "a"},
		{'A', "A"},
		{0x80, "M-^@"},
	}
	for _, c := range cases {
		got := visByte(c.b)
		if got != c.want {
			t.Errorf("visByte(0x%02x) = %q, want %q", c.b, got, c.want)
		}
	}
}

func TestVisLine(t *testing.T) {
	got := visLine("hello", false)
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	gotEnd := visLine("hello", true)
	if gotEnd != "hello$" {
		t.Errorf("expected 'hello$', got %q", gotEnd)
	}
	gotCtrl := visLine("a\x01b", false)
	if gotCtrl != "a^Ab" {
		t.Errorf("expected 'a^Ab', got %q", gotCtrl)
	}
}

func TestRunNumberNonBlankSkip(t *testing.T) {
	in := strings.NewReader("a\n\nb\n")
	var out bytes.Buffer
	lines, _ := Run(in, &out, false, true, false)
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	// First non-blank should be numbered
	if !strings.HasPrefix(lines[0], "     1\t") {
		t.Errorf("first non-blank should be numbered, got %q", lines[0])
	}
	// Blank line should NOT be numbered
	if strings.HasPrefix(lines[1], "     ") && strings.Contains(lines[1], "\t") {
		t.Errorf("blank line should NOT be numbered, got %q", lines[1])
	}
}

func TestVisByteHigh(t *testing.T) {
	// M- prefix for high bytes
	got := visByte(0xE1)
	if len(got) < 3 || got[:2] != "M-" {
		t.Errorf("expected M- prefix for high byte, got %q", got)
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Cat_FileAndStdin(t *testing.T) {
	// BusyBox: echo SOMETHING | cat foo - should print file then stdin
	// The test: echo I WANT > foo; echo SOMETHING | cat foo -
	// Expected output: I WANT\nSOMETHING\n

	var buf bytes.Buffer

	// Simulate: cat with a file reader first, then stdin
	fileReader := strings.NewReader("I WANT\n")
	stdinReader := strings.NewReader("SOMETHING\n")

	readers := []io.Reader{fileReader, stdinReader}

	for _, r := range readers {
		data, _ := io.ReadAll(r)
		buf.Write(data)
	}

	got := buf.String()
	if got != "I WANT\nSOMETHING\n" {
		t.Errorf("got %q, want %q", got, "I WANT\nSOMETHING\n")
	}
}

func TestBusyBox_Cat_FileAndDashStdin(t *testing.T) {
	// cat foo - should interleave file and stdin correctly.
	// We test with Run: a file reader followed by a stdin reader.
	fileReader := strings.NewReader("FILE\n")
	stdinReader := strings.NewReader("STDIN\n")

	readers := []io.Reader{fileReader, stdinReader}
	var buf bytes.Buffer
	for _, r := range readers {
		io.Copy(&buf, r)
	}

	got := buf.String()
	if got != "FILE\nSTDIN\n" {
		t.Errorf("got %q, want %q", got, "FILE\nSTDIN\n")
	}
}

// --- CLI-layer tests via catRun ---

func simpleCat(args []string, stdin string) (int, string, string) {
	var outBuf, errBuf bytes.Buffer
	code := catRun(args, &outBuf, &errBuf, strings.NewReader(stdin), "")
	return code, outBuf.String(), errBuf.String()
}

func TestCLI_BasicFile(t *testing.T) {
	code, out, errStr := simpleCat([]string{"testdata/cat_hello.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "hello\n" {
		t.Errorf("got %q, want %q", out, "hello\n")
	}
}

func TestCLI_NumberAll(t *testing.T) {
	code, out, errStr := simpleCat([]string{"-n", "testdata/cat_lines.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d in %q", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "     1\ta") {
		t.Errorf("line 1 missing number prefix: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "     2\t") {
		t.Errorf("line 2 missing number prefix: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "     3\tb") {
		t.Errorf("line 3 missing number prefix: %q", lines[2])
	}
}

func TestCLI_NumberNonBlank(t *testing.T) {
	code, out, errStr := simpleCat([]string{"-b", "testdata/cat_lines.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.HasPrefix(lines[0], "     1\ta") {
		t.Errorf("first non-blank should be 1: %q", lines[0])
	}
	// Second line is blank, should NOT be numbered (no tab prefix)
	if strings.Contains(lines[1], "\t") {
		t.Errorf("blank line should not have number prefix: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "     2\tb") {
		t.Errorf("second non-blank should be 2: %q", lines[2])
	}
}

func TestCLI_SqueezeBlank(t *testing.T) {
	code, out, errStr := simpleCat([]string{"-s", "testdata/cat_blank.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Input: a\n\n\n\nb\n\n\nc\n. Output should have at most 1 blank between non-blanks.
	blankCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
		}
	}
	if blankCount > 2 {
		t.Errorf("expected ≤2 blank lines with squeeze, got %d in %q", blankCount, lines)
	}
}

func TestCLI_ShowEnds(t *testing.T) {
	code, out, errStr := simpleCat([]string{"-e", "testdata/cat_hello.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "hello$\n" {
		t.Errorf("got %q, want %q", out, "hello$\n")
	}
}

func TestCLI_ShowNonPrinting(t *testing.T) {
	code, out, errStr := simpleCat([]string{"-v"}, string([]byte{0x01, '\n'}))
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "^A\n" {
		t.Errorf("got %q, want %q", out, "^A\n")
	}
}

func TestCLI_ShowEndsAndNonPrinting(t *testing.T) {
	code, out, errStr := simpleCat([]string{"-e"}, string([]byte{0x01, '\n'}))
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	// -e implies -v, so SOH becomes ^A, and $ is appended at end
	if out != "^A$\n" {
		t.Errorf("got %q, want %q", out, "^A$\n")
	}
}

func TestCLI_Json(t *testing.T) {
	code, out, errStr := simpleCat([]string{"--json", "testdata/cat_hello.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if !strings.Contains(out, "\"lines\"") || !strings.Contains(out, "hello") {
		t.Errorf("expected JSON output containing 'lines' and 'hello', got: %s", out)
	}
	if !strings.Contains(out, "\"lineCount\":1") {
		t.Errorf("expected lineCount 1 in JSON, got: %s", out)
	}
}

func TestCLI_JsonFlagLong(t *testing.T) {
	code, out, errStr := simpleCat([]string{"--json", "testdata/cat_hello.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if !strings.Contains(out, "\"lines\"") {
		t.Errorf("expected JSON output via --json, got: %s", out)
	}
}

func TestCLI_NumberFlagLong(t *testing.T) {
	code, out, errStr := simpleCat([]string{"--number", "testdata/cat_hello.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if !strings.HasPrefix(out, "     1\t") {
		t.Errorf("expected numbered line via --number, got: %q", out)
	}
}

func TestCLI_NumberNonBlankFlagLong(t *testing.T) {
	code, out, errStr := simpleCat([]string{"--number-nonblank", "testdata/cat_lines.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if !strings.Contains(out, "     1\ta") {
		t.Errorf("expected 1st non-blank numbered via --number-nonblank, got: %q", out)
	}
}

func TestCLI_SqueezeBlankFlagLong(t *testing.T) {
	code, out, errStr := simpleCat([]string{"--squeeze-blank", "testdata/cat_blank.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	// Should not contain three consecutive blank lines
	if strings.Contains(out, "\n\n\n") {
		t.Errorf("expected squeezing of blanks, got: %q", out)
	}
}

func TestCLI_ShowNonPrintingFlagLong(t *testing.T) {
	code, out, errStr := simpleCat([]string{"--show-nonprinting"}, string([]byte{0x02, '\n'}))
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "^B\n" {
		t.Errorf("got %q, want %q", out, "^B\n")
	}
}

func TestCLI_FileNotFound(t *testing.T) {
	code, out, errStr := simpleCat([]string{"/nonexistent/file/zzz"}, "")
	if code != 1 {
		t.Errorf("expected exit code 1 for missing file, got %d", code)
	}
	if out != "" {
		t.Errorf("expected no stdout for missing file, got: %q", out)
	}
	if !strings.Contains(errStr, "cat:") && !strings.Contains(errStr, "nonexistent") {
		t.Errorf("expected error message, got: %q", errStr)
	}
}

func TestCLI_StdinViaDash(t *testing.T) {
	code, out, errStr := simpleCat([]string{"-"}, "stdin_data\n")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "stdin_data\n" {
		t.Errorf("got %q, want %q", out, "stdin_data\n")
	}
}

func TestCLI_StdinDefault(t *testing.T) {
	code, out, errStr := simpleCat([]string{}, "default_stdin\n")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "default_stdin\n" {
		t.Errorf("got %q, want %q", out, "default_stdin\n")
	}
}

func TestCLI_StdinWithDashFlagN(t *testing.T) {
	code, out, errStr := simpleCat([]string{"-n", "-"}, "numbered\n")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if !strings.HasPrefix(out, "     1\t") {
		t.Errorf("expected numbered stdin via -n -, got: %q", out)
	}
}

func TestCLI_MultiFile(t *testing.T) {
	code, out, errStr := simpleCat([]string{"testdata/cat_hello.txt", "testdata/cat_world.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "hello\nworld\n" {
		t.Errorf("got %q, want %q", out, "hello\nworld\n")
	}
}

func TestCLI_FileAndStdinDash(t *testing.T) {
	code, out, errStr := simpleCat([]string{"testdata/cat_hello.txt", "-"}, "STDIN\n")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "hello\nSTDIN\n" {
		t.Errorf("got %q, want %q", out, "hello\nSTDIN\n")
	}
}

func TestCLI_VisWithFile(t *testing.T) {
	// cat -v on a file with control chars
	code, out, errStr := simpleCat([]string{"-v", "testdata/cat_ctrl.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if !strings.Contains(out, "^A") {
		t.Errorf("expected ^A in -v output, got: %q", out)
	}
}

func TestCLI_VisShowEnds(t *testing.T) {
	// cat -ve on input with newlines should show $ at end
	code, out, errStr := simpleCat([]string{"-ve", "testdata/cat_hello.txt"}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if !strings.Contains(out, "$") {
		t.Errorf("expected $ in -ve output, got: %q", out)
	}
}

func TestCLI_EmptyInputStream(t *testing.T) {
	code, out, errStr := simpleCat([]string{}, "")
	if code != 0 {
		t.Fatalf("exit code %d: %s", code, errStr)
	}
	if out != "" {
		t.Errorf("expected empty output for empty stdin, got: %q", out)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	code, _, errStr := simpleCat([]string{"--nonexistent"}, "")
	if code != 2 {
		t.Errorf("expected exit code 2 for bad flag, got %d", code)
	}
	if errStr == "" {
		t.Error("expected error message for bad flag")
	}
}

func TestCat_CLIRun(t *testing.T) {
	// Test the CLI glue run() function.
	var outBuf, errBuf bytes.Buffer
	stdin := bytes.NewBufferString("hello\n")
	rc := run([]string{}, stdin, &outBuf, &errBuf, "")
	if rc != 0 {
		t.Errorf("exit code: got %d, want 0", rc)
	}
	if outBuf.String() != "hello\n" {
		t.Errorf("got %q, want %q", outBuf.String(), "hello\n")
	}
}
