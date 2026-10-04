// Package echo implements the POSIX echo utility.
//
// echo prints its arguments to stdout, separated by spaces and
// terminated with a newline. It supports -n (suppress newline),
// -e (enable backslash escapes), and -E (disable escapes, default).
// Only --json is accepted as a long flag; everything else is literal text.
package echo

import (
	"fmt"
	"io"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// EchoResult is the structured result for --json mode.
type EchoResult struct {
	Text string `json:"text"`
}

// Run is the library function: given flags and words, return EchoResult.
func Run(noNewline, escape bool, words []string) EchoResult {
	text := strings.Join(words, " ")
	if escape {
		text = processEscapes(text)
	}
	return EchoResult{Text: text}
}

// processEscapes expands \n, \t, \\, \NNN (octal), etc. like echo -e.
func processEscapes(s string) string {
	return common.ExpandEscapes(s, common.EscapeArg)
}

// parseEchoFlags manually extracts echo-specific flags from the start of args.
// Only -n, -e, -E, and --json are recognized. All other arguments (including
// anything starting with -) are treated as literal text. This avoids the
// general ParseFlags which would choke on "---" or similar strings.
func parseEchoFlags(args []string) (noNewline, escape, jsonMode bool, words []string) {
	var i int
	for i < len(args) {
		a := args[i]
		// --json (long flag only, no -j short form to avoid collisions)
		if a == "--json" {
			jsonMode = true
			i++
			continue
		}
		// Short flag groups: only -n, -e, -E are recognized
		if len(a) >= 2 && a[0] == '-' && a[1] != '-' {
			chars := a[1:]
			// Accumulate flags first; only apply if all chars are valid.
			// This prevents partial side-effects when an invalid char
			// appears (e.g., -neEZ should be printed literally).
			nnl, esc, allValid := false, false, true
			hasE := false
			for _, c := range chars {
				switch c {
				case 'n':
					nnl = true
				case 'e':
					esc = true
				case 'E':
					esc = false
					hasE = true
				default:
					allValid = false
				}
			}
			if allValid {
				noNewline = nnl
				if esc || hasE {
					escape = esc
				}
				i++
				continue
			}
		}
		// Anything else: stop flag parsing, rest is literal text
		break
	}
	words = args[i:]
	return
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	noNewline, escape, jsonMode, words := parseEchoFlags(args)

	result := Run(noNewline, escape, words)

	common.Render("echo", result, jsonMode, stdout, func() {
		if noNewline {
			fmt.Fprint(stdout, result.Text)
		} else {
			fmt.Fprintln(stdout, result.Text)
		}
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "echo",
		Usage: "Display a line of text",
		Run:   run,
	})
}
