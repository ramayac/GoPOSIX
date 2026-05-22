package seq

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// errorWriter returns an error after a set number of bytes/writes
type errorWriter struct {
	writesLeft int
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	if w.writesLeft <= 0 {
		return 0, errors.New("broken pipe mock error")
	}
	w.writesLeft--
	return len(p), nil
}

func TestSeqParsing(t *testing.T) {
	// Test parseNum helper
	val, prec, width, err := parseNum("003.50")
	if err != nil {
		t.Fatal(err)
	}
	if val != 3.5 || prec != 2 || width != 3 {
		t.Errorf("expected 3.5, 2, 3; got %f, %d, %d", val, prec, width)
	}

	val, prec, width, err = parseNum("-04")
	if err != nil {
		t.Fatal(err)
	}
	if val != -4.0 || prec != 0 || width != 2 {
		t.Errorf("expected -4.0, 0, 2; got %f, %d, %d", val, prec, width)
	}

	// Test invalid parsing
	_, _, _, err = parseNum("abc")
	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}

func TestSeqErrorsAndBounds(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Flag error
	stdout.Reset()
	stderr.Reset()
	code := run([]string{"--invalid-flag"}, nil, &stdout, &stderr, "")
	if code != 2 {
		t.Errorf("expected exit 2 for flag error, got %d", code)
	}

	// Invalid number of arguments (0)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}

	// Invalid number of arguments (>3)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"1", "2", "3", "4"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}

	// Non-numeric arguments
	code = run([]string{"abc"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	code = run([]string{"1", "abc"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	code = run([]string{"1", "2", "abc"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestSeqPreProcessDetails(t *testing.T) {
	// Test explicit -- with count down
	var stdout, stderr bytes.Buffer
	code := run([]string{"--", "1", "-1", "-2"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d, stderr: %q", code, stderr.String())
	}
	expected := "1\n0\n-1\n-2\n"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}

	// Test --separator=value format
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--separator=:", "1", "3"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	expected = "1:2:3\n"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}
}

func TestSeqStepZero(t *testing.T) {
	// Infinite loop with step 0, tested via errorWriter
	var stderr bytes.Buffer
	w := &errorWriter{writesLeft: 3}
	code := run([]string{"1", "0", "3"}, nil, w, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}

	// Infinite loop with step 0 in JSON mode should fail
	var stdout bytes.Buffer
	stderr.Reset()
	code = run([]string{"--json", "1", "0", "3"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1 for infinite JSON sequence, got %d", code)
	}
}

func TestSeqCLI(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Test 1 argument
	stdout.Reset()
	stderr.Reset()
	code := run([]string{"3"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d, stderr: %q", code, stderr.String())
	}
	expected := "1\n2\n3\n"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}

	// Test 2 arguments
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"5", "7"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	expected = "5\n6\n7\n"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}

	// Test 3 arguments counting down
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"8", "-2", "4"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	expected = "8\n6\n4\n"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}

	// Test wrong way (should output nothing, exit 0)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"8", "2", "4"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected empty output, got %q", stdout.String())
	}

	// Test decimals precision
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"3", ".30", "4"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	expected = "3.00\n3.30\n3.60\n3.90\n"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}

	// Test padding with -w
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-w", "03", ".3", "0004"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	expected = "0003.0\n0003.3\n0003.6\n0003.9\n"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}

	// Test custom separator with -s
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-s", ", ", "1", "3"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	expected = "1, 2, 3\n"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}

	// Test JSON output
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--json", "1", "3"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"sequence"`) {
		t.Errorf("expected JSON sequence key, got %q", stdout.String())
	}
}
