// Package bzcat implements the POSIX-compliant bzcat utility.
package bzcat

import (
	"compress/bzip2"
	"io"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.DecompressSpec{
	ProgName: "bzcat",
	HelpText: "Usage: bzcat [FILE]...\n\n" +
		"Decompress FILEs to standard output.\n\n" +
		"Options:\n" +
		"  -h, --help     Print help",
	NewReader:  func(r io.Reader) (io.Reader, error) { return bzip2.NewReader(r), nil },
	CorruptMsg: "corrupted data",
	CatMode:    true,
	Flags: common.FlagSpec{
		Defs: []common.FlagDef{
			{Short: "h", Long: "help", Type: common.FlagBool},
			{Long: "json", Type: common.FlagBool},
		},
	},
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return common.DecompressMode(spec, args, stdin, stdout, stderr, cwd)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "bzcat",
		Usage: "Decompress bzip2 files to standard output",
		Run:   run,
	})
}
