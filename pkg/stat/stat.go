// Package stat implements the POSIX stat utility.
package stat

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// StatResult is the comprehensive --json output for a file.
type StatResult struct {
	Path   string    `json:"path"`
	Size   int64     `json:"size"`
	Mode   string    `json:"mode"`
	UID    uint32    `json:"uid"`
	GID    uint32    `json:"gid"`
	Atime  time.Time `json:"atime"`
	Mtime  time.Time `json:"mtime"`
	Ctime  time.Time `json:"ctime"`
	Inode  uint64    `json:"inode"`
	Links  uint64    `json:"links"`
	Blocks int64     `json:"blocks"`
	IsDir  bool      `json:"isDir"`
	IsLink bool      `json:"isLink"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stat: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	if len(flags.Positional) == 0 {
		fmt.Fprintln(os.Stderr, "stat: missing file operand")
		return 1
	}
	exitCode := 0
	for _, p := range flags.Positional {
		result, err := Run(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stat: %v\n", err)
			common.RenderError("stat", 1, "ESTAT", err.Error(), jsonMode, out)
			exitCode = 1
			continue
		}
		common.Render("stat", result, jsonMode, out, func() {
			fmt.Printf("  File: %s\n", result.Path)
			fmt.Printf("  Size: %-15d Blocks: %-10d  %s\n", result.Size, result.Blocks, result.Mode)
			fmt.Printf("  Inode: %-14d Links: %d\n", result.Inode, result.Links)
			fmt.Printf("  Uid: %-4d  Gid: %-4d\n", result.UID, result.GID)
			fmt.Printf("  Access: %s\n", result.Atime.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Modify: %s\n", result.Mtime.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Change: %s\n", result.Ctime.Format("2006-01-02 15:04:05"))
		})
	}
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "stat", Usage: "Display file status", Run: run})
}
