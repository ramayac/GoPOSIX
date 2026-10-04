// Package uncompress implements the POSIX-compliant uncompress utility.
package uncompress

import (
	"io"

	"github.com/hotei/dcompress"
	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.DecompressSpec{
	ProgName: "uncompress",
	HelpText: "Usage: uncompress [-cfkq] [FILE]...\n\n" +
		"Decompress .Z FILEs (default: stdin to stdout).\n\n" +
		"Options:\n" +
		"  -c, --stdout   Write to standard output\n" +
		"  -f, --force    Force overwrite of output files\n" +
		"  -k, --keep     Keep (don't delete) input files\n" +
		"  -q, --quiet    Suppress non-critical error messages",
	Suffixes: []common.DecompSuffix{{Ext: ".z", Cut: 2}},
	NewReader: func(r io.Reader) (io.Reader, error) {
		return dcompress.NewReader(r)
	},
	CorruptMsg:    "corrupted data",
	Quietable:     true,
	RecoverPanics: true,
	SilenceLog:    true,
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
		Name:  "uncompress",
		Usage: "Decompress .Z LZW compressed files",
		Run:   run,
	})
}
