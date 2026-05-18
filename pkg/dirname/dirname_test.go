package dirname

import (
	"bytes"
	"testing"
)

func TestBasic(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/path/to/file.txt", "/path/to"},
		{"/path/to/dir", "/path/to"},
		{"file.txt", "."},
		{"/", "/"},
		{"///", "/"},
		{"/usr", "/"},
		{"/usr/", "/usr"},
		{"./file.txt", "."},
		{"dir/", "dir"},
		{"", "."},
	}
	for _, c := range cases {
		got := Run(c.path).Result
		if got != c.want {
			t.Errorf("dirname(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestRunCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"/usr/bin/test"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "/usr/bin/test"}, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
}

func TestRunCLINoArgs(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, &buf)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}
