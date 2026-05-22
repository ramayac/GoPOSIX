package tsort

import (
	"bytes"
	"strings"
	"testing"
)

func TestTsortBasic(t *testing.T) {
	// 1. Simple 1-edge dependency
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("a b\n")
	gotExit := run([]string{}, stdin, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0, got %d. stderr: %s", gotExit, stderr.String())
	}
	out := stdout.String()
	if out != "a\nb\n" {
		t.Errorf("expected 'a\\nb\\n', got %q", out)
	}

	// 2. Self-loop / singleton
	stdout.Reset()
	stderr.Reset()
	stdin = strings.NewReader("a a\n")
	gotExit = run([]string{}, stdin, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0, got %d", gotExit)
	}
	if stdout.String() != "a\n" {
		t.Errorf("expected 'a\\n', got %q", stdout.String())
	}
}

func TestTsortEdgeCases(t *testing.T) {
	// 1. Odd number of words
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("a b c")
	gotExit := run([]string{}, stdin, &stdout, &stderr, "")
	if gotExit == 0 {
		t.Error("expected exit non-zero for odd number of words")
	}

	// 2. Cycle detection
	stdout.Reset()
	stderr.Reset()
	stdin = strings.NewReader("a b b a")
	gotExit = run([]string{}, stdin, &stdout, &stderr, "")
	if gotExit == 0 {
		t.Error("expected exit non-zero for cycle")
	}

	// 3. JSON mode
	stdout.Reset()
	stderr.Reset()
	stdin = strings.NewReader("a b b c")
	gotExit = run([]string{"--json"}, stdin, &stdout, &stderr, "")
	if gotExit != 0 {
		t.Errorf("expected exit 0 for json mode, got %d", gotExit)
	}
	out := stdout.String()
	if !strings.Contains(out, `"nodes":["a","b","c"]`) {
		t.Errorf("expected valid JSON nodes list, got: %s", out)
	}

	// 4. Too many arguments
	stdout.Reset()
	stderr.Reset()
	gotExit = run([]string{"file1", "file2"}, nil, &stdout, &stderr, "")
	if gotExit == 0 {
		t.Error("expected exit non-zero for multiple file arguments")
	}
}
