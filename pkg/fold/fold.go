// Package fold implements the POSIX fold utility.
//
// fold wraps input lines to a specified width.
// With -s, breaks occur at spaces when possible.
// With -b, width is counted in bytes rather than columns.
package fold

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// FoldResult is the --json output.
type FoldResult struct {
	Lines []string `json:"lines"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "w", Long: "width", Type: common.FlagValue},
		{Short: "b", Type: common.FlagBool},
		{Short: "s", Long: "spaces", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// --- Library layer ---

// Fold wraps text read from r to the given width.
// Returns the folded text with newlines.
func Fold(r io.Reader, width int, byteMode, spaceBreak bool) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(r)
	first := true
	for scanner.Scan() {
		if !first {
			result.WriteByte('\n')
		}
		first = false
		line := scanner.Text()
		result.Write(foldLine(line, width, byteMode, spaceBreak))
	}
	// If last line was empty, preserve the trailing newline.
	// Scanner.Text() returns "" for empty lines, so we detect EOF.
	return result.String(), scanner.Err()
}

// foldLine wraps a single line.
func foldLine(line string, width int, byteMode, spaceBreak bool) []byte {
	if len(line) == 0 {
		return []byte{'\n'}
	}
	if width <= 0 {
		return append([]byte(line), '\n')
	}

	// In byte mode, work with raw bytes. Otherwise, work with runes.
	if byteMode {
		return foldLineBytes(line, width, spaceBreak)
	}
	return foldLineRunes(line, width, spaceBreak)
}

// foldLineBytes wraps by byte count.
func foldLineBytes(line string, width int, spaceBreak bool) []byte {
	var result []byte
	pos := 0
	start := 0

	for i := 0; i < len(line); i++ {
		ch := line[i]
		advance := 1

		if pos+advance > width && pos > 0 {
			breakAt := i
			if spaceBreak {
				for j := i - 1; j >= start; j-- {
					if line[j] == ' ' {
						breakAt = j + 1
						break
					}
				}
			}
			result = append(result, line[start:breakAt]...)
			result = append(result, '\n')
			start = breakAt
			if spaceBreak && start < len(line) && line[start] == ' ' {
				start++
			}
			pos = 0
			i = start - 1
			continue
		}

		if pos == 0 && advance > width {
			result = append(result, ch)
			result = append(result, '\n')
			start = i + 1
			pos = 0
			continue
		}

		pos += advance
	}

	if start < len(line) {
		result = append(result, line[start:]...)
	}
	return result
}

// foldLineRunes wraps by character (rune) count with tab expansion.
func foldLineRunes(line string, width int, spaceBreak bool) []byte {
	runes := []rune(line)
	var result []byte
	pos := 0  // column position
	start := 0 // rune index

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		advance := 1
		if r == '\t' {
			advance = 8 - (pos % 8)
		}

		if pos+advance > width && pos > 0 {
			// Find break point in runes.
			breakAt := i
			if spaceBreak {
				for j := i - 1; j >= start; j-- {
					if runes[j] == ' ' {
						breakAt = j + 1
						break
					}
				}
			}
			result = append(result, []byte(string(runes[start:breakAt]))...)
			result = append(result, '\n')
			start = breakAt
			if spaceBreak && start < len(runes) && runes[start] == ' ' {
				start++
			}
			pos = 0
			i = start - 1
			continue
		}

		pos += advance
	}

	if start < len(runes) {
		result = append(result, []byte(string(runes[start:]))...)
	}
	return result
}

// --- CLI Glue ---

func foldRun(args []string, out, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "fold: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")
	byteMode := flags.Has("b")
	spaceBreak := flags.Has("s")

	width := 80
	if w := flags.Get("w"); w != "" {
		if v, e := strconv.Atoi(w); e == nil && v > 0 {
			width = v
		}
	}

	var reader io.Reader
	if len(flags.Positional) == 0 {
		reader = stdin
	} else {
		var readers []io.Reader
		for _, path := range flags.Positional {
			if path == "-" {
				readers = append(readers, stdin)
			} else {
				f, err := os.Open(path)
				if err != nil {
					fmt.Fprintf(errOut, "fold: %s: %v\n", path, err)
					return 1
				}
				defer f.Close()
				readers = append(readers, f)
			}
		}
		reader = io.MultiReader(readers...)
	}

	output, err := Fold(reader, width, byteMode, spaceBreak)
	if err != nil {
		fmt.Fprintf(errOut, "fold: %v\n", err)
		return 1
	}

	if jsonMode {
		lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
		common.Render("fold", FoldResult{Lines: lines}, true, out, func() {})
		return 0
	}

	fmt.Fprint(out, output)
	return 0
}

func run(args []string, out io.Writer) int {
	return foldRun(args, out, os.Stderr, os.Stdin)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "fold",
		Usage: "Wrap input lines to a specified width",
		Run:   run,
	})
}
