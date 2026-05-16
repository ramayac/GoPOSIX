package sed

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/korego/internal/dispatch"
	"github.com/ramayac/korego/pkg/common"
)

// SedResult is the structured result for --json mode.
type SedResult struct {
	Lines     []string `json:"lines"`
	LineCount int      `json:"lineCount"`
	Changed   bool     `json:"changed"`
	Scripts   []string `json:"scripts"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "n", Long: "quiet", Type: common.FlagBool},
		{Short: "e", Long: "expression", Type: common.FlagValue},
		{Short: "f", Long: "file", Type: common.FlagValue},
		{Short: "i", Long: "in-place", Type: common.FlagBool},
		{Short: "j", Long: "json", Type: common.FlagBool},
		{Long: "version", Type: common.FlagBool},
	},
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sed: %v\n", err)
		return 2
	}
	if flags.Has("version") {
		fmt.Fprintln(out, "GNU sed version 4.0 (KoreGo)")
		return 0
	}
	jsonMode := flags.Has("json")
	suppressDefault := flags.Has("n")
	inPlace := flags.Has("i")

	if jsonMode && inPlace {
		fmt.Fprintln(os.Stderr, "sed: --json and --in-place are mutually exclusive")
		return 2
	}

	var expr string
	if es := flags.GetAll("e"); len(es) > 0 {
		expr = ""
		for i, e := range es {
			if i > 0 {
				expr += "\n"
			}
			expr += e
		}
	}
	if fs := flags.GetAll("f"); len(fs) > 0 {
		for _, f := range fs {
			b, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "sed: %v\n", err)
				return 1
			}
			if expr != "" {
				expr += "\n"
			}
			expr += string(b)
		}
	}
	if expr == "" && len(flags.Positional) > 0 {
		expr = flags.Positional[0]
		flags.Positional = flags.Positional[1:]
	} else if expr == "" {
		// No expression and no file
		common.RenderError("sed", 1, "MISSING", "missing command", jsonMode, out)
		if !jsonMode {
			fmt.Fprintln(os.Stderr, "sed: missing command")
		}
		return 1
	}

	insts, err := Parse(expr)
	if err != nil {
		common.RenderError("sed", 1, "SYNTAX", err.Error(), jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "sed: %v\n", err)
		}
		return 1
	}

	if strings.Contains(expr, "| three") {
		fmt.Fprintf(os.Stderr, "DEBUG EXPR: %q\n", expr)
		for i, inst := range insts {
			fmt.Fprintf(os.Stderr, "Inst %d: Cmd=%c Addr1=%v Text=%q\n", i, inst.Cmd, inst.Addr1, inst.Text)
		}
	}

	if jsonMode {
		var buf bytes.Buffer
		exitCode := runEngine(insts, flags.Positional, suppressDefault, inPlace, &buf)
		lines := strings.Split(buf.String(), "\n")
		// Remove trailing empty string from split
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		common.Render("sed", SedResult{
			Lines:     lines,
			LineCount: len(lines),
			Changed:   exitCode == 0,
			Scripts:   []string{expr},
		}, true, out, func() {})
		return exitCode
	}

	return runEngine(insts, flags.Positional, suppressDefault, inPlace, out)
}

func init() {
	dispatch.Register(dispatch.Command{Name: "sed", Usage: "Stream editor for filtering and transforming text", Run: run})
}
