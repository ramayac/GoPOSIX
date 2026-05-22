package chown

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
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
		fmt.Fprintf(os.Stderr, "chown: %v\n", err)
		return 1
	}

	if len(flags.Positional) < 2 {
		fmt.Fprintln(os.Stderr, "chown: missing operand")
		return 1
	}

	ownerStr := flags.Positional[0]
	parts := strings.SplitN(ownerStr, ":", 2)

	uid := -1
	gid := -1

	if parts[0] != "" {
		uid = lookupUID(parts[0])
	}
	if len(parts) > 1 && parts[1] != "" {
		gid = lookupGID(parts[1])
	}

	var res []ChownResult
	exitCode := 0

	for _, path := range flags.Positional[1:] {
		err := os.Chown(path, uid, gid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chown: %v\n", err)
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

// lookupUID resolves a user identifier (name or numeric) to a UID.
// Returns -1 if the user cannot be found.
func lookupUID(name string) int {
	// Try numeric first
	if val, err := strconv.Atoi(name); err == nil {
		return val
	}
	// Try name lookup via /etc/passwd
	if u, err := user.Lookup(name); err == nil {
		if val, err := strconv.Atoi(u.Uid); err == nil {
			return val
		}
	}
	return -1
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
	dispatch.Register(dispatch.Command{Name: "chown", Usage: "Change file owner and group", Run: run})
}
