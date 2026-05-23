// Package uudecode implements the POSIX-compliant uudecode utility.
package uudecode

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "o", Long: "output-file", Type: common.FlagValue},
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "uudecode",
		Usage: "Decode a uuencoded file",
		Run:   run,
	})
}

// UudecodeResult represents the JSON response structure.
type UudecodeResult struct {
	Source       string `json:"source"`
	Destination  string `json:"destination"`
	BytesDecoded int64  `json:"bytesDecoded"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			break
		}
	}

	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		if jsonMode {
			common.RenderError("uudecode", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "uudecode: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: uudecode [-o OUTFILE] [INFILE]\n\n" +
			"Decode a uuencoded file (default: stdin).\n\n" +
			"Options:\n" +
			"  -o OUTFILE     Write to OUTFILE instead of the name specified in the header\n" +
			"                 (use '-' for standard output)\n" +
			"  -h, --help     Print help"
		common.Render("uudecode", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	pos := flags.Positional
	var infile string
	var reader io.Reader

	if len(pos) == 0 {
		infile = "-"
		reader = stdin
	} else if len(pos) == 1 {
		infile = pos[0]
		if infile == "-" {
			reader = stdin
		} else {
			absInPath := infile
			if !filepath.IsAbs(absInPath) {
				absInPath = filepath.Join(cwd, infile)
			}
			file, err := os.Open(absInPath)
			if err != nil {
				if jsonMode {
					common.RenderError("uudecode", 1, "OPEN_ERROR", err.Error(), true, stderr)
				} else {
					fmt.Fprintf(stderr, "uudecode: %v\n", err)
				}
				return 1
			}
			defer file.Close()
			reader = file
		}
	} else {
		if jsonMode {
			common.RenderError("uudecode", 1, "BAD_USAGE", "too many arguments", true, stderr)
		} else {
			fmt.Fprintln(stderr, "uudecode: too many arguments")
		}
		return 1
	}

	// Parse stream
	scanner := bufio.NewScanner(reader)
	var headerLine string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			if strings.HasPrefix(line, "begin ") || strings.HasPrefix(line, "begin-base64 ") {
				headerLine = line
				break
			}
		}
	}

	if headerLine == "" {
		if jsonMode {
			common.RenderError("uudecode", 1, "PARSE_ERROR", "missing header line", true, stderr)
		} else {
			fmt.Fprintln(stderr, "uudecode: missing header line")
		}
		return 1
	}

	fields := strings.Fields(headerLine)
	if len(fields) < 3 {
		if jsonMode {
			common.RenderError("uudecode", 1, "PARSE_ERROR", "invalid header format", true, stderr)
		} else {
			fmt.Fprintln(stderr, "uudecode: invalid header format")
		}
		return 1
	}

	isBase64 := fields[0] == "begin-base64"
	modeStr := fields[1]
	headerRemoteName := fields[2]

	// Determine output destination
	destFileArg := flags.Get("o")
	var destPath string
	var writeToStdout bool

	if destFileArg != "" {
		if destFileArg == "-" {
			writeToStdout = true
		} else {
			destPath = destFileArg
		}
	} else {
		if headerRemoteName == "-" {
			writeToStdout = true
		} else {
			destPath = headerRemoteName
		}
	}

	var decodedData []byte

	if isBase64 {
		// Base64 decoding
		var b64Buf strings.Builder
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "====" || line == "end" {
				break
			}
			b64Buf.WriteString(line)
		}
		data, err := base64.StdEncoding.DecodeString(b64Buf.String())
		if err != nil {
			// Try relaxed decoding (ignoring invalid chars)
			data, err = base64.StdEncoding.DecodeString(strings.Map(func(r rune) rune {
				if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
					return r
				}
				return -1
			}, b64Buf.String()))
			if err != nil {
				if jsonMode {
					common.RenderError("uudecode", 1, "DECODE_ERROR", err.Error(), true, stderr)
				} else {
					fmt.Fprintf(stderr, "uudecode: base64 decode error: %v\n", err)
				}
				return 1
			}
		}
		decodedData = data
	} else {
		// Traditional uuencode decoding
		var buf bytes.Buffer
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			if line == "end" {
				break
			}
			// Read length character
			lenChar := line[0]
			if lenChar == '`' || lenChar == ' ' {
				break
			}
			decLen := int(lenChar-32) & 0x3F
			if decLen <= 0 {
				break
			}

			// Parse 4-character chunks
			decodedLine := make([]byte, 0, decLen)
			chunkStart := 1
			for chunkStart < len(line) && len(decodedLine) < decLen {
				chunkEnd := chunkStart + 4
				if chunkEnd > len(line) {
					chunkEnd = len(line)
				}
				chunk := line[chunkStart:chunkEnd]
				if len(chunk) < 2 {
					break
				}
				var c1, c2, c3, c4 byte = 32, 32, 32, 32
				c1 = chunk[0]
				c2 = chunk[1]
				if len(chunk) > 2 {
					c3 = chunk[2]
				}
				if len(chunk) > 3 {
					c4 = chunk[3]
				}

				v1 := (c1 - 32) & 0x3F
				v2 := (c2 - 32) & 0x3F
				v3 := (c3 - 32) & 0x3F
				v4 := (c4 - 32) & 0x3F

				b1 := (v1 << 2) | (v2 >> 4)
				b2 := ((v2 & 15) << 4) | (v3 >> 2)
				b3 := ((v3 & 3) << 6) | v4

				decodedLine = append(decodedLine, b1)
				if len(decodedLine) < decLen {
					decodedLine = append(decodedLine, b2)
				}
				if len(decodedLine) < decLen {
					decodedLine = append(decodedLine, b3)
				}
				chunkStart += 4
			}
			buf.Write(decodedLine)
		}
		decodedData = buf.Bytes()
	}

	var bytesWritten int64
	if writeToStdout {
		n, err := stdout.Write(decodedData)
		if err != nil {
			if jsonMode {
				common.RenderError("uudecode", 1, "WRITE_ERROR", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "uudecode: %v\n", err)
			}
			return 1
		}
		bytesWritten = int64(n)
	} else {
		// Output to file
		var absDest string
		if filepath.IsAbs(destPath) {
			absDest = destPath
		} else {
			absDest = filepath.Join(cwd, destPath)
		}

		// Ensure directory exists
		dir := filepath.Dir(absDest)
		if err := os.MkdirAll(dir, 0755); err != nil {
			if jsonMode {
				common.RenderError("uudecode", 1, "MKDIR_ERROR", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "uudecode: %v\n", err)
			}
			return 1
		}

		// Parse permissions
		permMode := os.FileMode(0644)
		if modeVal, err := strconv.ParseUint(modeStr, 8, 32); err == nil {
			permMode = os.FileMode(modeVal)
		}

		if err := os.WriteFile(absDest, decodedData, permMode); err != nil {
			if jsonMode {
				common.RenderError("uudecode", 1, "WRITE_ERROR", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "uudecode: %v\n", err)
			}
			return 1
		}
		bytesWritten = int64(len(decodedData))
	}

	if jsonMode {
		common.Render("uudecode", UudecodeResult{
			Source:       infile,
			Destination:  destFileArg,
			BytesDecoded: bytesWritten,
		}, true, stdout, nil)
	}

	return 0
}
