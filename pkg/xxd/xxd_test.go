package xxd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDumpStandard(t *testing.T) {
	input := []byte("Hello World! 1234567890abcdef")
	in := bytes.NewReader(input)
	var out bytes.Buffer

	err := Run(in, &out, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	// Should contain offset, hex groups, and ASCII
	if !strings.Contains(got, "00000000:") {
		t.Errorf("expected offset 00000000 in output, got %q", got)
	}
	if !strings.Contains(got, "4865 6c6c") { // "Hell"
		t.Errorf("expected hex of 'Hell', got %q", got)
	}
	if !strings.Contains(got, "Hello World! 123") {
		t.Errorf("expected ASCII preview 'Hello World! 123', got %q", got)
	}
	if !strings.Contains(got, "00000010:") {
		t.Errorf("expected offset 00000010, got %q", got)
	}
}

func TestRunDumpPlain(t *testing.T) {
	input := []byte("Hello")
	in := bytes.NewReader(input)
	var out bytes.Buffer

	err := Run(in, &out, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	// Should contain only plain hex and newlines
	expected := "48656c6c6f\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRunReverseStandard(t *testing.T) {
	dump := "00000000: 4865 6c6c 6f20 576f 726c 6421  Hello World!\n"
	in := strings.NewReader(dump)
	var out bytes.Buffer

	err := Run(in, &out, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	expected := "Hello World!"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRunReversePlain(t *testing.T) {
	dump := "48656c6c6f\n"
	in := strings.NewReader(dump)
	var out bytes.Buffer

	err := Run(in, &out, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	expected := "Hello"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestReversePlainEdgeCases(t *testing.T) {
	// Plain reverser ignores whitespace and single bad chars, but truncates on 2 consecutive bad chars.
	// "4865 6c 6c6f" -> "Hello" (ignores spaces)
	// "48!65" -> "H" followed by bad character '!' and then '65' ('e')
	// Let's test single bad character behavior.
	// In xxd -p -r, single bad chars are skipped:
	// "48!65" -> "H" and then '!' is skipped, '6' is d1, '5' is d2 -> 'e'. Result: "He"
	t.Run("single bad char", func(t *testing.T) {
		in := strings.NewReader("48!65")
		var out bytes.Buffer
		err := Run(in, &out, true, true)
		if err != nil {
			t.Fatal(err)
		}
		if out.String() != "He" {
			t.Errorf("expected 'He', got %q", out.String())
		}
	})

	t.Run("nibble with bad 2nd char", func(t *testing.T) {
		// "4!65" -> '4' is d1. next is '!', which is bad. Discards '4'. Next is '6', then '5' -> "e"
		in := strings.NewReader("4!65")
		var out bytes.Buffer
		err := Run(in, &out, true, true)
		if err != nil {
			t.Fatal(err)
		}
		if out.String() != "e" {
			t.Errorf("expected 'e', got %q", out.String())
		}
	})

	t.Run("two consecutive bad chars", func(t *testing.T) {
		// "48!!65" -> '4' is d1, '8' is d2 -> "H". Then '!' is bad (badSeqCount=1). Next '!' is bad (badSeqCount=2) -> truncate.
		in := strings.NewReader("48!!65")
		var out bytes.Buffer
		err := Run(in, &out, true, true)
		if err != nil {
			t.Fatal(err)
		}
		if out.String() != "H" {
			t.Errorf("expected 'H', got %q", out.String())
		}
	})
}

func TestReverseStandardEdgeCases(t *testing.T) {
	t.Run("two consecutive spaces truncate", func(t *testing.T) {
		// Standard xxd -r truncates after two consecutive spaces.
		// "00000000: 4865 6c6c  Hello" -> "Hell" because the double space marks start of visual ASCII column.
		in := strings.NewReader("00000000: 4865 6c6c  Hello")
		var out bytes.Buffer
		err := Run(in, &out, false, true)
		if err != nil {
			t.Fatal(err)
		}
		if out.String() != "Hell" {
			t.Errorf("expected 'Hell', got %q", out.String())
		}
	})

	t.Run("no colon skip", func(t *testing.T) {
		in := strings.NewReader("no colon here")
		var out bytes.Buffer
		err := Run(in, &out, false, true)
		if err != nil {
			t.Fatal(err)
		}
		if out.Len() != 0 {
			t.Errorf("expected empty output, got %q", out.String())
		}
	})

	t.Run("whitespace prefix", func(t *testing.T) {
		in := strings.NewReader("  \t  00000000: 4865")
		var out bytes.Buffer
		err := Run(in, &out, false, true)
		if err != nil {
			t.Fatal(err)
		}
		if out.String() != "He" {
			t.Errorf("expected 'He', got %q", out.String())
		}
	})
}

func TestXxdRunCLI(t *testing.T) {
	// Create a temporary file to dump
	tempDir, err := os.MkdirTemp("", "xxd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(tempFile, []byte("Hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("dump file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := xxdRun([]string{tempFile}, &stdout, &stderr, nil, "")
		if code != 0 {
			t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "4865 6c6c 6f") {
			t.Errorf("expected stdout to contain '4865 6c6c 6f', got %q", stdout.String())
		}
	})

	t.Run("dump file plain", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := xxdRun([]string{"-p", tempFile}, &stdout, &stderr, nil, "")
		if code != 0 {
			t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if stdout.String() != "48656c6c6f\n" {
			t.Errorf("expected '48656c6c6f\\n', got %q", stdout.String())
		}
	})

	t.Run("dump stdin JSON", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader("Hello")
		code := xxdRun([]string{"--json"}, &stdout, &stderr, stdin, "")
		if code != 0 {
			t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		got := stdout.String()
		if !strings.Contains(got, `"lines"`) || !strings.Contains(got, "4865 6c6c") {
			t.Errorf("expected valid JSON structure, got %q", got)
		}
	})

	t.Run("invalid flag", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := xxdRun([]string{"-x"}, &stdout, &stderr, nil, "")
		if code != 2 {
			t.Errorf("expected exit code 2, got %d", code)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := xxdRun([]string{"nonexistent_file_xyz"}, &stdout, &stderr, nil, "")
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})
}
