// Package awk implements the POSIX awk text-processing utility.
//
// Wraps github.com/benhoyt/goawk (MIT, zero deps, pure Go) as a library.
// Provides both CLI access (via dispatch) and a Go library function (Run).
package awk

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/benhoyt/goawk/interp"
	"github.com/benhoyt/goawk/parser"
	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// Result is the structured output for --json mode.
type Result struct {
	Output string `json:"output"`
	Status int    `json:"status"`
}

// Run executes an AWK program on the given input.
//
// Parameters:
//   - source: the AWK program source text
//   - files: list of input file names (nil or empty means stdin only)
//   - fieldSep: field separator string (default " " — split on whitespace)
//   - vars: additional variable assignments in "name=value" format
//   - input: stdin reader
//   - out: stdout writer
//   - errOut: stderr writer
//
// Returns the exit status and any error.
func Run(source string, files []string, fieldSep string, vars []string,
	input io.Reader, out io.Writer, errOut io.Writer) (int, error) {

	prog, err := parser.ParseProgram([]byte(source), nil)
	if err != nil {
		fmt.Fprintf(errOut, "awk: %v\n", err)
		return 2, nil
	}

	// Build Vars slice: FS first, then user variables.
	// goawk expects alternating name/value pairs, so we split
	// "name=value" into ["name", "value"].
	allVars := []string{"FS", fieldSep}
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			allVars = append(allVars, parts[0], parts[1])
		}
	}

	config := &interp.Config{
		Stdin:  input,
		Output: out,
		Error:  errOut,
		Args:   files,
		Vars:   allVars,
	}
	return interp.ExecProgram(prog, config)
}

// run is the CLI entry point registered with the dispatcher.
//
// awk CLI syntax:
//
//	awk [-F fs] [-v var=value] [-f progfile] [--json] 'program' [file ...]
//	awk [-F fs] [-v var=value] [-f progfile] [--json] -f progfile [file ...]
//
// At least one of 'program' (positional) or '-f progfile' must be provided.
func run(args []string, out io.Writer) int {
	return awkRun(args, out, os.Stderr, os.Stdin)
}

// awkRun is the injectable entry point for testing.
func awkRun(args []string, out, errOut io.Writer, stdin io.Reader) int {
	// Manual flag parsing: awk program text can contain anything,
	// including strings starting with "-". Only -F, -v, -f, and --json
	// are recognized as flags. Everything else is positional.
	fieldSep := " "
	var vars []string
	var progFiles []string
	jsonMode := false
	var programText string
	var files []string

	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-F":
			i++
			if i < len(args) {
				fieldSep = args[i]
			}
		case strings.HasPrefix(a, "-F") && len(a) > 2:
			// -F: or -F\t etc.
			fieldSep = a[2:]
		case a == "-v":
			i++
			if i < len(args) {
				vars = append(vars, args[i])
			}
		case a == "-f":
			i++
			if i < len(args) {
				progFiles = append(progFiles, args[i])
			}
		case a == "--json":
			jsonMode = true
		case strings.HasPrefix(a, "-"):
			// Unknown flag in an awk context — treat as positional.
			// This handles cases like awk '-F:' which is invalid flag
			// syntax for us but might be intentional program text.
			// Since awk programs can start with -, we're lenient.
			if programText == "" && len(progFiles) == 0 {
				programText = a
			} else {
				files = append(files, a)
			}
		default:
			if programText == "" && len(progFiles) == 0 {
				programText = a
			} else {
				files = append(files, a)
			}
		}
		i++
	}

	// Build program source from -f files or positional program text
	var source string
	if len(progFiles) > 0 {
		var parts []string
		for _, pf := range progFiles {
			var data []byte
			var err error
			if pf == "-" {
				data, err = io.ReadAll(stdin)
			} else {
				data, err = os.ReadFile(pf)
			}
			if err != nil {
				fmt.Fprintf(errOut, "awk: %s: %v\n", pf, err)
				return 2
			}
			parts = append(parts, string(data))
		}
		source = strings.Join(parts, "\n")
	} else if programText != "" {
		source = programText
	} else {
		fmt.Fprintf(errOut, "awk: no program specified\n")
		return 2
	}

	status, err := Run(source, files, fieldSep, vars, stdin, out, errOut)
	if err != nil {
		fmt.Fprintf(errOut, "awk: %v\n", err)
		return 2
	}

	if jsonMode {
		common.Render("awk", Result{Output: "ok", Status: status}, true, out, func() {})
		return 0
	}

	return status
}

func init() {
	dispatch.Register(dispatch.Command{
		Name: "awk",
		Usage: "awk [-F fs] [-v var=value] [-f progfile] [--json] 'program' [file ...]\n" +
			"Pattern-directed scanning and processing language",
		Run: run,
	})
}
