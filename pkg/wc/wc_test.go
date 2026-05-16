package wc

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCountProper(t *testing.T) {
	in := strings.NewReader("hello world\nthis is a test\n")
	res, err := CountProper(in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines != 2 {
		t.Errorf("expected 2 lines, got %d", res.Lines)
	}
	if res.Words != 6 {
		t.Errorf("expected 6 words, got %d", res.Words)
	}
	if res.Bytes != 27 {
		t.Errorf("expected 27 bytes, got %d", res.Bytes)
	}
}

func TestCountProperUTF8(t *testing.T) {
	in := strings.NewReader("こんにちは\n")
	res, _ := CountProper(in)
	if res.Lines != 1 {
		t.Errorf("expected 1 line")
	}
	if res.Words != 1 {
		t.Errorf("expected 1 word")
	}
	if res.Chars != 6 {
		t.Errorf("expected 6 chars, got %d", res.Chars)
	}
	if res.Bytes != 16 {
		t.Errorf("expected 16 bytes, got %d", res.Bytes)
	}
}

func TestCountProperEmpty(t *testing.T) {
	in := strings.NewReader("")
	res, err := CountProper(in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines != 0 || res.Words != 0 || res.Bytes != 0 || res.Chars != 0 {
		t.Errorf("expected all zeros, got %+v", res)
	}
}

func TestCountProperSingleLine(t *testing.T) {
	in := strings.NewReader("hello")
	res, err := CountProper(in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines != 0 {
		t.Errorf("expected 0 lines (no newline), got %d", res.Lines)
	}
	if res.Words != 1 {
		t.Errorf("expected 1 word, got %d", res.Words)
	}
	if res.Bytes != 5 {
		t.Errorf("expected 5 bytes, got %d", res.Bytes)
	}
}

func TestCountProperWhitespace(t *testing.T) {
	in := strings.NewReader("   \t  \n")
	res, err := CountProper(in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines != 1 {
		t.Errorf("expected 1 line, got %d", res.Lines)
	}
	if res.Words != 0 {
		t.Errorf("expected 0 words, got %d", res.Words)
	}
}

func TestCount(t *testing.T) {
	in := strings.NewReader("hello world\n")
	res, err := Count(in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines != 1 {
		t.Errorf("expected 1 line, got %d", res.Lines)
	}
	if res.Words != 2 {
		t.Errorf("expected 2 words, got %d", res.Words)
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Wc_MaxLineLength(t *testing.T) {
	// BusyBox: echo "i'm a little teapot" | wc -L → 19
	// -L returns the length of the longest line.
	r := strings.NewReader("i'm a little teapot\n")
	res, err := CountProper(r)
	if err != nil {
		t.Fatal(err)
	}
	if res.MaxLineLength != 19 {
		t.Errorf("got %d, want 19", res.MaxLineLength)
	}
}

// --- CLI tests ---

func wcTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "wctest")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestCLI_BasicFile(t *testing.T) {
	f := wcTempFile(t, "hello world\nfoo bar\n")
	var out bytes.Buffer
	code := run([]string{f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "2") {
		t.Errorf("expected line count in output, got: %s", out.String())
	}
}

func TestCLI_LinesFlag(t *testing.T) {
	f := wcTempFile(t, "a\nb\nc\n")
	var out bytes.Buffer
	code := run([]string{"-l", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_WordsFlag(t *testing.T) {
	f := wcTempFile(t, "one two three\n")
	var out bytes.Buffer
	code := run([]string{"-w", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_BytesFlag(t *testing.T) {
	f := wcTempFile(t, "abc\n")
	var out bytes.Buffer
	code := run([]string{"-c", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_CharsFlag(t *testing.T) {
	f := wcTempFile(t, "abc\n")
	var out bytes.Buffer
	code := run([]string{"-m", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_MaxLineFlag(t *testing.T) {
	f := wcTempFile(t, "short\nlongest line here\nok\n")
	var out bytes.Buffer
	code := run([]string{"-L", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_JSON(t *testing.T) {
	f := wcTempFile(t, "hello\n")
	var out bytes.Buffer
	code := run([]string{"-j", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"lines\"") {
		t.Errorf("expected JSON, got: %s", out.String())
	}
}

func TestCLI_LongFlags(t *testing.T) {
	f := wcTempFile(t, "a\nb\n")
	var out bytes.Buffer
	code := run([]string{"--lines", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
}

func TestCLI_MultipleFiles(t *testing.T) {
	f1 := wcTempFile(t, "one\n")
	f2 := wcTempFile(t, "two\n")
	var out bytes.Buffer
	code := run([]string{f1, f2}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "total") {
		t.Errorf("expected total line, got: %s", out.String())
	}
}

func TestCLI_MissingFile(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"/nonexistent/wc/file"}, &out)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, &out)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}
