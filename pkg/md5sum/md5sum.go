// Package md5sum implements the POSIX md5sum utility.
package md5sum

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "c", Long: "check", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

var digestSpec = common.DigestSpec{
	ProgName:  "md5sum",
	Algorithm: "md5",
	New:       md5.New,
}

// HashFile computes the MD5 hash of an io.Reader.
func HashFile(r io.Reader) (string, error) {
	return common.DigestReader(md5.New(), r)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	if stdin == nil {
		stdin = os.Stdin
	}
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "md5sum: %v\n", err)
		return 1
	}

	jsonMode := flags.Has("json")
	checkMode := flags.Has("check")

	if checkMode {
		return common.DigestCheckMode(digestSpec, flags.Positional, jsonMode, stdin, stdout, stderr)
	}

	return common.DigestHashMode(digestSpec, flags.Positional, flags.Stdin, jsonMode, stdin, stdout, stderr)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "md5sum",
		Usage: "Compute and check MD5 message digest",
		Run:   run,
	})
}
