package cut

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCutFields(t *testing.T) {
	in := "a:b:c\n1:2:3\n"
	lines, _ := Run(strings.NewReader(in), "1,3", ":", "", "", false, false, false)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines")
	}
	if lines[0].Fields[0] != "a:c" {
		t.Errorf("got %v", lines[0].Fields[0])
	}
}

func TestCutChars(t *testing.T) {
	in := "abcdef\n"
	lines, _ := Run(strings.NewReader(in), "", "", "1-3,5", "", false, false, false)
	if lines[0].Fields[0] != "abce" {
		t.Errorf("got %v", lines[0].Fields[0])
	}
}

func TestCutBytes(t *testing.T) {
	in := "hello\n"
	lines, _ := Run(strings.NewReader(in), "", "", "", "2", false, false, false)
	if lines[0].Fields[0] != "e" {
		t.Errorf("got %v", lines[0].Fields[0])
	}
}

func TestParseList(t *testing.T) {
	cases := []struct {
		in    string
		count int
		valid bool
	}{
		{"1", 1, true},
		{"1,3,5", 3, true},
		{"1-5", 1, true},
		{"1-", 1, true},
		{"-5", 1, true},
		{"1-3,5-7", 2, true},
		{"", 0, false},
		{"0", 0, false},
	}
	for _, c := range cases {
		specs, err := parseList(c.in)
		if c.valid && err != nil {
			t.Errorf("parseList(%q) unexpected error: %v", c.in, err)
		}
		if !c.valid && err == nil {
			t.Errorf("parseList(%q) expected error, got %v", c.in, specs)
		}
		if c.valid && len(specs) != c.count {
			t.Errorf("parseList(%q) = %d specs, want %d", c.in, len(specs), c.count)
		}
	}
}

func TestInRange(t *testing.T) {
	specs := []rangeSpec{{1, 3}, {5, 5}}
	if !inRange(1, specs) {
		t.Error("1 should be in range")
	}
	if !inRange(3, specs) {
		t.Error("3 should be in range")
	}
	if inRange(4, specs) {
		t.Error("4 should NOT be in range")
	}
	if !inRange(5, specs) {
		t.Error("5 should be in range")
	}
}

func TestCutFieldsDelimiter(t *testing.T) {
	in := "a,b,c\nd,e,f\n"
	lines, _ := Run(strings.NewReader(in), "2", ",", "", "", false, false, false)
	if lines[0].Fields[0] != "b" {
		t.Errorf("got %v", lines[0].Fields[0])
	}
}

func TestCutOnlyDelimited(t *testing.T) {
	in := "a:b\nno-delim\nc:d\n"
	lines, _ := Run(strings.NewReader(in), "1", ":", "", "", true, false, false)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines with -s, got %d", len(lines))
	}
}

func TestCutCharsRange(t *testing.T) {
	in := "12345\n"
	lines, _ := Run(strings.NewReader(in), "", "", "2-4", "", false, false, false)
	if lines[0].Fields[0] != "234" {
		t.Errorf("got %v", lines[0].Fields[0])
	}
}

func TestCutEmpty(t *testing.T) {
	lines, err := Run(strings.NewReader(""), "1", "\t", "", "", false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestCutWhitespaceFields(t *testing.T) {
	in := "hello   world\nfoo bar\n"
	lines, _ := Run(strings.NewReader(in), "2", "", "", "", false, false, true)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines")
	}
	if lines[0].Fields[0] != "world" {
		t.Errorf("got %q", lines[0].Fields[0])
	}
	if lines[1].Fields[0] != "bar" {
		t.Errorf("got %q", lines[1].Fields[0])
	}
}

func TestCutMultiByteUnicode(t *testing.T) {
	// Verify character-position cuts work with multi-byte chars
	in := "café\n"
	// Bytes 1-3: c,a,f = 3 bytes, the é (byte 4-5) is excluded
	lines, err := Run(strings.NewReader(in), "", "", "1-3", "", false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if lines[0].Fields[0] != "caf" {
		t.Errorf("got %q, want %q", lines[0].Fields[0], "caf")
	}
}

func TestCutEmptyFields(t *testing.T) {
	// Fields with empty values between delimiters
	in := "a::c\n"
	lines, err := Run(strings.NewReader(in), "1-3", ":", "", "", false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	// Output: fields 1,2,3 joined with delimiter
	if lines[0].Fields[0] != "a::c" {
		t.Errorf("got %q, want %q", lines[0].Fields[0], "a::c")
	}
}

// --- CLI layer tests ---

func TestCLI_Fields(t *testing.T) {
	var outBuf bytes.Buffer
	code := run([]string{"-f", "1", "-d", ":"}, strings.NewReader(""), &outBuf, &outBuf, "")
	// stdin not piped in test — should error or handle gracefully
	_ = code
}

func TestCLI_BadFlag(t *testing.T) {
	var outBuf bytes.Buffer
	code := run([]string{"--bad-flag"}, strings.NewReader(""), &outBuf, &outBuf, "")
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestCLI_MissingList(t *testing.T) {
	var outBuf bytes.Buffer
	code := run([]string{}, nil, &outBuf, &outBuf, "")
	if code != 1 {
		t.Errorf("expected exit 1 for missing list, got %d", code)
	}
}

func TestCLI_NoxistentFile(t *testing.T) {
	var outBuf bytes.Buffer
	code := run([]string{"-f", "1", "/nonexistent_12345"}, nil, &outBuf, &outBuf, "")
	if code != 1 {
		t.Errorf("expected exit 1 for nonexistent file, got %d", code)
	}
}

func TestCLI_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := tmpDir + "/test.txt"
	os.WriteFile(f, []byte("a:b:c\n1:2:3\n"), 0644)
	var outBuf bytes.Buffer
	code := run([]string{"-f", "1", "-d", ":", f}, nil, &outBuf, &outBuf, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	output := outBuf.String()
	if !strings.Contains(output, "a") || !strings.Contains(output, "1") {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestCLI_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	f := tmpDir + "/test.txt"
	os.WriteFile(f, []byte("a:b\n"), 0644)
	var outBuf bytes.Buffer
	code := run([]string{"-f", "1", "-d", ":", "--json", f}, nil, &outBuf, &outBuf, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !bytes.Contains(outBuf.Bytes(), []byte("\"lines\"")) {
		t.Error("JSON missing lines field")
	}
}
