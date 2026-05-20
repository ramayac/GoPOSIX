// Package cut implements the POSIX cut utility.
package cut

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

// CutLine represents the fields/characters extracted from a single line.
type CutLine struct {
	Fields []string `json:"fields"`
}

// CutResult is the --json output for cut.
type CutResult struct {
	Lines []CutLine `json:"lines"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "f", Long: "fields", Type: common.FlagValue},
		{Short: "d", Long: "delimiter", Type: common.FlagValue},
		{Short: "c", Long: "characters", Type: common.FlagValue},
		{Short: "b", Long: "bytes", Type: common.FlagValue},
		{Short: "n", Long: "no-split-chars", Type: common.FlagBool},
		{Short: "s", Long: "only-delimited", Type: common.FlagBool},
		{Short: "D", Long: "no-suppress", Type: common.FlagBool},
		{Short: "F", Long: "whitespace-fields", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// rangeSpec defines an inclusive range.
type rangeSpec struct {
	start, end int
}

// parseList parses a list like "1,3,5" or "1-5" or "1-" or "-5"
func parseList(list string) ([]rangeSpec, error) {
	var specs []rangeSpec
	parts := strings.Split(list, ",")
	for _, p := range parts {
		if strings.Contains(p, "-") {
			sp := strings.SplitN(p, "-", 2)
			start := 1
			if sp[0] != "" {
				v, err := strconv.Atoi(sp[0])
				if err != nil || v < 1 {
					return nil, fmt.Errorf("invalid byte, character or field list")
				}
				start = v
			}
			end := int(^uint(0) >> 1) // math.MaxInt
			if sp[1] != "" {
				v, err := strconv.Atoi(sp[1])
				// POSIX says start <= end is not strictly required to be error, but busybox does
				if err != nil || v < 1 {
					return nil, fmt.Errorf("invalid byte, character or field list")
				}
				end = v
			}
			specs = append(specs, rangeSpec{start, end})
		} else {
			v, err := strconv.Atoi(p)
			if err != nil || v < 1 {
				return nil, fmt.Errorf("invalid byte, character or field list")
			}
			specs = append(specs, rangeSpec{v, v})
		}
	}
	return specs, nil
}

func inRange(idx int, specs []rangeSpec) bool {
	for _, s := range specs {
		if idx >= s.start && idx <= s.end {
			return true
		}
	}
	return false
}

func Run(r io.Reader, fields, delimiter, chars, bytesList string, onlyDelimited, noSuppress, whitespaceFields bool) ([]CutLine, error) {
	scanner := bufio.NewScanner(r)
	var lines []CutLine

	delim := "\t"
	if delimiter != "" {
		delim = delimiter
	}
	// -F overrides -d: split on repeated whitespace.
	if whitespaceFields {
		delim = "" // signals whitespace-split mode
	}

	var fieldSel []rangeSpec
	var err error
	if fields != "" {
		if fieldSel, err = parseList(fields); err != nil {
			return nil, err
		}
	}
	var charSel []rangeSpec
	if chars != "" {
		if charSel, err = parseList(chars); err != nil {
			return nil, err
		}
	}
	var byteSel []rangeSpec
	if bytesList != "" {
		if byteSel, err = parseList(bytesList); err != nil {
			return nil, err
		}
	}

	for scanner.Scan() {
		text := scanner.Text()
		var extracted []string

		if fields != "" {
			var parts []string
			if whitespaceFields {
				parts = strings.Fields(text)
			} else {
				parts = strings.Split(text, delim)
			}
			// If no delimiter found and not -F mode, POSIX cut by default prints the whole line
			// (unless -s is passed).
			hasDelim := whitespaceFields || strings.Contains(text, delim)
			if !hasDelim {
				if !onlyDelimited && !noSuppress {
					// Default: print whole line.
					extracted = append(extracted, text)
				} else if noSuppress {
					// -D: don't suppress, print empty (or partial match).
					extracted = append(extracted, "")
				} else {
					continue // Suppress line (-s)
				}
			} else {
				var selected []string
				sep := delim
				if whitespaceFields {
					sep = " "
				}
				// Select fields in the order specified by the field list.
				for _, spec := range fieldSel {
					for f := spec.start; f <= spec.end && f <= len(parts); f++ {
						if f > 0 && f <= len(parts) {
							selected = append(selected, parts[f-1])
						}
					}
				}
				extracted = append(extracted, strings.Join(selected, sep))
			}
		} else if chars != "" {
			runes := []rune(text)
			var sb strings.Builder
			for i, r := range runes {
				if inRange(i+1, charSel) {
					sb.WriteRune(r)
				}
			}
			extracted = append(extracted, sb.String())
		} else if bytesList != "" {
			var sb strings.Builder
			for i := 0; i < len(text); i++ {
				if inRange(i+1, byteSel) {
					sb.WriteByte(text[i])
				}
			}
			extracted = append(extracted, sb.String())
		}

		lines = append(lines, CutLine{Fields: extracted})
	}

	return lines, scanner.Err()
}

func run(args []string, out io.Writer) int {
	return cutRun(args, out, os.Stderr, os.Stdin)
}

func cutRun(args []string, out io.Writer, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "cut: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	fields := flags.Get("f")
	delimiter := flags.Get("d")
	chars := flags.Get("c")
	bytesList := flags.Get("b")
	onlyDelimited := flags.Has("s")
	noSuppress := flags.Has("D")
	whitespaceFields := flags.Has("F")

	// When -F is used without -f, the first positional is the field list.
	if whitespaceFields && fields == "" && len(flags.Positional) > 0 {
		fields = flags.Positional[0]
		flags.Positional = flags.Positional[1:]
	}

	if fields == "" && chars == "" && bytesList == "" {
		fmt.Fprintf(errOut, "cut: you must specify a list of bytes, characters, or fields\n")
		return 1
	}

	var readers []io.Reader
	if len(flags.Positional) == 0 {
		readers = append(readers, stdin)
	} else {
		for _, path := range flags.Positional {
			if path == "-" {
				readers = append(readers, stdin)
			} else {
				f, err := os.Open(path)
				if err != nil {
					fmt.Fprintf(errOut, "cut: %s: %v\n", path, err)
					return 1
				}
				defer f.Close()
				readers = append(readers, f)
			}
		}
	}

	var allLines []CutLine
	for _, r := range readers {
		lines, err := Run(r, fields, delimiter, chars, bytesList, onlyDelimited, noSuppress, whitespaceFields)
		if err != nil {
			fmt.Fprintf(errOut, "cut: %v\n", err)
			return 1
		}
		allLines = append(allLines, lines...)
	}

	if jsonMode {
		common.Render("cut", CutResult{Lines: allLines}, true, out, func() {})
	} else {
		for _, line := range allLines {
			if len(line.Fields) > 0 {
				fmt.Fprintln(out, line.Fields[0])
			}
		}
	}

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "cut", Usage: "Remove sections from each line of files", Run: run})
}
