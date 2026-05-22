package xargs

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestXargsJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json", "true"}, strings.NewReader(""), &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

// BusyBox hardening: xargs -0 should split input on NUL bytes.
func TestXargsNullDelimited(t *testing.T) {
	tmp, err := os.CreateTemp("", "xargs-null-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("hello")
	tmp.Write([]byte{0})
	tmp.WriteString("world")
	tmp.Write([]byte{0})
	tmp.Close()

	oldStdin := os.Stdin
	f, _ := os.Open(tmp.Name())
	os.Stdin = f
	defer func() { os.Stdin = oldStdin; f.Close() }()

	var out bytes.Buffer
	code := run([]string{"-0", "echo"}, f, &out, &out, "")
	if code != 0 {
		t.Fatalf("xargs -0 exited with %d, want 0", code)
	}
	if out.String() != "hello world\n" {
		t.Errorf("got %q, want %q", out.String(), "hello world\n")
	}
}

func TestXargs_MaxArgs(t *testing.T) {
	// -n2: at most 2 args per invocation.
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"-n", "2", "echo"}, &out, &errOut, strings.NewReader("a b c d e"), "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	// Should produce 3 invocations: "a b", "c d", "e"
	output := out.String()
	if !strings.Contains(output, "a b") {
		t.Errorf("expected 'a b' in output, got %q", output)
	}
	if !strings.Contains(output, "c d") {
		t.Errorf("expected 'c d' in output, got %q", output)
	}
	if !strings.Contains(output, "e") {
		t.Errorf("expected 'e' in output, got %q", output)
	}
}

func TestXargs_DefaultEchoCommand(t *testing.T) {
	// No command specified: defaults to echo.
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{}, &out, &errOut, strings.NewReader("hello world"), "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if !strings.Contains(out.String(), "hello world") {
		t.Errorf("expected 'hello world' in output, got %q", out.String())
	}
}

func TestXargs_Trace(t *testing.T) {
	// -t: print command trace to stderr.
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"-t", "echo", "hello"}, &out, &errOut, strings.NewReader(""), "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if !strings.Contains(errOut.String(), "echo") {
		t.Errorf("expected 'echo' in trace output, got %q", errOut.String())
	}
}

func TestXargs_ReplaceStr(t *testing.T) {
	// -I{}: replace {} in command args with each input line.
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"-I", "{}", "echo", "arg:{}"}, &out, &errOut,
		strings.NewReader("hello\nworld\n"), "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	output := out.String()
	if !strings.Contains(output, "arg:hello") {
		t.Errorf("expected 'arg:hello' in output, got %q", output)
	}
	if !strings.Contains(output, "arg:world") {
		t.Errorf("expected 'arg:world' in output, got %q", output)
	}
}

func TestXargs_ReplaceStr_EmptyLines(t *testing.T) {
	// -I{}: empty/whitespace lines are skipped.
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"-I", "{}", "echo", "{}"}, &out, &errOut,
		strings.NewReader("a\n  \nb\n"), "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	output := out.String()
	if strings.Count(output, "\n") > 2 {
		t.Errorf("empty lines should be skipped, got %q", output)
	}
}

func TestXargs_EOFString(t *testing.T) {
	// -E STOP: stop reading at line matching STOP.
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"-E", "STOP", "echo"}, &out, &errOut,
		strings.NewReader("a b STOP c"), "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	output := out.String()
	if strings.Contains(output, "c") {
		t.Errorf("'c' should not appear (stopped at STOP), got %q", output)
	}
	if !strings.Contains(output, "a b") {
		t.Errorf("expected 'a b' before STOP, got %q", output)
	}
}

func TestXargs_BadFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"--nonexistent"}, &out, &errOut, strings.NewReader(""), "")
	if rc != 1 {
		t.Errorf("expected exit 1 for bad flag, got %d", rc)
	}
}

func TestXargs_NoInputRunsOnce(t *testing.T) {
	// POSIX: xargs runs the command at least once even with no input.
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"true"}, &out, &errOut, strings.NewReader(""), "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
}

func TestXargs_CommandFails(t *testing.T) {
	// When the command fails, exit code should be 123.
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"false"}, &out, &errOut, strings.NewReader(""), "")
	// false exits 1, xargs maps 1→123
	if rc != 123 {
		t.Errorf("expected 123, got %d", rc)
	}
}

func TestXargs_JSONMode(t *testing.T) {
	var out, errOut bytes.Buffer
	rc := xargsRun([]string{"--json", "echo", "test"}, &out, &errOut, strings.NewReader(""), "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if !strings.Contains(out.String(), `"command"`) {
		t.Errorf("expected JSON with 'command', got %q", out.String())
	}
}
