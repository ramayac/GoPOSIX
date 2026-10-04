// Package common — shared digest core for the <algo>sum family (md5sum,
// sha1sum, sha256sum, sha512sum, sha3sum).
//
// DigestHashMode and DigestCheckMode implement the common file-walking,
// stdin handling, checksum-file parsing, and output rendering so each
// utility only supplies its hash constructor and program name.
package common

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// DigestSpec describes one digest utility.
type DigestSpec struct {
	// ProgName is the message prefix and --json command name (e.g. "md5sum").
	ProgName string
	// Algorithm is the JSON "algorithm" value (e.g. "md5", "sha1").
	Algorithm string
	// New returns a fresh hash for this algorithm.
	New func() hash.Hash
	// CheckNoFilesStdin makes check mode read checksum lines from stdin
	// when no checksum file is given (sha3sum behavior). When false, check
	// mode reports "no checksum file specified" (GNU md5sum/sha*sum behavior).
	CheckNoFilesStdin bool
	// CheckResolveAlg optionally picks the hash algorithm per checksum line
	// based on the expected digest length (sha3sum autodetection). When nil,
	// DigestSpec.New/Algorithm are used for every line.
	CheckResolveAlg func(expectedHash string) (hash.Hash, string, error)
}

// DigestHashResult is the --json output for hash mode.
type DigestHashResult struct {
	File      string `json:"file"`
	Hash      string `json:"hash"`
	Algorithm string `json:"algorithm"`
}

// DigestCheckResult is the --json output for check mode.
type DigestCheckResult struct {
	File   string `json:"file"`
	Status string `json:"status"` // "OK" or "FAILED"
}

// DigestReader computes the hex digest of r using h.
func DigestReader(h hash.Hash, r io.Reader) (string, error) {
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// DigestHashMode computes digests for files (or stdin when readStdin is set
// or files is empty) and renders "<hex>  <name>" lines or the JSON envelope.
// Returns the POSIX exit code.
func DigestHashMode(spec DigestSpec, files []string, readStdin bool, jsonMode bool, stdin io.Reader, stdout, stderr io.Writer) int {
	var results []DigestHashResult
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
			r = stdin
			name = "-"
		} else {
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(stderr, "%s: %s: %v\n", spec.ProgName, file, err)
				exitCode = 1
				continue
			}
			defer f.Close()
			r = f
			name = file
		}

		digest, err := DigestReader(spec.New(), r)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %s: %v\n", spec.ProgName, name, err)
			exitCode = 1
			continue
		}
		results = append(results, DigestHashResult{File: name, Hash: digest, Algorithm: spec.Algorithm})
	}

	if exitCode != 0 && jsonMode {
		RenderError(spec.ProgName, exitCode, "IO", "one or more files could not be read", true, stdout)
	} else {
		Render(spec.ProgName, results, jsonMode, stdout, func() {
			for _, r := range results {
				fmt.Fprintf(stdout, "%s  %s\n", r.Hash, r.File)
			}
		})
	}

	return exitCode
}

// DigestCheckMode verifies checksum files. Each line is "<hex>  <name>".
// Returns the POSIX exit code (1 when any check fails).
func DigestCheckMode(spec DigestSpec, files []string, jsonMode bool, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(files) == 0 {
		if spec.CheckNoFilesStdin {
			files = []string{"-"}
		} else {
			RenderError(spec.ProgName, 1, "MISSING_FILE", "no checksum file specified", jsonMode, stdout)
			if !jsonMode {
				fmt.Fprintf(stderr, "%s: no checksum file specified\n", spec.ProgName)
			}
			return 1
		}
	}

	exitCode := 0
	var results []DigestCheckResult

	for _, checksumFile := range files {
		var r io.Reader
		if checksumFile == "-" {
			r = stdin
		} else {
			f, err := os.Open(checksumFile)
			if err != nil {
				fmt.Fprintf(stderr, "%s: %s: %v\n", spec.ProgName, checksumFile, err)
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
					fmt.Fprintf(stderr, "%s: %s: improperly formatted checksum line\n", spec.ProgName, checksumFile)
					exitCode = 1
					continue
				}
				parts[1] = strings.TrimLeft(parts[1], " ")
			}

			expectedHash := parts[0]
			targetFile := parts[1]

			h, _, err := digestForLine(spec, expectedHash)
			if err != nil {
				fmt.Fprintf(stderr, "%s: %v\n", spec.ProgName, err)
				exitCode = 1
				continue
			}

			tf, err := os.Open(targetFile)
			if err != nil {
				fmt.Fprintf(stderr, "%s: FAILED open or read\n", targetFile)
				results = append(results, DigestCheckResult{File: targetFile, Status: "FAILED"})
				exitCode = 1
				continue
			}

			actualHash, err := DigestReader(h, tf)
			tf.Close()
			if err != nil {
				fmt.Fprintf(stderr, "%s: FAILED open or read\n", targetFile)
				results = append(results, DigestCheckResult{File: targetFile, Status: "FAILED"})
				exitCode = 1
				continue
			}

			if actualHash == expectedHash {
				results = append(results, DigestCheckResult{File: targetFile, Status: "OK"})
			} else {
				results = append(results, DigestCheckResult{File: targetFile, Status: "FAILED"})
				exitCode = 1
			}
		}
		if !hadLines {
			fmt.Fprintf(stderr, "%s: %s: no properly formatted checksum lines found\n", spec.ProgName, checksumFile)
			exitCode = 1
		}
	}

	Render(spec.ProgName, results, jsonMode, stdout, func() {
		for _, r := range results {
			fmt.Fprintf(stdout, "%s: %s\n", r.File, r.Status)
		}
	})

	return exitCode
}

// digestForLine returns the hasher for one checksum line.
func digestForLine(spec DigestSpec, expectedHash string) (hash.Hash, string, error) {
	if spec.CheckResolveAlg != nil {
		return spec.CheckResolveAlg(expectedHash)
	}
	return spec.New(), spec.Algorithm, nil
}
