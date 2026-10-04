package chgrp

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "R", Long: "recursive", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type ChgrpResult struct {
	Path string `json:"path"`
}

type ChgrpResp struct {
	Changed []ChgrpResult `json:"changed"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		return common.RenderFlagError("chgrp", args, err, stderr, 1)
	}

	if len(flags.Positional) < 2 {
		fmt.Fprintln(stderr, "chgrp: missing operand")
		return 1
	}

	groupStr := flags.Positional[0]
	gid := common.LookupGID(groupStr)
	if gid < 0 {
		fmt.Fprintf(stderr, "chgrp: invalid group: %s\n", groupStr)
		return 1
	}

	var res []ChgrpResult
	exitCode := 0

	for _, path := range flags.Positional[1:] {
		err := os.Chown(path, -1, gid)
		if err != nil {
			fmt.Fprintf(stderr, "chgrp: %v\n", err)
			exitCode = 1
		} else {
			res = append(res, ChgrpResult{Path: path})
		}
	}

	if flags.Has("json") {
		common.Render("chgrp", ChgrpResp{Changed: res}, true, stdout, func() {})
	}

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "chgrp", Usage: "Change group ownership", Run: run})
}
