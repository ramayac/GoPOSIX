package dispatch

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRegisterAndLookup(t *testing.T) {
	// Use a fresh blank registry by temporarily swapping.
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	Register(Command{Name: "test-cmd", Usage: "a test", Run: func([]string, io.Reader, io.Writer, io.Writer, string) int { return 0 }})
	cmd, ok := Lookup("test-cmd")
	if !ok {
		t.Fatal("expected to find test-cmd")
	}
	if cmd.Name != "test-cmd" {
		t.Errorf("name: got %q, want test-cmd", cmd.Name)
	}
}

func TestLookupMissing(t *testing.T) {
	_, ok := Lookup("does-not-exist-xyz")
	if ok {
		t.Error("expected Lookup to return false for unknown command")
	}
}

func TestDuplicateRegistrationPanics(t *testing.T) {
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	Register(Command{Name: "dup", Run: func([]string, io.Reader, io.Writer, io.Writer, string) int { return 0 }})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()
	Register(Command{Name: "dup", Run: func([]string, io.Reader, io.Writer, io.Writer, string) int { return 0 }})
}

func TestListAllSorted(t *testing.T) {
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	for _, name := range []string{"z", "a", "m"} {
		n := name
		Register(Command{Name: n, Run: func([]string, io.Reader, io.Writer, io.Writer, string) int { return 0 }})
	}
	all := ListAll()
	if len(all) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(all))
	}
	if all[0].Name != "a" || all[1].Name != "m" || all[2].Name != "z" {
		t.Errorf("not sorted: %v", all)
	}
}

func TestListAllEmpty(t *testing.T) {
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	all := ListAll()
	if len(all) != 0 {
		t.Errorf("expected 0 commands, got %d", len(all))
	}
}

func TestListCommands(t *testing.T) {
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	Register(Command{Name: "alpha", Usage: "first", Run: func([]string, io.Reader, io.Writer, io.Writer, string) int { return 0 }})
	Register(Command{Name: "beta", Usage: "second", Run: func([]string, io.Reader, io.Writer, io.Writer, string) int { return 0 }})
	Register(Command{Name: "sh", Usage: "shell (filtered)", Run: func([]string, io.Reader, io.Writer, io.Writer, string) int { return 0 }})

	// Redirect stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ListCommands()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// alpha and beta should appear; sh should be filtered out.
	if !strings.Contains(out, "alpha") {
		t.Error("expected 'alpha' in ListCommands output")
	}
	if !strings.Contains(out, "beta") {
		t.Error("expected 'beta' in ListCommands output")
	}
	if strings.Contains(out, "sh") {
		t.Error("'sh' should be filtered from ListCommands output")
	}
}

func TestPrintHelp(t *testing.T) {
	old := registry
	registry = map[string]Command{}
	defer func() { registry = old }()

	Register(Command{Name: "test-cmd", Usage: "a test command", Run: func([]string, io.Reader, io.Writer, io.Writer, string) int { return 0 }})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintHelp("mybin")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "mybin") {
		t.Error("expected binary name in help output")
	}
	if !strings.Contains(out, "test-cmd") {
		t.Error("expected command name in help output")
	}
	if !strings.Contains(out, "a test command") {
		t.Error("expected usage string in help output")
	}
	if !strings.Contains(out, "Usage:") {
		t.Error("expected 'Usage:' in help output")
	}
}
