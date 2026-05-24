package hexdump

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCanonicalNULs(t *testing.T) {
	in := bytes.NewReader([]byte{0, 0, 0, 0})
	var out bytes.Buffer

	// Build flags manually or call helper
	// For hexdump -C, we expect:
	// 00000000  00 00 00 00                                       |....|
	// 00000004
	fs1, err := parseFormatString(`"%08_ax  " 8/1 "%02x " " " 8/1 "%02x " " |"`)
	if err != nil {
		t.Fatal(err)
	}
	fs2, err := parseFormatString(`16/1 "%_p" "|\n"`)
	if err != nil {
		t.Fatal(err)
	}
	formatStrings := []FormatString{{Units: fs1}, {Units: fs2}}

	err = Run(in, &out, formatStrings, 0, -1, false)
	if err != nil {
		t.Fatal(err)
	}

	got := out.String()
	expected := "00000000  00 00 00 00                                       |....|\n00000004\n"
	if got != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, got)
	}
}

func TestRunFormatStringFolding(t *testing.T) {
	t.Run("11 NULs", func(t *testing.T) {
		in := bytes.NewReader(make([]byte, 11))
		var out bytes.Buffer

		units, err := parseFormatString(`1/1 "%02x|" 1/1 "%02x!\n"`)
		if err != nil {
			t.Fatal(err)
		}
		formatStrings := []FormatString{{Units: units}}

		err = Run(in, &out, formatStrings, 0, -1, false)
		if err != nil {
			t.Fatal(err)
		}

		got := out.String()
		expected := "00|00!\n*\n00|  !\n"
		if got != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, got)
		}
	})

	t.Run("12 NULs", func(t *testing.T) {
		in := bytes.NewReader(make([]byte, 12))
		var out bytes.Buffer

		units, err := parseFormatString(`1/1 "%02x|" 1/1 "%02x!\n"`)
		if err != nil {
			t.Fatal(err)
		}
		formatStrings := []FormatString{{Units: units}}

		err = Run(in, &out, formatStrings, 0, -1, false)
		if err != nil {
			t.Fatal(err)
		}

		got := out.String()
		expected := "00|00!\n*\n"
		if got != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, got)
		}
	})
}

func TestDefaultFormat(t *testing.T) {
	in := bytes.NewReader([]byte("abcdefgh"))
	var out bytes.Buffer

	// Default format is: "%07_ax" 8/2 " %04x" "\n"
	units, err := parseFormatString(`"%07_ax" 8/2 " %04x" "\n"`)
	if err != nil {
		t.Fatal(err)
	}
	formatStrings := []FormatString{{Units: units}}

	err = Run(in, &out, formatStrings, 0, -1, false)
	if err != nil {
		t.Fatal(err)
	}

	got := out.String()
	// abcdefgh => hex: a=61, b=62, c=63, d=64, e=65, f=66, g=67, h=68
	// Little-endian swaps: 6261 6463 6665 6867
	expected := "0000000 6261 6463 6665 6867\n0000008\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestHexdumpRunCLI(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hexdump_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(tempFile, []byte("abcdefgh"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("canonical dump file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := hexdumpRun([]string{"-C", tempFile}, &stdout, &stderr, nil, "")
		if code != 0 {
			t.Errorf("expected exit 0, got %d. stderr: %s", code, stderr.String())
		}
		got := stdout.String()
		if !strings.Contains(got, "61 62 63 64 65 66 67 68") {
			t.Errorf("expected canonical representation, got %q", got)
		}
	})

	t.Run("custom format string CLI", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := hexdumpRun([]string{"-e", `1/1 "%02x|"`, tempFile}, &stdout, &stderr, nil, "")
		if code != 0 {
			t.Errorf("expected exit 0, got %d. stderr: %s", code, stderr.String())
		}
		got := stdout.String()
		expected := "61|62|63|64|65|66|67|68|"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("JSON output CLI", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := hexdumpRun([]string{"--json", "-C", tempFile}, &stdout, &stderr, nil, "")
		if code != 0 {
			t.Errorf("expected exit 0, got %d. stderr: %s", code, stderr.String())
		}
		got := stdout.String()
		if !strings.Contains(got, `"lines"`) || !strings.Contains(got, "61 62 63 64") {
			t.Errorf("expected valid JSON output, got %q", got)
		}
	})

	t.Run("skip and length", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := hexdumpRun([]string{"-s", "2", "-n", "3", "-e", `1/1 "%c"`, tempFile}, &stdout, &stderr, nil, "")
		if code != 0 {
			t.Errorf("expected exit 0, got %d. stderr: %s", code, stderr.String())
		}
		got := stdout.String()
		expected := "cde"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("invalid flag", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := hexdumpRun([]string{"-z"}, &stdout, &stderr, nil, "")
		if code != 2 {
			t.Errorf("expected exit 2, got %d", code)
		}
	})

	t.Run("invalid format string", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := hexdumpRun([]string{"-e", `invalid_format_no_quotes`, tempFile}, &stdout, &stderr, nil, "")
		if code != 2 {
			t.Errorf("expected exit 2, got %d", code)
		}
	})
}
