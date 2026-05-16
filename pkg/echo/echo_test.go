package echo

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunBasic(t *testing.T) {
	result := Run(false, false, []string{"hello", "world"})
	if result.Text != "hello world" {
		t.Errorf("got %q, want %q", result.Text, "hello world")
	}
}

func TestRunEscapes(t *testing.T) {
	result := Run(false, true, []string{`hello\nworld`})
	if result.Text != "hello\nworld" {
		t.Errorf("got %q, want %q", result.Text, "hello\nworld")
	}
}

func TestRunNoEscapes(t *testing.T) {
	result := Run(false, false, []string{`hello\nworld`})
	if result.Text != `hello\nworld` {
		t.Errorf("got %q, want literal backslash-n", result.Text)
	}
}

func TestRunEmpty(t *testing.T) {
	result := Run(false, false, []string{})
	if result.Text != "" {
		t.Errorf("got %q, want empty string", result.Text)
	}
}

func TestRunMultipleEscapes(t *testing.T) {
	result := Run(false, true, []string{`a\tb`})
	if result.Text != "a\tb" {
		t.Errorf("got %q, want tab", result.Text)
	}
}

func TestProcessEscapes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{``, ``},
		{`hello`, `hello`},
		{`\n`, "\n"},
		{`\t`, "\t"},
		{`\r`, "\r"},
		{`\\`, "\\"},
		{`\a`, "\a"},
		{`\b`, "\b"},
		{`\v`, "\v"},
		{`\n\t\a`, "\n\t\a"},
		{`\x41`, "A"},        // hex
		{`\x41\x42`, "AB"},   // double hex
		{`\nhello\tworld`, "\nhello\tworld"},
		{`text\n`, "text\n"},
		{`\x`, "\\x"},
		{`\xG`, "\\xG"},
	}
	for _, c := range cases {
		got := processEscapes(c.in)
		if got != c.want {
			t.Errorf("processEscapes(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRunNoNewline(t *testing.T) {
	result := Run(true, false, []string{"hello"})
	if result.Text != "hello" {
		t.Errorf("got %q, want %q", result.Text, "hello")
	}
}

func TestRunSingleWord(t *testing.T) {
	result := Run(false, false, []string{"hello"})
	if result.Text != "hello" {
		t.Errorf("got %q, want %q", result.Text, "hello")
	}
}

func TestProcessEscapesOctal(t *testing.T) {
	got := processEscapes(`\0`)
	if got != "\x00" {
		t.Errorf("expected NUL byte, got %q (%v)", got, []byte(got))
	}
}

func TestProcessEscapesLiteralBackslash(t *testing.T) {
	got := processEscapes(`\q`)
	if got != `\q` {
		t.Errorf("expected literal backslash-q, got %q", got)
	}
}

// Verify strings.Builder usage pattern works identically for multiple words
func TestRunMultiWordEscapes(t *testing.T) {
	result := Run(false, true, []string{`line1\nline2`, `line3`})
	if !strings.Contains(result.Text, "line1\nline2") {
		t.Errorf("expected line1\\nline2, got %q", result.Text)
	}
	if !strings.HasSuffix(result.Text, "line3") {
		t.Errorf("expected to end with line3, got %q", result.Text)
	}
}

// CLI-level tests using run() directly (FlagSpec-based)

func TestRunCLIBasic(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"hello", "world"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "hello world\n" {
		t.Errorf("got %q, want %q", buf.String(), "hello world\n")
	}
}

func TestRunCLINoNewline(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-n", "hello"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "hello" {
		t.Errorf("got %q, want %q", buf.String(), "hello")
	}
}

func TestRunCLINoNewlineEscapes(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-ne", `hello\nworld`}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "hello\nworld" {
		t.Errorf("got %q, want %q", buf.String(), "hello\nworld")
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "hello", "world"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, buf.String())
	}
	data := env["data"].(map[string]interface{})
	if data["text"] != "hello world" {
		t.Errorf("text %q, want %q", data["text"], "hello world")
	}
}

func TestRunCLIJSONWithEscapes(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "-e", `hello\nworld`}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, buf.String())
	}
	data := env["data"].(map[string]interface{})
	if data["text"] != "hello\nworld" {
		t.Errorf("text %q, want %q", data["text"], "hello\\nworld")
	}
}

func TestRunCLIJSONNoNewline(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "-n", "hello"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v (%s)", err, buf.String())
	}
	data := env["data"].(map[string]interface{})
	if data["text"] != "hello" {
		t.Errorf("text %q, want %q", data["text"], "hello")
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Echo_ArgumentStartingWithDashes(t *testing.T) {
	// BusyBox: echo -ne "--- -\n+++ input" should print the text, not error.
	var buf bytes.Buffer
	code := run([]string{"-ne", "--- -\n+++ input"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	// Should contain --- and +++
	if buf.String() != "--- -\n+++ input" {
		t.Errorf("got %q, want %q", buf.String(), "--- -\n+++ input")
	}
}

func TestBusyBox_Echo_ArgumentIsDash(t *testing.T) {
	// BusyBox: echo -n "--- hello" should print literal text.
	var buf bytes.Buffer
	code := run([]string{"-n", "--- hello"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "--- hello" {
		t.Errorf("got %q, want %q", buf.String(), "--- hello")
	}
}

func TestBusyBox_Echo_NonOptsFlagGroup(t *testing.T) {
	// BusyBox CONFIG_DESKTOP test: echo -neEZ should print "-neEZ" literally
	// because Z is not a recognized flag.
	var buf bytes.Buffer
	code := run([]string{"-neEZ"}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "-neEZ\n" {
		t.Errorf("got %q, want %q", buf.String(), "-neEZ\n")
	}
}

func TestBusyBox_Echo_OctalEscapeNNN(t *testing.T) {
	// BusyBox: echo -ne '\41z' should output "!z" (octal 041 = 0x21 = '!')
	var buf bytes.Buffer
	code := run([]string{"-ne", `\41z`}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "!z" {
		t.Errorf("got %q (bytes: %v), want %q", buf.String(), []byte(buf.String()), "!z")
	}
}

func TestBusyBox_Echo_OctalEscapeZeroNNN(t *testing.T) {
	// BusyBox: echo -ne '\041' should output "!"
	var buf bytes.Buffer
	code := run([]string{"-ne", `\041`}, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "!" {
		t.Errorf("got %q, want %q", buf.String(), "!")
	}
}
