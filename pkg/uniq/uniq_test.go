package uniq

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestUniqBasic(t *testing.T) {
	in := "a\na\nb\nc\nc\n"
	res, _ := Run(strings.NewReader(in), false, false, false, false, 0, 0, 0)
	if len(res) != 3 || res[0].Line != "a" || res[1].Line != "b" || res[2].Line != "c" {
		t.Errorf("got %v", res)
	}
}

func TestUniqCount(t *testing.T) {
	in := "a\na\nb\n"
	res, _ := Run(strings.NewReader(in), true, false, false, false, 0, 0, 0)
	if len(res) != 2 || res[0].Count != 2 || res[1].Count != 1 {
		t.Errorf("got %v", res)
	}
}

func TestUniqDuplicates(t *testing.T) {
	in := "a\na\nb\nc\nc\n"
	res, _ := Run(strings.NewReader(in), false, true, false, false, 0, 0, 0)
	if len(res) != 2 || res[0].Line != "a" || res[1].Line != "c" {
		t.Errorf("got %v", res)
	}
}

func TestUniqUnique(t *testing.T) {
	in := "a\na\nb\nc\nc\n"
	res, _ := Run(strings.NewReader(in), false, false, true, false, 0, 0, 0)
	if len(res) != 1 || res[0].Line != "b" {
		t.Errorf("got %v", res)
	}
}

func TestUniqIgnoreCase(t *testing.T) {
	in := "a\nA\nb\n"
	res, _ := Run(strings.NewReader(in), false, false, false, true, 0, 0, 0)
	if len(res) != 2 || res[0].Line != "a" || res[0].Count != 2 {
		t.Errorf("got %v", res)
	}
}

func TestUniqSkipFields(t *testing.T) {
	// skipFields=1 on single-word lines consumes everything (no separator),
	// so all become "" → all duplicates of the first line.
	in := "hello\nworld\nfoo\n"
	res, _ := Run(strings.NewReader(in), false, false, false, false, 1, 0, 0)
	if len(res) != 1 {
		t.Errorf("expected 1 unique after skipping 1 field (single-word lines), got %d: %v", len(res), res)
	}
}

func TestUniqSkipFieldsMultiword(t *testing.T) {
	in := "hello there\nhello world\n"
	res, _ := Run(strings.NewReader(in), false, false, false, false, 1, 0, 0)
	if len(res) != 2 {
		t.Errorf("expected 2 unique, got %d", len(res))
	}
}

func TestUniqSkipChars(t *testing.T) {
	in := "001a\n002b\n003c\n"
	res, _ := Run(strings.NewReader(in), false, false, false, false, 0, 3, 0)
	if len(res) != 3 {
		t.Errorf("expected 3 unique, got %d", len(res))
	}
}

func TestUniqCheckChars(t *testing.T) {
	in := "abc\nabd\nabe\n"
	res, _ := Run(strings.NewReader(in), false, false, false, false, 0, 0, 2)
	if len(res) != 1 {
		t.Errorf("expected 1 unique (first 2 chars same), got %d", len(res))
	}
}

func TestUniqEmpty(t *testing.T) {
	res, err := Run(strings.NewReader(""), false, false, false, false, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 items for empty input, got %d", len(res))
	}
}

func TestUniqSingle(t *testing.T) {
	in := "hello\n"
	res, _ := Run(strings.NewReader(in), false, false, false, false, 0, 0, 0)
	if len(res) != 1 || res[0].Line != "hello" {
		t.Errorf("got %v", res)
	}
}

func TestUniqBothUniqueAndDuplicates(t *testing.T) {
	in := "a\na\nb\n"
	res, _ := Run(strings.NewReader(in), false, true, true, false, 0, 0, 0)
	if len(res) != 0 {
		t.Errorf("expected 0 items when both -d and -u, got %d", len(res))
	}
}

func TestExtractCompareKey(t *testing.T) {
	cases := []struct {
		line       string
		skipFields int
		skipChars  int
		checkChars int
		want       string
	}{
		// Skip 1 field: preserves the delimiter, so " world" (space + remainder)
		{"  hello world", 1, 0, 0, " world"},
		// Skip 2 fields on a 2-field line: last field consumed entirely → ""
		{"  hello world", 2, 0, 0, ""},
		{"abcdef", 0, 2, 0, "cdef"},
		{"abcdef", 0, 0, 3, "abc"},
		// Skip 1 field on a single word (no separator) → empty
		{"  abc", 1, 2, 0, ""},
	}
	for _, c := range cases {
		got := extractCompareKey(c.line, c.skipFields, c.skipChars, c.checkChars)
		if got != c.want {
			t.Errorf("extractCompareKey(%q, %d, %d, %d) = %q, want %q",
				c.line, c.skipFields, c.skipChars, c.checkChars, got, c.want)
		}
	}
}

// --- CLI tests ---

func uniqTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "uniqtest")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestCLI_Basic(t *testing.T) {
	f := uniqTempFile(t, "a\na\nb\nb\nc\n")
	var out bytes.Buffer
	code := run([]string{f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), out.String())
	}
}

func TestCLI_Count(t *testing.T) {
	f := uniqTempFile(t, "a\na\nb\n")
	var out bytes.Buffer
	code := run([]string{"-c", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "2 a") {
		t.Errorf("expected count prefix, got: %s", out.String())
	}
}

func TestCLI_Duplicates(t *testing.T) {
	f := uniqTempFile(t, "a\na\nb\nc\nc\n")
	var out bytes.Buffer
	code := run([]string{"-d", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "c" {
		t.Errorf("expected a\\nc, got %q", out.String())
	}
}

func TestCLI_Unique(t *testing.T) {
	f := uniqTempFile(t, "a\na\nb\nc\nc\n")
	var out bytes.Buffer
	code := run([]string{"-u", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimRight(out.String(), "\n") != "b" {
		t.Errorf("expected 'b', got %q", out.String())
	}
}

func TestCLI_IgnoreCase(t *testing.T) {
	f := uniqTempFile(t, "a\nA\nb\n")
	var out bytes.Buffer
	code := run([]string{"-i", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestCLI_JSON(t *testing.T) {
	f := uniqTempFile(t, "a\na\nb\n")
	var out bytes.Buffer
	code := run([]string{"--json", f}, nil, &out, &out, "")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.Len() == 0 {
		t.Error("expected JSON output")
	}
}

func TestCLI_FileNotFound(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"/nonexistent/uniq/file"}, nil, &out, &out, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out, &out, "")
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}
