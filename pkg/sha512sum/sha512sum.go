// Package sha512sum implements the POSIX sha512sum utility.
package sha512sum

import (
	"crypto/sha512"
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
	ProgName:          "sha512sum",
	Algorithm:         "sha512",
	New:               sha512.New,
	CheckNoFilesStdin: true,
}

// HashFile computes the SHA-512 hash of an io.Reader.
func HashFile(r io.Reader) (string, error) {
	return common.DigestReader(sha512.New(), r)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	if stdin == nil {
		stdin = os.Stdin
	}
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "sha512sum: %v\n", err)
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
		Name:  "sha512sum",
		Usage: "Compute and check SHA512 message digest",
		Run:   run,
	})
}
