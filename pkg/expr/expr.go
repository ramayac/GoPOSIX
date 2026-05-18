// Package expr implements the POSIX expr utility.
//
// Grammar (lowest to highest precedence):
//
//	expr   → or
//	or     → and ( '|' and )*
//	and    → cmp ( '&' cmp )*
//	cmp    → add ( ( '=' | '!=' | '<' | '<=' | '>' | '>=' ) add )*
//	add    → mul ( ( '+' | '-' ) mul )*
//	mul    → unary ( ( '*' | '/' | '%' ) unary )*
//	unary  → 'match' primary primary | 'substr' primary primary primary
//	       | 'index' primary primary | 'length' primary? | primary
//	primary→ '(' expr ')' | ATOM
package expr

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// ExprResult is the structured result for --json mode.
type ExprResult struct {
	Result   string `json:"result"`
	ExitCode int    `json:"exitCode"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// --- Evaluator ---

type parser struct {
	tokens []string
	pos    int
}

func (p *parser) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}

func (p *parser) next() string {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *parser) done() bool {
	return p.pos >= len(p.tokens)
}

// Eval evaluates a POSIX expr expression. Returns the result string and the
// POSIX exit code: 0 if result is non-null and non-zero, 1 otherwise, 2 on error.
func Eval(tokens []string) (string, int, error) {
	if len(tokens) == 0 {
		return "", 2, fmt.Errorf("missing operand")
	}
	p := &parser{tokens: tokens}
	result, err := p.parseOr()
	if err != nil {
		return "", 2, err
	}
	if !p.done() {
		return "", 2, fmt.Errorf("syntax error: unexpected %q", p.peek())
	}
	return result, exitCode(result), nil
}

func exitCode(result string) int {
	// POSIX: exit 1 if result is null or zero
	if result == "" || result == "0" {
		return 1
	}
	return 0
}

func (p *parser) parseOr() (string, error) {
	left, err := p.parseAnd()
	if err != nil {
		return "", err
	}
	for p.peek() == "|" {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return "", err
		}
		if left != "" && left != "0" {
			// left is non-null/non-zero, result is left
		} else {
			left = right
		}
	}
	return left, nil
}

func (p *parser) parseAnd() (string, error) {
	left, err := p.parseCmp()
	if err != nil {
		return "", err
	}
	for p.peek() == "&" {
		p.next()
		right, err := p.parseCmp()
		if err != nil {
			return "", err
		}
		leftNull := left == "" || left == "0"
		rightNull := right == "" || right == "0"
		if leftNull || rightNull {
			left = "0"
		}
		// else: left remains as-is
	}
	return left, nil
}

func (p *parser) parseCmp() (string, error) {
	left, err := p.parseAdd()
	if err != nil {
		return "", err
	}
	for {
		op := p.peek()
		switch op {
		case "=", "!=", "<", "<=", ">", ">=":
			p.next()
			right, err := p.parseAdd()
			if err != nil {
				return "", err
			}
			left = cmpResult(left, op, right)
		default:
			return left, nil
		}
	}
}

func cmpResult(left, op, right string) string {
	// Try integer comparison first
	ln, errL := strconv.ParseInt(left, 10, 64)
	rn, errR := strconv.ParseInt(right, 10, 64)

	if errL == nil && errR == nil {
		var result bool
		switch op {
		case "=":
			result = ln == rn
		case "!=":
			result = ln != rn
		case "<":
			result = ln < rn
		case "<=":
			result = ln <= rn
		case ">":
			result = ln > rn
		case ">=":
			result = ln >= rn
		}
		if result {
			return "1"
		}
		return "0"
	}

	// Fall back to string comparison
	cmp := strings.Compare(left, right)
	var result bool
	switch op {
	case "=":
		result = cmp == 0
	case "!=":
		result = cmp != 0
	case "<":
		result = cmp < 0
	case "<=":
		result = cmp <= 0
	case ">":
		result = cmp > 0
	case ">=":
		result = cmp >= 0
	}
	if result {
		return "1"
	}
	return "0"
}

func (p *parser) parseAdd() (string, error) {
	left, err := p.parseMul()
	if err != nil {
		return "", err
	}
	for p.peek() == "+" || p.peek() == "-" {
		op := p.next()
		right, err := p.parseMul()
		if err != nil {
			return "", err
		}
		ln, errL := strconv.ParseInt(left, 10, 64)
		rn, errR := strconv.ParseInt(right, 10, 64)
		if errL != nil || errR != nil {
			return "", fmt.Errorf("non-integer argument")
		}
		if op == "+" {
			left = strconv.FormatInt(ln+rn, 10)
		} else {
			left = strconv.FormatInt(ln-rn, 10)
		}
	}
	return left, nil
}

func (p *parser) parseMul() (string, error) {
	left, err := p.parseUnary()
	if err != nil {
		return "", err
	}
	for p.peek() == "*" || p.peek() == "/" || p.peek() == "%" {
		op := p.next()
		right, err := p.parseUnary()
		if err != nil {
			return "", err
		}
		ln, errL := strconv.ParseInt(left, 10, 64)
		rn, errR := strconv.ParseInt(right, 10, 64)
		if errL != nil || errR != nil {
			return "", fmt.Errorf("non-integer argument")
		}
		switch op {
		case "*":
			left = strconv.FormatInt(ln*rn, 10)
		case "/":
			if rn == 0 {
				return "", fmt.Errorf("division by zero")
			}
			left = strconv.FormatInt(ln/rn, 10)
		case "%":
			if rn == 0 {
				return "", fmt.Errorf("division by zero")
			}
			left = strconv.FormatInt(ln%rn, 10)
		}
	}
	return left, nil
}

func (p *parser) parseUnary() (string, error) {
	tok := p.peek()

	switch tok {
	case "match":
		p.next()
		str, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		pat, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		return matchStr(str, pat)

	case "substr":
		p.next()
		str, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		posStr, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		lenStr, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		return substrStr(str, posStr, lenStr)

	case "index":
		p.next()
		str, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		chars, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		return indexStr(str, chars), nil

	case "length":
		p.next()
		// length can take an optional argument
		if p.done() || p.peek() == ")" || p.peek() == "|" || p.peek() == "&" {
			return "", fmt.Errorf("syntax error")
		}
		str, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		return strconv.Itoa(len(str)), nil
	}

	return p.parsePrimary()
}

func (p *parser) parsePrimary() (string, error) {
	tok := p.peek()

	if tok == "(" {
		p.next()
		result, err := p.parseOr()
		if err != nil {
			return "", err
		}
		if p.peek() != ")" {
			return "", fmt.Errorf("syntax error: expected ')'")
		}
		p.next()
		return result, nil
	}

	if tok == "" {
		return "", fmt.Errorf("syntax error: unexpected end of expression")
	}

	// The ':' infix operator: STRING : REGEXP
	// We handle it here by checking if the next token is ':'
	p.next() // consume atom

	if p.peek() == ":" {
		p.next()
		pat, err := p.parsePrimary()
		if err != nil {
			return "", err
		}
		return matchStr(tok, pat)
	}

	return tok, nil
}

// matchStr implements POSIX expr match: anchored at start.
// Returns the matched string if the pattern has a group, or the length of the match.
func matchStr(str, pattern string) (string, error) {
	// POSIX: regex is always anchored at start
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %v", err)
	}
	m := re.FindStringSubmatch(str)
	if m == nil {
		// No match: if pattern had groups, return empty string, else 0
		if strings.Contains(pattern, "(") {
			return "", nil
		}
		return "0", nil
	}
	if len(m) > 1 {
		// Return first captured group
		return m[1], nil
	}
	// Return length of match
	return strconv.Itoa(len(m[0])), nil
}

func substrStr(str, posStr, lenStr string) (string, error) {
	pos, err := strconv.Atoi(posStr)
	if err != nil {
		return "", fmt.Errorf("non-integer argument")
	}
	length, err := strconv.Atoi(lenStr)
	if err != nil {
		return "", fmt.Errorf("non-integer argument")
	}
	// POSIX: positions are 1-based
	if pos < 1 {
		pos = 1
	}
	start := pos - 1
	if start >= len(str) {
		return "", nil
	}
	end := start + length
	if end > len(str) {
		end = len(str)
	}
	return str[start:end], nil
}

func indexStr(str, chars string) string {
	// Return position of first char in 'chars' found in 'str' (1-based), or 0
	for i, c := range str {
		if strings.ContainsRune(chars, c) {
			return strconv.Itoa(i + 1)
		}
	}
	return "0"
}

func run(args []string, out io.Writer) int {
	// Manual flag parsing: only --json/-j is accepted as a flag.
	// Everything else (including negative numbers) is positional.
	jsonMode := false
	posArgs := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		} else {
			posArgs = append(posArgs, a)
		}
	}

	tokens := posArgs

	result, exitCode, err := Eval(tokens)
	if err != nil {
		common.RenderError("expr", 2, "SYNTAX", err.Error(), jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "expr: %v\n", err)
		}
		return 2
	}

	common.Render("expr", ExprResult{Result: result, ExitCode: exitCode}, jsonMode, out, func() {
		fmt.Fprintln(out, result)
	})
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "expr",
		Usage: "Evaluate expressions",
		Run:   run,
	})
}
