// Package goposix is the public API entry point for building multicall binaries
// that compose GoPOSIX utilities with custom commands.
//
// Downstream projects (e.g., a bootable distro) import this package, blank-import
// utility packages to trigger their registration, and call Main() or Run().
//
//	package main
//
//	import (
//	    "os"
//
//	    "github.com/ramayac/goposix"
//
//	    // GoPOSIX's standard utilities
//	    _ "github.com/ramayac/goposix/pkg/ls"
//	    _ "github.com/ramayac/goposix/pkg/cat"
//	    // ...
//
//	    // Custom downstream utilities
//	    _ "github.com/ramayac/koreboot/pkg/init"
//	    _ "github.com/ramayac/koreboot/pkg/mount"
//	)
//
//	func main() {
//	    goposix.WellKnownNames = append(goposix.WellKnownNames, "koreboot")
//	    os.Exit(goposix.Main())
//	}
package goposix

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
)

// Version is set by ldflags at build time:
//
//	-X github.com/ramayac/goposix.Version=$(VERSION)
var Version = "0.1.0"

// WellKnownNames lists binary names that trigger subcommand dispatch.
// When argv[0] matches one of these names, argv[1] is treated as the
// command to run (e.g., "goposix ls"). Any other name triggers symlink
// dispatch where the binary name IS the command (e.g., "/bin/ls").
//
// Downstream projects should append their binary name before calling
// Main() to enable subcommand-style invocation.
var WellKnownNames = []string{"goposix", "busybox"}

// Main is the standard entry point for a multicall binary.
// It calls Run(os.Args) and is suitable as the entire body of main():
//
//	func main() { os.Exit(goposix.Main()) }
func Main() int {
	return Run(os.Args)
}

// Run dispatches to a registered command based on argv.
// Uses os.Stdout as the output writer. See RunWithWriter for injection.
func Run(argv []string) int {
	return RunWithWriter(argv, os.Stdout)
}

// RunWithWriter is the injectable variant of Run for testing and daemon use.
// It dispatches to a registered command based on argv.
//
// Dispatch modes:
//   - Subcommand mode: if filepath.Base(argv[0]) appears in WellKnownNames,
//     argv[1] is the command name and argv[2:] are its arguments.
//   - Symlink mode: otherwise, the binary name itself is the command name.
//
// Special flags recognized in subcommand mode (before the command name):
//
//	--help, -h          print command listing
//	--version           print binary name and version
//	--upgrade           self-upgrade to latest GitHub release
//	--list-commands     print one command name per line (for Dockerfile symlink generation)
//
// Returns a POSIX exit code (0 for success, 127 for unknown command, or the
// command's own exit code).
func RunWithWriter(argv []string, out io.Writer) int {
	binName := filepath.Base(argv[0])
	cmdName := binName

	if isWellKnown(cmdName) {
		if len(argv) < 2 {
			dispatch.PrintHelp(cmdName)
			return 0
		}
		switch argv[1] {
		case "--help", "-h":
			dispatch.PrintHelp(cmdName)
			return 0
		case "--version":
			fmt.Println(cmdName, "version", Version)
			return 0
		case "--upgrade":
			if err := Upgrade(Version); err != nil {
				fmt.Fprintf(os.Stderr, "%s: upgrade: %v\n", cmdName, err)
				return 1
			}
			return 0
		case "--list-commands":
			dispatch.ListCommands()
			return 0
		}
		cmdName = strings.TrimSpace(argv[1])
		argv = argv[1:] // shift so cmd sees argv[0] == cmdName
	}

	cmd, ok := dispatch.Lookup(cmdName)
	if !ok {
		fmt.Fprintf(os.Stderr, "%s: unknown command: %s\n", binName, cmdName)
		return 127
	}

	cwd, _ := os.Getwd()
	return cmd.Run(argv[1:], os.Stdin, out, os.Stderr, cwd)
}

func isWellKnown(name string) bool {
	for _, n := range WellKnownNames {
		if name == n {
			return true
		}
	}
	return false
}

// Command is an alias for dispatch.Command.
type Command = dispatch.Command

// Register adds a command to the global registry.
func Register(cmd Command) {
	dispatch.Register(cmd)
}
