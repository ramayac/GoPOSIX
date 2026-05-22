// Package md5sum implements the POSIX md5sum utility.
package md5sum

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
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
		{Long: "json", Type: common.FlagBool},
	},
}

// HashFile computes the MD5 hash of an io.Reader.
func HashFile(r io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "md5sum: %v\n", err)
		return 1
	}

	jsonMode := flags.Has("json")
	checkMode := flags.Has("check")

	if checkMode {
		return runCheck(flags.Positional, jsonMode, stdout)
	}

	return runHash(flags.Positional, flags.Stdin, jsonMode, stdout)
}

func runHash(files []string, readStdin bool, jsonMode bool, stdout io.Writer) int {
	var results []HashResult
	exitCode := 0

	if len(files) == 0 || readStdin {
		if len(files) == 0 {
			files = []string{"-"}
		}
	}

	for _, file := range files {
		var r io.Reader
		var name string
		if file == "-" {
			r = os.Stdin
			name = "-"
		} else {
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "md5sum: %s: %v\n", file, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			r = f
			name = file
		}

		hash, err := HashFile(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "md5sum: %s: %v\n", name, err)
			exitCode = 1
			continue
		}
		results = append(results, HashResult{File: name, Hash: hash, Algorithm: "md5"})
	}

	common.Render("md5sum", results, jsonMode, stdout, func() {
		for _, r := range results {
			fmt.Fprintf(stdout, "%s  %s\n", r.Hash, r.File)
		}
	})

	return exitCode
}

func runCheck(files []string, jsonMode bool, stdout io.Writer) int {
	if len(files) == 0 {
		common.RenderError("md5sum", 1, "MISSING_FILE", "no checksum file specified", jsonMode, stdout)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "md5sum: no checksum file specified\n")
		}
		return 1
	}

	exitCode := 0
	var results []CheckResult

	for _, checksumFile := range files {
		f, err := os.Open(checksumFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "md5sum: %s: %v\n", checksumFile, err)
			exitCode = 1
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
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
					fmt.Fprintf(os.Stderr, "md5sum: %s: improperly formatted checksum line\n", checksumFile)
					exitCode = 1
					continue
				}
				parts[1] = strings.TrimLeft(parts[1], " ")
			}

			expectedHash := parts[0]
			targetFile := parts[1]

			tf, err := os.Open(targetFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: FAILED open or read\n", targetFile)
				results = append(results, CheckResult{File: targetFile, Status: "FAILED"})
				exitCode = 1
				continue
			}

			actualHash, err := HashFile(tf)
			tf.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: FAILED open or read\n", targetFile)
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
			fmt.Fprintf(os.Stderr, "md5sum: %s: no properly formatted checksum lines found\n", checksumFile)
			exitCode = 1
		}
	}

	common.Render("md5sum", results, jsonMode, stdout, func() {
		for _, r := range results {
			fmt.Fprintf(stdout, "%s: %s\n", r.File, r.Status)
		}
	})

	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "md5sum",
		Usage: "Compute and check MD5 message digest",
		Run:   run,
	})
}
