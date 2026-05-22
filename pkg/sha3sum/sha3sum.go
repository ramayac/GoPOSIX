// Package sha3sum implements the POSIX-aligned sha3sum utility.
package sha3sum

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"golang.org/x/crypto/sha3"
)

// HashResult holds a single file hash result.
type HashResult struct {
	File      string `json:"file"`
	Hash      string `json:"hash"`
	Algorithm string `json:"algorithm"`
}

// CheckResult holds the result of verifying one line from a checksum file.
type CheckResult struct {
	File   string `json:"file"`
	Status string `json:"status"` // "OK" or "FAILED"
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "c", Long: "check", Type: common.FlagBool},
		{Short: "a", Long: "algorithm", Type: common.FlagValue}, // 224, 256, 384, 512
		{Long: "json", Type: common.FlagBool},
	},
}

// getHasher returns the appropriate SHA-3 hasher based on size string.
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
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
			common.RenderError("sha3sum", 1, "ALGORITHM_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "sha3sum: %v\n", err)
		}
		return 1
	}

	if checkMode {
		return runCheck(flags.Positional, alg, jsonMode, stdin, stdout, stderr)
	}

	return runHash(flags.Positional, flags.Stdin, alg, jsonMode, stdin, stdout, stderr)
}

func runHash(files []string, readStdin bool, alg string, jsonMode bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	var results []HashResult
	exitCode := 0

	if len(files) == 0 || readStdin {
		if len(files) == 0 {
			files = []string{"-"}
		}
	}

	_, algName, _ := getHasher(alg)

	for _, file := range files {
		var r io.Reader
		var name string
		if file == "-" {
			r = stdin
			name = "-"
		} else {
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(stderr, "sha3sum: %s: %v\n", file, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			r = f
			name = file
		}

		hash, err := HashFile(r, alg)
		if err != nil {
			fmt.Fprintf(stderr, "sha3sum: %s: %v\n", name, err)
			exitCode = 1
			continue
		}
		results = append(results, HashResult{File: name, Hash: hash, Algorithm: algName})
	}

	common.Render("sha3sum", results, jsonMode, stdout, func() {
		for _, r := range results {
			fmt.Fprintf(stdout, "%s  %s\n", r.Hash, r.File)
		}
	})

	return exitCode
}

func runCheck(files []string, alg string, jsonMode bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(files) == 0 {
		files = []string{"-"}
	}

	exitCode := 0
	var results []CheckResult

	for _, checksumFile := range files {
		var r io.Reader
		if checksumFile == "-" {
			r = stdin
		} else {
			f, err := os.Open(checksumFile)
			if err != nil {
				fmt.Fprintf(stderr, "sha3sum: %s: %v\n", checksumFile, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			r = f
		}

		scanner := bufio.NewScanner(r)
		hadLines := false
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			hadLines = true

			parts := strings.SplitN(line, "  ", 2)
			if len(parts) != 2 {
				parts = strings.SplitN(line, " ", 2)
				if len(parts) != 2 {
					fmt.Fprintf(stderr, "sha3sum: %s: improperly formatted checksum line\n", checksumFile)
					exitCode = 1
					continue
				}
				parts[1] = strings.TrimLeft(parts[1], " ")
			}

			expectedHash := parts[0]
			targetFile := parts[1]

			// Autodetect algorithm size based on hash length if check mode doesn't lock it down
			checkAlg := alg
			if checkAlg == "" {
				switch len(expectedHash) {
				case 56:
					checkAlg = "224"
				case 64:
					checkAlg = "256"
				case 96:
					checkAlg = "384"
				case 128:
					checkAlg = "512"
				}
			}

			tf, err := os.Open(targetFile)
			if err != nil {
				fmt.Fprintf(stderr, "%s: FAILED open or read\n", targetFile)
				results = append(results, CheckResult{File: targetFile, Status: "FAILED"})
				exitCode = 1
				continue
			}

			actualHash, err := HashFile(tf, checkAlg)
			tf.Close()
			if err != nil {
				fmt.Fprintf(stderr, "%s: FAILED open or read\n", targetFile)
				results = append(results, CheckResult{File: targetFile, Status: "FAILED"})
				exitCode = 1
				continue
			}

			if actualHash == expectedHash {
				results = append(results, CheckResult{File: targetFile, Status: "OK"})
			} else {
				results = append(results, CheckResult{File: targetFile, Status: "FAILED"})
				exitCode = 1
			}
		}
		if !hadLines {
			fmt.Fprintf(stderr, "sha3sum: %s: no properly formatted checksum lines found\n", checksumFile)
			exitCode = 1
		}
	}

	common.Render("sha3sum", results, jsonMode, stdout, func() {
		for _, r := range results {
			fmt.Fprintf(stdout, "%s: %s\n", r.File, r.Status)
		}
	})

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "sha3sum",
		Usage: "Compute and check SHA3 message digest",
		Run:   run,
	})
}
