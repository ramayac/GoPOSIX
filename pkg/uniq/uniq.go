// Package uniq implements the POSIX uniq utility.
package uniq

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// UniqItem is the output for json format
type UniqItem struct {
	Line  string `json:"line"`
	Count int    `json:"count"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "c", Long: "count", Type: common.FlagBool},
		{Short: "d", Long: "repeated", Type: common.FlagBool},
		{Short: "u", Long: "unique", Type: common.FlagBool},
		{Short: "i", Long: "ignore-case", Type: common.FlagBool},
		{Short: "f", Long: "skip-fields", Type: common.FlagValue},
		{Short: "s", Long: "skip-chars", Type: common.FlagValue},
		{Short: "w", Long: "check-chars", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

func extractCompareKey(line string, skipFields, skipChars, checkChars int) string {
	// Skip fields
	for i := 0; i < skipFields; i++ {
		line = strings.TrimLeft(line, " \t")
		idx := strings.IndexAny(line, " \t")
		if idx == -1 {
			line = ""
			break
		}
		line = line[idx:]
	}
	// Skip chars
	if skipChars > 0 {
		runes := []rune(line)
		if len(runes) > skipChars {
			line = string(runes[skipChars:])
		} else {
			line = ""
		}
	}
	// Check chars
	if checkChars > 0 {
		runes := []rune(line)
		if len(runes) > checkChars {
			line = string(runes[:checkChars])
		}
	}
	return line
}

func Run(r io.Reader, countMode, duplicatesOnly, uniqueOnly, ignoreCase bool, skipFields, skipChars, checkChars int) ([]UniqItem, error) {
	scanner := bufio.NewScanner(r)
	var items []UniqItem

	if uniqueOnly && duplicatesOnly {
		return items, scanner.Err()
	}

	var prev string
	var prevOrig string
	count := 0
	first := true

	for scanner.Scan() {
		orig := scanner.Text()
		line := orig
		if ignoreCase {
			line = strings.ToLower(line)
		}
		line = extractCompareKey(line, skipFields, skipChars, checkChars)

		if first {
			prev = line
			prevOrig = orig
			count = 1
			first = false
			continue
		}

		if line == prev {
			count++
		} else {
			if (!duplicatesOnly && !uniqueOnly) || (duplicatesOnly && count > 1) || (uniqueOnly && count == 1) {
				items = append(items, UniqItem{Line: prevOrig, Count: count})
			}
			prev = line
			prevOrig = orig
			count = 1
		}
	}

	if !first {
		if (!duplicatesOnly && !uniqueOnly) || (duplicatesOnly && count > 1) || (uniqueOnly && count == 1) {
			items = append(items, UniqItem{Line: prevOrig, Count: count})
		}
	}

	return items, scanner.Err()
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return uniqRun(args, stdout, stderr, stdin, cwd)
}

func uniqRun(args []string, stdout io.Writer, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "uniq: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	countMode := flags.Has("c")
	duplicatesOnly := flags.Has("d")
	uniqueOnly := flags.Has("u")
	ignoreCase := flags.Has("i")
	skipFields := 0
	if val := flags.Get("f"); val != "" {
		fmt.Sscanf(val, "%d", &skipFields)
	}
	skipChars := 0
	if val := flags.Get("s"); val != "" {
		fmt.Sscanf(val, "%d", &skipChars)
	}
	checkChars := 0
	if val := flags.Get("w"); val != "" {
		fmt.Sscanf(val, "%d", &checkChars)
	}

	var in io.Reader = stdin
	if len(flags.Positional) > 0 && flags.Positional[0] != "-" {
		f, err := os.Open(flags.Positional[0])
		if err != nil {
			fmt.Fprintf(errOut, "uniq: %v\n", err)
			return 1
		}
		defer f.Close()
		in = f
	}

	if !jsonMode {
		// stdout already passed as parameter; use output file if second positional
		if len(flags.Positional) > 1 && flags.Positional[1] != "-" {
			f, err := os.Create(flags.Positional[1])
			if err != nil {
				fmt.Fprintf(errOut, "uniq: %v\n", err)
				return 1
			}
			defer f.Close()
			stdout = f
		}
	}

	items, err := Run(in, countMode, duplicatesOnly, uniqueOnly, ignoreCase, skipFields, skipChars, checkChars)
	if err != nil {
		fmt.Fprintf(errOut, "uniq: %v\n", err)
		return 1
	}

	if jsonMode {
		common.Render("uniq", items, true, stdout, func() {})
	} else {
		for _, item := range items {
			if countMode {
				fmt.Fprintf(stdout, "%7d %s\n", item.Count, item.Line)
			} else {
				fmt.Fprintln(stdout, item.Line)
			}
		}
	}

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "uniq",
		Usage: "Report or omit repeated lines",
		Run:   run,
	})
}
