// Package paste implements the POSIX paste utility.
//
// paste merges lines of files horizontally, separated by a delimiter.
// With multiple "-" (stdin) arguments, lines are distributed round-robin.
package paste

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// PasteResult is the --json output.
type PasteResult struct {
	Records [][]string `json:"records"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "d", Long: "delimiters", Type: common.FlagValue},
		{Short: "s", Long: "serial", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// --- Library layer ---

// Merge reads multiple scanners and returns merged records.
// Lines are read one-at-a-time from each scanner, cycling through them
// for each output row (round-robin for parallel mode, sequential for serial).
func Merge(scanners []*bufio.Scanner, serial bool) (records [][]string) {
	if len(scanners) == 0 {
		return
	}

	if serial {
		// Serial mode: each file → one output line (all lines concatenated).
		for _, sc := range scanners {
			var lines []string
			for sc.Scan() {
				lines = append(lines, sc.Text())
			}
			records = append(records, lines)
		}
		return
	}

	// Parallel mode: read line-by-line from all scanners, interleaved.
	for {
		record := make([]string, len(scanners))
		any := false
		for i, sc := range scanners {
			if sc.Scan() {
				record[i] = sc.Text()
				any = true
			}
		}
		if !any {
			break
		}
		// Trim trailing empty fields.
		for len(record) > 0 && record[len(record)-1] == "" {
			record = record[:len(record)-1]
		}
		records = append(records, record)
	}
	return
}

// Format produces the standard paste output.
// delimiters is cycled through for each gap between fields.
// An empty delimiter means no separator at that position.
func Format(records [][]string, delimiters []string) string {
	if len(delimiters) == 0 {
		delimiters = []string{"\t"}
	}
	var b strings.Builder
	for _, record := range records {
		for i, field := range record {
			if i > 0 {
				delim := delimiters[(i-1)%len(delimiters)]
				b.WriteString(delim)
			}
			b.WriteString(field)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// parseDelimiters parses the -d flag value into delimiter strings.
// Handles \t, \n, \0 (empty), \\, and literal characters.
func parseDelimiters(s string) []string {
	var result []string
	escape := false
	for i := 0; i < len(s); i++ {
		if escape {
			switch s[i] {
			case 't':
				result = append(result, "\t")
			case 'n':
				result = append(result, "\n")
			case '0':
				result = append(result, "") // empty delimiter
			case '\\':
				result = append(result, "\\")
			default:
				result = append(result, string(s[i]))
			}
			escape = false
		} else if s[i] == '\\' {
			escape = true
		} else {
			result = append(result, string(s[i]))
		}
	}
	if escape {
		result = append(result, "\\")
	}
	return result
}

// --- CLI Glue ---

func pasteRun(args []string, out, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "paste: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")
	serial := flags.Has("s")

	delimStr := flags.Get("d")
	delimiters := parseDelimiters(delimStr)
	if len(delimiters) == 0 && delimStr == "" {
		delimiters = []string{"\t"}
	}

	files := flags.Positional
	if len(files) == 0 {
		files = []string{"-"}
	}

	// If multiple "-" args, pre-read stdin and distribute round-robin.
	stdinCount := 0
	for _, f := range files {
		if f == "-" {
			stdinCount++
		}
	}

	var sharedStdinLines []string
	if stdinCount > 1 {
		sc := bufio.NewScanner(stdin)
		for sc.Scan() {
			sharedStdinLines = append(sharedStdinLines, sc.Text())
		}
	}

	// Build scanners. For shared stdin, each "-" gets a dedicated stream
	// of its round-robin-assigned lines.
	var scanners []*bufio.Scanner
	var closers []io.Closer
	stdinIdx := 0
	for _, f := range files {
		if f == "-" {
			if stdinCount > 1 {
				// Round-robin: pick every Nth line.
				var buf strings.Builder
				for i := stdinIdx; i < len(sharedStdinLines); i += stdinCount {
					buf.WriteString(sharedStdinLines[i])
					buf.WriteByte('\n')
				}
				scanners = append(scanners, bufio.NewScanner(strings.NewReader(buf.String())))
				stdinIdx++
			} else {
				scanners = append(scanners, bufio.NewScanner(stdin))
			}
		} else {
			file, err := os.Open(f)
			if err != nil {
				fmt.Fprintf(errOut, "paste: %s: %v\n", f, err)
				// Clean up already-opened files.
				for _, c := range closers {
					c.Close()
				}
				return 1
			}
			closers = append(closers, file)
			scanners = append(scanners, bufio.NewScanner(file))
		}
	}
	defer func() {
		for _, c := range closers {
			c.Close()
		}
	}()

	records := Merge(scanners, serial)

	if jsonMode {
		common.Render("paste", PasteResult{Records: records}, true, out, func() {})
		return 0
	}

	fmt.Fprint(out, Format(records, delimiters))
	return 0
}

func run(args []string, out io.Writer) int {
	return pasteRun(args, out, os.Stderr, os.Stdin)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "paste",
		Usage: "Merge lines of files",
		Run:   run,
	})
}
