// expand: convert tabs to spaces
package expand

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

type ExpandResult struct {
	Lines []string `json:"lines"`
}

var expSpec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "t", Long: "tabs", Type: common.FlagValue},
		{Short: "i", Long: "initial", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func expandLine(line string, tabWidth int, initialOnly bool) string {
	var result strings.Builder
	col := 0
	for i := 0; i < len(line); {
		r, size := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && size == 1 {
			result.WriteByte(line[i])
			col++
			i++
			continue
		}
		if r == '\t' {
			spaces := tabWidth - (col % tabWidth)
			result.WriteString(strings.Repeat(" ", spaces))
			col += spaces
			i += size
		} else {
			result.WriteString(line[i : i+size])
			col++
			i += size
			if initialOnly && r != ' ' {
				result.WriteString(line[i:])
				return result.String()
			}
		}
	}
	return result.String()
}

// Transform applies expandLine to each line of the input text.
func Transform(input string, tabWidth int, initialOnly bool) string {
	if input == "" {
		return ""
	}
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		lines[i] = expandLine(line, tabWidth, initialOnly)
	}
	return strings.Join(lines, "\n")
}

func expandRun(args []string, stdout, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, expSpec)
	if err != nil {
		fmt.Fprintf(errOut, "expand: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	initialOnly := flags.Has("i")

	tabWidth := 8
	if tw := flags.Get("t"); tw != "" {
		if v, e := strconv.Atoi(tw); e == nil && v > 0 {
			tabWidth = v
		}
	}

	var input []byte
	if len(flags.Positional) == 0 {
		input, _ = io.ReadAll(stdin)
	} else {
		for _, f := range flags.Positional {
			d, _ := os.ReadFile(f)
			input = append(input, d...)
		}
	}

	text := string(input)
	if len(text) > 0 && text[len(text)-1] == '\n' {
		text = text[:len(text)-1]
	}
	lines := strings.Split(text, "\n")
	var outLines []string
	for _, line := range lines {
		outLines = append(outLines, expandLine(line, tabWidth, initialOnly))
	}

	if jsonMode {
		common.Render("expand", ExpandResult{Lines: outLines}, true, stdout, func() {})
		return 0
	}

	fmt.Fprint(stdout, strings.Join(outLines, "\n"))
	if len(input) > 0 && input[len(input)-1] == '\n' {
		fmt.Fprint(stdout, "\n")
	}
	return 0
}

func run(args []string, stdin io.Reader, stdout io.Writer) int { return expandRun(args, stdout, os.Stderr, os.Stdin) }
func init() {
	dispatch.Register(dispatch.Command{Name: "expand", Usage: "Convert tabs to spaces", Run: run})
}
