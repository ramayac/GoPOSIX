package main

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/ramayac/goposix"
	"github.com/ramayac/goposix/internal/dispatch"
)

// captureRun runs goposix.Run with argv and captures stdout.
func captureRun(argv []string) (int, string) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := goposix.Run(argv)
	w.Close()
	os.Stdout = oldStdout
	var buf []byte
	for {
		b := make([]byte, 1024)
		n, _ := r.Read(b)
		if n == 0 {
			break
		}
		buf = append(buf, b[:n]...)
	}
	return code, string(buf)
}

func TestRun_Help(t *testing.T) {
	code, out := captureRun([]string{"goposix", "--help"})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out, "goposix") {
		t.Errorf("expected 'goposix' in help, got: %s", out)
	}
}

func TestRun_HelpShortFlag(t *testing.T) {
	code, out := captureRun([]string{"goposix", "-h"})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out, "goposix") {
		t.Errorf("expected help, got: %s", out)
	}
}

func TestRun_Version(t *testing.T) {
	code, out := captureRun([]string{"goposix", "--version"})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out, "version") {
		t.Errorf("expected version string, got: %s", out)
	}
}

func TestRun_ListCommands(t *testing.T) {
	code, out := captureRun([]string{"goposix", "--list-commands"})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out, "ls") && !strings.Contains(out, "echo") {
		t.Errorf("expected 'ls' or 'echo' in command list, got: %s", out)
	}
}

func TestRun_NoArgs(t *testing.T) {
	code, out := captureRun([]string{"goposix"})
	if code != 0 {
		t.Errorf("expected exit 0 for no args, got %d", code)
	}
	if !strings.Contains(out, "goposix") {
		t.Errorf("expected help, got: %s", out)
	}
}

func TestRun_Subcommand(t *testing.T) {
	code, _ := captureRun([]string{"goposix", "echo", "hello"})
	if code != 0 {
		t.Errorf("expected exit 0 for subcommand, got %d", code)
	}
}

func TestRun_SymlinkMode(t *testing.T) {
	// Symlink mode: binary name IS the command (not a well-known name)
	code, _ := captureRun([]string{"/some/path/echo", "hello"})
	if code != 0 {
		t.Errorf("expected exit 0 for symlink mode, got %d", code)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	code, _ := captureRun([]string{"goposix", "nonexistent_command_zzz"})
	if code != 127 {
		t.Errorf("expected exit 127 for unknown command, got %d", code)
	}
}

func TestRun_UnknownSymlink(t *testing.T) {
	code, _ := captureRun([]string{"/bin/nonexistent_cmd_zzz"})
	if code != 127 {
		t.Errorf("expected exit 127 for unknown symlink, got %d", code)
	}
}

func TestRun_UnknownSymlinkNoArgs(t *testing.T) {
	code, _ := captureRun([]string{"/bin/nonexistent_cmd_zzz"})
	if code != 127 {
		t.Errorf("expected exit 127, got %d", code)
	}
}

func TestRun_CommandWithArgs(t *testing.T) {
	code, _ := captureRun([]string{"goposix", "true"})
	if code != 0 {
		t.Errorf("expected exit 0 for goposix true, got %d", code)
	}
}

func TestRun_CommandReturnsError(t *testing.T) {
	code, _ := captureRun([]string{"goposix", "rm", "/nonexistent/file/zzz"})
	// rm should return non-zero for missing file
	if code == 0 {
		t.Error("expected non-zero exit for missing file")
	}
}

// cmdPkgMapping maps pkg/ directory names to the dispatch command names
// they register.  Most are 1:1, but some register aliases.
var cmdPkgMapping = map[string][]string{
	// 1:1 mappings — omitted (default: []string{pkgName})
	"grep":      {"grep", "egrep", "fgrep"},
	"gzip":      {"gzip", "gunzip"},
	"truefalse": {"true", "false"},
	"testcmd":   {"test"},
	"shell":     {"shell", "sh"},
}

// TestListCommandsMatchesPkgDir verifies that every utility package under
// pkg/ registers at least one command, and that every registered command maps
// to an existing pkg/ directory (excluding libraries like common, client).
func TestListCommandsMatchesPkgDir(t *testing.T) {
	// 1. Build the set of command names from --list-commands output.
	_, raw := captureRun([]string{"goposix", "--list-commands"})
	registered := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		name := strings.TrimSpace(line)
		if name != "" && name != "[" && name != "]" {
			// --list-commands outputs one name per line (JSON array)
			registered[name] = true
		}
	}

	// Fallback: if capture fails, use dispatch.ListAll() directly.
	if len(registered) == 0 {
		for _, c := range dispatch.ListAll() {
			registered[c.Name] = true
		}
	}

	// 2. Walk pkg/ directories on disk.
	entries, err := os.ReadDir("../../pkg")
	if err != nil {
		t.Fatalf("cannot read pkg/ directory: %v", err)
	}

	// Library packages that are not expected to register commands.
	libs := map[string]bool{
		"common": true,
		"client": true,
	}

	var missingRegistrations []string
	var orphanCommands []string

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pkgName := e.Name()
		if libs[pkgName] {
			continue
		}

		// Expected command names for this package.
		expected, ok := cmdPkgMapping[pkgName]
		if !ok {
			expected = []string{pkgName}
		}

		// Verify each expected command is registered.
		for _, cmdName := range expected {
			if !registered[cmdName] {
				// Check via dispatch.ListAll() as authoritative source.
				_, found := dispatch.Lookup(cmdName)
				if !found {
					missingRegistrations = append(missingRegistrations,
						pkgName+" → "+cmdName)
				} else {
					// It IS registered; capture just parsed badly.
					registered[cmdName] = true
				}
			}
		}
	}

	// 3. Check for commands that don't map to any pkg/ directory.
	for cmdName := range registered {
		found := false
		for _, e := range entries {
			if !e.IsDir() || libs[e.Name()] {
				continue
			}
			expected, ok := cmdPkgMapping[e.Name()]
			if !ok {
				expected = []string{e.Name()}
			}
			for _, exp := range expected {
				if exp == cmdName {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			orphanCommands = append(orphanCommands, cmdName)
		}
	}

	sort.Strings(missingRegistrations)
	sort.Strings(orphanCommands)

	for _, m := range missingRegistrations {
		t.Errorf("pkg/ package not registered in dispatch: %s "+
			"(missing blank import in cmd/goposix/main.go?)", m)
	}
	for _, o := range orphanCommands {
		t.Errorf("registered command %q has no corresponding pkg/ directory", o)
	}

	// Report total counts as diagnostic.
	if t.Failed() {
		t.Logf("pkg/ dirs: %d packages, %d registered, %d missing, %d orphans",
			len(entries)-len(libs), len(registered),
			len(missingRegistrations), len(orphanCommands))
	}
}
