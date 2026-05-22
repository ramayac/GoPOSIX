// nl: line numbering utility
package nl

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

type NlLine struct {
	Number int    `json:"number,omitempty"`
	Text   string `json:"text"`
}
type NlResult struct {
	Lines []NlLine `json:"lines"`
}

var nlSpec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "b", Long: "body-numbering", Type: common.FlagValue},
		{Short: "v", Long: "starting-line-number", Type: common.FlagValue},
		{Short: "w", Long: "number-width", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// NumberLines is the library-layer entry point for line numbering.
// Returns formatted lines and a structured result.
func NumberLines(r io.Reader, bodyType string, startNum, width int) ([]string, NlResult) {
	var lines []string
	var result NlResult
	sc := bufio.NewScanner(r)
	num := startNum
	for sc.Scan() {
		line := sc.Text()
		nl := NlLine{Text: line}
		var formatted string
		switch bodyType {
		case "a":
			nl.Number = num
			formatted = fmt.Sprintf("%*d\t%s", width, num, line)
			num++
		case "t":
			if strings.TrimSpace(line) != "" {
				nl.Number = num
				formatted = fmt.Sprintf("%*d\t%s", width, num, line)
				num++
			} else {
				formatted = fmt.Sprintf("%*s%s", width+1, "", line)
			}
		case "n":
			formatted = fmt.Sprintf("%*s%s", width+1, "", line)
		}
		lines = append(lines, formatted)
		result.Lines = append(result.Lines, nl)
	}
	return lines, result
}

func nlRun(args []string, stdout, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, nlSpec)
	if err != nil {
		fmt.Fprintf(errOut, "nl: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	bodyType := flags.Get("b")
	if bodyType == "" {
		bodyType = "t"
	}
	startNum := 1
	if v := flags.Get("v"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			startNum = n
		}
	}
	width := 6
	if w := flags.Get("w"); w != "" {
		if n, e := strconv.Atoi(w); e == nil && n > 0 {
			width = n
		}
	}

	var reader io.Reader = stdin
	if len(flags.Positional) > 0 {
		f, err := os.Open(flags.Positional[0])
		if err != nil {
			fmt.Fprintf(errOut, "nl: %v\n", err)
			return 1
		}
		defer f.Close()
		reader = f
	}

	var result []NlLine
	sc := bufio.NewScanner(reader)
	num := startNum
	for sc.Scan() {
		line := sc.Text()
		nl := NlLine{Text: line}
		switch bodyType {
		case "a":
			nl.Number = num
			num++
		case "t":
			if strings.TrimSpace(line) != "" {
				nl.Number = num
				num++
			}
		case "n":
			// no numbering
		}
		result = append(result, nl)
	}

	if jsonMode {
		common.Render("nl", NlResult{Lines: result}, true, stdout, func() {})
		return 0
	}

	for _, nl := range result {
		if nl.Number > 0 {
			fmt.Fprintf(stdout, "%*d\t%s\n", width, nl.Number, nl.Text)
		} else {
			fmt.Fprintf(stdout, "%*s%s\n", width+1, "", nl.Text)
		}
	}
	return 0
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return nlRun(args, stdout, stderr, stdin, cwd)
}
func init() {
	dispatch.Register(dispatch.Command{Name: "nl", Usage: "Number lines of files", Run: run})
}
