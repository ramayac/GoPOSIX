// Package xxd implements the POSIX/BusyBox xxd utility.
package xxd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "p", Long: "plain", Type: common.FlagBool},
		{Short: "r", Long: "revert", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// XxdResult is the structured output for --json mode.
type XxdResult struct {
	Lines []string `json:"lines"`
}

// Run executes the core xxd logic.
func Run(r io.Reader, w io.Writer, plainMode, reverseMode bool) error {
	if reverseMode {
		if plainMode {
			return reversePlain(r, w)
		}
		return reverseStandard(r, w)
	}

	if plainMode {
		return dumpPlain(r, w)
	}
	return dumpStandard(r, w)
}

func dumpStandard(r io.Reader, w io.Writer) error {
	bw := bufio.NewWriterSize(w, 32*1024)
	defer bw.Flush()

	buf := make([]byte, 16)
	var offset int64

	for {
		n, err := r.Read(buf)
		if n > 0 {
			// Print offset
			fmt.Fprintf(bw, "%08x: ", offset)
			offset += int64(n)

			// Format hex bytes
			var hexParts []string
			for j := 0; j < 16; j += 2 {
				if j < n {
					if j+1 < n {
						hexParts = append(hexParts, fmt.Sprintf("%02x%02x", buf[j], buf[j+1]))
					} else {
						hexParts = append(hexParts, fmt.Sprintf("%02x", buf[j]))
					}
				}
			}
			hexCol := strings.Join(hexParts, " ")
			// Pad hexCol to 39 characters
			if len(hexCol) < 39 {
				hexCol += strings.Repeat(" ", 39-len(hexCol))
			}

			// Format ASCII
			var asciiBuilder strings.Builder
			for j := 0; j < n; j++ {
				b := buf[j]
				if b >= 32 && b <= 126 {
					asciiBuilder.WriteByte(b)
				} else {
					asciiBuilder.WriteByte('.')
				}
			}

			fmt.Fprintf(bw, "%s  %s\n", hexCol, asciiBuilder.String())
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func dumpPlain(r io.Reader, w io.Writer) error {
	bw := bufio.NewWriterSize(w, 32*1024)
	defer bw.Flush()

	buf := make([]byte, 30) // 30 bytes = 60 hex chars per line
	for {
		n, err := r.Read(buf)
		if n > 0 {
			for j := 0; j < n; j++ {
				fmt.Fprintf(bw, "%02x", buf[j])
			}
			fmt.Fprintln(bw)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func reversePlain(r io.Reader, w io.Writer) error {
	bw := bufio.NewWriterSize(w, 32*1024)
	defer bw.Flush()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		i := 0
		n := len(line)
		badSeqCount := 0
		truncated := false

		for i < n {
			// Find first hex digit
			var d1 byte
			foundD1 := false

			for i < n {
				char := line[i]
				i++
				if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
					badSeqCount = 0
					continue
				}

				isHex, val := hexVal(char)
				if isHex {
					d1 = val
					foundD1 = true
					badSeqCount = 0
					break
				} else {
					badSeqCount++
					if badSeqCount >= 2 {
						truncated = true
						break
					}
				}
			}

			if truncated || !foundD1 {
				break
			}

			// Find second hex digit
			var d2 byte
			foundD2 := false

			for i < n {
				char := line[i]
				i++
				if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
					badSeqCount = 0
					continue
				}

				isHex, val := hexVal(char)
				if isHex {
					d2 = val
					foundD2 = true
					badSeqCount = 0
					break
				} else {
					// Second char is bad: discard d1
					badSeqCount++
					if badSeqCount >= 2 {
						truncated = true
					}
					break // break inner loop, skip this byte, d1 is discarded
				}
			}

			if truncated {
				break
			}

			if foundD2 {
				bw.WriteByte((d1 << 4) | d2)
			}
		}
	}
	return scanner.Err()
}

func reverseStandard(r io.Reader, w io.Writer) error {
	bw := bufio.NewWriterSize(w, 32*1024)
	defer bw.Flush()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip leading whitespace
		trimmed := strings.TrimLeft(line, " \t\r\n")
		if trimmed == "" {
			continue
		}

		colonIdx := strings.Index(trimmed, ":")
		if colonIdx == -1 {
			continue // invalid line, skip
		}

		hexContent := trimmed[colonIdx+1:]
		if doubleSpaceIdx := strings.Index(hexContent, "  "); doubleSpaceIdx != -1 {
			hexContent = hexContent[:doubleSpaceIdx]
		}

		// Now parse hex bytes from hexContent
		i := 0
		n := len(hexContent)
		badSeqCount := 0
		truncated := false

		for i < n {
			var d1 byte
			foundD1 := false

			for i < n {
				char := hexContent[i]
				i++
				if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
					badSeqCount = 0
					continue
				}

				isHex, val := hexVal(char)
				if isHex {
					d1 = val
					foundD1 = true
					badSeqCount = 0
					break
				} else {
					badSeqCount++
					if badSeqCount >= 2 {
						truncated = true
						break
					}
				}
			}

			if truncated || !foundD1 {
				break
			}

			var d2 byte
			foundD2 := false

			for i < n {
				char := hexContent[i]
				i++
				if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
					badSeqCount = 0
					continue
				}

				isHex, val := hexVal(char)
				if isHex {
					d2 = val
					foundD2 = true
					badSeqCount = 0
					break
				} else {
					badSeqCount++
					if badSeqCount >= 2 {
						truncated = true
					}
					break
				}
			}

			if truncated {
				break
			}

			if foundD2 {
				bw.WriteByte((d1 << 4) | d2)
			}
		}
	}
	return scanner.Err()
}

func hexVal(char byte) (bool, byte) {
	if char >= '0' && char <= '9' {
		return true, char - '0'
	}
	if char >= 'a' && char <= 'f' {
		return true, char - 'a' + 10
	}
	if char >= 'A' && char <= 'F' {
		return true, char - 'A' + 10
	}
	return false, 0
}

func xxdRun(args []string, stdout, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "xxd: %v\n", err)
		return 2
	}

	plainMode := flags.Has("p")
	reverseMode := flags.Has("r")
	jsonMode := flags.Has("json")

	var input io.Reader = stdin
	if len(flags.Positional) > 0 {
		file, err := os.Open(flags.Positional[0])
		if err != nil {
			fmt.Fprintf(errOut, "xxd: %v\n", err)
			return 1
		}
		defer file.Close()
		input = file
	}

	if jsonMode {
		var buf bytes.Buffer
		err = Run(input, &buf, plainMode, reverseMode)
		if err != nil {
			common.RenderError("xxd", 1, "ERR", err.Error(), true, stdout)
			return 1
		}
		lines := strings.Split(buf.String(), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		common.Render("xxd", XxdResult{Lines: lines}, true, stdout, func() {})
		return 0
	}

	err = Run(input, stdout, plainMode, reverseMode)
	if err != nil {
		fmt.Fprintf(errOut, "xxd: %v\n", err)
		return 1
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "xxd",
		Usage: "Make a hexdump or do the reverse",
		Run: func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
			return xxdRun(args, stdout, stderr, stdin, cwd)
		},
	})
}
