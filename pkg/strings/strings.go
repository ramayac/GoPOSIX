// strings: extract printable strings from binary files
package strings

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

type StringEntry struct {
	Offset int    `json:"offset"`
	Value  string `json:"value"`
}
type StringsResult struct {
	Strings []StringEntry `json:"strings"`
}

var strSpec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "n", Long: "bytes", Type: common.FlagValue},
		{Short: "t", Long: "radix", Type: common.FlagValue},
		{Short: "a", Long: "all", Type: common.FlagBool},
		{Short: "f", Long: "print-file-name", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func isPrintable(b byte) bool {
	return (b >= 0x20 && b <= 0x7E) || b == '\t'
}

// FormatOffset formats an offset in the specified radix for output.
func FormatOffset(offset int64, radix byte) string {
	switch radix {
	case 'x':
		return strconv.FormatInt(offset, 16)
	case 'o':
		return strconv.FormatInt(offset, 8)
	default:
		return strconv.FormatInt(offset, 10)
	}
}

// Scan extracts printable strings of at least minLen from the reader.
func Scan(r io.Reader, minLen int) ([]StringEntry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var entries []StringEntry
	var run []byte
	runStart := -1
	for i, b := range data {
		if isPrintable(b) || b == '\t' {
			if runStart < 0 {
				runStart = i
			}
			run = append(run, b)
		} else {
			if len(run) >= minLen {
				entries = append(entries, StringEntry{Offset: runStart, Value: string(run)})
			}
			run = nil
			runStart = -1
		}
	}
	if len(run) >= minLen {
		entries = append(entries, StringEntry{Offset: runStart, Value: string(run)})
	}
	return entries, nil
}

func stringsRun(args []string, stdout, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, strSpec)
	if err != nil {
		fmt.Fprintf(errOut, "strings: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	minLen := 4
	if n := flags.Get("n"); n != "" {
		if v, e := strconv.Atoi(n); e == nil && v > 0 {
			minLen = v
		}
	}
	radix := flags.Get("t")
	printName := flags.Has("f")

	var data []byte
	files := flags.Positional
	if len(files) == 0 {
		data, _ = io.ReadAll(stdin)
	} else {
		for _, f := range files {
			d, _ := os.ReadFile(f)
			data = append(data, d...)
		}
	}

	var entries []StringEntry
	var run []byte
	runStart := -1

	for i, b := range data {
		if isPrintable(b) || b == '\t' {
			if runStart < 0 {
				runStart = i
			}
			run = append(run, b)
		} else {
			if len(run) >= minLen {
				entries = append(entries, StringEntry{Offset: runStart, Value: string(run)})
			}
			run = nil
			runStart = -1
		}
	}
	if len(run) >= minLen {
		entries = append(entries, StringEntry{Offset: runStart, Value: string(run)})
	}

	if jsonMode {
		common.Render("strings", StringsResult{Strings: entries}, true, stdout, func() {})
		return 0
	}

	for _, e := range entries {
		if radix == "x" {
			fmt.Fprintf(stdout, "%7x ", e.Offset)
		} else if radix == "d" {
			fmt.Fprintf(stdout, "%7d ", e.Offset)
		} else if radix == "o" {
			fmt.Fprintf(stdout, "%7o ", e.Offset)
		}
		if printName && len(files) > 0 {
			fmt.Fprintf(stdout, "%s: ", files[0])
		}
		fmt.Fprintln(stdout, e.Value)
	}
	return 0
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return stringsRun(args, stdout, stderr, stdin, cwd)
}
func init() {
	dispatch.Register(dispatch.Command{Name: "strings", Usage: "Extract printable strings from files", Run: run})
}
