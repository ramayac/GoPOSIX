// sum: BSD/SysV checksum utility
package sum

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

type FileSum struct {
	File     string `json:"file"`
	Checksum int    `json:"checksum"`
	Blocks   int    `json:"blocks"`
}
type SumResult struct {
	Files []FileSum `json:"files"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "r", Type: common.FlagBool},
		{Short: "s", Long: "sysv", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func sumBSD(data []byte) (int, int) {
	if len(data) == 0 {
		return 0, 0
	}
	var sum int
	for _, b := range data {
		sum = (sum >> 1) + ((sum & 1) << 15) + int(b)
		sum &= 0xFFFF
	}
	blocks := (len(data) + 1023) / 1024
	return sum, blocks
}

func sumSysV(data []byte) (int, int) {
	if len(data) == 0 {
		return 0, 0
	}
	var sum uint32
	for _, b := range data {
		sum += uint32(b)
	}
	blocks := (len(data) + 511) / 512
	return int((sum & 0xFFFFFFFF) % 0x10000), blocks
}

// Run computes a checksum of the data. sysv selects SysV algorithm;
// otherwise BSD is used. Returns checksum and block count.
func Run(r io.Reader, sysv bool) (int, int) {
	data, _ := io.ReadAll(r)
	if sysv {
		return sumSysV(data)
	}
	return sumBSD(data)
}

func sumRun(args []string, stdout, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "sum: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	sysv := flags.Has("s")

	files := flags.Positional
	if len(files) == 0 {
		files = []string{"-"}
	}

	var results []FileSum
	for _, path := range files {
		var data []byte
		var name string
		if path == "-" {
			data, err = io.ReadAll(stdin)
			name = ""
		} else {
			data, err = os.ReadFile(path)
			name = path
		}
		if err != nil {
			fmt.Fprintf(errOut, "sum: %s: %v\n", path, err)
			return 1
		}
		var cs, blk int
		if sysv {
			cs, blk = sumSysV(data)
		} else {
			cs, blk = sumBSD(data)
		}
		results = append(results, FileSum{File: name, Checksum: cs, Blocks: blk})
	}

	if jsonMode {
		common.Render("sum", SumResult{Files: results}, true, stdout, func() {})
		return 0
	}

	multi := len(results) > 1
	for _, r := range results {
		if sysv {
			if multi || r.File != "" {
				fmt.Fprintf(stdout, "%d %d %s\n", r.Checksum, r.Blocks, r.File)
			} else {
				fmt.Fprintf(stdout, "%d %d\n", r.Checksum, r.Blocks)
			}
		} else {
			if multi {
				fmt.Fprintf(stdout, "%05d %5d %s\n", r.Checksum, r.Blocks, r.File)
			} else {
				fmt.Fprintf(stdout, "%05d %5d\n", r.Checksum, r.Blocks)
			}
		}
	}
	return 0
}

func run(args []string, stdin io.Reader, stdout io.Writer) int { return sumRun(args, stdout, os.Stderr, os.Stdin) }
func init() {
	dispatch.Register(dispatch.Command{Name: "sum", Usage: "Compute checksum and block count", Run: run})
}
