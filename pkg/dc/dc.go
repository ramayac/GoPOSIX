// Package dc implements the POSIX-compliant dc (desk calculator) utility
// with arbitrary-precision arithmetic using Go's math/big.
package dc

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// DcResult is the JSON output structure.
type DcResult struct {
	Output []string `json:"output"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "expression", Short: "e", Type: common.FlagValue},
		{Long: "file", Short: "f", Type: common.FlagValue},
		{Long: "x", Short: "x", Type: common.FlagValue}, // extended register mode (same as -e in GoPOSIX)
		{Long: "json", Type: common.FlagBool},
	},
}

// dcState holds the execution state of dc.
type dcState struct {
	stack []dcValue          // main stack
	regs  map[rune][]dcValue // registers (each holds a stack)
	scale int                // current scale
}

// dcValue is either a *big.Rat (number) or string.
type dcValue struct {
	isStr      bool
	str        string
	rat        *big.Rat
	fracDigits int // original number of fractional digits (for Z command)
}

func newNumStr(s string) (*big.Rat, int, error) {
	s = strings.TrimLeft(s, " \t")

	negative := false
	if strings.HasPrefix(s, "_") {
		negative = true
		s = s[1:]
	}
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	// Remove leading zeros (but keep at least one digit)
	for len(s) > 1 && s[0] == '0' && s[1] != '.' {
		s = s[1:]
	}

	if s == "" || s == "." {
		s = "0"
	}

	dotIdx := strings.Index(s, ".")
	if dotIdx == -1 {
		// Integer
		if negative {
			s = "-" + s
		}
		r := new(big.Rat)
		if _, ok := r.SetString(s); !ok {
			return nil, 0, fmt.Errorf("invalid number: %q", s)
		}
		return r, 0, nil
	}

	// Decimal: build as integer * 10^(-fracLen)
	intPart := s[:dotIdx]
	fracPart := s[dotIdx+1:]
	if intPart == "" {
		intPart = "0"
	}
	if fracPart == "" {
		fracPart = "0"
	}

	combined := intPart + fracPart
	if negative {
		combined = "-" + combined
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(fracPart))), nil)
	num := new(big.Int)
	if _, ok := num.SetString(combined, 10); !ok {
		return nil, 0, fmt.Errorf("invalid number: %q", s)
	}
	numer := new(big.Int).Set(num)
	if numer.Sign() < 0 {
		numer = new(big.Int).Neg(num)
	}
	r := new(big.Rat).SetFrac(num, denom)
	return r, len(fracPart), nil
}

func pushNumDC(state *dcState, s string) error {
	r, decimals, err := newNumStr(s)
	if err != nil {
		return err
	}
	state.stack = append(state.stack, dcValue{rat: r, fracDigits: decimals})
	return nil
}

func pushStrDC(state *dcState, s string) {
	state.stack = append(state.stack, dcValue{isStr: true, str: s})
}

func popVal(state *dcState) (dcValue, bool) {
	if len(state.stack) == 0 {
		return dcValue{}, false
	}
	v := state.stack[len(state.stack)-1]
	state.stack = state.stack[:len(state.stack)-1]
	return v, true
}

// popNumVal pops the top value as a number, also returning its fracDigits.
func popNumValFD(state *dcState) (*big.Rat, int, bool) {
	v, ok := popVal(state)
	if !ok {
		return nil, 0, false
	}
	if v.isStr {
		return new(big.Rat), 0, true
	}
	return v.rat, v.fracDigits, true
}

func popNumVal(state *dcState) (*big.Rat, bool) {
	r, _, ok := popNumValFD(state)
	return r, ok
}

// popTwoNumFD pops the top two values as numbers, also returning their fracDigits.
func popTwoNumFD(state *dcState) (a, b *big.Rat, aFrac, bFrac int, ok bool) {
	b, bFrac, ok = popNumValFD(state)
	if !ok {
		return
	}
	a, aFrac, ok = popNumValFD(state)
	if !ok {
		state.stack = append(state.stack, dcValue{rat: b, fracDigits: bFrac})
		return
	}
	return
}

// popTwo pops the top two values as numbers
func popTwoNumVal(state *dcState) (a, b *big.Rat, ok bool) {
	b, ok = popNumVal(state)
	if !ok {
		return
	}
	a, ok = popNumVal(state)
	if !ok {
		state.stack = append(state.stack, dcValue{rat: b})
		return
	}
	return
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func dupVal(v dcValue) dcValue {
	if v.isStr {
		return dcValue{isStr: true, str: v.str}
	}
	return dcValue{rat: new(big.Rat).Set(v.rat), fracDigits: v.fracDigits}
}

func ratToInt64(r *big.Rat) int64 {
	if r.IsInt() {
		return r.Num().Int64()
	}
	q := new(big.Int).Quo(r.Num(), r.Denom())
	return q.Int64()
}

func formatRat(r *big.Rat, scale int) string {
	if r.Sign() == 0 {
		return "0"
	}
	if r.IsInt() && scale == 0 {
		return r.Num().String()
	}

	// Get sign
	sign := ""
	if r.Sign() < 0 {
		sign = "-"
	}

	a := new(big.Rat).Set(r)
	if a.Sign() < 0 {
		a.Neg(a)
	}

	num := a.Num()
	den := a.Denom()

	intPart := new(big.Int).Quo(num, den)
	rem := new(big.Int).Rem(num, den)

	maxDigits := scale + 10
	if maxDigits < 200 {
		maxDigits = 200
	}

	var fracStr strings.Builder
	for rem.Sign() != 0 && fracStr.Len() < maxDigits {
		rem.Mul(rem, big.NewInt(10))
		digit := new(big.Int).Quo(rem, den)
		fracStr.WriteByte('0' + byte(digit.Int64()))
		rem.Rem(rem, den)
	}

	fs := fracStr.String()

	if scale > 0 {
		if len(fs) > scale {
			fs = fs[:scale]
		}
		for len(fs) < scale {
			fs += "0"
		}
		intStr := intPart.String()
		if intStr == "0" && sign == "" {
			return "." + fs
		}
		if intStr == "0" && sign == "-" {
			return "-." + fs
		}
		return sign + intStr + "." + fs
	}

	// Trim trailing zeros when no explicit scale
	fs = strings.TrimRight(fs, "0")

	if fs != "" {
		intStr := intPart.String()
		// Strip leading zero for values < 1 (BusyBox convention)
		if intStr == "0" && sign == "" {
			return "." + fs
		}
		if intStr == "0" && sign == "-" {
			return "-." + fs
		}
		return sign + intStr + "." + fs
	}
	return sign + intPart.String()
}

func (v dcValue) String(scale int) string {
	if v.isStr {
		return v.str
	}
	// Use per-number fracDigits for formatting precision.
	// fracDigits == 0 means "no explicit decimal places" — format as-is.
	return formatRat(v.rat, v.fracDigits)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "dc: %v\n", err)
		return 2
	}

	jsonMode := flags.Has("json")

	state := &dcState{
		regs: make(map[rune][]dcValue),
	}

	var buf strings.Builder

	for _, expr := range flags.GetAll("expression") {
		buf.WriteString(expr)
		buf.WriteByte(' ')
	}
	for _, expr := range flags.GetAll("x") {
		buf.WriteString(expr)
		buf.WriteByte(' ')
	}
	for _, file := range flags.GetAll("file") {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(stderr, "dc: %v\n", err)
			return 1
		}
		buf.Write(data)
		buf.WriteByte(' ')
	}

	if buf.Len() == 0 {
		posArgs := flags.Positional
		if len(posArgs) > 0 {
			// Standard dc convention: positional args are filenames.
			// If a file exists, read it; otherwise treat as literal expression.
			hasFiles := false
			for _, arg := range posArgs {
				if _, err := os.Stat(arg); err == nil {
					data, ferr := os.ReadFile(arg)
					if ferr == nil {
						buf.Write(data)
						buf.WriteByte(' ')
						hasFiles = true
						continue
					}
				}
				buf.WriteString(arg)
				buf.WriteByte(' ')
			}
			_ = hasFiles
		} else if stdin != nil {
			data, _ := io.ReadAll(stdin)
			buf.Write(data)
		}
	}

	var output []string
	err = evalDC(state, buf.String(), stdin, &output)
	if err != nil {
		// BusyBox dc prints errors but always exits 0; match that behavior
		fmt.Fprintf(stderr, "dc: %v\n", err)
	}

	// Wrap long lines to match BusyBox dc output format
	output = wrapOutput(output)

	result := DcResult{Output: output}
	common.Render("dc", result, jsonMode, stdout, func() {
		for _, line := range output {
			fmt.Fprintln(stdout, line)
		}
	})
	return 0
}

// wrapOutput wraps long output lines with \\ continuation (matching BusyBox dc).
func wrapOutput(lines []string) []string {
	var result []string
	for _, line := range lines {
		const maxLen = 69
		for len(line) > maxLen {
			result = append(result, line[:maxLen]+"\\")
			line = line[maxLen:]
		}
		result = append(result, line)
	}
	return result
}

// Returns the number string and new position.
func parseNumber(runes []rune, pos int) (string, int) {
	start := pos
	i := pos
	// Handle negative prefix
	if pos < len(runes) && runes[pos] == '_' {
		i = pos + 1
	}
	// Parse integer part
	for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
		i++
	}
	// Parse decimal part
	if i < len(runes) && runes[i] == '.' {
		i++
		for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
			i++
		}
	}
	// Must have consumed at least one digit (check first digit position)
	numStart := start
	if start < len(runes) && runes[start] == '_' {
		numStart++
	}
	digitStart := numStart
	if digitStart < len(runes) && runes[digitStart] == '.' {
		digitStart++
	}
	if i > digitStart && digitStart < len(runes) && runes[digitStart] >= '0' && runes[digitStart] <= '9' {
		return string(runes[start:i]), i
	}
	return "", pos
}

func evalDC(state *dcState, input string, stdin io.Reader, output *[]string) error {
	runes := []rune(input)
	i := 0
	n := len(runes)

	for i < n {
		ch := runes[i]

		// Skip whitespace
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			i++
			continue
		}

		// Comment
		if ch == '#' {
			for i < n && runes[i] != '\n' {
				i++
			}
			continue
		}

		// String literal
		if ch == '[' {
			i++ // skip [
			var sb strings.Builder
			depth := 1
			for i < n && depth > 0 {
				c := runes[i]
				i++
				if c == '[' {
					depth++
					sb.WriteRune(c)
				} else if c == ']' {
					depth--
					if depth > 0 {
						sb.WriteRune(c)
					}
				} else if c == '\\' && i < n && runes[i] != '[' && runes[i] != ']' {
					next := runes[i]
					i++
					sb.WriteRune('\\')
					sb.WriteRune(next)
				} else if c == '\\' {
					sb.WriteRune('\\')
				} else {
					sb.WriteRune(c)
				}
			}
			pushStrDC(state, sb.String())
			continue
		}

		// Number (including _neg and .decimals)
		if (ch >= '0' && ch <= '9') || ch == '_' || ch == '.' {
			numStr, next := parseNumber(runes, i)
			if numStr != "" {
				if err := pushNumDC(state, numStr); err != nil {
					return err
				}
				i = next
				continue
			}
			i++
			continue
		}

		i++ // consume the command character

		switch ch {

		// ---- Output ----
		case 'p':
			if len(state.stack) > 0 {
				v := state.stack[len(state.stack)-1]
				*output = append(*output, v.String(state.scale))
			}

		case 'n':
			if v, ok := popVal(state); ok {
				s := v.String(state.scale)
				if len(*output) > 0 {
					(*output)[len(*output)-1] += s
				} else {
					*output = append(*output, s)
				}
			}

		case 'P':
			if v, ok := popVal(state); ok && v.isStr {
				s := parseStringEscapes(v.str)
				if len(*output) > 0 {
					(*output)[len(*output)-1] += s
				} else {
					*output = append(*output, s)
				}
			}

		case 'f':
			for j := len(state.stack) - 1; j >= 0; j-- {
				*output = append(*output, state.stack[j].String(state.scale))
			}

		// ---- Stack ops ----
		case 'c':
			state.stack = nil

		case 'd':
			if len(state.stack) > 0 {
				state.stack = append(state.stack, dupVal(state.stack[len(state.stack)-1]))
			}

		case 'r':
			if len(state.stack) >= 2 {
				state.stack[len(state.stack)-1], state.stack[len(state.stack)-2] =
					state.stack[len(state.stack)-2], state.stack[len(state.stack)-1]
			}

		case 'R':
			popVal(state)

		case 'z':
			r := new(big.Rat).SetInt64(int64(len(state.stack)))
			state.stack = append(state.stack, dcValue{rat: r})

		case 'Z':
			if len(state.stack) > 0 {
				v := state.stack[len(state.stack)-1]
				var length int64
				if v.isStr {
					length = int64(len(v.str))
				} else {
					s := formatRat(v.rat, state.scale)
					// Remove minus, decimal point, leading zeros
					s = strings.TrimPrefix(s, "-")
					s = strings.ReplaceAll(s, ".", "")
					// Pad with trailing zeros to restore precision lost during
					// big.Rat simplification (e.g., 0.00120 → 3/2500 → ".0012")
					if v.fracDigits > 0 {
						for len(s) < v.fracDigits {
							s += "0"
						}
					}
					s = strings.TrimLeft(s, "0")
					if s == "" {
						length = 1
					} else {
						length = int64(len(s))
					}
				}
				r := new(big.Rat).SetInt64(length)
				state.stack = append(state.stack, dcValue{rat: r})
			}

		// ---- Arithmetic ----
		case '+':
			a, b, aFrac, bFrac, ok := popTwoNumFD(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			r := new(big.Rat).Add(a, b)
			state.stack = append(state.stack, dcValue{rat: r, fracDigits: maxInt(aFrac, bFrac)})

		case '-':
			a, b, aFrac, bFrac, ok := popTwoNumFD(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			r := new(big.Rat).Sub(a, b)
			state.stack = append(state.stack, dcValue{rat: r, fracDigits: maxInt(aFrac, bFrac)})

		case '*':
			a, b, aFrac, bFrac, ok := popTwoNumFD(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			r := new(big.Rat).Mul(a, b)
			// BusyBox dc: result scale = min(scale(a)+scale(b), max(scale, scale(a), scale(b)))
			fd := aFrac + bFrac
			maxScale := maxInt(maxInt(state.scale, aFrac), bFrac)
			if fd > maxScale {
				fd = maxScale
			}
			state.stack = append(state.stack, dcValue{rat: r, fracDigits: fd})

		case '/':
			a, b, _, _, ok := popTwoNumFD(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			if b.Sign() == 0 {
				return fmt.Errorf("divide by zero")
			}
			r := new(big.Rat).Quo(a, b)
			state.stack = append(state.stack, dcValue{rat: r, fracDigits: state.scale})

		case '%':
			a, b, aFrac, bFrac, ok := popTwoNumFD(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			if b.Sign() == 0 {
				return fmt.Errorf("remainder by zero")
			}
			// Truncation toward zero (C/Go semantics)
			quot := new(big.Rat).Quo(a, b)
			qInt := new(big.Int).Quo(quot.Num(), quot.Denom())
			prod := new(big.Rat).SetInt(qInt)
			prod.Mul(prod, b)
			r := new(big.Rat).Sub(a, prod)
			state.stack = append(state.stack, dcValue{rat: r, fracDigits: maxInt(maxInt(aFrac, bFrac), state.scale)})

		case '~':
			a, b, aFrac, bFrac, ok := popTwoNumFD(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			if b.Sign() == 0 {
				return fmt.Errorf("divide by zero")
			}
			quot := new(big.Rat).Quo(a, b)
			qInt := new(big.Int).Quo(quot.Num(), quot.Denom())
			qRat := new(big.Rat).SetInt(qInt)
			prod := new(big.Rat).Mul(b, qRat)
			rem := new(big.Rat).Sub(a, prod)
			remScale := maxInt(maxInt(aFrac, bFrac), state.scale)
			state.stack = append(state.stack, dcValue{rat: qRat, fracDigits: state.scale})
			state.stack = append(state.stack, dcValue{rat: rem, fracDigits: remScale})

		case '^':
			a, b, aFrac, _, ok := popTwoNumFD(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			exp := ratToInt64(b)
			r := new(big.Rat)
			var resultFrac int
			if exp == 0 {
				r.SetInt64(1)
			} else if a.Sign() == 0 {
				// 0^n → 0 (BusyBox convention for all n != 0)
				r.SetInt64(0)
			} else if exp < 0 {
				r = ratPowInt(a, -exp)
				r.Inv(r)
				resultFrac = state.scale
			} else {
				r = ratPowInt(a, exp)
				// Result scale: min(exp * base_scale, max(global_scale, base_scale))
				fd := int(exp) * aFrac
				maxScale := maxInt(state.scale, aFrac)
				if fd > maxScale {
					fd = maxScale
				}
				resultFrac = fd
			}
			state.stack = append(state.stack, dcValue{rat: r, fracDigits: resultFrac})

		case 'v':
			a, aFrac, ok := popNumValFD(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			if a.Sign() < 0 {
				return fmt.Errorf("square root of negative number")
			}
			r := ratSqrtNewton(a, state.scale)
			state.stack = append(state.stack, dcValue{rat: r, fracDigits: maxInt(state.scale, aFrac)})

		case '|':
			// Modular exponentiation: mod, exp, base → base^exp % mod
			mod, ok := popNumVal(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			exp, ok := popNumVal(state)
			if !ok {
				state.stack = append(state.stack, dcValue{rat: mod})
				return fmt.Errorf("stack empty")
			}
			base, ok := popNumVal(state)
			if !ok {
				state.stack = append(state.stack, dcValue{rat: exp})
				state.stack = append(state.stack, dcValue{rat: mod})
				return fmt.Errorf("stack empty")
			}
			r := ratModExpVal(base, exp, mod)
			state.stack = append(state.stack, dcValue{rat: r})

		// ---- Scale ----
		case 'k':
			v, ok := popNumVal(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			s := int(ratToInt64(v))
			if s < 0 {
				s = 0
			}
			state.scale = s

		case 'K':
			r := new(big.Rat).SetInt64(int64(state.scale))
			state.stack = append(state.stack, dcValue{rat: r, fracDigits: state.scale})

		// ---- Registers ----
		case 's':
			if i < n {
				reg := runes[i]
				i++
				if v, ok := popVal(state); ok {
					state.regs[reg] = []dcValue{dupVal(v)}
				} else {
					state.regs[reg] = []dcValue{{isStr: true, str: ""}}
				}
			}

		case 'l':
			if i < n {
				reg := runes[i]
				i++
				if vals, ok := state.regs[reg]; ok && len(vals) > 0 {
					state.stack = append(state.stack, dupVal(vals[len(vals)-1]))
				} else {
					// Undefined register → push 0
					state.stack = append(state.stack, dcValue{rat: new(big.Rat)})
				}
			}

		case 'S':
			if i < n {
				reg := runes[i]
				i++
				if len(state.stack) > 0 {
					v := state.stack[len(state.stack)-1]
					state.regs[reg] = append(state.regs[reg], dupVal(v))
				}
			}

		case 'L':
			if i < n {
				reg := runes[i]
				i++
				if vals, ok := state.regs[reg]; ok && len(vals) > 0 {
					v := vals[len(vals)-1]
					state.regs[reg] = vals[:len(vals)-1]
					state.stack = append(state.stack, dupVal(v))
				}
			}

		// ---- Macro execution ----
		case 'x':
			if v, ok := popVal(state); ok {
				if v.isStr {
					if err := evalDC(state, v.str, stdin, output); err != nil {
						return err
					}
				} else {
					// Push back non-string values (BusyBox dc doesn't pop them)
					state.stack = append(state.stack, v)
				}
			}

		// ---- Comparison (push result) ----
		case '(':
			a, b, ok := popTwoNumVal(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			r := new(big.Rat)
			if a.Cmp(b) > 0 {
				r.SetInt64(1)
			}
			state.stack = append(state.stack, dcValue{rat: r})

		case '{':
			a, b, ok := popTwoNumVal(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			r := new(big.Rat)
			if a.Cmp(b) >= 0 {
				r.SetInt64(1)
			}
			state.stack = append(state.stack, dcValue{rat: r})

		case 'G':
			a, b, ok := popTwoNumVal(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			r := new(big.Rat)
			if a.Cmp(b) == 0 {
				r.SetInt64(1)
			}
			state.stack = append(state.stack, dcValue{rat: r})

		case 'N':
			v, ok := popVal(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			r := new(big.Rat)
			if v.isStr && v.str == "" {
				r.SetInt64(1)
			} else if !v.isStr && v.rat.Sign() == 0 {
				r.SetInt64(1)
			}
			state.stack = append(state.stack, dcValue{rat: r})

		// ---- Conditional execute ----
		case '>', '<', '=':
			if i < n {
				condOp := ch
				reg := runes[i]
				i++
				a, b, ok := popTwoNumVal(state)
				if !ok {
					return fmt.Errorf("stack empty")
				}
				var execute bool
				switch condOp {
				case '>':
					execute = b.Cmp(a) > 0
				case '<':
					execute = b.Cmp(a) < 0
				case '=':
					execute = b.Cmp(a) == 0
				}
				if execute {
					if vals, ok := state.regs[reg]; ok && len(vals) > 0 && vals[len(vals)-1].isStr {
						evalDC(state, vals[len(vals)-1].str, stdin, output)
					}
				} else {
					// Check for else clause (>aeb)
					if i < n && runes[i] == 'e' && i+1 < n {
						i++
						elseReg := runes[i]
						i++
						if vals, ok := state.regs[elseReg]; ok && len(vals) > 0 && vals[len(vals)-1].isStr {
							evalDC(state, vals[len(vals)-1].str, stdin, output)
						}
					}
				}
			}

		// ---- Negated conditional execute ----
		case '!':
			if i < n {
				condOp := runes[i]
				if condOp == '>' || condOp == '<' || condOp == '=' {
					i++
					if i < n {
						reg := runes[i]
						i++
						a, b, ok := popTwoNumVal(state)
						if !ok {
							return fmt.Errorf("stack empty")
						}
						var execute bool
						switch condOp {
						case '>':
							execute = !(b.Cmp(a) > 0)
						case '<':
							execute = !(b.Cmp(a) < 0)
						case '=':
							execute = !(b.Cmp(a) == 0)
						}
						if execute {
							if vals, ok := state.regs[reg]; ok && len(vals) > 0 && vals[len(vals)-1].isStr {
								evalDC(state, vals[len(vals)-1].str, stdin, output)
							}
						} else {
							if i < n && runes[i] == 'e' && i+1 < n {
								i++
								elseReg := runes[i]
								i++
								if vals, ok := state.regs[elseReg]; ok && len(vals) > 0 && vals[len(vals)-1].isStr {
									evalDC(state, vals[len(vals)-1].str, stdin, output)
								}
							}
						}
					}
				}
			}

		// ---- Read from stdin ----
		case '?':
			if stdin != nil {
				line, err := readLine(stdin)
				if err == nil && line != "" {
					evalDC(state, line, stdin, output)
				}
			}

		// ---- Convert to ASCII string ----
		case 'a':
			v, ok := popVal(state)
			if !ok {
				return fmt.Errorf("stack empty")
			}
			var s string
			if v.isStr {
				s = v.str
			} else {
				code := int(ratToInt64(v.rat)) & 0xFF
				s = string(rune(code))
			}
			pushStrDC(state, s)

		// Default: ignore unknown characters (dc silently ignores them)
		default:
			// ignored
		}
	}
	return nil
}

func readLine(r io.Reader) (string, error) {
	var buf []byte
	one := make([]byte, 1)
	for {
		_, err := r.Read(one)
		if err != nil {
			return string(buf), err
		}
		if one[0] == '\n' {
			return string(buf), nil
		}
		buf = append(buf, one[0])
	}
}

func ratPowInt(base *big.Rat, exp int64) *big.Rat {
	r := new(big.Rat).SetInt64(1)
	b := new(big.Rat).Set(base)
	for exp > 0 {
		if exp&1 == 1 {
			r.Mul(r, b)
		}
		b.Mul(b, b)
		exp >>= 1
	}
	return r
}

func ratSqrtNewton(a *big.Rat, scale int) *big.Rat {
	if a.Sign() == 0 {
		return new(big.Rat)
	}

	// Work with scaled integers: sqrt(a) = sqrt(num/den) = sqrt(num*den)/den.
	// Scale up by 10^(2*scale) for precision, then divide back.
	num := new(big.Int).Set(a.Num())
	den := new(big.Int).Set(a.Denom())

	// Compute intSqrt(num * den * 10^(2*scale))
	scalePow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(2*(scale+2))), nil)
	prod := new(big.Int).Mul(num, den)
	prod.Mul(prod, scalePow)

	n := new(big.Int).Sqrt(prod)

	// Result = n / (den * 10^(scale+2))
	denPow := new(big.Int).Mul(den, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale+2)), nil))

	result := new(big.Rat).SetFrac(n, denPow)
	return result
}

func ratModExpVal(base, exp, mod *big.Rat) *big.Rat {
	// Exponent 0 always yields 1 (BusyBox dc convention)
	if exp.Sign() == 0 {
		return new(big.Rat).SetInt64(1)
	}

	b := new(big.Int)
	if base.IsInt() {
		b.Set(base.Num())
	} else {
		b.Quo(base.Num(), base.Denom())
	}

	e := new(big.Int).Quo(exp.Num(), exp.Denom())

	m := new(big.Int)
	if mod.IsInt() {
		m.Set(mod.Num())
	} else {
		m.Quo(mod.Num(), mod.Denom())
	}

	if m.Sign() == 0 {
		return new(big.Rat)
	}
	result := new(big.Int).Exp(b, e, m)
	return new(big.Rat).SetInt(result)
}

func parseStringEscapes(s string) string {
	var sb strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) {
			switch runes[i+1] {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			case '\\':
				sb.WriteRune('\\')
			default:
				sb.WriteRune(runes[i])
				sb.WriteRune(runes[i+1])
			}
			i++
		} else {
			sb.WriteRune(runes[i])
		}
	}
	return sb.String()
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "dc",
		Usage: "Desk calculator (arbitrary precision RPN)",
		Run:   run,
	})
}
