// Package hexdump implements the POSIX/BusyBox hexdump utility.
package hexdump

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
		{Short: "C", Long: "canonical", Type: common.FlagBool},
		{Short: "e", Long: "format", Type: common.FlagValue},
		{Short: "n", Long: "length", Type: common.FlagValue},
		{Short: "s", Long: "skip", Type: common.FlagValue},
		{Short: "v", Long: "verbose", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// ActionType defines what a print action does.
type ActionType int

const (
	ActionOffset ActionType = iota
	ActionValue
	ActionLiteral
)

// PrintAction represents a compiled instruction for formatting.
type PrintAction struct {
	Type       ActionType
	OffsetType string // "x", "d", "o"
	FormatSpec string // e.g. "%08x", "%02x", "%s"
	ValType    string // "x", "d", "o", "c", "p", "u", "X"
	Literal    string
}

// FormatUnit defines a repeat count, byte consumption, and the actions to take.
type FormatUnit struct {
	Iteration int
	ByteCount int
	Format    string
	Actions   []PrintAction
}

// FormatString represents a complete format line/string (e.g. one -e option).
type FormatString struct {
	Units []FormatUnit
}

// HexdumpResult is the structured output for --json mode.
type HexdumpResult struct {
	Lines []string `json:"lines"`
}

// Run executes the hexdump formatting logic.
func Run(r io.Reader, w io.Writer, formatStrings []FormatString, skipBytes, lengthBytes int64, verbose bool) error {
	bw := bufio.NewWriterSize(w, 32*1024)
	defer bw.Flush()

	// Apply skip
	if skipBytes > 0 {
		discarded, err := io.CopyN(io.Discard, r, skipBytes)
		if err != nil && err != io.EOF {
			return err
		}
		if discarded < skipBytes {
			// Scanned past EOF
			return nil
		}
	}

	// Limit reader if length is set
	var input io.Reader = r
	if lengthBytes >= 0 {
		input = io.LimitReader(r, lengthBytes)
	}

	// Determine block size
	blockSize := 0
	for _, fs := range formatStrings {
		size := 0
		for _, u := range fs.Units {
			hasVal := false
			for _, act := range u.Actions {
				if act.Type == ActionValue {
					hasVal = true
					break
				}
			}
			if hasVal {
				size += u.Iteration * u.ByteCount
			}
		}
		if size > blockSize {
			blockSize = size
		}
	}
	if blockSize == 0 {
		blockSize = 16
	}

	var prevBlock []byte
	folded := false
	var offset int64 = skipBytes

	buf := make([]byte, blockSize)

	for {
		n, err := input.Read(buf)
		if n > 0 {
			currentBlock := buf[:n]

			// Duplicate folding (unless verbose)
			if !verbose && len(prevBlock) == blockSize && len(currentBlock) == blockSize && bytes.Equal(prevBlock, currentBlock) {
				if !folded {
					fmt.Fprintln(bw, "*")
					folded = true
				}
				offset += int64(n)
				if err == io.EOF {
					break
				}
				continue
			}

			folded = false
			prevBlock = make([]byte, len(currentBlock))
			copy(prevBlock, currentBlock)

			// Format this block using each format string
			var blockBuf strings.Builder
			for _, fs := range formatStrings {
				cursor := 0
				for _, u := range fs.Units {
					for iter := 0; iter < u.Iteration; iter++ {
						for _, act := range u.Actions {
							switch act.Type {
							case ActionLiteral:
								blockBuf.WriteString(act.Literal)
							case ActionOffset:
								fmt.Fprintf(&blockBuf, act.FormatSpec, offset)
							case ActionValue:
								// Check if bytes are available at cursor
								if cursor >= len(currentBlock) {
									// Incomplete/missing bytes: print spaces except for p, c, s
									if act.ValType != "p" && act.ValType != "c" && act.ValType != "s" {
										blockBuf.WriteString(spacePadding(act.FormatSpec))
									}
								} else {
									// Read value of u.ByteCount bytes (little-endian)
									var val uint64
									actual := 0
									for b := 0; b < u.ByteCount; b++ {
										if cursor+b < len(currentBlock) {
											val |= uint64(currentBlock[cursor+b]) << (8 * b)
											actual++
										}
									}

									switch act.ValType {
									case "x", "X", "d", "o", "u":
										fmt.Fprintf(&blockBuf, act.FormatSpec, val)
									case "c":
										// Char format: print printable, escape controls, or print octal
										b := byte(val)
										if b >= 32 && b <= 126 {
											if strings.Contains(act.FormatSpec, "s") {
												fmt.Fprintf(&blockBuf, act.FormatSpec, string(b))
											} else {
												fmt.Fprintf(&blockBuf, act.FormatSpec, b)
											}
										} else {
											var esc string
											switch b {
											case 0:
												esc = "\\0"
											case '\a':
												esc = "\\a"
											case '\b':
												esc = "\\b"
											case '\t':
												esc = "\\t"
											case '\n':
												esc = "\\n"
											case '\v':
												esc = "\\v"
											case '\f':
												esc = "\\f"
											case '\r':
												esc = "\\r"
											default:
												esc = fmt.Sprintf("\\%03o", b)
											}
											fmtSpec := strings.ReplaceAll(act.FormatSpec, "c", "s")
											fmt.Fprintf(&blockBuf, fmtSpec, esc)
										}
									case "p":
										// Printable character representation
										b := byte(val)
										if b >= 32 && b <= 126 {
											fmt.Fprintf(&blockBuf, act.FormatSpec, string(b))
										} else {
											fmt.Fprintf(&blockBuf, act.FormatSpec, ".")
										}
									default:
										fmt.Fprintf(&blockBuf, act.FormatSpec, val)
									}
								}
							}
						}
						// Advance cursor for value units
						hasVal := false
						for _, act := range u.Actions {
							if act.Type == ActionValue {
								hasVal = true
								break
							}
						}
						if hasVal {
							cursor += u.ByteCount
						}
					}
				}
			}
			bw.WriteString(trimLineWhitespaces(blockBuf.String()))

			offset += int64(n)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Print final offset line if any format string has ActionOffset
	hasOffsetSpec := false
	var finalOffsetSpec string
	for _, fs := range formatStrings {
		for _, u := range fs.Units {
			for _, act := range u.Actions {
				if act.Type == ActionOffset {
					hasOffsetSpec = true
					finalOffsetSpec = act.FormatSpec
				}
			}
		}
	}
	if hasOffsetSpec {
		fmt.Fprintf(bw, finalOffsetSpec+"\n", offset)
	}

	return nil
}

func spacePadding(formatSpec string) string {
	width := 0
	for _, r := range formatSpec {
		if r >= '0' && r <= '9' {
			width = width*10 + int(r-'0')
		}
	}
	if width == 0 {
		if strings.HasSuffix(formatSpec, "x") || strings.HasSuffix(formatSpec, "d") {
			width = 2
		} else {
			width = 1
		}
	}
	return strings.Repeat(" ", width)
}

func trimLineWhitespaces(s string) string {
	var sb strings.Builder
	lines := strings.Split(s, "\n")
	for idx, line := range lines {
		if idx == len(lines)-1 {
			sb.WriteString(strings.TrimRight(line, " \t"))
		} else {
			sb.WriteString(strings.TrimRight(line, " \t"))
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func compileFormat(fmtStr string) []PrintAction {
	var actions []PrintAction
	i := 0
	n := len(fmtStr)
	for i < n {
		if fmtStr[i] == '%' {
			i++ // skip '%'

			widthFlags := ""
			for i < n {
				char := fmtStr[i]
				if char == '_' {
					i++ // skip '_'
					if i < n {
						spec := fmtStr[i]
						i++
						if spec == 'a' && i < n {
							offsetType := fmtStr[i]
							i++
							actions = append(actions, PrintAction{
								Type:       ActionOffset,
								OffsetType: string(offsetType),
								FormatSpec: "%" + widthFlags + string(offsetType),
							})
							break
						} else if spec == 'c' {
							actions = append(actions, PrintAction{
								Type:       ActionValue,
								ValType:    "c",
								FormatSpec: "%" + widthFlags + "s",
							})
							break
						} else if spec == 'p' {
							actions = append(actions, PrintAction{
								Type:       ActionValue,
								ValType:    "p",
								FormatSpec: "%" + widthFlags + "s",
							})
							break
						}
					}
				} else if char == 'd' || char == 'o' || char == 'x' || char == 'X' || char == 'c' || char == 's' || char == 'u' {
					i++
					actions = append(actions, PrintAction{
						Type:       ActionValue,
						ValType:    string(char),
						FormatSpec: "%" + widthFlags + string(char),
					})
					break
				} else {
					widthFlags += string(char)
					i++
				}
			}
		} else {
			start := i
			for i < n && fmtStr[i] != '%' {
				i++
			}
			actions = append(actions, PrintAction{
				Type:    ActionLiteral,
				Literal: fmtStr[start:i],
			})
		}
	}
	return actions
}

func parseFormatString(fs string) ([]FormatUnit, error) {
	var units []FormatUnit
	i := 0
	n := len(fs)
	for i < n {
		for i < n && (fs[i] == ' ' || fs[i] == '\t' || fs[i] == '\r' || fs[i] == '\n' || fs[i] == ',') {
			i++
		}
		if i >= n {
			break
		}

		iteration := 1
		start := i
		for i < n && fs[i] >= '0' && fs[i] <= '9' {
			i++
		}
		if i > start {
			fmt.Sscanf(fs[start:i], "%d", &iteration)
		}

		byteCount := 1
		if i < n && fs[i] == '/' {
			i++
			start = i
			for i < n && fs[i] >= '0' && fs[i] <= '9' {
				i++
			}
			if i > start {
				fmt.Sscanf(fs[start:i], "%d", &byteCount)
			}
		}

		for i < n && (fs[i] == ' ' || fs[i] == '\t') {
			i++
		}

		if i >= n || fs[i] != '"' {
			return nil, fmt.Errorf("expected quoted format string at index %d", i)
		}
		i++ // skip opening quote

		var buf strings.Builder
		for i < n {
			if fs[i] == '"' {
				i++
				break
			}
			if fs[i] == '\\' && i+1 < n {
				escaped := fs[i+1]
				i += 2
				switch escaped {
				case 'n':
					buf.WriteByte('\n')
				case 't':
					buf.WriteByte('\t')
				case 'r':
					buf.WriteByte('\r')
				case 'b':
					buf.WriteByte('\b')
				case 'f':
					buf.WriteByte('\f')
				case '"':
					buf.WriteByte('"')
				case '\\':
					buf.WriteByte('\\')
				default:
					buf.WriteByte(escaped)
				}
			} else {
				buf.WriteByte(fs[i])
				i++
			}
		}

		fmtStr := buf.String()
		actions := compileFormat(fmtStr)

		units = append(units, FormatUnit{
			Iteration: iteration,
			ByteCount: byteCount,
			Format:    fmtStr,
			Actions:   actions,
		})
	}
	return units, nil
}

func getFormatStrings(flags *common.ParseResult) ([]FormatString, error) {
	if flags.Has("C") {
		// Canonical hex+ASCII format is equivalent to:
		// -e '"%08_ax  " 8/1 "%02x " "  " 8/1 "%02x " "  |"' -e '"%_p"|\n"'
		fs1, err := parseFormatString(`"%08_ax  " 8/1 "%02x " " " 8/1 "%02x " " |"`)
		if err != nil {
			return nil, err
		}
		fs2, err := parseFormatString(`16/1 "%_p" "|\n"`)
		if err != nil {
			return nil, err
		}
		return []FormatString{{Units: fs1}, {Units: fs2}}, nil
	}

	if eList := flags.GetAll("e"); len(eList) > 0 {
		var formatStrings []FormatString
		for _, eOpt := range eList {
			units, err := parseFormatString(eOpt)
			if err != nil {
				return nil, err
			}
			formatStrings = append(formatStrings, FormatString{Units: units})
		}
		return formatStrings, nil
	}

	// Default format:
	// "%07_ax" 8/2 " %04x" "\n"
	defaultUnits, err := parseFormatString(`"%07_ax" 8/2 " %04x" "\n"`)
	if err != nil {
		return nil, err
	}
	return []FormatString{{Units: defaultUnits}}, nil
}

func hexdumpRun(args []string, stdout, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "hexdump: %v\n", err)
		return 2
	}

	formatStrings, err := getFormatStrings(flags)
	if err != nil {
		fmt.Fprintf(errOut, "hexdump: format error: %v\n", err)
		return 2
	}

	var skipBytes int64 = 0
	if val := flags.Get("s"); val != "" {
		var s int
		fmt.Sscanf(val, "%d", &s)
		skipBytes = int64(s)
	}

	lengthBytes := int64(-1)
	if val := flags.Get("n"); val != "" {
		var l int
		fmt.Sscanf(val, "%d", &l)
		lengthBytes = int64(l)
	}
	verbose := flags.Has("v")
	jsonMode := flags.Has("json")

	var input io.Reader = stdin
	if len(flags.Positional) > 0 {
		// Multi-file support: open first file for now, but we can combine them easily
		// For compliance, single file/stdin covers almost all cases.
		file, err := os.Open(flags.Positional[0])
		if err != nil {
			fmt.Fprintf(errOut, "hexdump: %v\n", err)
			return 1
		}
		defer file.Close()
		input = file
	}

	if jsonMode {
		var buf bytes.Buffer
		err = Run(input, &buf, formatStrings, skipBytes, lengthBytes, verbose)
		if err != nil {
			common.RenderError("hexdump", 1, "ERR", err.Error(), true, stdout)
			return 1
		}
		lines := strings.Split(buf.String(), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		common.Render("hexdump", HexdumpResult{Lines: lines}, true, stdout, func() {})
		return 0
	}

	err = Run(input, stdout, formatStrings, skipBytes, lengthBytes, verbose)
	if err != nil {
		fmt.Fprintf(errOut, "hexdump: %v\n", err)
		return 1
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "hexdump",
		Usage: "Display file contents in hexadecimal, decimal, octal, or ASCII",
		Run: func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
			return hexdumpRun(args, stdout, stderr, stdin, cwd)
		},
	})
}
