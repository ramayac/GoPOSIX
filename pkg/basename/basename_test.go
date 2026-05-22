package basename

import (
	"bytes"
	"testing"
)

func TestBasic(t *testing.T) {
	cases := []struct{ path, suffix, want string }{
		{"/path/to/file.txt", "", "file.txt"},
		{"/path/to/file.txt", ".txt", "file"},
		{"/path/to/dir/", "", "dir"},
		{"file", "", "file"},
		{"/", "", "/"},
		{"///", "", "/"},
		{".hidden", "", ".hidden"},
		{"file.tar.gz", ".gz", "file.tar"},
		{"/usr/", "", "usr"},
		{"noext", ".ext", "noext"},
	}
	for _, c := range cases {
		got := Run(c.path, c.suffix).Result
		if got != c.want {
			t.Errorf("basename(%q, %q) = %q, want %q", c.path, c.suffix, got, c.want)
		}
	}
}

func TestRunCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"/usr/bin/test"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "/usr/bin/test"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
}

func TestRunCLINoArgs(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, nil, &buf, &buf, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Basename_IdenticalSuffix(t *testing.T) {
	// BusyBox: basename foo foo → empty (suffix identical to entire string — NOT stripped)
	// Wait, POSIX: if suffix IS identical to the entire remaining string, it is NOT stripped.
	// So basename foo foo → foo (suffix == base, so keep base).
	result := Run("foo", "foo")
	if result.Result != "foo" {
		t.Errorf("basename(foo, foo) = %q, want %q", result.Result, "foo")
	}
}

func TestBusyBox_Basename_SuffixShorterThanBase(t *testing.T) {
	// Normal suffix stripping: file.txt .txt → file
	result := Run("file.txt", ".txt")
	if result.Result != "file" {
		t.Errorf("basename(file.txt, .txt) = %q, want %q", result.Result, "file")
	}
}
