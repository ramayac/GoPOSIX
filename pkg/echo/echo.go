// Package echo implements the POSIX echo utility.
//
// echo prints its arguments to stdout, separated by spaces and
// terminated with a newline. It supports -n (suppress newline),
// -e (enable backslash escapes), and -E (disable escapes, default).
// Only --json is accepted as a long flag; everything else is literal text.
package echo

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// EchoResult is the structured result for --json mode.
type EchoResult struct {
	Text string `json:"text"`
}

// Run is the library function: given flags and words, return EchoResult.
func Run(noNewline, escape bool, words []string) EchoResult {
	text := strings.Join(words, " ")
	if escape {
		text = processEscapes(text)
	}
	return EchoResult{Text: text}
}

// processEscapes expands \n, \t, \\, \NNN (octal), etc. like echo -e.
func processEscapes(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			c := s[i]
			switch c {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case 'a':
				sb.WriteByte('\a')
			case 'b':
				sb.WriteByte('\b')
			case 'v':
				sb.WriteByte('\v')
			case 'x':
				// hex
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
						sb.WriteByte(byte(hex))
						i = j - 1
					} else {
						sb.WriteByte('\\')
						sb.WriteByte('x')
					}
				} else {
					sb.WriteByte('\\')
					sb.WriteByte('x')
				}
			default:
				// Handle octal escape: \NNN where N is 0-7 (1-3 digits).
				// \0 is special: the 0 is a marker, digits start after it.
				// \1-\7: the digit IS the first octal digit.
				if c == '0' {
					// \0: consume up to 3 octal digits starting from i+1.
					start := i + 1
					end := start
					for end < len(s) && end < start+3 && s[end] >= '0' && s[end] <= '7' {
						end++
					}
					if end > start {
						oct, _ := strconv.ParseUint(s[start:end], 8, 8)
						sb.WriteByte(byte(oct))
						i = end - 1
					} else {
						sb.WriteByte(0)
					}
				} else if c >= '1' && c <= '7' {
					oct := int(c - '0')
					j := i + 1
					for ; j < len(s) && j < i+3 && s[j] >= '0' && s[j] <= '7'; j++ {
						oct = oct*8 + int(s[j]-'0')
					}
					sb.WriteByte(byte(oct))
					i = j - 1
				} else {
					// Unknown escape: preserve backslash + char
					sb.WriteByte('\\')
					sb.WriteByte(c)
				}
			}
		} else {
			sb.WriteByte(s[i])
		}
	}
	return sb.String()
}

// parseEchoFlags manually extracts echo-specific flags from the start of args.
// Only -n, -e, -E, and --json are recognized. All other arguments (including
// anything starting with -) are treated as literal text. This avoids the
// general ParseFlags which would choke on "---" or similar strings.
func parseEchoFlags(args []string) (noNewline, escape, jsonMode bool, words []string) {
	var i int
	for i < len(args) {
		a := args[i]
		// --json (long flag only, no -j short form to avoid collisions)
		if a == "--json" {
			jsonMode = true
			i++
			continue
		}
		// Short flag groups: only -n, -e, -E are recognized
		if len(a) >= 2 && a[0] == '-' && a[1] != '-' {
			chars := a[1:]
			// Accumulate flags first; only apply if all chars are valid.
			// This prevents partial side-effects when an invalid char
			// appears (e.g., -neEZ should be printed literally).
			nnl, esc, allValid := false, false, true
			hasE := false
			for _, c := range chars {
				switch c {
				case 'n':
					nnl = true
				case 'e':
					esc = true
				case 'E':
					esc = false
					hasE = true
				default:
					allValid = false
				}
			}
			if allValid {
				noNewline = nnl
				if esc || hasE {
					escape = esc
				}
				i++
				continue
			}
		}
		// Anything else: stop flag parsing, rest is literal text
		break
	}
	words = args[i:]
	return
}

func run(args []string, out io.Writer) int {
	noNewline, escape, jsonMode, words := parseEchoFlags(args)

	result := Run(noNewline, escape, words)

	common.Render("echo", result, jsonMode, out, func() {
		if noNewline {
			fmt.Fprint(out, result.Text)
		} else {
			fmt.Fprintln(out, result.Text)
		}
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "echo",
		Usage: "Display a line of text",
		Run:   run,
	})
}
