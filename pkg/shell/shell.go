// Package shell implements the shell command for GoPOSIX.
//
// The shell command wraps internal/shell.Exec() (which uses mvdan.cc/sh/v3)
// and exposes it as a CLI utility. It registers both "shell" and "sh" as
// dispatch commands, supporting:
//
//   - Script file:   goposix shell /etc/rc
//   - Inline script:  goposix shell -c "echo hello"
//   - Stdin pipe:     echo "ls" | goposix shell
//   - Interactive:    goposix shell   (when stdin is a terminal)
//
// Shebang handling:
// The Linux kernel passes everything after #! as a single argument with a
// leading space.  #!/bin/goposixos shell → exec("/bin/goposixos", " shell", "/etc/rc")
// We trim leading whitespace from the first argument to handle this.
package shell

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/internal/shell"
)

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return shellRun(args, stdin, stdout, stderr, cwd)
}

// shellRun is the injectable entry point for testing.
func shellRun(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	// Parse flags. The dispatch framework strips the command name before
	// passing args, so args[0] is the first actual argument.
	// Handle shebang space in args[0] if present (symlink mode may pass
	// the binary name with leading space from kernel shebang parsing).
	if len(args) > 0 {
		args[0] = strings.TrimSpace(args[0])
	}

	var inlineScript string
	positional := []string{}

	i := 0
	for i < len(args) {
		a := args[i]
		if a == "-c" {
			if i+1 < len(args) {
				inlineScript = args[i+1]
				i += 2
				continue
			}
			fmt.Fprintln(stderr, "shell: -c requires an argument")
			return 2
		}
		if a == "--help" || a == "-h" {
			fmt.Fprintln(stdout, `Usage: shell [-c SCRIPT] [FILE]
Execute shell scripts or start an interactive shell.

Options:
  -c SCRIPT  Execute inline script
  --help     Show this help`)
			return 0
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			// Unknown flag: stop parsing, remaining args are script file path.
			break
		}
		positional = append(positional, a)
		i++
	}
	// Add any remaining args after flag parsing stopped.
	positional = append(positional, args[i:]...)

	// Determine mode
	var script string
	var exitCode int

	switch {
	case inlineScript != "":
		script = inlineScript
		result := shell.Exec(script, "", nil)
		fmt.Fprint(stdout, result.Stdout)
		if result.Stderr != "" {
			fmt.Fprint(stderr, result.Stderr)
		}
		return int(result.ExitCode)

	case len(positional) > 0:
		// Script file mode
		data, err := os.ReadFile(positional[0])
		if err != nil {
			fmt.Fprintf(stderr, "shell: %v\n", err)
			return 1
		}
		script = string(data)
		result := shell.Exec(script, "", nil)
		fmt.Fprint(stdout, result.Stdout)
		if result.Stderr != "" {
			fmt.Fprint(stderr, result.Stderr)
		}
		return int(result.ExitCode)

	default:
		// Check if stdin is a terminal (only possible when stdin is an *os.File).
		// Mock readers (bytes.Buffer, strings.Reader) are treated as pipe mode.
		if f, ok := stdin.(*os.File); ok && isTerminal(f) {
			exitCode = interactive(stdin, stdout, stderr)
		} else {
			// Pipe mode: read all stdin
			data, err := io.ReadAll(stdin)
			if err != nil {
				fmt.Fprintf(stderr, "shell: %v\n", err)
				return 1
			}
			script = string(data)
			result := shell.Exec(script, "", nil)
			fmt.Fprint(stdout, result.Stdout)
			if result.Stderr != "" {
				fmt.Fprint(stderr, result.Stderr)
			}
			return int(result.ExitCode)
		}
	}
	_ = script // used above in switch cases
	return exitCode
}

// interactive runs a simple REPL (read-eval-print loop).
func interactive(stdin io.Reader, stdout, stderr io.Writer) int {
	reader := bufio.NewReader(stdin)
	for {
		fmt.Fprint(stdout, "$ ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(stdout)
			}
			return 0
		}
		line = strings.TrimSpace(line)
		if line == "exit" || line == "quit" {
			return 0
		}
		if line == "" {
			continue
		}
		// Pass the process CWD so cd changes persist across commands.
		cwd, _ := os.Getwd()
		result := shell.Exec(line, cwd, nil)
		if result.Stdout != "" {
			fmt.Fprint(stdout, result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Fprint(stderr, result.Stderr)
		}
	}
}

// isTerminal checks whether f is a terminal (character device).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "shell",
		Usage: "Execute shell scripts or interactive shell",
		Run:   run,
	})
	dispatch.Register(dispatch.Command{
		Name:  "sh",
		Usage: "Execute shell scripts or interactive shell (alias)",
		Run:   run,
	})
	dispatch.Register(dispatch.Command{
		Name:  "ash",
		Usage: "Execute shell scripts or interactive shell (ash alias)",
		Run:   run,
	})
}
