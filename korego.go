// Package korego is the public API entry point for building multicall binaries
// that compose KoreGo utilities with custom commands.
//
// Downstream projects (e.g., a bootable distro) import this package, blank-import
// utility packages to trigger their registration, and call Main() or Run().
//
//	package main
//
//	import (
//	    "os"
//
//	    "github.com/ramayac/korego"
//
//	    // KoreGo's standard utilities
//	    _ "github.com/ramayac/korego/pkg/ls"
//	    _ "github.com/ramayac/korego/pkg/cat"
//	    // ...
//
//	    // Custom downstream utilities
//	    _ "github.com/ramayac/koreboot/pkg/init"
//	    _ "github.com/ramayac/koreboot/pkg/mount"
//	)
//
//	func main() {
//	    korego.WellKnownNames = append(korego.WellKnownNames, "koreboot")
//	    os.Exit(korego.Main())
//	}
package korego

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ramayac/korego/internal/dispatch"
)

// Version is set by ldflags at build time:
//
//	-X github.com/ramayac/korego.Version=$(VERSION)
var Version = "0.1.0"

// WellKnownNames lists binary names that trigger subcommand dispatch.
// When argv[0] matches one of these names, argv[1] is treated as the
// command to run (e.g., "korego ls"). Any other name triggers symlink
// dispatch where the binary name IS the command (e.g., "/bin/ls").
//
// Downstream projects should append their binary name before calling
// Main() to enable subcommand-style invocation.
var WellKnownNames = []string{"korego", "busybox"}

// Main is the standard entry point for a multicall binary.
// It calls Run(os.Args) and is suitable as the entire body of main():
//
//	func main() { os.Exit(korego.Main()) }
func Main() int {
	return Run(os.Args)
}

// Run dispatches to a registered command based on argv.
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
//	--list-commands     print one command name per line (for Dockerfile symlink generation)
//
// Returns a POSIX exit code (0 for success, 127 for unknown command, or the
// command's own exit code).
func Run(argv []string) int {
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
		case "--list-commands":
			dispatch.ListCommands()
			return 0
		}
		cmdName = argv[1]
		argv = argv[1:] // shift so cmd sees argv[0] == cmdName
	}

	cmd, ok := dispatch.Lookup(cmdName)
	if !ok {
		fmt.Fprintf(os.Stderr, "%s: unknown command: %s\n", binName, cmdName)
		return 127
	}

	return cmd.Run(argv[1:], os.Stdout)
}

func isWellKnown(name string) bool {
	for _, n := range WellKnownNames {
		if name == n {
			return true
		}
	}
	return false
}
