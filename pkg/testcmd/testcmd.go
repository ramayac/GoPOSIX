// Package testcmd implements the POSIX test / [ utility.
//
// POSIX test evaluates conditional expressions and returns exit code 0 (true)
// or 1 (false). When invoked as '[', a closing ']' argument is required.
package testcmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/ramayac/korego/internal/dispatch"
	"github.com/ramayac/korego/pkg/common"
)

// TestResult is the --json output.
type TestResult struct {
	Result bool `json:"result"`
}

// --- Library layer ---

// Evaluate evaluates a POSIX test expression given as tokens.
// Returns true/false and an optional error for syntax problems.
func Evaluate(tokens []string) (bool, error) {
	if len(tokens) == 0 {
		return false, nil // no args = false
	}
	p := &testParser{tokens: tokens}
	result, err := p.parseExpr()
	if err != nil {
		return false, err
	}
	if !p.done() {
		return false, fmt.Errorf("unexpected argument %q", p.peek())
	}
	return result, nil
}

type testParser struct {
	tokens []string
	pos    int
}

func (p *testParser) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}

func (p *testParser) next() string {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *testParser) done() bool {
	return p.pos >= len(p.tokens)
}

// Grammar:
//   expr     → orExpr
//   orExpr   → andExpr ( '-o' andExpr )*
//   andExpr  → notExpr ( '-a' notExpr )*
//   notExpr  → '!' notExpr | primary
//   primary  → '(' expr ')' | unaryTest | binaryTest | singleArg

func (p *testParser) parseExpr() (bool, error) {
	return p.parseOr()
}

func (p *testParser) parseOr() (bool, error) {
	left, err := p.parseAnd()
	if err != nil {
		return false, err
	}
	for p.peek() == "-o" {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return false, err
		}
		left = left || right
	}
	return left, nil
}

func (p *testParser) parseAnd() (bool, error) {
	left, err := p.parseNot()
	if err != nil {
		return false, err
	}
	for p.peek() == "-a" {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return false, err
		}
		left = left && right
	}
	return left, nil
}

func (p *testParser) parseNot() (bool, error) {
	if p.peek() == "!" {
		p.next()
		result, err := p.parseNot()
		if err != nil {
			return false, err
		}
		return !result, nil
	}
	return p.parsePrimary()
}

func (p *testParser) parsePrimary() (bool, error) {
	tok := p.peek()

	// Parenthesized expression
	if tok == "(" {
		p.next()
		result, err := p.parseExpr()
		if err != nil {
			return false, err
		}
		if p.peek() != ")" {
			return false, fmt.Errorf("missing ')'")
		}
		p.next()
		return result, nil
	}

	// Unary file/string tests
	switch tok {
	case "-e", "-f", "-d", "-s", "-r", "-w", "-x", "-L", "-h",
		"-b", "-c", "-p", "-S", "-g", "-u", "-k", "-t",
		"-z", "-n":
		p.next()
		if p.done() {
			return false, fmt.Errorf("missing argument for %s", tok)
		}
		arg := p.next()
		return evalUnary(tok, arg)
	}

	if p.done() {
		return false, fmt.Errorf("unexpected end of expression")
	}

	// Must be either a binary test or a bare string test.
	// Look ahead to see if next token is a binary operator.
	left := p.next()

	if !p.done() {
		op := p.peek()
		if isBinaryOp(op) {
			p.next()
			if p.done() {
				return false, fmt.Errorf("missing argument after %s", op)
			}
			right := p.next()
			return evalBinary(left, op, right)
		}
	}

	// Single arg: true if non-empty
	return left != "", nil
}

func isBinaryOp(op string) bool {
	switch op {
	case "=", "==", "!=",
		"-eq", "-ne", "-lt", "-le", "-gt", "-ge":
		return true
	}
	return false
}

func evalUnary(op, arg string) (bool, error) {
	switch op {
	case "-z":
		return arg == "", nil
	case "-n":
		return arg != "", nil
	}

	// File tests
	info, err := os.Lstat(arg)

	switch op {
	case "-e":
		return err == nil, nil
	case "-f":
		return err == nil && info.Mode().IsRegular(), nil
	case "-d":
		return err == nil && info.IsDir(), nil
	case "-s":
		return err == nil && info.Size() > 0, nil
	case "-L", "-h":
		return err == nil && info.Mode()&os.ModeSymlink != 0, nil
	case "-b":
		return err == nil && info.Mode()&os.ModeDevice != 0, nil
	case "-c":
		return err == nil && info.Mode()&os.ModeCharDevice != 0, nil
	case "-p":
		return err == nil && info.Mode()&os.ModeNamedPipe != 0, nil
	case "-S":
		return err == nil && info.Mode()&os.ModeSocket != 0, nil
	case "-g":
		return err == nil && info.Mode()&os.ModeSetgid != 0, nil
	case "-u":
		return err == nil && info.Mode()&os.ModeSetuid != 0, nil
	case "-k":
		return err == nil && info.Mode()&os.ModeSticky != 0, nil
	}

	// Permission tests need Stat (follows symlinks)
	if op == "-r" || op == "-w" || op == "-x" {
		info, err = os.Stat(arg)
		if err != nil {
			return false, nil
		}
		mode := info.Mode().Perm()
		switch op {
		case "-r":
			return mode&0444 != 0, nil
		case "-w":
			return mode&0222 != 0, nil
		case "-x":
			return mode&0111 != 0, nil
		}
	}

	return false, fmt.Errorf("unknown unary operator: %s", op)
}

func evalBinary(left, op, right string) (bool, error) {
	switch op {
	case "=", "==":
		return left == right, nil
	case "!=":
		return left != right, nil
	}

	// Integer comparisons
	ln, errL := strconv.ParseInt(left, 10, 64)
	rn, errR := strconv.ParseInt(right, 10, 64)
	if errL != nil || errR != nil {
		return false, fmt.Errorf("integer expression expected")
	}

	switch op {
	case "-eq":
		return ln == rn, nil
	case "-ne":
		return ln != rn, nil
	case "-lt":
		return ln < rn, nil
	case "-le":
		return ln <= rn, nil
	case "-gt":
		return ln > rn, nil
	case "-ge":
		return ln >= rn, nil
	}

	return false, fmt.Errorf("unknown binary operator: %s", op)
}

// spec for test command -- only the json flag (expression args are passthrough)
var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "j", Long: "json", Type: common.FlagBool},
	},
}

// --- CLI layer ---

func runTest(args []string, out io.Writer) int {
	// Pre-process: extract --json/-j before passing to flag parser
	// because test's expression language uses single-dash flags (-n, -z, -eq etc.)
	jsonMode := false
	cleanArgs := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "--json", "-j":
			jsonMode = true
		default:
			cleanArgs = append(cleanArgs, a)
		}
	}

	result, err := Evaluate(cleanArgs)
	if err != nil {
		common.RenderError("test", 2, "SYNTAX", err.Error(), jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "test: %v\n", err)
		}
		return 2
	}

	exitCode := 1
	if result {
		exitCode = 0
	}

	if jsonMode {
		common.Render("test", TestResult{Result: result}, true, out, func() {})
	}
	return exitCode
}

func runBracket(args []string, out io.Writer) int {
	// Pre-process: extract --json/-j
	jsonMode := false
	cleanArgs := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "--json", "-j":
			jsonMode = true
		default:
			cleanArgs = append(cleanArgs, a)
		}
	}

	// Validate closing ']'
	if len(cleanArgs) == 0 || cleanArgs[len(cleanArgs)-1] != "]" {
		msg := "missing ']'"
		common.RenderError("[", 2, "SYNTAX", msg, jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "[: %s\n", msg)
		}
		return 2
	}
	// Strip the closing ']' and delegate to test evaluation
	exprArgs := cleanArgs[:len(cleanArgs)-1]

	result, err := Evaluate(exprArgs)
	if err != nil {
		common.RenderError("[", 2, "SYNTAX", err.Error(), jsonMode, out)
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "[: %v\n", err)
		}
		return 2
	}

	exitCode := 1
	if result {
		exitCode = 0
	}

	if jsonMode {
		common.Render("[", TestResult{Result: result}, true, out, func() {})
	}
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "test",
		Usage: "Evaluate conditional expressions",
		Run:   runTest,
	})
	dispatch.Register(dispatch.Command{
		Name:  "[",
		Usage: "Evaluate conditional expressions (bracket form)",
		Run:   runBracket,
	})
}
