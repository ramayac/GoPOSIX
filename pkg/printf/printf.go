// Package printf implements the POSIX printf utility.
//
// printf formats and prints data according to a format string. It supports
// standard POSIX conversions (d, i, o, u, x, X, f, F, e, E, g, G, c, s, %),
// width/precision via * (from arguments), length modifiers (h, hh, l, ll, j, z, t, L),
// backslash escapes (\n, \t, \\, \0NNN, \xNN), and the \c escape to stop output.
// Only --json is accepted as a flag; all other arguments (including those
// starting with -) are positional arguments to the format string.
package printf

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// PrintfResult is the structured result for --json mode.
type PrintfResult struct {
	Output string `json:"output"`
}

// runState holds mutable formatting state.
type runState struct {
	format string   // escape-processed format string
	args   []string // positional arguments
	out    *strings.Builder
	argIdx int  // current argument index (cycles)
	hadErr bool // any conversion error occurred
	pos    int  // current position in format string (0-based byte index)
}

// Format interprets a POSIX printf format string with arguments.
// It processes the format string, reusing it as necessary to consume all args.
// Returns the combined output (stdout + interleaved stderr), and whether any
// conversion error occurred. Stderr messages are interleaved in the output
// as they occur (following POSIX printf behavior).
func Format(format string, args []string) (string, bool) {
	state := &runState{
		format: processEscapes(format),
		args:   args,
		out:    &strings.Builder{},
	}

	// POSIX: reuse format string as often as needed to satisfy all args.
	for {
		if len(state.format) == 0 {
			break
		}
		if !state.processOnePass() {
			break
		}
		// If all args are consumed, stop.
		if state.argIdx >= len(state.args) {
			break
		}
		// Reset format position and continue.
		state.pos = 0
	}

	return state.out.String(), state.hadErr
}

// processOnePass makes one pass through the format string, consuming args.
// Returns false if a fatal error occurred (e.g., invalid format char).
func (state *runState) processOnePass() bool {
	for state.pos < len(state.format) {
		// Check for \c (stop-output marker) at the Format level.
		if state.format[state.pos] == '\\' && state.pos+1 < len(state.format) && state.format[state.pos+1] == 'c' {
			state.pos += 2
			return false // stop format reuse
		}

		c := state.format[state.pos]
		state.pos++

		if c != '%' {
			state.out.WriteByte(c)
			continue
		}

		// Handle bare '%' at end of format
		if state.pos >= len(state.format) {
			state.formatError("invalid format", "%")
			return false
		}

		// Handle %% — literal percent
		if state.format[state.pos] == '%' {
			state.out.WriteByte('%')
			state.pos++
			continue
		}

		// --- Parse conversion specification ---
		// Flags: -, +, ' ', 0, #
		flags := ""
		for state.pos < len(state.format) {
			b := state.format[state.pos]
			if b == '-' || b == '+' || b == ' ' || b == '0' || b == '#' {
				flags += string(b)
				state.pos++
			} else {
				break
			}
		}

		// Field width: decimal digits or *
		width := -1
		if state.pos < len(state.format) && state.format[state.pos] == '*' {
			state.pos++
			arg := state.nextArg()
			n, err := strconv.Atoi(arg)
			if err != nil {
				width = 0
			} else {
				width = n
			}
		} else {
			for state.pos < len(state.format) && state.format[state.pos] >= '0' && state.format[state.pos] <= '9' {
				if width < 0 {
					width = 0
				}
				width = width*10 + int(state.format[state.pos]-'0')
				state.pos++
			}
		}

		// Precision: . followed by decimal digits or *
		precision := -1
		hasPrecision := false
		if state.pos < len(state.format) && state.format[state.pos] == '.' {
			hasPrecision = true
			state.pos++
			if state.pos < len(state.format) && state.format[state.pos] == '*' {
				state.pos++
				arg := state.nextArg()
				n, err := strconv.Atoi(arg)
				if err != nil {
					precision = 0
				} else {
					precision = n
				}
			} else {
				precision = 0
				for state.pos < len(state.format) && state.format[state.pos] >= '0' && state.format[state.pos] <= '9' {
					precision = precision*10 + int(state.format[state.pos]-'0')
					state.pos++
				}
			}
		}

		// Length modifier: hh, h, l, ll, j, z, t, L
		if state.pos < len(state.format) {
			switch state.format[state.pos] {
			case 'h':
				state.pos++
				if state.pos < len(state.format) && state.format[state.pos] == 'h' {
					state.pos++ // hh
				}
			case 'l':
				state.pos++
				if state.pos < len(state.format) && state.format[state.pos] == 'l' {
					state.pos++ // ll
				}
			case 'j', 'z', 't', 'L':
				state.pos++
			}
		}

		// Conversion specifier
		if state.pos >= len(state.format) {
			state.formatError("invalid format", "%"+flags)
			return false
		}
		conv := state.format[state.pos]
		state.pos++

		switch conv {
		case 'd', 'i':
			state.doIntConv(conv, flags, width, precision, hasPrecision)
		case 'o', 'u', 'x', 'X':
			state.doUintConv(conv, flags, width, precision, hasPrecision)
		case 'f', 'F', 'e', 'E', 'g', 'G':
			state.doFloatConv(conv, flags, width, precision, hasPrecision)
		case 's':
			state.doStringConv(flags, width, precision, hasPrecision)
		case 'c':
			state.doCharConv()
		case '%':
			state.out.WriteByte('%')
		case 'b':
			state.doBConv()
		default:
			state.formatError("invalid format", "%"+string(conv))
			return false
		}
	}
	return true
}

func (s *runState) nextArg() string {
	if len(s.args) == 0 {
		return ""
	}
	arg := s.args[s.argIdx%len(s.args)]
	s.argIdx++
	return arg
}

func (s *runState) formatError(msg, detail string) {
	// Interleave error message in the output (separated by newline)
	if s.out.Len() > 0 && s.out.String()[s.out.Len()-1] != '\n' {
		s.out.WriteByte('\n')
	}
	s.out.WriteString("printf: ")
	// Format depends on error type:
	// - "invalid number" → printf: invalid number 'arg'
	// - "invalid format" → printf: %: invalid format
	if msg == "invalid format" {
		s.out.WriteString(detail)
		s.out.WriteString(": ")
		s.out.WriteString(msg)
	} else {
		s.out.WriteString(msg)
		s.out.WriteString(" '")
		s.out.WriteString(detail)
		s.out.WriteString("'")
	}
	s.out.WriteByte('\n')
	s.hadErr = true
}

func (s *runState) doIntConv(conv byte, flags string, width, precision int, hasPrecision bool) {
	arg := s.nextArg()

	// Handle character constants like '"x' or '\n'
	if len(arg) >= 2 && arg[0] == '\'' && arg[len(arg)-1] == '\'' {
		arg = arg[1 : len(arg)-1]
		if len(arg) >= 2 && arg[0] == '\\' {
			arg = string(unescapeChar(arg[1:]))
		}
		if len(arg) > 0 {
			s.formatInt(int64(arg[0]), conv, flags, width, precision, hasPrecision)
			return
		}
	}

	// Also handle single-quote prefix: '"x → char 'x', "'y → char 'y'
	if len(arg) >= 1 && arg[0] == '"' {
		// Parse as character constant
		val := int64(arg[1])
		if len(arg) > 2 && arg[1] == '\\' {
			val = int64(unescapeChar(arg[2:]))
		}
		s.formatInt(val, conv, flags, width, precision, hasPrecision)
		return
	}

	val, remaining := parseInt(arg)
	if remaining != "" || arg == "" && remaining == "" {
		// Check if it's a single-quote-prefix char
		if len(arg) >= 2 && arg[0] == '\'' {
			s.formatInt(int64(arg[1]), conv, flags, width, precision, hasPrecision)
			return
		}
	}
	if remaining == arg {
		// Entirely unparseable
		s.formatError("invalid number", arg)
		// Output 0 as replacement
		s.formatInt(0, conv, flags, width, precision, hasPrecision)
		return
	}
	if remaining != "" {
		s.formatError("invalid number", arg)
	}
	s.formatInt(val, conv, flags, width, precision, hasPrecision)
}

func (s *runState) doUintConv(conv byte, flags string, width, precision int, hasPrecision bool) {
	arg := s.nextArg()
	val, remaining := parseUint(arg)
	if remaining == arg {
		s.formatError("invalid number", arg)
		s.formatUint(0, conv, flags, width, precision, hasPrecision)
		return
	}
	if remaining != "" {
		s.formatError("invalid number", arg)
	}
	s.formatUint(val, conv, flags, width, precision, hasPrecision)
}

func (s *runState) doFloatConv(conv byte, flags string, width, precision int, hasPrecision bool) {
	arg := s.nextArg()
	val, remaining := parseFloat(arg)
	if remaining == arg {
		s.formatError("invalid number", arg)
		s.formatFloat(0.0, conv, flags, width, precision, hasPrecision)
		return
	}
	if remaining != "" {
		s.formatError("invalid number", arg)
	}
	s.formatFloat(val, conv, flags, width, precision, hasPrecision)
}

func (s *runState) doStringConv(flags string, width, precision int, hasPrecision bool) {
	arg := s.nextArg()
	s.formatString(arg, flags, width, precision, hasPrecision)
}

func (s *runState) doCharConv() {
	arg := s.nextArg()
	if len(arg) > 0 {
		s.out.WriteByte(arg[0])
	}
}

func (s *runState) doBConv() {
	arg := s.nextArg()
	// Process escape sequences in the argument
	processed := processEscapesForB(arg)
	s.out.WriteString(processed)
}

// processEscapesForB handles escape sequences for %b (similar to echo -e but
// without octal/hex expansion of the full argument — just \ sequences).
func processEscapesForB(s string) string {
	var buf strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case 'r':
				buf.WriteByte('\r')
			case '\\':
				buf.WriteByte('\\')
			case 'a':
				buf.WriteByte('\a')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case 'v':
				buf.WriteByte('\v')
			case '0':
				// Octal escape: \0NNN
				end := i + 1
				for end < len(s) && end < i+4 && s[end] >= '0' && s[end] <= '7' {
					end++
				}
				if end > i+1 {
					val, err := strconv.ParseUint(s[i+1:end], 8, 8)
					if err == nil {
						buf.WriteByte(byte(val))
						i = end - 1
						break
					}
				}
				buf.WriteByte(0)
			case 'x':
				// Hex escape: \xNN
				if i+1 < len(s) {
					hex := 0
					j := i + 1
					for ; j < len(s) && j < i+3; j++ {
						h := s[j]
						if h >= '0' && h <= '9' {
							hex = hex*16 + int(h-'0')
						} else if h >= 'a' && h <= 'f' {
							hex = hex*16 + int(h-'a'+10)
						} else if h >= 'A' && h <= 'F' {
							hex = hex*16 + int(h-'A'+10)
						} else {
							break
						}
					}
					if j > i+1 {
						buf.WriteByte(byte(hex))
						i = j - 1
						break
					}
				}
				buf.WriteByte('\\')
				buf.WriteByte('x')
			default:
				// Unknown escape with backslash: output backslash + char
				buf.WriteByte('\\')
				buf.WriteByte(s[i])
			}
			i++
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String()
}

func (s *runState) formatInt(val int64, conv byte, flags string, width, precision int, hasPrecision bool) {
	// Go's fmt package doesn't support %i — map it to %d.
	goConv := conv
	if goConv == 'i' {
		goConv = 'd'
	}

	// Handle negative width: left-justify (like '-' flag).
	if width < 0 {
		flags = "-" + flags
		width = -width
	}

	// Handle negative precision: treat as no precision (precision 0).
	if hasPrecision && precision < 0 {
		precision = 0
	}

	fmtStr := "%"
	if flags != "" {
		fmtStr += flags
	}
	if width >= 0 && hasPrecision {
		// When we have both width and precision, include width
		fmtStr += strconv.Itoa(width)
	} else if width > 0 {
		fmtStr += strconv.Itoa(width)
	}
	if hasPrecision {
		fmtStr += "."
		if precision >= 0 {
			fmtStr += strconv.Itoa(precision)
		}
	}
	fmtStr += string(goConv)
	s.out.WriteString(fmt.Sprintf(fmtStr, val))
}

func (s *runState) formatUint(val uint64, conv byte, flags string, width, precision int, hasPrecision bool) {
	// Handle negative width: left-justify.
	if width < 0 {
		flags = "-" + flags
		width = -width
	}
	// Handle negative precision: treat as no precision.
	if hasPrecision && precision < 0 {
		precision = 0
	}

	fmtStr := "%"
	if flags != "" {
		fmtStr += flags
	}
	if width > 0 {
		fmtStr += strconv.Itoa(width)
	}
	if hasPrecision {
		fmtStr += "."
		if precision >= 0 {
			fmtStr += strconv.Itoa(precision)
		}
	}
	fmtStr += string(conv)
	s.out.WriteString(fmt.Sprintf(fmtStr, val))
}

func (s *runState) formatFloat(val float64, conv byte, flags string, width, precision int, hasPrecision bool) {
	// Handle negative width: left-justify.
	if width < 0 {
		flags = "-" + flags
		width = -width
	}
	// Handle negative precision: treat as if precision was omitted (default 6).
	if hasPrecision && precision < 0 {
		hasPrecision = false
		precision = -1
	}

	fmtStr := "%"
	if flags != "" {
		fmtStr += flags
	}
	if width > 0 {
		fmtStr += strconv.Itoa(width)
	}
	if hasPrecision {
		fmtStr += "."
		if precision >= 0 {
			fmtStr += strconv.Itoa(precision)
		}
	}
	fmtStr += string(conv)
	s.out.WriteString(fmt.Sprintf(fmtStr, val))
}

func (s *runState) formatString(arg, flags string, width, precision int, hasPrecision bool) {
	if hasPrecision && precision >= 0 && precision < len(arg) {
		arg = arg[:precision]
	}

	// Only pad if width was explicitly specified (>= 0).
	// -1 means width was not specified.
	if width >= 0 && width > len(arg) {
		pad := width - len(arg)
		if strings.Contains(flags, "-") {
			s.out.WriteString(arg)
			writePad(s.out, pad)
		} else {
			writePad(s.out, pad)
			s.out.WriteString(arg)
		}
	} else {
		s.out.WriteString(arg)
	}
}

// parseInt parses an integer from a string, returning value and remaining unparsed suffix.
func parseInt(s string) (int64, string) {
	s = strings.TrimLeft(s, " ")
	if s == "" {
		return 0, ""
	}
	if s[0] == '+' {
		s = s[1:]
	}
	val, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0, s
	}
	return val, ""
}

// parseUint parses an unsigned integer.
func parseUint(s string) (uint64, string) {
	s = strings.TrimLeft(s, " ")
	if s == "" {
		return 0, ""
	}
	if s[0] == '+' {
		s = s[1:]
	}
	val, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		return 0, s
	}
	return val, ""
}

// parseFloat parses a float.
func parseFloat(s string) (float64, string) {
	s = strings.TrimLeft(s, " ")
	if s == "" {
		return 0, ""
	}
	if s[0] == '+' {
		s = s[1:]
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, s
	}
	return val, ""
}

// unescapeChar unescapes a single escaped character (like 'n' → '\n').
func unescapeChar(s string) byte {
	if len(s) == 0 {
		return 0
	}
	switch s[0] {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case '\\':
		return '\\'
	case 'a':
		return '\a'
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	case 'v':
		return '\v'
	case '0':
		return 0
	default:
		if len(s) >= 3 && s[0] >= '0' && s[0] <= '7' {
			val := 0
			for i := 0; i < 3 && i < len(s) && s[i] >= '0' && s[i] <= '7'; i++ {
				val = val*8 + int(s[i]-'0')
			}
			return byte(val)
		}
		return s[0]
	}
}

func writePad(b *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		b.WriteByte(' ')
	}
}

// processEscapes expands POSIX backslash escape sequences in a format string.
// \c causes it to stop expansion and return a marker.
func processEscapes(s string) string {
	var buf strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			c := s[i+1]
			switch c {
			case 'n':
				buf.WriteByte('\n')
				i += 2
			case 't':
				buf.WriteByte('\t')
				i += 2
			case 'r':
				buf.WriteByte('\r')
				i += 2
			case '\\':
				buf.WriteByte('\\')
				i += 2
			case 'a':
				buf.WriteByte('\a')
				i += 2
			case 'b':
				buf.WriteByte('\b')
				i += 2
			case 'f':
				buf.WriteByte('\f')
				i += 2
			case 'v':
				buf.WriteByte('\v')
				i += 2
			case 'c':
				// \c is handled at the Format level (stops output).
				// Pass through as literal \ and c for Format to detect.
				buf.WriteByte('\\')
				buf.WriteByte('c')
				i += 2
			case '0':
				// Octal escape: \0NNN (1-3 octal digits)
				end := i + 2
				for end < len(s) && end < i+5 && s[end] >= '0' && s[end] <= '7' {
					end++
				}
				if end > i+2 {
					val, err := strconv.ParseUint(s[i+2:end], 8, 8)
					if err == nil {
						buf.WriteByte(byte(val))
						i = end
						continue
					}
				}
				// Just \0 = NUL byte
				buf.WriteByte(0)
				i += 2
			case 'x':
				// Hex escape: \xNN
				j := i + 2
				for j < len(s) && j < i+4 {
					h := s[j]
					if (h >= '0' && h <= '9') || (h >= 'a' && h <= 'f') || (h >= 'A' && h <= 'F') {
						j++
					} else {
						break
					}
				}
				if j > i+2 {
					val, _ := strconv.ParseUint(s[i+2:j], 16, 8)
					buf.WriteByte(byte(val))
					i = j
				} else {
					buf.WriteByte('\\')
					buf.WriteByte('x')
					i += 2
				}
			default:
				// Unknown escape: preserve backslash + char
				buf.WriteByte('\\')
				buf.WriteByte(c)
				i += 2
			}
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String()
}

func run(args []string, out io.Writer) int {
	// Manual flag parsing: only --json is accepted as a flag.
	// Everything else (including -...) is a positional argument.
	jsonMode := false
	posArgs := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		} else {
			posArgs = append(posArgs, a)
		}
	}

	if len(posArgs) == 0 {
		common.RenderError("printf", 1, "MISSING_OPERAND", "missing operand", jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "printf: missing operand\n")
		}
		return 1
	}

	format := posArgs[0]
	fmtArgs := posArgs[1:]

	output, hadErr := Format(format, fmtArgs)

	exitCode := 0
	if hadErr {
		exitCode = 1
	}

	common.Render("printf", PrintfResult{Output: output}, jsonMode, out, func() {
		fmt.Fprint(out, output)
	})
	return exitCode
}

// countSpecifiers counts the number of conversion specifiers in a format string.
func countSpecifiers(format string) int {
	count := 0
	i := 0
	for i < len(format) {
		if format[i] == '%' {
			if i+1 < len(format) && format[i+1] == '%' {
				i += 2
				continue
			}
			count++
			i++ // skip %
			// skip flags, width, precision, length
			for i < len(format) && strings.ContainsRune("-+ 0#", rune(format[i])) {
				i++
			}
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
				i++
			}
			if i < len(format) && format[i] == '.' {
				i++
				for i < len(format) && format[i] >= '0' && format[i] <= '9' {
					i++
				}
			}
			// length modifiers
			if i < len(format) {
				switch format[i] {
				case 'h':
					i++
					if i < len(format) && format[i] == 'h' {
						i++
					}
				case 'l':
					i++
					if i < len(format) && format[i] == 'l' {
						i++
					}
				case 'j', 'z', 't', 'L':
					i++
				}
			}
			if i < len(format) {
				i++ // conversion char
			}
		} else {
			i++
		}
	}
	return count
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "printf",
		Usage: "Format and print data",
		Run:   run,
	})
}
