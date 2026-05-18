// Package whoami implements the POSIX whoami utility.
package whoami

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// WhoamiResult is the structured result for --json mode.
type WhoamiResult struct {
	User string `json:"user"`
	UID  int    `json:"uid"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// Run returns the current user information.
func Run() (WhoamiResult, error) {
	u, err := user.Current()
	if err != nil {
		return WhoamiResult{}, err
	}
	uid, _ := strconv.Atoi(u.Uid)
	return WhoamiResult{User: u.Username, UID: uid}, nil
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "whoami: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	result, err := Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "whoami: %v\n", err)
		common.RenderError("whoami", 1, "EUSER", err.Error(), jsonMode, out)
		return 1
	}

	common.Render("whoami", result, jsonMode, out, func() {
		fmt.Println(result.User)
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "whoami",
		Usage: "Print effective user name",
		Run:   run,
	})
}
