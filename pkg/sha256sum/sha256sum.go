// Package sha256sum implements the POSIX sha256sum utility.
package sha256sum

import (
	"crypto/sha256"
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
	ProgName:  "sha256sum",
	Algorithm: "sha256",
	New:       sha256.New,
}

// HashFile computes the SHA-256 hash of an io.Reader.
func HashFile(r io.Reader) (string, error) {
	return common.DigestReader(sha256.New(), r)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	if stdin == nil {
		stdin = os.Stdin
	}
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "sha256sum: %v\n", err)
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
		Name:  "sha256sum",
		Usage: "Compute and check SHA256 message digest",
		Run:   run,
	})
}
