package chgrp

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"

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
		fmt.Fprintf(os.Stderr, "chgrp: %v\n", err)
		return 1
	}

	if len(flags.Positional) < 2 {
		fmt.Fprintln(os.Stderr, "chgrp: missing operand")
		return 1
	}

	groupStr := flags.Positional[0]
	gid := lookupGID(groupStr)
	if gid < 0 {
		fmt.Fprintf(os.Stderr, "chgrp: invalid group: %s\n", groupStr)
		return 1
	}

	var res []ChgrpResult
	exitCode := 0

	for _, path := range flags.Positional[1:] {
		err := os.Chown(path, -1, gid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chgrp: %v\n", err)
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

// lookupGID resolves a group identifier (name or numeric) to a GID.
// Returns -1 if the group cannot be found.
func lookupGID(name string) int {
	// Try numeric first
	if val, err := strconv.Atoi(name); err == nil {
		return val
	}
	// Try name lookup via /etc/group
	if g, err := user.LookupGroup(name); err == nil {
		if val, err := strconv.Atoi(g.Gid); err == nil {
			return val
		}
	}
	return -1
}

func init() {
	dispatch.Register(dispatch.Command{Name: "chgrp", Usage: "Change group ownership", Run: run})
}
