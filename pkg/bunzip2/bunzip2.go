// Package bunzip2 implements the POSIX-compliant bunzip2 utility.
package bunzip2

import (
	"compress/bzip2"
	"io"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.DecompressSpec{
	ProgName: "bunzip2",
	HelpText: "Usage: bunzip2 [-cfkq] [FILE]...\n\n" +
		"Decompress FILEs (default: stdin to stdout).\n\n" +
		"Options:\n" +
		"  -c, --stdout   Write to standard output\n" +
		"  -f, --force    Force overwrite of output files\n" +
		"  -k, --keep     Keep (don't delete) input files\n" +
		"  -q, --quiet    Suppress non-critical error messages",
	Suffixes: []common.DecompSuffix{
		{Ext: ".bz2", Cut: 4},
		{Ext: ".tbz2", Cut: 5, Append: ".tar"},
		{Ext: ".tbz", Cut: 4, Append: ".tar"},
	},
	NewReader:  func(r io.Reader) (io.Reader, error) { return bzip2.NewReader(r), nil },
	CorruptMsg: "bunzip error -5",
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
		Name:  "bunzip2",
		Usage: "Decompress bzip2 compressed files",
		Run:   run,
	})
}
