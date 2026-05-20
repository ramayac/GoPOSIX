package tail

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunBasic(t *testing.T) {
	in := strings.NewReader("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n")
	var out bytes.Buffer
	lines, err := Run(in, &out, 10, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}
}

func TestRunShort(t *testing.T) {
	in := strings.NewReader("1\n2\n")
	var out bytes.Buffer
	lines, _ := Run(in, &out, 10, 0, false)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestRunFromStart(t *testing.T) {
	in := strings.NewReader("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n")
	var out bytes.Buffer
	lines, err := Run(in, &out, 5, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 6 {
		t.Errorf("expected at least 6 lines from +5, got %d", len(lines))
	}
	if lines[0] != "5" {
		t.Errorf("first line should be '5', got %q", lines[0])
	}
}

func TestRunBytesMode(t *testing.T) {
	in := strings.NewReader("abcdefghij")
	var out bytes.Buffer
	lines, err := Run(in, &out, 0, 5, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Errorf("expected 1 line from byte mode, got %d", len(lines))
	}
	if out.String() != "fghij" {
		t.Errorf("expected last 5 bytes 'fghij', got %q", out.String())
	}
}

func TestRunBytesFromStart(t *testing.T) {
	in := strings.NewReader("abcdefghij")
	var out bytes.Buffer
	_, err := Run(in, &out, 0, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "cdefghij" {
		t.Errorf("expected 'cdefghij' (skip first 2 bytes +c3), got %q", out.String())
	}
}

func TestRunBytesExact(t *testing.T) {
	in := strings.NewReader("abc")
	var out bytes.Buffer
	_, err := Run(in, &out, 0, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "abc" {
		t.Errorf("expected 'abc', got %q", out.String())
	}
}

func TestRunEmpty(t *testing.T) {
	in := strings.NewReader("")
	var out bytes.Buffer
	lines, err := Run(in, &out, 10, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestRunFromStartShort(t *testing.T) {
	in := strings.NewReader("1\n2\n")
	var out bytes.Buffer
	lines, err := Run(in, &out, 10, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	// +10 on 2 lines skips all, producing nothing
	if len(lines) != 0 {
		t.Errorf("expected 0 lines (skip 9 of 2), got %d", len(lines))
	}
}

// --- CLI tests via run() ---

func tailTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "tailtest")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestCLI_BasicFile(t *testing.T) {
	f := tailTempFile(t, "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n")
	var out bytes.Buffer
	code := run([]string{f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d: %q", len(lines), out.String())
	}
}

func TestCLI_NumLines(t *testing.T) {
	f := tailTempFile(t, "1\n2\n3\n4\n5\n")
	var out bytes.Buffer
	code := run([]string{"-n", "2", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %q", len(lines), out.String())
	}
}

func TestCLI_TraditionalDashNum(t *testing.T) {
	f := tailTempFile(t, "1\n2\n3\n4\n5\n")
	var out bytes.Buffer
	code := run([]string{"-2", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines from -2, got %d: %q", len(lines), out.String())
	}
}

func TestCLI_PlusNum(t *testing.T) {
	f := tailTempFile(t, "1\n2\n3\n4\n5\n")
	var out bytes.Buffer
	code := run([]string{"-n", "+3", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines from +3, got %d: %q", len(lines), out.String())
	}
}

func TestCLI_BytesCount(t *testing.T) {
	f := tailTempFile(t, "abcdefghij")
	var out bytes.Buffer
	code := run([]string{"-c", "4", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "ghij" {
		t.Errorf("expected 'ghij', got %q", out.String())
	}
}

func TestCLI_PlusBytesCount(t *testing.T) {
	f := tailTempFile(t, "abcdefghij")
	var out bytes.Buffer
	code := run([]string{"-c", "+5", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "efghij" {
		t.Errorf("expected 'efghij', got %q", out.String())
	}
}

func TestCLI_MultiFile(t *testing.T) {
	f1 := tailTempFile(t, "a\nb\nc\n")
	f2 := tailTempFile(t, "x\ny\nz\n")
	var out bytes.Buffer
	code := run([]string{f1, f2}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Multi-file output should have ==> headers
	if !strings.Contains(out.String(), "==>") {
		t.Errorf("expected multi-file headers, got: %s", out.String())
	}
}

func TestCLI_JSON(t *testing.T) {
	f := tailTempFile(t, "hello\nworld\n")
	var out bytes.Buffer
	code := run([]string{"--json", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"lines\"") {
		t.Errorf("expected JSON output, got: %s", out.String())
	}
}

func TestCLI_LongFlags(t *testing.T) {
	f := tailTempFile(t, "1\n2\n3\n4\n5\n")
	var out bytes.Buffer
	code := run([]string{"--lines", "2", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestCLI_FileNotFound(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"/nonexistent/tail/file"}, nil, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for missing file, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out)
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestCLI_IllegalLineCount(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-n", "abc", "/dev/null"}, nil, &out)
	if code != 2 {
		t.Errorf("expected exit 2 for illegal line count, got %d", code)
	}
}

func TestCLI_IllegalByteCount(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-c", "xyz", "/dev/null"}, nil, &out)
	if code != 2 {
		t.Errorf("expected exit 2 for illegal byte count, got %d", code)
	}
}

func TestCLI_EmptyInput(t *testing.T) {
	f := tailTempFile(t, "")
	var out bytes.Buffer
	code := run([]string{f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if out.String() != "" {
		t.Errorf("expected empty output, got %q", out.String())
	}
}

func TestCLI_ShortFile(t *testing.T) {
	f := tailTempFile(t, "one\ntwo\n")
	var out bytes.Buffer
	code := run([]string{"-n", "10", f}, nil, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines from short file, got %d", len(lines))
	}
}
