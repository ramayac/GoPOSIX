// Package ln implements the POSIX ln utility.
package ln

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// LnResult is the --json output.
type LnResult struct {
	Links []struct {
		Target string `json:"target"`
		Link   string `json:"link"`
	} `json:"links"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Long: "symbolic", Type: common.FlagBool},
		{Short: "f", Long: "force", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ln: %v\n", err)
		return 2
	}
	if len(flags.Positional) < 2 {
		fmt.Fprintln(os.Stderr, "ln: missing file operand")
		return 1
	}
	jsonMode := flags.Has("json")
	symbolic := flags.Has("s")
	force := flags.Has("f")

	target := flags.Positional[0]
	link := flags.Positional[1]

	if force {
		os.Remove(link)
	}

	var linkErr error
	if symbolic {
		linkErr = os.Symlink(target, link)
	} else {
		linkErr = os.Link(target, link)
	}
	if linkErr != nil {
		fmt.Fprintf(os.Stderr, "ln: %v\n", linkErr)
		common.RenderError("ln", 1, "ELN", linkErr.Error(), jsonMode, out)
		return 1
	}

	result := LnResult{}
	result.Links = append(result.Links, struct {
		Target string `json:"target"`
		Link   string `json:"link"`
	}{Target: target, Link: link})
	common.Render("ln", result, jsonMode, out, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "ln", Usage: "Make links between files", Run: run})
}
