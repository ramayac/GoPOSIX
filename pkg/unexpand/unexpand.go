// Package unexpand implements the POSIX unexpand utility.
//
// unexpand converts spaces to tabs in each line of input.
// By default, only leading blanks (spaces and tabs) are converted.
// With -a, all sequences of two or more blanks are converted.
package unexpand

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"unicode/utf8"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// UnexpandResult is the --json output.
type UnexpandResult struct {
	Lines []string `json:"lines"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "a", Long: "all", Type: common.FlagBool},
		{Short: "t", Long: "tabs", Type: common.FlagValue},
		{Short: "f", Long: "first-only", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// --- Library layer ---

// Transform converts spaces to tabs in the input text.
// If allBlanks is true, all blank sequences are converted;
// otherwise only leading blanks are converted.
func Transform(input string, tabWidth int, allBlanks bool) string {
	if len(input) == 0 {
		return input
	}
	hadTrailingNewline := input[len(input)-1] == '\n'
	var result []byte
	lines := splitLines(input)
	for i, line := range lines {
		if i > 0 {
			result = append(result, '\n')
		}
		result = append(result, unexpandLine(line, tabWidth, allBlanks)...)
	}
	if hadTrailingNewline {
		result = append(result, '\n')
	}
	return string(result)
}

// --- Core transformation ---

// unexpandLine converts blanks to tabs in a single line.
// allBlanks: if true, convert ALL blank sequences.
// If false (default mode): convert leading blanks AND any blank
// sequence that contains a tab character (BusyBox behavior).
func unexpandLine(line string, tabWidth int, allBlanks bool) string {
	if len(line) == 0 {
		return line
	}

	// If the line contains multi-byte UTF-8, use all-blanks mode.
	// This matches BusyBox CONFIG_UNICODE_SUPPORT behavior.
	if !allBlanks {
		for i := 0; i < len(line); i++ {
			if line[i] >= 0x80 {
				allBlanks = true
				break
			}
		}
	}

	var result []byte
	col := 0

	// Iterate by rune for proper Unicode column counting.
	for i := 0; i < len(line); {
		r, size := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte, treat as single column.
			result = append(result, line[i])
			col++
			i++
			continue
		}

		if r == ' ' || r == '\t' {
			// Check if this blank sequence should be converted.
			shouldConvert := allBlanks || col == 0 || blankRunContainsTab(line[i:])
			if shouldConvert {
				startCol := col
				for i < len(line) {
					r2, sz2 := utf8.DecodeRuneInString(line[i:])
					if r2 != ' ' && r2 != '\t' {
						break
					}
					col = advanceCol(col, r2, tabWidth)
					i += sz2
				}
				result = append(result, blanksToTabs(startCol, col, tabWidth)...)
				continue
			}
		}

		// Non-blank or non-converted blank character.
		col = advanceCol(col, r, tabWidth)
		result = append(result, line[i:i+size]...)
		i += size
	}
	return string(result)
}

// blankRunContainsTab checks if the blank run starting at s contains a tab.
// s is the substring starting at the first blank character (byte offset).
func blankRunContainsTab(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r != ' ' && r != '\t' {
			return false
		}
		if r == '\t' {
			return true
		}
		i += size
	}
	return false
}

func advanceCol(col int, r rune, tabWidth int) int {
	if r == '\t' {
		return col + tabWidth - (col % tabWidth)
	}
	return col + 1
}

// blanksToTabs converts a run of blanks occupying visual columns
// [startCol, endCol) into the minimum number of tabs and spaces.
func blanksToTabs(startCol, endCol, tabWidth int) []byte {
	var result []byte
	col := startCol
	for col < endCol {
		nextTabStop := col + tabWidth - (col % tabWidth)
		if nextTabStop <= endCol {
			result = append(result, '\t')
			col = nextTabStop
		} else {
			result = append(result, ' ')
			col++
		}
	}
	return result
}

// splitLines splits s at newline boundaries.
// If s ends with a newline, that newline is consumed (not producing an
// empty trailing line). The Transform function re-inserts newlines
// between lines.
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	// Strip trailing newline for consistent processing.
	if s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	if s == "" {
		return []string{""}
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// --- CLI Glue ---

// unexpandRun is the injectable entry point.
func unexpandRun(args []string, stdout, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "unexpand: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")
	allBlanks := flags.Has("a")
	firstOnly := flags.Has("f") || flags.Has("first-only")

	tabWidth := 8
	hasTabFlag := false
	if tw := flags.Get("t"); tw != "" {
		if v, e := strconv.Atoi(tw); e == nil && v > 0 {
			tabWidth = v
			hasTabFlag = true
		}
	}

	// BusyBox behavior: -t N without --first-only or -f implies -a.
	if hasTabFlag && !firstOnly {
		allBlanks = true
	}

	// Read from positional files or stdin.
	var input []byte
	if len(flags.Positional) == 0 {
		data, readErr := io.ReadAll(stdin)
		if readErr != nil {
			fmt.Fprintf(errOut, "unexpand: %v\n", readErr)
			return 1
		}
		input = data
	} else {
		for _, path := range flags.Positional {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				fmt.Fprintf(errOut, "unexpand: %s: %v\n", path, readErr)
				return 1
			}
			input = append(input, data...)
		}
	}

	result := Transform(string(input), tabWidth, allBlanks)

	if jsonMode {
		lines := splitLines(result)
		common.Render("unexpand", UnexpandResult{Lines: lines}, true, stdout, func() {})
		return 0
	}

	fmt.Fprint(stdout, result)
	return 0
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return unexpandRun(args, stdout, stderr, stdin, cwd)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "unexpand",
		Usage: "Convert spaces to tabs",
		Run:   run,
	})
}
