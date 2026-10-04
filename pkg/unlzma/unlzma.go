// Package unlzma implements the POSIX-compliant unlzma utility.
package unlzma

import (
	"io"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"github.com/ulikunitz/xz/lzma"
)

var spec = common.DecompressSpec{
	ProgName: "unlzma",
	HelpText: "Usage: unlzma [-cfkq] [FILE]...\n\n" +
		"Decompress FILEs (default: stdin to stdout).\n\n" +
		"Options:\n" +
		"  -c, --stdout   Write to standard output\n" +
		"  -f, --force    Force overwrite of output files\n" +
		"  -k, --keep     Keep (don't delete) input files\n" +
		"  -q, --quiet    Suppress non-critical error messages",
	Suffixes:   []common.DecompSuffix{{Ext: ".lzma", Cut: 5}},
	NewReader:  func(r io.Reader) (io.Reader, error) { return lzma.NewReader(r) },
	CorruptMsg: "corrupted data",
	Quietable:  true,
	Flags: common.FlagSpec{
		Defs: []common.FlagDef{
			{Short: "c", Long: "stdout", Type: common.FlagBool},
			{Short: "f", Long: "force", Type: common.FlagBool},
			{Short: "k", Long: "keep", Type: common.FlagBool},
			{Short: "q", Long: "quiet", Type: common.FlagBool},
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
		Name:  "unlzma",
		Usage: "Decompress lzma compressed files",
		Run:   run,
	})
}
