// Package comm implements the POSIX comm utility.
//
// comm reads two sorted files and produces three columns:
//   - Column 1: lines only in file1
//   - Column 2: lines only in file2 (preceded by a tab)
//   - Column 3: lines appearing in both files (preceded by two tabs)
package comm

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// CommResult is the --json output.
type CommResult struct {
	OnlyFile1 []string `json:"only_file1"`
	OnlyFile2 []string `json:"only_file2"`
	Both      []string `json:"both"`
}

// toResult splits entries into the JSON result struct.
func toResult(entries []Entry) CommResult {
	var r CommResult
	for _, e := range entries {
		switch e.Col {
		case 1:
			r.OnlyFile1 = append(r.OnlyFile1, e.Text)
		case 2:
			r.OnlyFile2 = append(r.OnlyFile2, e.Text)
		case 3:
			r.Both = append(r.Both, e.Text)
		}
	}
	return r
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "1", Type: common.FlagBool},
		{Short: "2", Type: common.FlagBool},
		{Short: "3", Type: common.FlagBool},
		{Long: "total", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// --- Library layer ---

// Entry represents one line in the merged output.
type Entry struct {
	Text string
	Col  int // 1, 2, or 3
}

// Compare performs the comm comparison of two sorted line lists.
// It returns the merged output in sorted order, one Entry per unique line.
// suppress[0] suppresses col1, suppress[1] suppresses col2, suppress[2] suppresses col3.
func Compare(file1, file2 []string, suppress [3]bool) (entries []Entry) {
	i, j := 0, 0
	for i < len(file1) && j < len(file2) {
		cmp := strings.Compare(file1[i], file2[j])
		switch {
		case cmp < 0:
			if !suppress[0] {
				entries = append(entries, Entry{Text: file1[i], Col: 1})
			}
			i++
		case cmp > 0:
			if !suppress[1] {
				entries = append(entries, Entry{Text: file2[j], Col: 2})
			}
			j++
		default:
			if !suppress[2] {
				entries = append(entries, Entry{Text: file1[i], Col: 3})
			}
			i++
			j++
		}
	}
	for ; i < len(file1); i++ {
		if !suppress[0] {
			entries = append(entries, Entry{Text: file1[i], Col: 1})
		}
	}
	for ; j < len(file2); j++ {
		if !suppress[1] {
			entries = append(entries, Entry{Text: file2[j], Col: 2})
		}
	}
	return
}

// Format produces the standard comm output text.
func Format(entries []Entry) string {
	var b strings.Builder
	for _, e := range entries {
		switch e.Col {
		case 1:
			b.WriteString(e.Text)
		case 2:
			b.WriteByte('\t')
			b.WriteString(e.Text)
		case 3:
			b.WriteString("\t\t")
			b.WriteString(e.Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Counts returns the count of entries in each column.
func Counts(entries []Entry) (c1, c2, c3 int) {
	for _, e := range entries {
		switch e.Col {
		case 1:
			c1++
		case 2:
			c2++
		case 3:
			c3++
		}
	}
	return
}

// readLines reads all lines from a reader. Unterminated last lines are
// treated as complete lines (adding an implicit newline).
func readLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	// If the last line was not newline-terminated, include it anyway.
	if err := scanner.Err(); err != nil {
		return lines, err
	}
	return lines, nil
}

// --- CLI Glue ---

func commRun(args []string, stdout, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "comm: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")
	suppress1 := flags.Has("1")
	suppress2 := flags.Has("2")
	suppress3 := flags.Has("3")
	showTotal := flags.Has("total")

	// Determine file1 and file2 from positional args or "-" for stdin.
	files := flags.Positional
	if len(files) == 0 {
		fmt.Fprintf(errOut, "comm: missing file operands\n")
		return 2
	}
	if len(files) < 2 {
		fmt.Fprintf(errOut, "comm: missing second file operand\n")
		return 2
	}

	// If both files are "-", read stdin once into memory so both
	// references share the same data.
	var stdinData []string
	stdinConsumed := false
	if (len(files) >= 2 && files[0] == "-" && files[1] == "-") ||
		(len(files) >= 1 && files[0] == "-" && (len(files) < 2 || files[1] == "-")) {
		var err error
		stdinData, err = readLines(stdin)
		if err != nil {
			fmt.Fprintf(errOut, "comm: stdin: %v\n", err)
			return 1
		}
		stdinConsumed = true
	}

	getLines := func(path string) ([]string, error) {
		if path == "-" {
			if stdinConsumed {
				return stdinData, nil
			}
			stdinConsumed = true
			return readLines(stdin)
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return readLines(f)
	}

	lines1, err := getLines(files[0])
	if err != nil {
		fmt.Fprintf(errOut, "comm: %s: %v\n", files[0], err)
		return 1
	}
	lines2, err := getLines(files[1])
	if err != nil {
		fmt.Fprintf(errOut, "comm: %s: %v\n", files[1], err)
		return 1
	}

	entries := Compare(lines1, lines2, [3]bool{suppress1, suppress2, suppress3})

	if jsonMode {
		common.Render("comm", toResult(entries), true, stdout, func() {})
		return 0
	}

	text := Format(entries)
	fmt.Fprint(stdout, text)

	if showTotal {
		c1, c2, c3 := Counts(entries)
		fmt.Fprintf(errOut, "%d\t%d\t%d\n", c1, c2, c3)
	}

	return 0
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return commRun(args, stdout, stderr, stdin, cwd)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "comm",
		Usage: "Compare two sorted files line by line",
		Run:   run,
	})
}
