// Package link implements the POSIX link utility — create hard links.
package link

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// LinkResult is the --json output.
type LinkResult struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// Run creates a hard link from src to dst.
func Run(src, dst string) error {
	return os.Link(src, dst)
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "link: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	if len(flags.Positional) != 2 {
		fmt.Fprintln(os.Stderr, "link: missing file operand")
		common.RenderError("link", 1, "EARGS", "missing file operand", jsonMode, out)
		return 1
	}

	src := flags.Positional[0]
	dst := flags.Positional[1]

	if err := Run(src, dst); err != nil {
		fmt.Fprintf(os.Stderr, "link: %v\n", err)
		common.RenderError("link", 1, "ELINK", err.Error(), jsonMode, out)
		return 1
	}

	result := LinkResult{Source: src, Target: dst}
	common.Render("link", result, jsonMode, out, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "link",
		Usage: "Create a hard link to a file",
		Run:   run,
	})
}
