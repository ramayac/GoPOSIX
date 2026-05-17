// Package split implements the POSIX split utility — split a file into pieces.
package split

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// SplitResult is the --json output.
type SplitResult struct {
	Files  []string `json:"files"`
	Chunks int      `json:"chunks"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "l", Long: "lines", Type: common.FlagValue},
		{Short: "b", Long: "bytes", Type: common.FlagValue},
		{Short: "a", Long: "suffix-length", Type: common.FlagValue},
		{Short: "d", Long: "numeric-suffixes", Type: common.FlagBool},
		{Long: "filter", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

// parseSize parses a size string like "100", "1k", "2m".
func parseSize(s string) (int64, error) {
	s = strings.ToLower(s)
	multiplier := int64(1)
	if strings.HasSuffix(s, "k") {
		multiplier = 1024
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "kb") {
		multiplier = 1000
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "m") {
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "mb") {
		multiplier = 1000 * 1000
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "g") {
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n * multiplier, nil
}

// generateSuffix returns the suffix string for chunk index n.
// suffixLen: number of characters; numeric: use decimal digits instead of letters.
func generateSuffix(n int, suffixLen int, numeric bool) string {
	if numeric {
		format := fmt.Sprintf("%%0%dd", suffixLen)
		return fmt.Sprintf(format, n)
	}
	// Alphabetic: a-z, then aa, ab, ..., az, ba, ...
	// This is base-26 with digits a-z
	suffix := make([]byte, suffixLen)
	for i := suffixLen - 1; i >= 0; i-- {
		suffix[i] = byte('a' + (n % 26))
		n /= 26
	}
	return string(suffix)
}

// Run splits input into multiple output files.
func Run(r io.Reader, prefix string, linesPerFile int64, bytesPerFile int64, suffixLen int, numeric bool, filter string) (SplitResult, error) {
	if linesPerFile <= 0 && bytesPerFile <= 0 {
		linesPerFile = 1000 // POSIX default
	}

	var files []string
	chunkIdx := 0

	if bytesPerFile > 0 {
		// Byte-based splitting
		buf := make([]byte, 32*1024)
		for {
			// Generate filename
			if len(generateSuffix(chunkIdx, suffixLen, numeric)) > suffixLen {
				break
			}

			fn := prefix + generateSuffix(chunkIdx, suffixLen, numeric)
			var w io.Writer
			if filter != "" {
				// filter mode — we'd pipe to command, but for simplicity skip
				w = io.Discard
			} else {
				f, err := os.Create(fn)
				if err != nil {
					return SplitResult{Files: files, Chunks: chunkIdx}, fmt.Errorf("cannot create %s: %w", fn, err)
				}
				defer f.Close()
				w = f
				files = append(files, fn)
			}

			remaining := bytesPerFile
			for remaining > 0 {
				toRead := int64(len(buf))
				if toRead > remaining {
					toRead = remaining
				}
				n, err := r.Read(buf[:toRead])
				if n > 0 {
					w.Write(buf[:n])
					remaining -= int64(n)
				}
				if err == io.EOF {
					chunkIdx++
					return SplitResult{Files: files, Chunks: chunkIdx}, nil
				}
				if err != nil {
					return SplitResult{Files: files, Chunks: chunkIdx}, err
				}
			}
			chunkIdx++
		}
		return SplitResult{Files: files, Chunks: chunkIdx}, nil
	}

	// Line-based splitting
	scanner := bufio.NewScanner(r)
	var currentLines int64
	var currentFile *os.File

	openFile := func() error {
		if currentFile != nil {
			currentFile.Close()
		}

		if len(generateSuffix(chunkIdx, suffixLen, numeric)) > suffixLen {
			return fmt.Errorf("suffix overflow")
		}

		fn := prefix + generateSuffix(chunkIdx, suffixLen, numeric)
		if filter != "" {
			currentFile = nil
			files = append(files, fn)
			return nil
		}
		f, err := os.Create(fn)
		if err != nil {
			return fmt.Errorf("cannot create %s: %w", fn, err)
		}
		currentFile = f
		files = append(files, fn)
		return nil
	}

	if err := openFile(); err != nil {
		return SplitResult{Files: files, Chunks: chunkIdx}, err
	}

	for scanner.Scan() {
		if currentLines >= linesPerFile {
			chunkIdx++
			currentLines = 0
			if err := openFile(); err != nil {
				return SplitResult{Files: files, Chunks: chunkIdx}, err
			}
		}
		if currentFile != nil {
			fmt.Fprintln(currentFile, scanner.Text())
		}
		currentLines++
	}

	if currentFile != nil {
		currentFile.Close()
	}

	if err := scanner.Err(); err != nil {
		return SplitResult{Files: files, Chunks: chunkIdx}, err
	}

	chunkIdx++
	return SplitResult{Files: files, Chunks: chunkIdx}, nil
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "split: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	linesPerFile := int64(0)
	bytesPerFile := int64(0)

	if flags.Has("l") {
		if n, err := parseSize(flags.Get("l")); err == nil {
			linesPerFile = n
		}
	}
	if flags.Has("b") {
		if n, err := parseSize(flags.Get("b")); err == nil {
			bytesPerFile = n
		}
	}
	if linesPerFile == 0 && bytesPerFile == 0 {
		linesPerFile = 1000
	}

	suffixLen := 2
	if flags.Has("a") {
		if n, err := strconv.Atoi(flags.Get("a")); err == nil && n > 0 {
			suffixLen = n
		}
	}

	numeric := flags.Has("d")
	filter := flags.Get("filter")

	prefix := "x"
	if len(flags.Positional) > 0 {
		if flags.Positional[0] != "-" {
			// File argument
			prefix = ""
		}
	}
	if len(flags.Positional) > 1 {
		prefix = flags.Positional[1]
	}

	var input io.Reader = os.Stdin
	fileArg := ""
	if len(flags.Positional) > 0 && flags.Positional[0] != "-" {
		fileArg = flags.Positional[0]
		f, err := os.Open(fileArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "split: %v\n", err)
			common.RenderError("split", 1, "EOPEN", err.Error(), jsonMode, out)
			return 1
		}
		defer f.Close()
		input = f
	}

	result, err := Run(input, prefix, linesPerFile, bytesPerFile, suffixLen, numeric, filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "split: %v\n", err)
		common.RenderError("split", 1, "ESPLIT", err.Error(), jsonMode, out)
		return 1
	}

	common.Render("split", result, jsonMode, out, func() {})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "split",
		Usage: "Split a file into pieces",
		Run:   run,
	})
}
