// cmp: compare two files byte by byte
package cmp

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

type CmpResult struct {
	Equal   bool `json:"equal"`
	BytePos int  `json:"byte_pos,omitempty"`
	LineNum int  `json:"line_num,omitempty"`
	Val1    int  `json:"val1,omitempty"`
	Val2    int  `json:"val2,omitempty"`
}

var cmpSpec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Long: "quiet", Type: common.FlagBool},
		{Short: "l", Long: "verbose", Type: common.FlagBool},
		{Short: "n", Long: "bytes", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// DiffEntry represents a single byte difference.
type DiffEntry struct {
	Byte int
	Line int
}

// Compare reads two files byte by byte and returns differences.
// limit caps the number of bytes compared; 0 means no limit.
// verbose: if true, all diffs are collected; if false, only first diff.
func Compare(r1, r2 io.Reader, limit int, verbose bool) ([]DiffEntry, bool) {
	d1, _ := io.ReadAll(r1)
	d2, _ := io.ReadAll(r2)
	if limit > 0 {
		if limit < len(d1) {
			d1 = d1[:limit]
		}
		if limit < len(d2) {
			d2 = d2[:limit]
		}
	}
	var diffs []DiffEntry
	bytePos := 0
	lineNum := 1
	for bytePos < len(d1) && bytePos < len(d2) {
		if d1[bytePos] == '\n' {
			lineNum++
		}
		if d1[bytePos] != d2[bytePos] {
			diffs = append(diffs, DiffEntry{Byte: bytePos + 1, Line: lineNum})
			if !verbose {
				return diffs, false
			}
		}
		bytePos++
	}
	if len(d1) != len(d2) {
		diffs = append(diffs, DiffEntry{Byte: bytePos + 1, Line: lineNum})
		return diffs, false
	}
	return diffs, len(diffs) == 0
}

func cmpRun(args []string, out, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, cmpSpec)
	if err != nil {
		fmt.Fprintf(errOut, "cmp: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	silent := flags.Has("s")
	listMode := flags.Has("l")

	limit := -1
	if n := flags.Get("n"); n != "" {
		if v, e := strconv.Atoi(n); e == nil {
			limit = v
		}
	}

	files := flags.Positional
	if len(files) < 2 {
		fmt.Fprintf(errOut, "cmp: missing operand\n")
		return 2
	}

	// If both files are "-", cache stdin so both reads see the same data.
	var stdinCache []byte
	stdinCount := 0
	for _, f := range files {
		if f == "-" {
			stdinCount++
		}
	}
	if stdinCount > 1 {
		stdinCache, _ = io.ReadAll(stdin)
	}
	stdinUsed := false

	readFile := func(path string) ([]byte, error) {
		if path == "-" {
			if stdinCount > 1 {
				return stdinCache, nil
			}
			if !stdinUsed {
				stdinUsed = true
				return io.ReadAll(stdin)
			}
			// Second "-" with count==1 shouldn't happen, but handle gracefully.
			return nil, nil
		}
		return os.ReadFile(path)
	}

	d1, err := readFile(files[0])
	if err != nil {
		fmt.Fprintf(errOut, "cmp: %s: %v\n", files[0], err)
		return 2
	}
	d2, err := readFile(files[1])
	if err != nil {
		fmt.Fprintf(errOut, "cmp: %s: %v\n", files[1], err)
		return 2
	}

	if limit >= 0 {
		if limit < len(d1) {
			d1 = d1[:limit]
		}
		if limit < len(d2) {
			d2 = d2[:limit]
		}
	}

	// Find first difference
	bytePos := 0
	lineNum := 1
	for bytePos < len(d1) && bytePos < len(d2) {
		if d1[bytePos] == '\n' {
			lineNum++
		}
		if d1[bytePos] != d2[bytePos] {
			if jsonMode {
				common.Render("cmp", CmpResult{Equal: false, BytePos: bytePos + 1, LineNum: lineNum, Val1: int(d1[bytePos]), Val2: int(d2[bytePos])}, true, out, func() {})
				return 1
			}
			if silent {
				return 1
			}
			if listMode {
				fmt.Fprintf(out, "%d %o %o\n", bytePos+1, d1[bytePos], d2[bytePos])
			} else {
				fmt.Fprintf(out, "%s %s differ: byte %d, line %d\n", files[0], files[1], bytePos+1, lineNum)
			}
			return 1
		}
		bytePos++
	}

	// One file may be longer
	if len(d1) != len(d2) {
		if jsonMode {
			common.Render("cmp", CmpResult{Equal: false, BytePos: bytePos + 1, LineNum: lineNum}, true, out, func() {})
			return 1
		}
		if silent {
			return 1
		}
		fmt.Fprintf(errOut, "cmp: EOF on %s\n", files[1])
		return 1
	}

	if jsonMode {
		common.Render("cmp", CmpResult{Equal: true}, true, out, func() {})
	}
	return 0
}

func run(args []string, out io.Writer) int { return cmpRun(args, out, os.Stderr, os.Stdin) }
func init() {
	dispatch.Register(dispatch.Command{Name: "cmp", Usage: "Compare two files byte by byte", Run: run})
}
