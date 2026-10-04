package chown

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "R", Long: "recursive", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type ChownResult struct {
	Path string `json:"path"`
}

type ChownResp struct {
	Changed []ChownResult `json:"changed"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		return common.RenderFlagError("chown", args, err, stderr, 1)
	}

	if len(flags.Positional) < 2 {
		fmt.Fprintln(stderr, "chown: missing operand")
		return 1
	}

	ownerStr := flags.Positional[0]
	parts := strings.SplitN(ownerStr, ":", 2)

	uid := -1
	gid := -1

	if parts[0] != "" {
		uid = common.LookupUID(parts[0])
	}
	if len(parts) > 1 && parts[1] != "" {
		gid = common.LookupGID(parts[1])
	}

	var res []ChownResult
	exitCode := 0

	for _, path := range flags.Positional[1:] {
		err := os.Chown(path, uid, gid)
		if err != nil {
			fmt.Fprintf(stderr, "chown: %v\n", err)
			exitCode = 1
		} else {
			res = append(res, ChownResult{Path: path})
		}
	}

	if flags.Has("json") {
		common.Render("chown", ChownResp{Changed: res}, true, stdout, func() {})
	}

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "chown", Usage: "Change file owner and group", Run: run})
}
