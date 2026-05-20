package head

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ramayac/goposix/internal/dispatch"
)

// captureOutput runs fn with os.Stdout and os.Stderr redirected to pipes.
// Returns (stdout, stderr) as strings.
func captureOutput(fn func()) (string, string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return bufOut.String(), bufErr.String()
}

// captureOutputWithStdin runs fn with os.Stdin set to input and stdout/stderr captured.
func captureOutputWithStdin(input string, fn func()) (string, string) {
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rIn, wIn, _ := os.Pipe()
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdin = rIn
	os.Stdout = wOut
	os.Stderr = wErr

	go func() {
		wIn.Write([]byte(input))
		wIn.Close()
	}()

	fn()

	wOut.Close()
	wErr.Close()

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	os.Stdin = oldStdin
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return bufOut.String(), bufErr.String()
}

// helper: get command and run
func runHead(args []string) (int, string, string) {
	cmd, ok := dispatch.Lookup("head")
	if !ok {
		panic("head not registered; import pkg/head in test")
	}
	var jsonBuf bytes.Buffer

	var exitCode int
	_, stderr := captureOutput(func() {
		exitCode = cmd.Run(args, &jsonBuf)
	})
	return exitCode, jsonBuf.String(), stderr
}

func runHeadWithStdin(stdin string, args []string) (int, string, string) {
	cmd, ok := dispatch.Lookup("head")
	if !ok {
		panic("head not registered; import pkg/head in test")
	}
	var jsonBuf bytes.Buffer

	var exitCode int
	_, stderr := captureOutputWithStdin(stdin, func() {
		exitCode = cmd.Run(args, &jsonBuf)
	})
	return exitCode, jsonBuf.String(), stderr
}

// ========== Existing library-layer tests (kept for regression) ==========

func TestRunBasic(t *testing.T) {
	in := strings.NewReader("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n")
	var out bytes.Buffer
	lines, err := Run(in, &out, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}
	if !strings.Contains(out.String(), "10") || strings.Contains(out.String(), "11") {
		t.Error("output mismatch")
	}
}

func TestRunShort(t *testing.T) {
	in := strings.NewReader("1\n2\n")
	var out bytes.Buffer
	lines, _ := Run(in, &out, 10)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestRunBytes(t *testing.T) {
	in := strings.NewReader("abcdefghij")
	var out bytes.Buffer
	lines, err := runBytes(in, &out, 5)
	if err != nil {
		t.Fatal(err)
	}
	if lines != nil {
		t.Error("runBytes should return nil lines")
	}
	if out.String() != "abcde" {
		t.Errorf("expected 'abcde', got %q", out.String())
	}
}

func TestRunBytesExact(t *testing.T) {
	in := strings.NewReader("abc")
	var out bytes.Buffer
	_, err := runBytes(in, &out, 5)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "abc" {
		t.Errorf("expected 'abc', got %q", out.String())
	}
}

func TestRunBytesEmpty(t *testing.T) {
	in := strings.NewReader("")
	var out bytes.Buffer
	_, err := runBytes(in, &out, 5)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "" {
		t.Errorf("expected empty, got %q", out.String())
	}
}

func TestRunNegative(t *testing.T) {
	in := strings.NewReader("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n")
	var out bytes.Buffer
	lines, err := runNegative(in, &out, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 8 {
		t.Errorf("expected 8 lines (10-2), got %d", len(lines))
	}
}

func TestRunNegativeSkipAll(t *testing.T) {
	in := strings.NewReader("a\nb\n")
	var out bytes.Buffer
	lines, err := runNegative(in, &out, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestRunZeroLines(t *testing.T) {
	in := strings.NewReader("1\n2\n3\n")
	var out bytes.Buffer
	lines, _ := Run(in, &out, 0)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestRunEmpty(t *testing.T) {
	in := strings.NewReader("")
	var out bytes.Buffer
	lines, err := Run(in, &out, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

// ========== CLI dispatch-call tests ==========

const testdataDir = "../../testdata/head/"

func TestRunCLI_Basic(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{testdataDir + "ten.txt"})
	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d: %q", len(lines), stdout)
	}
	if lines[0] != "1" || lines[9] != "10" {
		t.Errorf("unexpected output: %q", stdout)
	}
}

func TestRunCLI_LinesFlag(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"-n", "3", testdataDir + "ten.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_LinesFlagAttached(t *testing.T) {
	// -n5 without space (POSIX allows this)
	exitCode, stdout, stderr := runHead([]string{"-n5", testdataDir + "ten.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_NegativeLines(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"-n", "-3", testdataDir + "ten.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 7 {
		t.Errorf("expected 7 lines (10-3), got %d: %q", len(lines), stdout)
	}
	if lines[6] != "7" {
		t.Errorf("last line should be 7, got %q", lines[6])
	}
}

func TestRunCLI_NegativeAttached(t *testing.T) {
	// -n-2 in one arg
	exitCode, stdout, stderr := runHead([]string{"-n-2", testdataDir + "ten.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 8 {
		t.Errorf("expected 8 lines (10-2), got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_BytesFlag(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"-c", "5", testdataDir + "bytes.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	if stdout != "abcde" {
		t.Errorf("expected 'abcde', got %q", stdout)
	}
}

func TestRunCLI_BytesFlagZero(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"-c", "0", testdataDir + "ten.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	if stdout != "" {
		t.Errorf("expected empty, got %q", stdout)
	}
}

func TestRunCLI_MultiFile(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{testdataDir + "ten.txt", testdataDir + "five.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	// Should have headers and output for both files
	if !strings.Contains(stdout, "==> "+testdataDir+"ten.txt <==") {
		t.Errorf("missing header for ten.txt: %q", stdout)
	}
	if !strings.Contains(stdout, "==> "+testdataDir+"five.txt <==") {
		t.Errorf("missing header for five.txt: %q", stdout)
	}
	// ten.txt has 10 lines, head defaults to 10 lines → all 10
	// five.txt has 5 lines → all 5
	if !strings.Contains(stdout, "10") {
		t.Error("expected '10' in output")
	}
	if !strings.Contains(stdout, "e") {
		t.Error("expected 'e' in output")
	}
}

func TestRunCLI_Stdin(t *testing.T) {
	exitCode, stdout, stderr := runHeadWithStdin("a\nb\nc\nd\n", []string{"-n", "2"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines from stdin, got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_StdinDash(t *testing.T) {
	exitCode, stdout, stderr := runHeadWithStdin("x\ny\nz\n", []string{"-n", "2", "-"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines from 'head -n 2 -', got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_Json(t *testing.T) {
	cmd, ok := dispatch.Lookup("head")
	if !ok {
		t.Fatal("head not registered")
	}
	var jsonBuf bytes.Buffer
	exitCode := cmd.Run([]string{"--json", testdataDir + "five.txt"}, &jsonBuf)
	if exitCode != 0 {
		t.Fatalf("exit %d", exitCode)
	}
	jsonOutput := jsonBuf.String()
	if jsonOutput == "" {
		t.Fatal("expected JSON output")
	}
	// JSON is wrapped in the common envelope: {"data": {...}}
	var envelope struct {
		Data HeadResult `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonOutput), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v (output: %q)", err, jsonOutput)
	}
	result := envelope.Data
	if result.LineCount != 5 {
		t.Errorf("expected 5 lines in JSON, got %d", result.LineCount)
	}
	if len(result.Lines) != 5 {
		t.Errorf("expected 5 lines array, got %d: %v", len(result.Lines), result.Lines)
	}
}

func TestRunCLI_FileNotFound(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"nonexistent_file_xyz.txt"})
	if exitCode != 1 {
		t.Errorf("expected exit 1 for missing file, got %d", exitCode)
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
	if stderr == "" {
		t.Error("expected stderr for missing file")
	}
}

func TestRunCLI_InvalidLinesCount(t *testing.T) {
	exitCode, _, stderr := runHead([]string{"-n", "abc", testdataDir + "ten.txt"})
	if exitCode != 2 {
		t.Errorf("expected exit 2 for invalid -n, got %d", exitCode)
	}
	if !strings.Contains(stderr, "illegal line count") {
		t.Errorf("expected 'illegal line count' in stderr, got %q", stderr)
	}
}

func TestRunCLI_InvalidBytesCount(t *testing.T) {
	exitCode, _, stderr := runHead([]string{"-c", "abc", testdataDir + "ten.txt"})
	if exitCode != 2 {
		t.Errorf("expected exit 2 for invalid -c, got %d", exitCode)
	}
	if !strings.Contains(stderr, "illegal byte count") {
		t.Errorf("expected 'illegal byte count' in stderr, got %q", stderr)
	}
}

func TestRunCLI_ShortFile(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"-n", "20", testdataDir + "five.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 5 {
		t.Errorf("expected all 5 lines from short file, got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_NegativeAll(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"-n", "-100", testdataDir + "five.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty output (skip 100 from 5 lines), got %q", stdout)
	}
}

func TestRunCLI_NoArgs(t *testing.T) {
	exitCode, stdout, stderr := runHeadWithStdin("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n", []string{})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines (default), got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_BytesLongFlag(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"--bytes=8", testdataDir + "bytes.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	if stdout != "abcdefgh" {
		t.Errorf("expected 'abcdefgh', got %q", stdout)
	}
}

func TestRunCLI_LinesLongFlag(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"--lines=4", testdataDir + "ten.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 lines from --lines=4, got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_MultiFileWithDash(t *testing.T) {
	exitCode, stdout, stderr := runHeadWithStdin("stdin1\nstdin2\n", []string{"-", testdataDir + "five.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "==> standard input <==") {
		t.Errorf("missing 'standard input' header: %q", stdout)
	}
	if !strings.Contains(stdout, "==> "+testdataDir+"five.txt <==") {
		t.Errorf("missing five.txt header: %q", stdout)
	}
}

func TestRunCLI_FileNotFoundContinues(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"nonexistent_1.txt", testdataDir + "five.txt"})
	if exitCode != 1 {
		t.Errorf("expected exit 1 for missing file, got %d", exitCode)
	}
	// Should still have processed five.txt
	if !strings.Contains(stdout, "a") {
		t.Errorf("expected five.txt output, got %q", stdout)
	}
	if !strings.Contains(stderr, "nonexistent_1.txt") {
		t.Errorf("expected error about missing file, got %q", stderr)
	}
}

func TestRunCLI_UnknownFlag(t *testing.T) {
	exitCode, _, stderr := runHead([]string{"--bogus", testdataDir + "ten.txt"})
	if exitCode != 2 {
		t.Errorf("expected exit 2 for unknown flag, got %d", exitCode)
	}
	if stderr == "" {
		t.Error("expected stderr for unknown flag")
	}
}

func TestRunCLI_LargeLineCount(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"-n", "999999", testdataDir + "five.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines (short file), got %d", len(lines))
	}
}

func TestRunCLI_LongFlagLinesSpace(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"--lines", "3", testdataDir + "ten.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines from --lines 3, got %d: %q", len(lines), stdout)
	}
}

func TestRunCLI_LongFlagBytesSpace(t *testing.T) {
	exitCode, stdout, stderr := runHead([]string{"--bytes", "6", testdataDir + "bytes.txt"})
	if exitCode != 0 {
		t.Fatalf("exit %d: %s", exitCode, stderr)
	}
	if stdout != "abcdef" {
		t.Errorf("expected 'abcdef', got %q", stdout)
	}
}
