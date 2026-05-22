package cmp

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCompare_Identical(t *testing.T) {
	r1 := strings.NewReader("hello\nworld\n")
	r2 := strings.NewReader("hello\nworld\n")
	diffs, equal := Compare(r1, r2, 0, false)
	if !equal {
		t.Error("identical files should be equal")
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs, got %d", len(diffs))
	}
}

func TestCompare_FirstByteDiff(t *testing.T) {
	r1 := strings.NewReader("hello\n")
	r2 := strings.NewReader("jello\n")
	diffs, equal := Compare(r1, r2, 0, false)
	if equal {
		t.Error("differing files should not be equal")
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Byte != 1 {
		t.Errorf("byte: got %d, want 1", diffs[0].Byte)
	}
	if diffs[0].Line != 1 {
		t.Errorf("line: got %d, want 1", diffs[0].Line)
	}
}

func TestCompare_MidLineDiff(t *testing.T) {
	r1 := strings.NewReader("hello\nworld\n")
	r2 := strings.NewReader("hello\nworxd\n")
	diffs, equal := Compare(r1, r2, 0, false)
	if equal {
		t.Error("should not be equal")
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	// 'l' vs 'x' at position 10 on line 2
	if diffs[0].Byte != 10 {
		t.Errorf("byte: got %d, want 10", diffs[0].Byte)
	}
	if diffs[0].Line != 2 {
		t.Errorf("line: got %d, want 2", diffs[0].Line)
	}
}

func TestCompare_VerboseAllDiffs(t *testing.T) {
	r1 := strings.NewReader("abc")
	r2 := strings.NewReader("axc")
	diffs, equal := Compare(r1, r2, 0, true)
	if equal {
		t.Error("should not be equal")
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff in 'abc' vs 'axc', got %d", len(diffs))
	}
	if diffs[0].Byte != 2 {
		t.Errorf("byte: got %d, want 2", diffs[0].Byte)
	}
}

func TestCompare_VerboseMultipleDiffs(t *testing.T) {
	r1 := strings.NewReader("hello\n")
	r2 := strings.NewReader("hxllo\n")
	diffs, equal := Compare(r1, r2, 0, true)
	if equal {
		t.Error("should not be equal")
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
}

func TestCompare_File1Shorter(t *testing.T) {
	r1 := strings.NewReader("ab")
	r2 := strings.NewReader("abc")
	diffs, equal := Compare(r1, r2, 0, false)
	if equal {
		t.Error("different lengths should not be equal")
	}
	if len(diffs) == 0 {
		t.Fatal("expected diffs for shorter file")
	}
}

func TestCompare_File2Shorter(t *testing.T) {
	r1 := strings.NewReader("abcd")
	r2 := strings.NewReader("ab")
	_, equal := Compare(r1, r2, 0, false)
	if equal {
		t.Error("different lengths should not be equal")
	}
}

func TestCompare_LimitBytes(t *testing.T) {
	r1 := strings.NewReader("different")
	r2 := strings.NewReader("different")
	diffs, equal := Compare(r1, r2, 3, false)
	if len(diffs) != 0 || !equal {
		t.Error("first 3 bytes are identical")
	}

	r1 = strings.NewReader("difxerent")
	r2 = strings.NewReader("different")
	diffs, equal = Compare(r1, r2, 5, false)
	if !equal {
		// x at byte 4 should be caught within first 5
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff within limit, got %d", len(diffs))
		}
		if diffs[0].Byte != 4 {
			t.Errorf("byte: got %d, want 4", diffs[0].Byte)
		}
	} else {
		t.Error("should detect diff within limit")
	}
}

func TestCompare_EmptyFiles(t *testing.T) {
	r1 := strings.NewReader("")
	r2 := strings.NewReader("")
	diffs, equal := Compare(r1, r2, 0, false)
	if !equal {
		t.Error("empty files should be equal")
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for empty files, got %d", len(diffs))
	}
}

func TestCompare_EmptyVsNonEmpty(t *testing.T) {
	r1 := strings.NewReader("")
	r2 := strings.NewReader("a")
	diffs, equal := Compare(r1, r2, 0, false)
	if equal {
		t.Error("empty vs non-empty should not be equal")
	}
	if len(diffs) == 0 {
		t.Error("expected diffs for empty vs non-empty")
	}
}

// --- CLI layer tests ---

func TestCmpRun_SilentMode(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	rc := cmpRun([]string{"-s", "-", "-"}, &outBuf, &errBuf, strings.NewReader("hello\n"), "")
	if rc != 0 {
		t.Errorf("silent identical: got rc=%d, want 0 (stderr: %s)", rc, errBuf.String())
	}
	if outBuf.Len() != 0 {
		t.Errorf("silent mode should produce no output, got %q", outBuf.String())
	}
}

func TestCmpRun_DifferSilent(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := tmpDir + "/a.txt"
	f2 := tmpDir + "/b.txt"
	os.WriteFile(f1, []byte("hello"), 0644)
	os.WriteFile(f2, []byte("hallo"), 0644)
	var outBuf, errBuf bytes.Buffer
	rc := cmpRun([]string{"-s", f1, f2}, &outBuf, &errBuf, nil, "")
	if rc != 1 {
		t.Errorf("silent differ: got rc=%d, want 1", rc)
	}
	if outBuf.Len() != 0 {
		t.Errorf("silent mode should produce no output, got %q", outBuf.String())
	}
}

func TestCmpRun_JsonFlag(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := tmpDir + "/a.txt"
	f2 := tmpDir + "/b.txt"
	os.WriteFile(f1, []byte("hello"), 0644)
	os.WriteFile(f2, []byte("hello"), 0644)
	var outBuf, errBuf bytes.Buffer
	rc := cmpRun([]string{"--json", f1, f2}, &outBuf, &errBuf, nil, "")
	if rc != 0 {
		t.Fatalf("json identical: got rc=%d", rc)
	}
	if !bytes.Contains(outBuf.Bytes(), []byte("\"equal\"")) {
		t.Error("JSON output missing equal field")
	}
}

func TestCmpRun_JsonDiffer(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := tmpDir + "/a.txt"
	f2 := tmpDir + "/b.txt"
	os.WriteFile(f1, []byte("hello"), 0644)
	os.WriteFile(f2, []byte("hallo"), 0644)
	var outBuf, errBuf bytes.Buffer
	rc := cmpRun([]string{"--json", f1, f2}, &outBuf, &errBuf, nil, "")
	if rc != 1 {
		t.Fatalf("json differ: got rc=%d, want 1", rc)
	}
	if !bytes.Contains(outBuf.Bytes(), []byte("\"equal\"")) {
		t.Error("JSON output missing equal field")
	}
}

func TestCmpRun_MissingFile(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	rc := cmpRun([]string{"/nonexistent_12345", "/nonexistent_67890"}, &outBuf, &errBuf, nil, "")
	if rc != 2 {
		t.Errorf("missing file: got rc=%d, want 2", rc)
	}
}

func TestCmpRun_BadFlag(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	rc := cmpRun([]string{"--bad-flag"}, &outBuf, &errBuf, nil, "")
	if rc != 2 {
		t.Errorf("bad flag: got rc=%d, want 2", rc)
	}
}

func TestCmpRun_MissingOperand(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	rc := cmpRun([]string{}, &outBuf, &errBuf, nil, "")
	if rc != 2 {
		t.Errorf("missing operand: got rc=%d, want 2", rc)
	}
}

func TestCmp_CLIRun(t *testing.T) {
	// Test the CLI glue run() function.
	var outBuf, errBuf bytes.Buffer
	rc := run([]string{}, nil, &outBuf, &errBuf, "")
	if rc != 2 {
		t.Errorf("expected exit 2 for missing operand, got %d", rc)
	}
}
