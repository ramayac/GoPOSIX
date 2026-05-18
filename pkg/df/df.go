package df

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "h", Long: "human-readable", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type FSInfo struct {
	Filesystem string `json:"filesystem"`
	Size       uint64 `json:"size"`
	Used       uint64 `json:"used"`
	Avail      uint64 `json:"avail"`
	Mountpoint string `json:"mountpoint"`
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "df: %v\n", err)
		return 1
	}

	path := "/"
	if len(flags.Positional) > 0 {
		path = flags.Positional[0]
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		fmt.Fprintf(os.Stderr, "df: %s: %v\n", path, err)
		return 1
	}

	size := stat.Blocks * uint64(stat.Bsize)
	avail := stat.Bavail * uint64(stat.Bsize)
	used := size - avail

	info := FSInfo{
		Filesystem: "unknown",
		Size:       size,
		Used:       used,
		Avail:      avail,
		Mountpoint: path,
	}

	human := flags.Has("h")
	jsonMode := flags.Has("json")

	common.Render("df", []FSInfo{info}, jsonMode, out, func() {
		fmt.Fprintf(out, "Filesystem\tSize\tUsed\tAvail\tMounted on\n")
		if human {
			fmt.Fprintf(out, "unknown\t%dM\t%dM\t%dM\t%s\n", size/1024/1024, used/1024/1024, avail/1024/1024, path)
		} else {
			fmt.Fprintf(out, "unknown\t%d\t%d\t%d\t%s\n", size/1024, used/1024, avail/1024, path)
		}
	})

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "df", Usage: "Report file system disk space usage", Run: run})
}
