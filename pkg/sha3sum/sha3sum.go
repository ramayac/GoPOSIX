// Package sha3sum implements the POSIX-aligned sha3sum utility.
package sha3sum

import (
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"golang.org/x/crypto/sha3"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "c", Long: "check", Type: common.FlagBool},
		{Short: "a", Long: "algorithm", Type: common.FlagValue}, // 224, 256, 384, 512
		{Long: "json", Type: common.FlagBool},
	},
}

// getHasher returns the SHA-3 hasher and algorithm name for a size string.
func getHasher(alg string) (hash.Hash, string, error) {
	switch alg {
	case "", "224":
		return sha3.New224(), "sha3-224", nil
	case "256":
		return sha3.New256(), "sha3-256", nil
	case "384":
		return sha3.New384(), "sha3-384", nil
	case "512":
		return sha3.New512(), "sha3-512", nil
	default:
		return nil, "", fmt.Errorf("invalid SHA-3 algorithm: %s", alg)
	}
}

// HashFile computes the SHA-3 hash of an io.Reader.
func HashFile(r io.Reader, alg string) (string, error) {
	h, _, err := getHasher(alg)
	if err != nil {
		return "", err
	}
	return common.DigestReader(h, r)
}

// digestSpec builds the shared digest spec for a locked algorithm. When alg
// is empty, check mode autodetects the size per checksum line.
func digestSpec(alg string) common.DigestSpec {
	_, name, _ := getHasher(alg)
	spec := common.DigestSpec{
		ProgName:          "sha3sum",
		Algorithm:         name,
		New:               func() hash.Hash { hh, _, _ := getHasher(alg); return hh },
		CheckNoFilesStdin: true,
	}
	if alg == "" {
		spec.CheckResolveAlg = func(expectedHash string) (hash.Hash, string, error) {
			switch len(expectedHash) {
			case 56:
				return getHasher("224")
			case 64:
				return getHasher("256")
			case 96:
				return getHasher("384")
			case 128:
				return getHasher("512")
			default:
				return getHasher("")
			}
		}
	}
	return spec
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	if stdin == nil {
		stdin = os.Stdin
	}
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "sha3sum: %v\n", err)
		return 1
	}

	jsonMode := flags.Has("json")
	checkMode := flags.Has("check")
	alg := flags.Get("algorithm")

	// Validate algorithm early
	if _, _, err := getHasher(alg); err != nil {
		if jsonMode {
			common.RenderError("sha3sum", 1, "ALGORITHM_ERROR", err.Error(), true, stdout)
		} else {
			fmt.Fprintf(stderr, "sha3sum: %v\n", err)
		}
		return 1
	}

	if checkMode {
		return common.DigestCheckMode(digestSpec(alg), flags.Positional, jsonMode, stdin, stdout, stderr)
	}

	return common.DigestHashMode(digestSpec(alg), flags.Positional, flags.Stdin, jsonMode, stdin, stdout, stderr)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "sha3sum",
		Usage: "Compute and check SHA3 message digest",
		Run:   run,
	})
}
