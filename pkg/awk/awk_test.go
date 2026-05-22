package awk

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// helper runs the awk CLI and returns combined stdout+stderr + exit code.
func helper(t *testing.T, args []string, stdin string) (string, int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	stdinReader := strings.NewReader(stdin)
	code := awkRun(args, &outBuf, &errBuf, stdinReader, "")
	return outBuf.String() + errBuf.String(), code
}

func TestFieldSplitting(t *testing.T) {
	out, code := helper(t, []string{"-F", ":", "{ print $1 }"}, "a:b:c\nd:e:f\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "a") || !strings.Contains(out, "d") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestFieldSplittingConcat(t *testing.T) {
	out, code := helper(t, []string{"-F:", "{ print $1 }"}, "a:b:c\nd:e:f\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "a") || !strings.Contains(out, "d") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestDefaultWhitespace(t *testing.T) {
	out, code := helper(t, []string{"{ print $1 }"}, "alice 90\nbob 85\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestPrintDollarZero(t *testing.T) {
	out, code := helper(t, []string{"{ print $0 }"}, "hello world\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestBEGINBlock(t *testing.T) {
	out, code := helper(t, []string{"BEGIN { print \"start\" }"}, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "start") {
		t.Errorf("expected 'start' in output: %q", out)
	}
}

func TestENDBlock(t *testing.T) {
	out, code := helper(t, []string{"{ sum += $1 } END { print sum }"}, "10\n20\n30\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "60") {
		t.Errorf("expected sum 60 in output: %q", out)
	}
}

func TestPatternMatching(t *testing.T) {
	out, code := helper(t, []string{"/foo/ { print $1 }"}, "foo bar\nbaz qux\nfoo baz\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if strings.Count(out, "foo") != 2 {
		t.Errorf("expected 2 matches, got: %q", out)
	}
}

func TestExpressionPattern(t *testing.T) {
	out, code := helper(t, []string{"$2 > 50 { print $1 }"}, "alice 90\nbob 30\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if strings.Contains(out, "bob") || !strings.Contains(out, "alice") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestNRAndNF(t *testing.T) {
	out, code := helper(t, []string{"{ print NR, NF }"}, "a b c\nd e\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "1 3") || !strings.Contains(out, "2 2") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestUserVariables(t *testing.T) {
	out, code := helper(t, []string{"{ total += $2 } END { print total }"}, "a 10\nb 20\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "30") {
		t.Errorf("expected 30, got: %q", out)
	}
}

func TestLengthFunction(t *testing.T) {
	out, code := helper(t, []string{"{ print length($0) }"}, "hello\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "5") {
		t.Errorf("expected 5, got: %q", out)
	}
}

func TestSubstrFunction(t *testing.T) {
	out, code := helper(t, []string{"{ print substr($0, 2, 3) }"}, "abcdef\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "bcd") {
		t.Errorf("expected bcd, got: %q", out)
	}
}

func TestSubFunction(t *testing.T) {
	out, code := helper(t, []string{"{ sub(/foo/, \"bar\") } 1"}, "hello foo world\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "hello bar world") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestGsubFunction(t *testing.T) {
	out, code := helper(t, []string{"{ gsub(/foo/, \"bar\") } 1"}, "foo foo foo\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if strings.Count(out, "bar") != 3 {
		t.Errorf("expected 3 bars, got: %q", out)
	}
}

func TestIntFunction(t *testing.T) {
	out, code := helper(t, []string{"{ print int($1) }"}, "42.9\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("expected 42, got: %q", out)
	}
}

func TestArithmetic(t *testing.T) {
	out, code := helper(t, []string{"{ print $1 + $2 }"}, "10 5\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "15") {
		t.Errorf("expected 15, got: %q", out)
	}
}

func TestIfElse(t *testing.T) {
	out, code := helper(t, []string{"{ if ($1 > 5) print \"big\"; else print \"small\" }"}, "3\n8\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "small") || !strings.Contains(out, "big") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestWhileLoop(t *testing.T) {
	out, code := helper(t, []string{"{ i = 1; while (i <= 3) { print i; i++ } }"}, "\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if strings.Count(out, "1") < 1 || strings.Count(out, "2") < 1 || strings.Count(out, "3") < 1 {
		t.Errorf("expected 1 2 3, got: %q", out)
	}
}

func TestForLoop(t *testing.T) {
	out, code := helper(t, []string{"{ for (i = 1; i <= 3; i++) print i }"}, "\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if strings.Count(out, "1") < 1 || strings.Count(out, "3") < 1 {
		t.Errorf("expected 1 2 3, got: %q", out)
	}
}

func TestForInLoop(t *testing.T) {
	out, code := helper(t, []string{"{ arr[$1] = $2 } END { for (k in arr) print k, arr[k] }"}, "a 1\nb 2\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "a 1") || !strings.Contains(out, "b 2") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestArrays(t *testing.T) {
	out, code := helper(t, []string{"{ a[$1] = $2 } END { print a[\"x\"] }"}, "x 99\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "99") {
		t.Errorf("expected 99, got: %q", out)
	}
}

func TestDeleteArray(t *testing.T) {
	out, code := helper(t, []string{"BEGIN { a[1]=10; a[2]=20; delete a[1]; print length(a) }"}, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "1") {
		t.Errorf("expected length 1, got: %q", out)
	}
}

func TestVFlag(t *testing.T) {
	out, code := helper(t, []string{"-v", "threshold=50", "$2 > threshold { print $1 }"}, "alice 90\nbob 30\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if strings.Contains(out, "bob") || !strings.Contains(out, "alice") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestFFlagFromFile(t *testing.T) {
	tmp, err := os.CreateTemp("", "awk-prog-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("{ print $1 }\n")
	tmp.Close()

	out, code := helper(t, []string{"-f", tmp.Name()}, "hello world\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected 'hello', got: %q", out)
	}
}

func TestCloser(t *testing.T) {
	out, code := helper(t, []string{"{ close(\"/dev/null\") } 1"}, "ok\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("expected 'ok', got: %q", out)
	}
}

func TestSyntaxError(t *testing.T) {
	_, code := helper(t, []string{"{"}, "")
	if code != 2 {
		t.Errorf("exit code %d, want 2", code)
	}
}

func TestNoProgram(t *testing.T) {
	_, code := helper(t, []string{}, "")
	if code != 2 {
		t.Errorf("exit code %d, want 2", code)
	}
}

func TestEmptyProgram(t *testing.T) {
	_, code := helper(t, []string{"BEGIN { exit 0 }"}, "")
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
}

func TestInvalidFFlag(t *testing.T) {
	_, code := helper(t, []string{"-f", "/nonexistent/awk/prog"}, "")
	if code != 2 {
		t.Errorf("exit code %d, want 2", code)
	}
}

func TestFieldSeparatorDefault(t *testing.T) {
	out, code := helper(t, []string{"{ print NF }"}, "a\tb\tc\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	// Default FS splits on whitespace, tabs included
	if !strings.Contains(out, "3") {
		t.Errorf("expected NF=3, got: %q", out)
	}
}

func TestAssignmentExpression(t *testing.T) {
	out, code := helper(t, []string{"{ x = $1 + 10; print x }"}, "5\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "15") {
		t.Errorf("expected 15, got: %q", out)
	}
}

func TestSplitFunction(t *testing.T) {
	out, code := helper(t, []string{"{ n = split($0, a, \",\"); print n, a[2] }"}, "x,y,z\n")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(out, "3") || !strings.Contains(out, "y") {
		t.Errorf("unexpected output: %q", out)
	}
}

// TestAWKRunInjectsWriters tests that the injectable entry point
// awkRun correctly routes stdout and stderr to the provided writers.
func TestAWKRunInjectsWriters(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("hello\n")
	code := awkRun([]string{"{ print $1 }"}, &outBuf, &errBuf, stdin, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if !strings.Contains(outBuf.String(), "hello") {
		t.Errorf("expected output to contain 'hello', got: %q", outBuf.String())
	}
	if errBuf.Len() > 0 {
		t.Errorf("expected no stderr, got: %q", errBuf.String())
	}
}

// TestJSONMode tests that --json captures awk output into structured lines.
func TestJSONMode(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("alice 90\nbob 85\n")
	code := awkRun([]string{"--json", "{ print $1, $2 }"}, &outBuf, &errBuf, stdin, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0 (stderr: %q)", code, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, `"command":"awk"`) {
		t.Errorf("expected 'command':'awk' in JSON output, got: %q", out)
	}
	if !strings.Contains(out, `"exitCode":0`) {
		t.Errorf("expected 'exitCode':0, got: %q", out)
	}
	if !strings.Contains(out, `"lines"`) {
		t.Errorf("expected 'lines' in data, got: %q", out)
	}
	if !strings.Contains(out, `"alice 90"`) {
		t.Errorf("expected 'alice 90' in lines, got: %q", out)
	}
	if !strings.Contains(out, `"bob 85"`) {
		t.Errorf("expected 'bob 85' in lines, got: %q", out)
	}
}

// TestJSONModeNoOutput tests --json with a program that produces no output.
func TestJSONModeNoOutput(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("")
	code := awkRun([]string{"--json", "BEGIN { x = 1 }"}, &outBuf, &errBuf, stdin, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0 (stderr: %q)", code, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, `"lineCount":0`) {
		t.Errorf("expected 'lineCount':0 in output, got: %q", out)
	}
}

func TestRunLibraryFunc(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("hello world\nfoo bar\n")
	status, err := Run("{ print $1 }", nil, " ", nil, stdin, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if status != 0 {
		t.Errorf("status %d, want 0", status)
	}
	out := outBuf.String()
	if !strings.Contains(out, "hello") || !strings.Contains(out, "foo") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRunLibraryFuncWithFiles(t *testing.T) {
	// Create temp file
	tmp, err := os.CreateTemp("", "awk-input-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("line one\nline two\n")
	tmp.Close()

	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("")
	status, err := Run("{ print NR, $0 }", []string{tmp.Name()}, " ", nil, stdin, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if status != 0 {
		t.Errorf("status %d, want 0", status)
	}
	out := outBuf.String()
	if !strings.Contains(out, "1 line one") || !strings.Contains(out, "2 line two") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRunLibraryFuncWithVars(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("alice 90\nbob 30\n")
	status, err := Run("$2 > threshold { print $1 }", nil, " ",
		[]string{"threshold=50"}, stdin, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if status != 0 {
		t.Errorf("status %d, want 0", status)
	}
	out := outBuf.String()
	if strings.Contains(out, "bob") || !strings.Contains(out, "alice") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRunLibraryFuncSyntaxError(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("")
	status, err := Run("{", nil, " ", nil, stdin, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 2 {
		t.Errorf("status %d, want 2", status)
	}
	if errBuf.Len() == 0 {
		t.Error("expected error message on stderr")
	}
}

func TestRunCapture(t *testing.T) {
	var errBuf bytes.Buffer
	stdin := strings.NewReader("alice 90\nbob 85\n")
	lines, status, err := RunCapture("{ print $1 }", nil, " ", nil, stdin, &errBuf)
	if err != nil {
		t.Fatalf("RunCapture error: %v", err)
	}
	if status != 0 {
		t.Errorf("status %d, want 0", status)
	}
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %q", len(lines), lines)
	}
	if lines[0] != "alice" || lines[1] != "bob" {
		t.Errorf("unexpected lines: %q", lines)
	}
}

func TestRunCaptureWithFiles(t *testing.T) {
	tmp, err := os.CreateTemp("", "awk-input-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("line one\nline two\n")
	tmp.Close()

	var errBuf bytes.Buffer
	stdin := strings.NewReader("")
	lines, status, err := RunCapture("{ print NR, $0 }", []string{tmp.Name()}, " ", nil, stdin, &errBuf)
	if err != nil {
		t.Fatalf("RunCapture error: %v", err)
	}
	if status != 0 {
		t.Errorf("status %d, want 0", status)
	}
	if lines[0] != "1 line one" || lines[1] != "2 line two" {
		t.Errorf("unexpected lines: %q", lines)
	}
}

func TestRunCaptureNoOutput(t *testing.T) {
	var errBuf bytes.Buffer
	stdin := strings.NewReader("")
	lines, status, err := RunCapture("BEGIN { x = 1 }", nil, " ", nil, stdin, &errBuf)
	if err != nil {
		t.Fatalf("RunCapture error: %v", err)
	}
	if status != 0 {
		t.Errorf("status %d, want 0", status)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d: %q", len(lines), lines)
	}
}

func TestRunCaptureSyntaxError(t *testing.T) {
	var errBuf bytes.Buffer
	stdin := strings.NewReader("")
	lines, status, err := RunCapture("{", nil, " ", nil, stdin, &errBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 2 {
		t.Errorf("status %d, want 2", status)
	}
	if lines != nil {
		t.Errorf("expected nil lines on error, got: %q", lines)
	}
}

func TestJSONModeErrorExit(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	stdin := strings.NewReader("")
	code := awkRun([]string{"--json", "BEGIN { exit 3 }"}, &outBuf, &errBuf, stdin, "")
	if code != 0 {
		t.Fatalf("exit code %d, want 0 — JSON mode always returns 0", code)
	}
	out := outBuf.String()
	if !strings.Contains(out, `"status":3`) {
		t.Errorf("expected 'status':3 in JSON, got: %q", out)
	}
}
