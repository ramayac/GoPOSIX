// Package cat implements the POSIX cat utility.
package cat

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// CatResult is the --json output for cat (used for small files).
type CatResult struct {
	Lines     []string `json:"lines"`
	LineCount int      `json:"lineCount"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "n", Long: "number", Type: common.FlagBool},
		{Short: "b", Long: "number-nonblank", Type: common.FlagBool},
		{Short: "s", Long: "squeeze-blank", Type: common.FlagBool},
		{Short: "e", Type: common.FlagBool},
		{Short: "v", Long: "show-nonprinting", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// visByte converts a single byte to its visible representation for cat -v.
// Tab and newline pass through unchanged. Other control chars become ^X.
// DEL (0x7F) becomes ^?. High bytes (>=0x80) get M- prefix.
func visByte(b byte) string {
	switch {
	case b == '\t' || b == '\n':
		return string(b)
	case b < 0x20:
		return "^" + string(b+0x40)
	case b == 0x7F:
		return "^?"
	case b >= 0x80:
		b &^= 0x80
		inner := visByte(b)
		if inner[0] == '^' {
			return "M-" + inner
		}
		return "M-" + string(b)
	default:
		return string(b)
	}
}

// visLine applies cat -v transformation to a line and appends $ if showEnd is true.
func visLine(line string, showEnd bool) string {
	var buf strings.Builder
	for i := 0; i < len(line); i++ {
		buf.WriteString(visByte(line[i]))
	}
	if showEnd {
		buf.WriteByte('$')
	}
	return buf.String()
}

// Run reads from r and writes processed lines to w.
// Returns lines for JSON mode.
func Run(r io.Reader, w io.Writer, numberAll, numberNonBlank, squeezeBlank bool) ([]string, error) {
	scanner := bufio.NewScanner(r)
	lineNum := 0
	prevBlank := false
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()
		isBlank := strings.TrimSpace(line) == ""

		if squeezeBlank && isBlank && prevBlank {
			continue
		}
		prevBlank = isBlank

		var prefix string
		if numberAll {
			lineNum++
			prefix = fmt.Sprintf("%6d\t", lineNum)
		} else if numberNonBlank && !isBlank {
			lineNum++
			prefix = fmt.Sprintf("%6d\t", lineNum)
		}

		out := prefix + line
		fmt.Fprintln(w, out)
		lines = append(lines, out)
	}
	return lines, scanner.Err()
}

// catRun is the injectable CLI entry point for cat. It mirrors the POSIX
// cat command: it parses flags, opens files, reads stdin, and writes to out.
// errOut receives error messages. stdin is the reader for "-" and default stdin.
func catRun(args []string, out, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "cat: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	numberAll := flags.Has("n")
	numberNonBlank := flags.Has("b")
	squeezeBlank := flags.Has("s")
	showNonPrinting := flags.Has("v")
	showEnds := flags.Has("e")

	// Collect readers: files or stdin.
	var readers []io.Reader
	// Only default to stdin when NO positional args are given.
	// When "-" is explicitly in the positional args, it will be
	// handled in the loop below. This prevents reading stdin twice
	// when "-" is present (once from the implicit default, once
	// from the explicit "-").
	if len(flags.Positional) == 0 {
		readers = append(readers, stdin)
	}
	var openedFiles []*os.File
	for _, path := range flags.Positional {
		if path == "-" {
			readers = append(readers, stdin)
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(errOut, "cat: %s: %v\n", path, err)
			return 1
		}
		openedFiles = append(openedFiles, f)
		readers = append(readers, f)
	}
	defer func() {
		for _, f := range openedFiles {
			f.Close()
		}
	}()

	if jsonMode {
		var allLines []string
		for _, r := range readers {
			lines, err := Run(r, io.Discard, numberAll, numberNonBlank, squeezeBlank)
			if err != nil {
				fmt.Fprintf(errOut, "cat: %v\n", err)
				return 1
			}
			allLines = append(allLines, lines...)
		}
		common.Render("cat", CatResult{Lines: allLines, LineCount: len(allLines)}, true, out, func() {})
		return 0
	}

	if !numberAll && !numberNonBlank && !squeezeBlank && !showNonPrinting && !showEnds {
		for _, r := range readers {
			if _, err := io.Copy(out, r); err != nil {
				fmt.Fprintf(errOut, "cat: %v\n", err)
				return 1
			}
		}
		return 0
	}

	// cat -v / -e: per-character vis processing.
	if showNonPrinting || showEnds {
		return catRunVis(readers, showEnds || showNonPrinting, showEnds, jsonMode, out, errOut)
	}

	for _, r := range readers {
		if _, err := Run(r, out, numberAll, numberNonBlank, squeezeBlank); err != nil {
			fmt.Fprintf(errOut, "cat: %v\n", err)
			return 1
		}
	}
	return 0
}

func run(args []string, out io.Writer) int {
	return catRun(args, out, os.Stderr, os.Stdin)
}

// catRunVis handles cat -v and/or cat -e: per-line vis transformation.
func catRunVis(readers []io.Reader, doVis, showEnds, jsonMode bool, out, errOut io.Writer) int {
	var allLines []string
	for _, r := range readers {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if doVis {
				line = visLine(line, showEnds)
			} else if showEnds {
				line += "$"
			}
			fmt.Fprintln(out, line)
			allLines = append(allLines, line)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(errOut, "cat: %v\n", err)
			return 1
		}
	}
	if jsonMode {
		common.Render("cat", CatResult{Lines: allLines, LineCount: len(allLines)}, true, out, func() {})
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "cat",
		Usage: "Concatenate and print files",
		Run:   run,
	})
}
