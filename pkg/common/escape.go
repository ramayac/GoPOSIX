// Package common provides shared helpers for goposix utilities.
package common

import (
	"strconv"
	"strings"
)

// EscapeMode selects the octal and hexadecimal semantics of ExpandEscapes.
type EscapeMode int

const (
	// EscapeFormat is the printf format-string mode: \0NNN is a full octal
	// escape and \xNN accepts up to two digits.
	EscapeFormat EscapeMode = iota
	// EscapeArg is the echo -e / printf %b mode: a bare \0 is a NUL byte and
	// \1–\7 start an octal escape; \xNN accepts up to two digits.
	EscapeArg
)

// ExpandEscapes expands POSIX backslash escape sequences in s.
//
// Supported escapes: \n, \t, \r, \\, \a, \b, \f, \v, octal, and hexadecimal
// (\xNN). The \c escape is passed through unchanged as the two characters
// "\c" so callers can stop output at the format level. Unknown escapes keep
// their backslash.
func ExpandEscapes(s string, mode EscapeMode) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			sb.WriteByte(s[i])
			continue
		}
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
		case 'f':
			sb.WriteByte('\f')
		case 'v':
			sb.WriteByte('\v')
		case 'c':
			// \c stops output; the caller sees it and stops. Preserve it.
			sb.WriteByte('\\')
			sb.WriteByte('c')
		case 'x':
			// Hex escape: \xNN (up to two digits).
			j := i + 1
			for j < len(s) && j < i+3 && isHexDigit(s[j]) {
				j++
			}
			if j > i+1 {
				if val, err := strconv.ParseUint(s[i+1:j], 16, 8); err == nil {
					sb.WriteByte(byte(val))
					i = j - 1
					break
				}
			}
			sb.WriteByte('\\')
			sb.WriteByte('x')
		case '0':
			if mode == EscapeFormat {
				// \0NNN: one to three octal digits; a bare \0 is a NUL byte.
				end := i + 1
				for end < len(s) && end < i+4 && isOctalDigit(s[end]) {
					end++
				}
				if end > i+1 {
					if val, err := strconv.ParseUint(s[i+1:end], 8, 8); err == nil {
						sb.WriteByte(byte(val))
						i = end - 1
						break
					}
				}
				sb.WriteByte(0)
			} else {
				// A bare \0 is a NUL byte; following octal digits are consumed.
				start := i + 1
				end := start
				for end < len(s) && end < start+3 && isOctalDigit(s[end]) {
					end++
				}
				if end > start {
					if val, err := strconv.ParseUint(s[start:end], 8, 8); err == nil {
						sb.WriteByte(byte(val))
						i = end - 1
						break
					}
				}
				sb.WriteByte(0)
			}
		default:
			if mode == EscapeArg && c >= '1' && c <= '7' {
				// \1–\7: the digit is the first octal digit.
				oct := int(c - '0')
				j := i + 1
				for j < len(s) && j < i+3 && isOctalDigit(s[j]) {
					oct = oct*8 + int(s[j]-'0')
					j++
				}
				sb.WriteByte(byte(oct))
				i = j - 1
			} else {
				// Unknown escape: keep backslash and the character.
				sb.WriteByte('\\')
				sb.WriteByte(c)
			}
		}
	}
	return sb.String()
}

func isOctalDigit(b byte) bool { return b >= '0' && b <= '7' }

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
