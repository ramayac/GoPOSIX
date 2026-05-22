package goposix

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ramayac/goposix/internal/dispatch"
)

func init() {
	// Register a test command for the tests below.
	dispatch.Register(dispatch.Command{
		Name:  "test-hello",
		Usage: "print a greeting",
		Run: func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
			stdout.Write([]byte("hello"))
			return 0
		},
	})
	dispatch.Register(dispatch.Command{
		Name:  "test-exit",
		Usage: "exit with given code",
		Run: func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
			return 42
		},
	})
	dispatch.Register(dispatch.Command{
		Name:  "test-echo-args",
		Usage: "echo args",
		Run: func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
			stdout.Write([]byte(strings.Join(args, " ")))
			return 0
		},
	})
}

// captureStderr runs f and returns what was written to stderr.
func captureStderr(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	f()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestRun_SubcommandDispatch(t *testing.T) {
	exit := Run([]string{"goposix", "test-hello"})
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestRun_SymlinkDispatch(t *testing.T) {
	exit := Run([]string{"/bin/test-hello"})
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	err := captureStderr(func() {
		exit := Run([]string{"goposix", "no-such-cmd"})
		if exit != 127 {
			t.Errorf("expected exit 127, got %d", exit)
		}
	})
	if !strings.Contains(err, "goposix: unknown command: no-such-cmd") {
		t.Errorf("unexpected stderr: %q", err)
	}
}

func TestRun_UnknownSymlink(t *testing.T) {
	err := captureStderr(func() {
		exit := Run([]string{"/bin/no-such-cmd"})
		if exit != 127 {
			t.Errorf("expected exit 127, got %d", exit)
		}
	})
	if !strings.Contains(err, "no-such-cmd: unknown command: no-such-cmd") {
		t.Errorf("unexpected stderr: %q", err)
	}
}

func TestRun_Version(t *testing.T) {
	old := Version
	Version = "1.2.3-test"
	defer func() { Version = old }()

	// Can't easily capture stdout here since Run uses os.Stdout directly,
	// but we can verify it doesn't panic and returns 0.
	exit := Run([]string{"goposix", "--version"})
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestRun_ListCommands(t *testing.T) {
	exit := Run([]string{"goposix", "--list-commands"})
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestRun_Help(t *testing.T) {
	exit := Run([]string{"goposix", "--help"})
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestRun_NoArgsShowsHelp(t *testing.T) {
	exit := Run([]string{"goposix"})
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestRun_BusyboxMode(t *testing.T) {
	exit := Run([]string{"busybox", "test-hello"})
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestRun_CommandExitCode(t *testing.T) {
	exit := Run([]string{"goposix", "test-exit"})
	if exit != 42 {
		t.Errorf("expected exit 42, got %d", exit)
	}
}

func TestRun_ArgForwarding(t *testing.T) {
	// test-echo-args writes its args to stdout separated by spaces.
	// We can't easily capture stdout but we can verify it doesn't crash.
	exit := Run([]string{"goposix", "test-echo-args", "a", "b", "c"})
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestWellKnownNames(t *testing.T) {
	if !isWellKnown("goposix") {
		t.Error("expected 'goposix' to be well-known")
	}
	if !isWellKnown("busybox") {
		t.Error("expected 'busybox' to be well-known")
	}
	if isWellKnown("ls") {
		t.Error("expected 'ls' NOT to be well-known")
	}
}

func TestMain(t *testing.T) {
	// Main() just calls Run(os.Args). Verify it returns 0 for a valid command.
	// We need to set os.Args to mimic a well-known binary invocation.
	origArgs := os.Args
	os.Args = []string{"goposix", "test-hello"}
	defer func() { os.Args = origArgs }()

	exit := Main()
	if exit != 0 {
		t.Errorf("expected exit 0, got %d", exit)
	}
}

func TestMain_NoArgs(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"goposix"}
	defer func() { os.Args = origArgs }()

	exit := Main()
	if exit != 0 {
		t.Errorf("expected exit 0 for no-args, got %d", exit)
	}
}

func TestMain_Symlink(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"/bin/test-hello"}
	defer func() { os.Args = origArgs }()

	exit := Main()
	if exit != 0 {
		t.Errorf("expected exit 0 for symlink invocation, got %d", exit)
	}
}

func TestRegister(t *testing.T) {
	// Register should not panic and the command should be lookuppable.
	called := false
	Register(Command{
		Name:  "test-registered",
		Usage: "verifies Register works",
		Run: func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
			called = true
			return 0
		},
	})

	cmd, ok := dispatch.Lookup("test-registered")
	if !ok {
		t.Fatal("expected command to be registered")
	}
	cmd.Run(nil, nil, nil, nil, "")
	if !called {
		t.Error("expected registered Run to be called")
	}
}

func TestRunWithWriter_Upgrade(t *testing.T) {
	// --upgrade attempts to contact GitHub; it will fail in tests but should
	// not panic and should exercise the upgrade error path.
	errOutput := captureStderr(func() {
		exit := RunWithWriter([]string{"goposix", "--upgrade"}, os.Stdout)
		if exit != 1 {
			t.Logf("note: upgrade exit code is %d (expected 1 if upgrade fails)", exit)
		}
	})
	_ = errOutput // error will vary depending on network availability
}

func TestRunWithWriter_HelpShortFlag(t *testing.T) {
	exit := Run([]string{"goposix", "-h"})
	if exit != 0 {
		t.Errorf("expected exit 0 for -h, got %d", exit)
	}
}

func TestWellKnownNames_Append(t *testing.T) {
	orig := make([]string, len(WellKnownNames))
	copy(orig, WellKnownNames)
	defer func() { WellKnownNames = orig }()

	WellKnownNames = append(WellKnownNames, "koreboot")
	if !isWellKnown("koreboot") {
		t.Error("expected 'koreboot' to be well-known after append")
	}
	// Subcommand dispatch should now work with koreboot
	exit := Run([]string{"koreboot", "test-hello"})
	if exit != 0 {
		t.Errorf("expected exit 0 with koreboot binary name, got %d", exit)
	}
}
