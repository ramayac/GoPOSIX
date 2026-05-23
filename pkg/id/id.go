package id

import (
	"fmt"
	"io"
	"os/user"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "u", Long: "user", Type: common.FlagBool},
		{Short: "g", Long: "group", Type: common.FlagBool},
		{Short: "G", Long: "groups", Type: common.FlagBool},
		{Short: "n", Long: "name", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type IDInfo struct {
	UID    int      `json:"uid"`
	User   string   `json:"user"`
	GID    int      `json:"gid"`
	Group  string   `json:"group"`
	Groups []string `json:"groups"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "id: %v\n", err)
		return 1
	}

	u, err := user.Current()
	if err != nil {
		fmt.Fprintf(stderr, "id: %v\n", err)
		return 1
	}

	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	g, _ := user.LookupGroupId(u.Gid)
	groupName := u.Gid
	if g != nil {
		groupName = g.Name
	}

	gids, _ := u.GroupIds()

	info := IDInfo{
		UID:    uid,
		User:   u.Username,
		GID:    gid,
		Group:  groupName,
		Groups: gids,
	}

	jsonMode := flags.Has("json")

	if !jsonMode {
		uOpt := flags.Has("u")
		gOpt := flags.Has("g")
		GOpt := flags.Has("G")
		nOpt := flags.Has("n")

		if uOpt {
			if nOpt {
				fmt.Fprintln(stdout, u.Username)
			} else {
				fmt.Fprintln(stdout, uid)
			}
			return 0
		}

		if gOpt {
			if nOpt {
				fmt.Fprintln(stdout, groupName)
			} else {
				fmt.Fprintln(stdout, gid)
			}
			return 0
		}

		if GOpt {
			var list []string
			for _, gg := range gids {
				if nOpt {
					if goBj, err := user.LookupGroupId(gg); err == nil {
						list = append(list, goBj.Name)
					} else {
						list = append(list, gg)
					}
				} else {
					list = append(list, gg)
				}
			}
			fmt.Fprintln(stdout, strings.Join(list, " "))
			return 0
		}
	}

	common.Render("id", info, jsonMode, stdout, func() {
		fmt.Fprintf(stdout, "uid=%d(%s) gid=%d(%s)", uid, u.Username, gid, groupName)
		if len(gids) > 0 {
			fmt.Fprint(stdout, " groups=")
			for i, gg := range gids {
				gn := gg
				if goBj, err := user.LookupGroupId(gg); err == nil {
					gn = goBj.Name
				}
				if i > 0 {
					fmt.Fprint(stdout, ",")
				}
				fmt.Fprintf(stdout, "%s(%s)", gg, gn)
			}
		}
		fmt.Fprintln(stdout)
	})

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "id", Usage: "Print real and effective user and group IDs", Run: run})
}
