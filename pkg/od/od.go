// Package od implements the POSIX od utility — dump files in octal and other formats.
package od

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "b", Long: "", Type: common.FlagBool},
		{Short: "c", Long: "", Type: common.FlagBool},
		{Short: "f", Long: "", Type: common.FlagBool},
		{Short: "x", Long: "", Type: common.FlagBool},
		{Short: "t", Long: "", Type: common.FlagValue},
		{Short: "N", Long: "", Type: common.FlagValue},
		{Short: "A", Long: "", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
		{Long: "traditional", Type: common.FlagBool},
	},
}

// OdResult holds structured od output.
type OdResult struct {
	Records []string `json:"records"`
}

// Run reads from r and produces an od dump. args contains flags + optional filename.
func Run(args []string, r io.Reader, w io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "od: %v\n", err)
		return 1
	}

	var in io.Reader = r
	if len(flags.Positional) > 0 {
		f, ferr := os.Open(flags.Positional[0])
		if ferr != nil {
			fmt.Fprintf(os.Stderr, "od: %s: %v\n", flags.Positional[0], ferr)
			return 1
		}
		defer f.Close()
		in = f
	}

	maxBytes := int64(-1)
	if ns := flags.Get("N"); ns != "" {
		n, _ := strconv.ParseInt(ns, 10, 64)
		if n > 0 {
			maxBytes = n
		}
	}

	jsonMode := flags.Has("json")

	// Determine output format. -t takes precedence if present.
	if t := flags.Get("t"); t != "" {
		switch t {
		case "x1", "x":
			return dumpHexBytes(in, w, maxBytes, jsonMode)
		case "x2":
			return dumpHex(in, w, maxBytes, jsonMode)
		case "o1", "b":
			return dumpBytes(in, w, maxBytes, jsonMode)
		case "o2", "o":
			return dumpShorts(in, w, maxBytes, jsonMode)
		case "c":
			return dumpChars(in, w, maxBytes, jsonMode)
		case "f":
			return dumpFloat(in, w, maxBytes, jsonMode)
		}
	}

	switch {
	case flags.Has("b"):
		return dumpBytes(in, w, maxBytes, jsonMode)
	case flags.Has("c"):
		return dumpChars(in, w, maxBytes, jsonMode)
	case flags.Has("f"):
		return dumpFloat(in, w, maxBytes, jsonMode)
	case flags.Has("x"):
		return dumpHex(in, w, maxBytes, jsonMode)
	default:
		return dumpShorts(in, w, maxBytes, jsonMode)
	}
}

// dumpBytes outputs 1-byte octal values (od -b).
func dumpBytes(r io.Reader, w io.Writer, maxBytes int64, jsonMode bool) int {
	const perLine = 16
	buf := make([]byte, perLine)
	var offset int64
	var records []string

	emit := func(s string) {
		records = append(records, s)
		if !jsonMode {
			fmt.Fprint(w, s)
		}
	}

	for {
		n, err := io.ReadFull(r, buf)
		if n == 0 && err == io.EOF {
			break
		}
		if maxBytes >= 0 {
			remaining := maxBytes - offset
			if remaining <= 0 {
				break
			}
			if int64(n) > remaining {
				n = int(remaining)
			}
		}

		line := fmt.Sprintf("%07o", offset)
		for i := 0; i < n; i++ {
			line += fmt.Sprintf(" %03o", buf[i])
		}
		line += "\n"
		emit(line)
		offset += int64(n)
		if err == io.EOF {
			break
		}
	}

	// Final offset line
	emit(fmt.Sprintf("%07o\n", offset))

	if jsonMode {
		common.Render("od", OdResult{Records: records}, true, w, func() {})
	}
	return 0
}

// dumpChars outputs character dump (od -c).
func dumpChars(r io.Reader, w io.Writer, maxBytes int64, jsonMode bool) int {
	const perLine = 16
	buf := make([]byte, perLine)
	var offset int64
	var records []string

	emit := func(s string) {
		records = append(records, s)
		if !jsonMode {
			fmt.Fprint(w, s)
		}
	}

	for {
		n, err := io.ReadFull(r, buf)
		if n == 0 && err == io.EOF {
			break
		}
		if maxBytes >= 0 {
			remaining := maxBytes - offset
			if remaining <= 0 {
				break
			}
			if int64(n) > remaining {
				n = int(remaining)
			}
		}

		line := fmt.Sprintf("%07o", offset)
		for i := 0; i < n; i++ {
			line += " " + charEscape(buf[i])
		}
		line += "\n"
		emit(line)
		offset += int64(n)
		if err == io.EOF {
			break
		}
	}

	emit(fmt.Sprintf("%07o\n", offset))

	if jsonMode {
		common.Render("od", OdResult{Records: records}, true, w, func() {})
	}
	return 0
}

// charEscape returns a 3-char representation of a byte for od -c output.
func charEscape(b byte) string {
	switch b {
	case '\n':
		return " \\n"
	case '\t':
		return " \\t"
	case '\r':
		return " \\r"
	case '\f':
		return " \\f"
	case '\b':
		return " \\b"
	case '\a':
		return " \\a"
	case '\v':
		return " \\v"
	case '\\':
		return " \\\\"
	case 0:
		return " \\0"
	}
	if b >= 0x20 && b <= 0x7E {
		return fmt.Sprintf("  %c", b)
	}
	return fmt.Sprintf(" %03o", b)
}

// dumpFloat outputs 4-byte float dump (od -f).
func dumpFloat(r io.Reader, w io.Writer, maxBytes int64, jsonMode bool) int {
	const perLine = 16
	buf := make([]byte, perLine)
	var offset int64
	var records []string

	emit := func(s string) {
		records = append(records, s)
		if !jsonMode {
			fmt.Fprint(w, s)
		}
	}

	for {
		n, err := io.ReadFull(r, buf)
		if n == 0 && err == io.EOF {
			break
		}
		if maxBytes >= 0 {
			remaining := maxBytes - offset
			if remaining <= 0 {
				break
			}
			if int64(n) > remaining {
				n = int(remaining)
			}
		}

		line := fmt.Sprintf("%07o", offset)
		for i := 0; i < n; i += 4 {
			if i+4 <= n {
				val := float32fromBytes(buf[i : i+4])
				line += fmt.Sprintf("% 16.7e", val)
			}
		}
		line += "\n"
		emit(line)
		offset += int64(n)
		if err == io.EOF {
			break
		}
	}

	emit(fmt.Sprintf("%07o\n", offset))

	if jsonMode {
		common.Render("od", OdResult{Records: records}, true, w, func() {})
	}
	return 0
}

// float32fromBytes converts 4 bytes (native endian) to a float32.
func float32fromBytes(b []byte) float32 {
	bits := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return math.Float32frombits(bits)
}

// dumpHexBytes outputs 1-byte hex dump (od -t x1).
func dumpHexBytes(r io.Reader, w io.Writer, maxBytes int64, jsonMode bool) int {
	const perLine = 16
	buf := make([]byte, perLine)
	var offset int64
	var records []string

	emit := func(s string) {
		records = append(records, s)
		if !jsonMode {
			fmt.Fprint(w, s)
		}
	}

	for {
		n, err := io.ReadFull(r, buf)
		if n == 0 && err == io.EOF {
			break
		}
		if maxBytes >= 0 {
			remaining := maxBytes - offset
			if remaining <= 0 {
				break
			}
			if int64(n) > remaining {
				n = int(remaining)
			}
		}

		line := fmt.Sprintf("%07o", offset)
		for i := 0; i < n; i++ {
			line += fmt.Sprintf(" %02x", buf[i])
		}
		line += "\n"
		emit(line)
		offset += int64(n)
		if err == io.EOF {
			break
		}
	}

	emit(fmt.Sprintf("%07o\n", offset))

	if jsonMode {
		common.Render("od", OdResult{Records: records}, true, w, func() {})
	}
	return 0
}

// dumpHex outputs 2-byte hex dump (od -x).
func dumpHex(r io.Reader, w io.Writer, maxBytes int64, jsonMode bool) int {
	const perLine = 16
	buf := make([]byte, perLine)
	var offset int64
	var records []string

	emit := func(s string) {
		records = append(records, s)
		if !jsonMode {
			fmt.Fprint(w, s)
		}
	}

	for {
		n, err := io.ReadFull(r, buf)
		if n == 0 && err == io.EOF {
			break
		}
		if maxBytes >= 0 {
			remaining := maxBytes - offset
			if remaining <= 0 {
				break
			}
			if int64(n) > remaining {
				n = int(remaining)
			}
		}

		line := fmt.Sprintf("%07o", offset)
		for i := 0; i < n; i += 2 {
			if i+1 < n {
				// Two bytes: native-endian 16-bit word
				val := uint16(buf[i]) | uint16(buf[i+1])<<8
				line += fmt.Sprintf(" %04x", val)
			} else {
				// Single trailing byte
				line += fmt.Sprintf(" %04x", uint16(buf[i]))
			}
		}
		line += "\n"
		emit(line)
		offset += int64(n)
		if err == io.EOF {
			break
		}
	}

	emit(fmt.Sprintf("%07o\n", offset))

	if jsonMode {
		common.Render("od", OdResult{Records: records}, true, w, func() {})
	}
	return 0
}

// dumpShorts outputs 2-byte octal shorts (default od).
func dumpShorts(r io.Reader, w io.Writer, maxBytes int64, jsonMode bool) int {
	const perLine = 16
	buf := make([]byte, perLine)
	var offset int64
	var records []string

	emit := func(s string) {
		records = append(records, s)
		if !jsonMode {
			fmt.Fprint(w, s)
		}
	}

	for {
		n, err := io.ReadFull(r, buf)
		if n == 0 && err == io.EOF {
			break
		}
		if maxBytes >= 0 {
			remaining := maxBytes - offset
			if remaining <= 0 {
				break
			}
			if int64(n) > remaining {
				n = int(remaining)
			}
		}

		line := fmt.Sprintf("%07o", offset)
		for i := 0; i < n; i += 2 {
			if i+1 < n {
				val := uint16(buf[i]) | uint16(buf[i+1])<<8
				line += fmt.Sprintf(" %07o", val)
			} else {
				line += fmt.Sprintf(" %07o", uint16(buf[i]))
			}
		}
		line += "\n"
		emit(line)
		offset += int64(n)
		if err == io.EOF {
			break
		}
	}

	emit(fmt.Sprintf("%07o\n", offset))

	if jsonMode {
		common.Render("od", OdResult{Records: records}, true, w, func() {})
	}
	return 0
}

// ---------------------------------------------------------------------------
// CLI glue
// ---------------------------------------------------------------------------

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	return odRun(args, os.Stdin, stdout)
}

// odRun is the testable entry point for the od CLI.
func odRun(args []string, stdin io.Reader, stdout io.Writer) int {
	return Run(args, stdin, stdout)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "od",
		Usage: "od [-bcdox] [-N count] [file...] — dump files in octal and other formats",
		Run:   run,
	})
}
