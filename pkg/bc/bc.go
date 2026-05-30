// Package bc implements the POSIX/BusyBox bc calculator utility.
package bc

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "l", Long: "mathlib", Type: common.FlagBool},
		{Short: "q", Long: "quiet", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// BcResult is the structured output for --json mode.
type BcResult struct {
	Lines []string `json:"lines"`
}

// TokenType defines the lexical token types for bc.
type TokenType int

const (
	TokEOF TokenType = iota
	TokError
	TokNewline
	TokNumber
	TokString
	TokIdent
	TokAssign
	TokPlus
	TokMinus
	TokMul
	TokDiv
	TokMod
	TokPower
	TokInc // ++
	TokDec // --
	TokEq  // ==
	TokLe  // <=
	TokGe  // >=
	TokNe  // !=
	TokLt  // <
	TokGt  // >
	TokAnd // &&
	TokOr  // ||
	TokNot // !
	TokLparen
	TokRparen
	TokLbracket
	TokRbracket
	TokLbrace
	TokRbrace
	TokComma
	TokSemicolon
	TokKeywordIf
	TokKeywordElse
	TokKeywordWhile
	TokKeywordFor
	TokKeywordDefine
	TokKeywordAuto
	TokKeywordReturn
	TokKeywordBreak
	TokKeywordContinue
	TokKeywordHalt
	TokKeywordPrint
	TokPlusAssign  // +=
	TokMinusAssign // -=
	TokMulAssign   // *=
	TokDivAssign   // /=
	TokPowAssign   // ^=
	TokModAssign   // %=
	TokDot         // . (alias for 'last')
)

type Token struct {
	Type TokenType
	Val  string
}

// Lexer tokenizes the bc input stream.
type Lexer struct {
	input []rune
	pos   int
}

func NewLexer(input string) *Lexer {
	// Pre-process: ignore backslash newlines
	var processed []rune
	runes := []rune(input)
	n := len(runes)
	for i := 0; i < n; i++ {
		if runes[i] == '\\' && i+1 < n && runes[i+1] == '\n' {
			i++ // skip both '\\' and '\n'
			continue
		}
		processed = append(processed, runes[i])
	}
	return &Lexer{input: processed, pos: 0}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	return ch
}

func (l *Lexer) NextToken() Token {
	for {
		ch := l.peek()
		if ch == 0 {
			return Token{Type: TokEOF}
		}

		// Skip comments and whitespaces (except newlines)
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.next()
			continue
		}

		// Line comments: # ...
		if ch == '#' {
			for l.peek() != '\n' && l.peek() != 0 {
				l.next()
			}
			continue
		}

		// Multiline comments: /* ... */
		if ch == '/' {
			l.next()
			if l.peek() == '*' {
				l.next()
				for {
					c := l.next()
					if c == 0 {
						break
					}
					if c == '*' && l.peek() == '/' {
						l.next()
						break
					}
				}
				continue
			} else if l.peek() == '=' {
				// /= compound assignment
				l.next()
				return Token{Type: TokDivAssign, Val: "/="}
			} else {
				// Division
				return Token{Type: TokDiv, Val: "/"}
			}
		}

		if ch == '\n' {
			l.next()
			return Token{Type: TokNewline, Val: "\n"}
		}

		// Division slash (re-evaluated after skipping comments)
		if ch == '/' {
			l.next()
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokDivAssign, Val: "/="}
			}
			return Token{Type: TokDiv, Val: "/"}
		}

		// Parentheses and delimiters
		switch ch {
		case '(':
			l.next()
			return Token{Type: TokLparen, Val: "("}
		case ')':
			l.next()
			return Token{Type: TokRparen, Val: ")"}
		case '[':
			l.next()
			return Token{Type: TokLbracket, Val: "["}
		case ']':
			l.next()
			return Token{Type: TokRbracket, Val: "]"}
		case '{':
			l.next()
			return Token{Type: TokLbrace, Val: "{"}
		case '}':
			l.next()
			return Token{Type: TokRbrace, Val: "}"}
		case ',':
			l.next()
			return Token{Type: TokComma, Val: ","}
		case ';':
			l.next()
			return Token{Type: TokSemicolon, Val: ";"}
		case '+':
			l.next()
			if l.peek() == '+' {
				l.next()
				return Token{Type: TokInc, Val: "++"}
			}
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokPlusAssign, Val: "+="}
			}
			return Token{Type: TokPlus, Val: "+"}
		case '-':
			l.next()
			if l.peek() == '-' {
				l.next()
				return Token{Type: TokDec, Val: "--"}
			}
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokMinusAssign, Val: "-="}
			}
			return Token{Type: TokMinus, Val: "-"}
		case '*':
			l.next()
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokMulAssign, Val: "*="}
			}
			return Token{Type: TokMul, Val: "*"}
		case '%':
			l.next()
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokModAssign, Val: "%="}
			}
			return Token{Type: TokMod, Val: "%"}
		case '^':
			l.next()
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokPowAssign, Val: "^="}
			}
			return Token{Type: TokPower, Val: "^"}
		case '.':
			// If '.' is followed by a digit, it's a number literal like .5
			// Otherwise it's the 'last' special variable
			nextPos := l.pos + 1
			if nextPos < len(l.input) && isDigit(l.input[nextPos]) {
				// Fall through to number parsing below — do nothing in the switch
			} else {
				l.next()
				return Token{Type: TokDot, Val: "."}
			}
		case '=':
			l.next()
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokEq, Val: "=="}
			}
			return Token{Type: TokAssign, Val: "="}
		case '<':
			l.next()
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokLe, Val: "<="}
			}
			return Token{Type: TokLt, Val: "<"}
		case '>':
			l.next()
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokGe, Val: ">="}
			}
			return Token{Type: TokGt, Val: ">"}
		case '!':
			l.next()
			if l.peek() == '=' {
				l.next()
				return Token{Type: TokNe, Val: "!="}
			}
			return Token{Type: TokNot, Val: "!"}
		case '&':
			l.next()
			if l.peek() == '&' {
				l.next()
				return Token{Type: TokAnd, Val: "&&"}
			}
			return Token{Type: TokError, Val: "&"}
		case '|':
			l.next()
			if l.peek() == '|' {
				l.next()
				return Token{Type: TokOr, Val: "||"}
			}
			return Token{Type: TokError, Val: "|"}
		case '"':
			l.next() // skip '"'
			var sb strings.Builder
			for {
				c := l.next()
				if c == 0 {
					return Token{Type: TokError, Val: "unterminated string"}
				}
				if c == '"' {
					break
				}
				if c == '\\' {
					nextChar := l.next()
					if nextChar == 0 {
						return Token{Type: TokError, Val: "unterminated string"}
					}
					sb.WriteRune('\\')
					sb.WriteRune(nextChar)
				} else {
					sb.WriteRune(c)
				}
			}
			return Token{Type: TokString, Val: sb.String()}
		}

		// Identifiers (keywords and variables)
		if isStartOfIdent(ch) {
			var sb strings.Builder
			for isLetter(l.peek()) || isDigit(l.peek()) {
				sb.WriteRune(l.next())
			}
			val := sb.String()
			switch val {
			case "if":
				return Token{Type: TokKeywordIf, Val: val}
			case "else":
				return Token{Type: TokKeywordElse, Val: val}
			case "while":
				return Token{Type: TokKeywordWhile, Val: val}
			case "for":
				return Token{Type: TokKeywordFor, Val: val}
			case "define":
				return Token{Type: TokKeywordDefine, Val: val}
			case "auto":
				return Token{Type: TokKeywordAuto, Val: val}
			case "return":
				return Token{Type: TokKeywordReturn, Val: val}
			case "break":
				return Token{Type: TokKeywordBreak, Val: val}
			case "continue":
				return Token{Type: TokKeywordContinue, Val: val}
			case "halt":
				return Token{Type: TokKeywordHalt, Val: val}
			case "print":
				return Token{Type: TokKeywordPrint, Val: val}
			default:
				isAllUpper := true
				for _, r := range val {
					if (r >= 'a' && r <= 'z') || r == '_' {
						isAllUpper = false
						break
					}
				}
				if isAllUpper && len(val) > 0 {
					return Token{Type: TokNumber, Val: val}
				}
				return Token{Type: TokIdent, Val: val}
			}
		}

		// Numbers: standard bc matches dot-separated hex/digits starting with digit, dot, or uppercase letter A-Z
		if isDigit(ch) || ch == '.' || (ch >= 'A' && ch <= 'Z') {
			var sb strings.Builder
			hasDot := false
			for {
				p := l.peek()
				if p == '.' {
					if hasDot {
						break
					}
					hasDot = true
					sb.WriteRune(l.next())
				} else if isDigit(p) || (p >= 'A' && p <= 'Z') {
					sb.WriteRune(l.next())
				} else {
					break
				}
			}
			val := sb.String()
			if val == "." {
				return Token{Type: TokError, Val: "."}
			}
			return Token{Type: TokNumber, Val: val}
		}

		// Unrecognized token
		l.next()
		return Token{Type: TokError, Val: string(ch)}
	}
}

func isStartOfIdent(c rune) bool {
	return (c >= 'a' && c <= 'z') || c == '_'
}

func isLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

// AST nodes for bc parser.
type Stmt interface{}
type Expr interface{}

type ExprStmt struct {
	Expr Expr
}

type BlockStmt struct {
	Stmts []Stmt
}

type IfStmt struct {
	Cond Expr
	Then Stmt
	Else Stmt
}

type WhileStmt struct {
	Cond Expr
	Body Stmt
}

type ForStmt struct {
	Init Expr
	Cond Expr
	Step Expr
	Body Stmt
}

type ReturnStmt struct {
	Val Expr
}

type BreakStmt struct{}
type ContinueStmt struct{}
type HaltStmt struct{}

type PrintStmt struct {
	Args []Expr
}

type NumExpr struct {
	Val string
}

type StringExpr struct {
	Val string
}

type VarExpr struct {
	Name string
}

type ArrayAccessExpr struct {
	Name  string
	Index Expr
}

type BinaryExpr struct {
	Op  TokenType
	Lhs Expr
	Rhs Expr
}

type UnaryExpr struct {
	Op         TokenType
	Expr       Expr
	PostIncDec bool // true for postfix ++/--
}

type CallExpr struct {
	Name string
	Args []Expr
}

type AssignExpr struct {
	Lhs Expr
	Rhs Expr
}

// ParenExpr wraps a parenthesized expression so isTopLevelPrintable
// can detect that (x = 5) should print but bare x = 5 should not.
type ParenExpr struct {
	Expr Expr
}

// Parser parses a slice of Tokens into Statement AST.
type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(lexer *Lexer) *Parser {
	var tokens []Token
	for {
		tok := lexer.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokEOF {
			break
		}
	}
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	tok := p.peek()
	if tok.Type != TokEOF {
		p.pos++
	}
	return tok
}

func (p *Parser) match(t TokenType) bool {
	if p.peek().Type == t {
		p.next()
		return true
	}
	return false
}

func (p *Parser) skipNewlines() {
	for p.peek().Type == TokNewline {
		p.next()
	}
}

func (p *Parser) Parse() ([]Stmt, error) {
	var stmts []Stmt
	for p.peek().Type != TokEOF {
		p.skipNewlines()
		if p.peek().Type == TokEOF {
			break
		}
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	return stmts, nil
}

func (p *Parser) parseStmt() (Stmt, error) {
	if p.match(TokSemicolon) {
		return nil, nil // empty statement
	}
	tok := p.peek()
	switch tok.Type {
	case TokLbrace:
		p.next()
		var stmts []Stmt
		for p.peek().Type != TokRbrace && p.peek().Type != TokEOF {
			p.skipNewlines()
			if p.peek().Type == TokRbrace {
				break
			}
			st, err := p.parseStmt()
			if err != nil {
				return nil, err
			}
			if st != nil {
				stmts = append(stmts, st)
			}
		}
		if !p.match(TokRbrace) {
			return nil, fmt.Errorf("expected }")
		}
		return &BlockStmt{Stmts: stmts}, nil

	case TokKeywordIf:
		p.next()
		if !p.match(TokLparen) {
			return nil, fmt.Errorf("expected (")
		}
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if !p.match(TokRparen) {
			return nil, fmt.Errorf("expected )")
		}
		p.skipNewlines()
		thenStmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		p.skipNewlines()
		var elseStmt Stmt
		if p.match(TokKeywordElse) {
			p.skipNewlines()
			elseStmt, err = p.parseStmt()
			if err != nil {
				return nil, err
			}
		}
		return &IfStmt{Cond: cond, Then: thenStmt, Else: elseStmt}, nil

	case TokKeywordWhile:
		p.next()
		if !p.match(TokLparen) {
			return nil, fmt.Errorf("expected (")
		}
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if !p.match(TokRparen) {
			return nil, fmt.Errorf("expected )")
		}
		p.skipNewlines()
		body, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		return &WhileStmt{Cond: cond, Body: body}, nil

	case TokKeywordFor:
		p.next()
		if !p.match(TokLparen) {
			return nil, fmt.Errorf("expected (")
		}
		var initExpr Expr
		var err error
		if p.peek().Type != TokSemicolon {
			initExpr, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		if !p.match(TokSemicolon) {
			return nil, fmt.Errorf("expected ; in for loop init")
		}
		var condExpr Expr
		if p.peek().Type != TokSemicolon {
			condExpr, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		if !p.match(TokSemicolon) {
			return nil, fmt.Errorf("expected ; in for loop cond")
		}
		var stepExpr Expr
		if p.peek().Type != TokRparen {
			stepExpr, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		if !p.match(TokRparen) {
			return nil, fmt.Errorf("expected ) in for loop step")
		}
		p.skipNewlines()
		body, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		return &ForStmt{Init: initExpr, Cond: condExpr, Step: stepExpr, Body: body}, nil

	case TokKeywordReturn:
		p.next()
		var retExpr Expr
		if p.peek().Type != TokNewline && p.peek().Type != TokSemicolon && p.peek().Type != TokEOF && p.peek().Type != TokRbrace {
			var err error
			retExpr, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		// consume optional semicolon or newline
		p.match(TokSemicolon)
		return &ReturnStmt{Val: retExpr}, nil

	case TokKeywordBreak:
		p.next()
		p.match(TokSemicolon)
		return &BreakStmt{}, nil

	case TokKeywordContinue:
		p.next()
		p.match(TokSemicolon)
		return &ContinueStmt{}, nil

	case TokKeywordHalt:
		p.next()
		p.match(TokSemicolon)
		return &HaltStmt{}, nil

	case TokKeywordPrint:
		p.next()
		var args []Expr
		for {
			expr, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			args = append(args, expr)
			if !p.match(TokComma) {
				break
			}
		}
		p.match(TokSemicolon)
		return &PrintStmt{Args: args}, nil

	case TokKeywordDefine:
		return p.parseDefine()

	default:
		// Expression statement
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		p.match(TokSemicolon)
		p.match(TokNewline)
		return &ExprStmt{Expr: expr}, nil
	}
}

type FuncDecl struct {
	Name        string
	Params      []string
	ParamArrays []bool
	ParamRefs   []bool
	Autos       []string
	AutoArrays  []string
	Body        Stmt
	Void        bool
}

func (p *Parser) parseDefine() (Stmt, error) {
	p.next() // skip 'define'
	isVoid := false
	if p.peek().Type == TokIdent && p.peek().Val == "void" {
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokIdent {
			p.next() // skip void keyword
			isVoid = true
		}
	}
	nameTok := p.next()
	if nameTok.Type != TokIdent {
		return nil, fmt.Errorf("expected function name")
	}
	if !p.match(TokLparen) {
		return nil, fmt.Errorf("expected (")
	}
	var params []string
	var paramArrays []bool
	var paramRefs []bool
	for p.peek().Type != TokRparen && p.peek().Type != TokEOF {
		isRef := false
		if p.match(TokMul) {
			isRef = true
		}
		paramTok := p.next()
		if paramTok.Type != TokIdent {
			return nil, fmt.Errorf("expected param name")
		}
		params = append(params, paramTok.Val)
		isArray := false
		if p.match(TokLbracket) {
			if !p.match(TokRbracket) {
				return nil, fmt.Errorf("expected ] in param array")
			}
			isArray = true
		}
		if isRef && !isArray {
			return nil, fmt.Errorf("reference operator * only allowed on array parameters")
		}
		paramArrays = append(paramArrays, isArray)
		paramRefs = append(paramRefs, isRef)
		if !p.match(TokComma) {
			break
		}
	}
	if !p.match(TokRparen) {
		return nil, fmt.Errorf("expected )")
	}

	p.skipNewlines()
	if !p.match(TokLbrace) {
		return nil, fmt.Errorf("expected {")
	}

	var autos []string
	var autoArrays []string
	p.skipNewlines()
	if p.match(TokKeywordAuto) {
		for {
			autoTok := p.next()
			if autoTok.Type != TokIdent {
				return nil, fmt.Errorf("expected auto variable name")
			}
			isArray := false
			if p.match(TokLbracket) {
				if !p.match(TokRbracket) {
					return nil, fmt.Errorf("expected ] in auto array")
				}
				isArray = true
			}
			if isArray {
				autoArrays = append(autoArrays, autoTok.Val)
			} else {
				autos = append(autos, autoTok.Val)
			}
			if !p.match(TokComma) {
				break
			}
		}
		p.match(TokSemicolon)
		p.skipNewlines()
	}

	var stmts []Stmt
	for p.peek().Type != TokRbrace && p.peek().Type != TokEOF {
		p.skipNewlines()
		if p.peek().Type == TokRbrace {
			break
		}
		st, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		if st != nil {
			stmts = append(stmts, st)
		}
	}
	if !p.match(TokRbrace) {
		return nil, fmt.Errorf("expected } in define body")
	}

	return &FuncDecl{
		Name:        nameTok.Val,
		Params:      params,
		ParamArrays: paramArrays,
		ParamRefs:   paramRefs,
		Autos:       autos,
		AutoArrays:  autoArrays,
		Body:        &BlockStmt{Stmts: stmts},
		Void:        isVoid,
	}, nil
}

// Expression parsing with operator precedence.
func (p *Parser) parseExpr() (Expr, error) {
	return p.parseAssignment()
}

func (p *Parser) parseAssignment() (Expr, error) {
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	tok := p.peek()
	switch tok.Type {
	case TokAssign:
		p.next()
		rhs, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		return &AssignExpr{Lhs: expr, Rhs: rhs}, nil
	case TokPlusAssign, TokMinusAssign, TokMulAssign, TokDivAssign, TokPowAssign, TokModAssign:
		p.next()
		rhs, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}
		// Desugar: x op= y  →  x = x op y
		var binOp TokenType
		switch tok.Type {
		case TokPlusAssign:
			binOp = TokPlus
		case TokMinusAssign:
			binOp = TokMinus
		case TokMulAssign:
			binOp = TokMul
		case TokDivAssign:
			binOp = TokDiv
		case TokPowAssign:
			binOp = TokPower
		case TokModAssign:
			binOp = TokMod
		}
		return &AssignExpr{Lhs: expr, Rhs: &BinaryExpr{Op: binOp, Lhs: expr, Rhs: rhs}}, nil
	}
	return expr, nil
}

func (p *Parser) parseOr() (Expr, error) {
	expr, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokOr {
		tok := p.next()
		rhs, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: tok.Type, Lhs: expr, Rhs: rhs}
	}
	return expr, nil
}

func (p *Parser) parseAnd() (Expr, error) {
	expr, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokAnd {
		tok := p.next()
		rhs, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: tok.Type, Lhs: expr, Rhs: rhs}
	}
	return expr, nil
}

func (p *Parser) parseComparison() (Expr, error) {
	expr, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek().Type
		if t == TokEq || t == TokNe || t == TokLt || t == TokLe || t == TokGt || t == TokGe {
			tok := p.next()
			rhs, err := p.parseAddSub()
			if err != nil {
				return nil, err
			}
			expr = &BinaryExpr{Op: tok.Type, Lhs: expr, Rhs: rhs}
		} else {
			break
		}
	}
	return expr, nil
}

func (p *Parser) parseAddSub() (Expr, error) {
	expr, err := p.parseMulDivMod()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek().Type
		if t == TokPlus || t == TokMinus {
			tok := p.next()
			rhs, err := p.parseMulDivMod()
			if err != nil {
				return nil, err
			}
			expr = &BinaryExpr{Op: tok.Type, Lhs: expr, Rhs: rhs}
		} else {
			break
		}
	}
	return expr, nil
}

func (p *Parser) parseMulDivMod() (Expr, error) {
	expr, err := p.parsePower()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek().Type
		if t == TokMul || t == TokDiv || t == TokMod {
			tok := p.next()
			rhs, err := p.parsePower()
			if err != nil {
				return nil, err
			}
			expr = &BinaryExpr{Op: tok.Type, Lhs: expr, Rhs: rhs}
		} else {
			break
		}
	}
	return expr, nil
}

func (p *Parser) parsePower() (Expr, error) {
	expr, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if p.peek().Type == TokPower {
		tok := p.next()
		rhs, err := p.parsePower() // Right associative
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: tok.Type, Lhs: expr, Rhs: rhs}
	}
	return expr, nil
}

func (p *Parser) parseUnary() (Expr, error) {
	t := p.peek().Type
	if t == TokMinus || t == TokPlus || t == TokNot || t == TokInc || t == TokDec {
		tok := p.next()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: tok.Type, Expr: expr, PostIncDec: false}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (Expr, error) {
	tok := p.peek()
	switch tok.Type {
	case TokNumber:
		p.next()
		return &NumExpr{Val: tok.Val}, nil
	case TokString:
		p.next()
		return &StringExpr{Val: tok.Val}, nil
	case TokIdent:
		p.next()
		name := tok.Val
		// Function call
		if p.match(TokLparen) {
			var args []Expr
			for p.peek().Type != TokRparen && p.peek().Type != TokEOF {
				arg, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if !p.match(TokComma) {
					break
				}
			}
			if !p.match(TokRparen) {
				return nil, fmt.Errorf("expected ) in call")
			}
			return &CallExpr{Name: name, Args: args}, nil
		}
		// Array access
		if p.match(TokLbracket) {
			if p.match(TokRbracket) {
				return &ArrayAccessExpr{Name: name, Index: nil}, nil
			}
			idx, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if !p.match(TokRbracket) {
				return nil, fmt.Errorf("expected ] in array access")
			}
			return &ArrayAccessExpr{Name: name, Index: idx}, nil
		}
		// Variable expression.
		// Check for postfix ++/--
		expr := Expr(&VarExpr{Name: name})
		if p.peek().Type == TokInc || p.peek().Type == TokDec {
			postTok := p.next()
			expr = &UnaryExpr{Op: postTok.Type, Expr: expr, PostIncDec: true}
		}
		return expr, nil

	case TokLparen:
		p.next()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if !p.match(TokRparen) {
			return nil, fmt.Errorf("expected )")
		}
		return &ParenExpr{Expr: expr}, nil

	case TokDot:
		p.next()
		// '.' is an alias for the 'last' special variable
		expr := Expr(&VarExpr{Name: "last"})
		if p.peek().Type == TokInc || p.peek().Type == TokDec {
			postTok := p.next()
			expr = &UnaryExpr{Op: postTok.Type, Expr: expr, PostIncDec: true}
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unexpected token %v", tok)
	}
}

// Interpreter execution state and variables.
type ValType int

const (
	ValNum ValType = iota
	ValStr
	ValVoid
	ValArrayRef
)

type Val struct {
	Type     ValType
	Rat      *big.Rat
	Str      string
	Scale    int
	IsNeg    bool
	ArrayRef map[string]Val
}

func newValNum(r *big.Rat, scale int) Val {
	return Val{Type: ValNum, Rat: r, Scale: scale, IsNeg: r.Sign() < 0}
}

func newValNumNeg(r *big.Rat, scale int, isNeg bool) Val {
	return Val{Type: ValNum, Rat: r, Scale: scale, IsNeg: isNeg || r.Sign() < 0}
}

func newValStr(s string) Val {
	return Val{Type: ValStr, Str: s, Scale: 0}
}

func newValVoid() Val {
	return Val{Type: ValVoid, Scale: 0}
}

func newValArrayRef(arr map[string]Val) Val {
	return Val{Type: ValArrayRef, ArrayRef: arr, Scale: 0}
}

func (v Val) IsTrue() bool {
	if v.Type != ValNum {
		return false
	}
	return v.Rat.Sign() != 0
}

type Scope struct {
	Vars   map[string]Val
	Arrays map[string]map[string]Val
	Parent *Scope
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		Vars:   make(map[string]Val),
		Arrays: make(map[string]map[string]Val),
		Parent: parent,
	}
}

func (s *Scope) Get(name string) Val {
	if val, ok := s.Vars[name]; ok {
		return val
	}
	if s.Parent != nil {
		return s.Parent.Get(name)
	}
	return newValNum(big.NewRat(0, 1), 0)
}

func (s *Scope) Set(name string, val Val) {
	curr := s
	for curr != nil {
		if _, ok := curr.Vars[name]; ok {
			curr.Vars[name] = val
			return
		}
		curr = curr.Parent
	}
	s.Vars[name] = val
}

func (s *Scope) GetArray(name string, index string) Val {
	curr := s
	for curr != nil {
		if arr, ok := curr.Arrays[name]; ok {
			if val, exists := arr[index]; exists {
				return val
			}
			return newValNum(big.NewRat(0, 1), 0)
		}
		curr = curr.Parent
	}
	return newValNum(big.NewRat(0, 1), 0)
}

func (s *Scope) SetArray(name string, index string, val Val) {
	curr := s
	for curr != nil {
		if arr, ok := curr.Arrays[name]; ok {
			arr[index] = val
			return
		}
		curr = curr.Parent
	}
	if s.Arrays[name] == nil {
		s.Arrays[name] = make(map[string]Val)
	}
	s.Arrays[name][index] = val
}

func (s *Scope) GetArrayRef(name string) map[string]Val {
	curr := s
	for curr != nil {
		if arr, ok := curr.Arrays[name]; ok {
			return arr
		}
		curr = curr.Parent
	}
	arr := make(map[string]Val)
	s.Arrays[name] = arr
	return arr
}

type Interpreter struct {
	Globals     *Scope
	Locals      *Scope
	Functions   map[string]*FuncDecl
	Scale       int
	Ibase       int
	Obase       int
	Stdout      io.Writer
	Stdin       io.Reader
	StdinReader *bufio.Reader
	Halted      bool
	Last        Val // 'last' / '.' special variable
}

func NewInterpreter(stdout io.Writer, stdin io.Reader, mathLib bool) *Interpreter {
	g := NewScope(nil)
	scale := 0
	if mathLib {
		scale = 20
	}
	var stdinReader *bufio.Reader
	if stdin != nil {
		stdinReader = bufio.NewReader(stdin)
	}
	ip := &Interpreter{
		Globals:     g,
		Locals:      g,
		Functions:   make(map[string]*FuncDecl),
		Scale:       scale,
		Ibase:       10,
		Obase:       10,
		Stdout:      stdout,
		Stdin:       stdin,
		StdinReader: stdinReader,
		Last:        newValNum(big.NewRat(0, 1), 0),
	}
	if mathLib {
		lex := NewLexer(MathLibSource)
		parser := NewParser(lex)
		stmts, err := parser.Parse()
		if err != nil {
			panic("failed to parse math lib: " + err.Error())
		}
		err = ip.Execute(stmts)
		if err != nil {
			panic("failed to execute math lib: " + err.Error())
		}
	}
	return ip
}

func (ip *Interpreter) Execute(stmts []Stmt) error {
	for _, stmt := range stmts {
		if ip.Halted {
			break
		}
		_, err := ip.execStmt(stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

type FlowControl int

const (
	FlowNormal FlowControl = iota
	FlowReturn
	FlowBreak
	FlowContinue
)

type ExecResult struct {
	Flow FlowControl
	Val  Val
}

func (ip *Interpreter) execStmt(stmt Stmt) (ExecResult, error) {
	switch s := stmt.(type) {
	case *FuncDecl:
		ip.Functions[s.Name] = s
		return ExecResult{Flow: FlowNormal}, nil

	case *ExprStmt:
		val, err := ip.eval(s.Expr)
		if err != nil {
			return ExecResult{Flow: FlowNormal}, err
		}
		if val.Type == ValNum && isTopLevelPrintable(s.Expr) {
			ip.Last = val // track 'last' printed value
			printWrapped(ip.Stdout, formatRat(val.Rat, ip.Obase, val.Scale, val.IsNeg))
			fmt.Fprintln(ip.Stdout)
		} else if val.Type == ValStr && isTopLevelPrintable(s.Expr) {
			fmt.Fprint(ip.Stdout, val.Str)
		}
		return ExecResult{Flow: FlowNormal}, nil

	case *BlockStmt:
		for _, st := range s.Stmts {
			res, err := ip.execStmt(st)
			if err != nil {
				return ExecResult{Flow: FlowNormal}, err
			}
			if res.Flow != FlowNormal {
				return res, nil
			}
		}
		return ExecResult{Flow: FlowNormal}, nil

	case *IfStmt:
		condVal, err := ip.eval(s.Cond)
		if err != nil {
			return ExecResult{Flow: FlowNormal}, err
		}
		if condVal.IsTrue() {
			return ip.execStmt(s.Then)
		} else if s.Else != nil {
			return ip.execStmt(s.Else)
		}
		return ExecResult{Flow: FlowNormal}, nil

	case *WhileStmt:
		for {
			condVal, err := ip.eval(s.Cond)
			if err != nil {
				return ExecResult{Flow: FlowNormal}, err
			}
			if !condVal.IsTrue() {
				break
			}
			res, err := ip.execStmt(s.Body)
			if err != nil {
				return ExecResult{Flow: FlowNormal}, err
			}
			if res.Flow == FlowReturn {
				return res, nil
			}
			if res.Flow == FlowBreak {
				break
			}
		}
		return ExecResult{Flow: FlowNormal}, nil

	case *ForStmt:
		if s.Init != nil {
			_, err := ip.eval(s.Init)
			if err != nil {
				return ExecResult{Flow: FlowNormal}, err
			}
		}
		for {
			if s.Cond != nil {
				condVal, err := ip.eval(s.Cond)
				if err != nil {
					return ExecResult{Flow: FlowNormal}, err
				}
				if !condVal.IsTrue() {
					break
				}
			}
			res, err := ip.execStmt(s.Body)
			if err != nil {
				return ExecResult{Flow: FlowNormal}, err
			}
			if res.Flow == FlowReturn {
				return res, nil
			}
			if res.Flow == FlowBreak {
				break
			}
			if s.Step != nil {
				_, err := ip.eval(s.Step)
				if err != nil {
					return ExecResult{Flow: FlowNormal}, err
				}
			}
		}
		return ExecResult{Flow: FlowNormal}, nil

	case *ReturnStmt:
		val := newValVoid()
		if s.Val != nil {
			var err error
			val, err = ip.eval(s.Val)
			if err != nil {
				return ExecResult{Flow: FlowNormal}, err
			}
		}
		return ExecResult{Flow: FlowReturn, Val: val}, nil

	case *BreakStmt:
		return ExecResult{Flow: FlowBreak}, nil

	case *ContinueStmt:
		return ExecResult{Flow: FlowContinue}, nil

	case *HaltStmt:
		ip.Halted = true
		return ExecResult{Flow: FlowNormal}, nil

	case *PrintStmt:
		for _, arg := range s.Args {
			val, err := ip.eval(arg)
			if err != nil {
				return ExecResult{Flow: FlowNormal}, err
			}
			if val.Type == ValNum {
				printWrapped(ip.Stdout, formatRat(val.Rat, ip.Obase, val.Scale, val.IsNeg))
			} else if val.Type == ValStr {
				fmt.Fprint(ip.Stdout, unescapeBcString(val.Str))
			}
		}
		return ExecResult{Flow: FlowNormal}, nil
	}
	return ExecResult{Flow: FlowNormal}, nil
}

func isTopLevelPrintable(expr Expr) bool {
	switch expr.(type) {
	case *AssignExpr:
		// Bare assignment: x = 5 → don't print
		return false
	case *ParenExpr:
		// Parenthesized expression: (x = 5) → always print
		return true
	default:
		return true
	}
}

func (ip *Interpreter) eval(expr Expr) (Val, error) {
	switch e := expr.(type) {
	case *ParenExpr:
		// Transparently evaluate inner expression
		return ip.eval(e.Expr)

	case *NumExpr:
		r, err := parseNumberInBase(e.Val, ip.Ibase)
		if err != nil {
			return newValVoid(), err
		}
		scale := 0
		if idx := strings.Index(e.Val, "."); idx != -1 {
			scale = len(e.Val) - idx - 1
		}
		return newValNum(r, scale), nil

	case *StringExpr:
		return newValStr(e.Val), nil

	case *VarExpr:
		switch e.Name {
		case "scale":
			return newValNum(big.NewRat(int64(ip.Scale), 1), 0), nil
		case "ibase":
			return newValNum(big.NewRat(int64(ip.Ibase), 1), 0), nil
		case "obase":
			return newValNum(big.NewRat(int64(ip.Obase), 1), 0), nil
		case "last":
			return ip.Last, nil
		default:
			return ip.Locals.Get(e.Name), nil
		}

	case *ArrayAccessExpr:
		if e.Index == nil {
			arr := ip.Locals.GetArrayRef(e.Name)
			return newValArrayRef(arr), nil
		}
		idxVal, err := ip.eval(e.Index)
		if err != nil {
			return newValVoid(), err
		}
		idxStr := idxString(idxVal.Rat)
		return ip.Locals.GetArray(e.Name, idxStr), nil

	case *AssignExpr:
		rhsVal, err := ip.eval(e.Rhs)
		if err != nil {
			return newValVoid(), err
		}
		if rhsVal.Type != ValNum {
			return newValVoid(), fmt.Errorf("cannot assign non-numeric value")
		}

		switch lhs := e.Lhs.(type) {
		case *VarExpr:
			switch lhs.Name {
			case "scale":
				ip.Scale = int(ratToInt64(rhsVal.Rat))
			case "ibase":
				ip.Ibase = int(ratToInt64(rhsVal.Rat))
				if ip.Ibase < 2 {
					ip.Ibase = 2
				}
				if ip.Ibase > 36 {
					ip.Ibase = 36
				}
			case "obase":
				ip.Obase = int(ratToInt64(rhsVal.Rat))
			case "last":
				ip.Last = rhsVal
			default:
				ip.Locals.Set(lhs.Name, rhsVal)
			}
		case *ArrayAccessExpr:
			idxVal, err := ip.eval(lhs.Index)
			if err != nil {
				return newValVoid(), err
			}
			idxStr := idxString(idxVal.Rat)
			ip.Locals.SetArray(lhs.Name, idxStr, rhsVal)
		default:
			return newValVoid(), fmt.Errorf("invalid left-hand side of assignment")
		}
		return rhsVal, nil

	case *BinaryExpr:
		lhsVal, err := ip.eval(e.Lhs)
		if err != nil {
			return newValVoid(), err
		}
		rhsVal, err := ip.eval(e.Rhs)
		if err != nil {
			return newValVoid(), err
		}

		if lhsVal.Type != ValNum || rhsVal.Type != ValNum {
			return newValVoid(), fmt.Errorf("binary operations only supported on numbers")
		}

		res := big.NewRat(0, 1)
		resScale := 0
		switch e.Op {
		case TokPlus, TokMinus:
			if e.Op == TokPlus {
				res.Add(lhsVal.Rat, rhsVal.Rat)
			} else {
				res.Sub(lhsVal.Rat, rhsVal.Rat)
			}
			resScale = lhsVal.Scale
			if rhsVal.Scale > resScale {
				resScale = rhsVal.Scale
			}
		case TokMul:
			res.Mul(lhsVal.Rat, rhsVal.Rat)
			resScale = lhsVal.Scale + rhsVal.Scale
			maxInherent := lhsVal.Scale
			if rhsVal.Scale > maxInherent {
				maxInherent = rhsVal.Scale
			}
			limit := ip.Scale
			if maxInherent > limit {
				limit = maxInherent
			}
			if resScale > limit {
				resScale = limit
			}
			res = truncateRat(res, resScale)
			if res.Sign() == 0 {
				resScale = 0
			}
		case TokDiv:
			if rhsVal.Rat.Sign() == 0 {
				return newValVoid(), fmt.Errorf("division by zero")
			}
			res.Quo(lhsVal.Rat, rhsVal.Rat)
			resScale = ip.Scale
			res = truncateRat(res, resScale)
			if res.Sign() == 0 {
				resScale = 0
			}
		case TokMod:
			if rhsVal.Rat.Sign() == 0 {
				return newValVoid(), fmt.Errorf("modulo by zero")
			}
			div := big.NewRat(0, 1).Quo(lhsVal.Rat, rhsVal.Rat)
			divTruncated := truncateRat(div, ip.Scale)
			term := big.NewRat(0, 1).Mul(rhsVal.Rat, divTruncated)
			res.Sub(lhsVal.Rat, term)

			resScale = ip.Scale + rhsVal.Scale
			if lhsVal.Scale > resScale {
				resScale = lhsVal.Scale
			}
			res = truncateRat(res, resScale)
			if res.Sign() == 0 {
				resScale = 0
			}
		case TokPower:
			exponent := ratToInt64(rhsVal.Rat)
			res = ratPower(lhsVal.Rat, exponent)

			if res.Sign() == 0 {
				resScale = 0
			} else if exponent < 0 {
				resScale = ip.Scale
			} else {
				resScale = lhsVal.Scale * int(exponent)
				limit := ip.Scale
				if lhsVal.Scale > limit {
					limit = lhsVal.Scale
				}
				if resScale > limit {
					resScale = limit
				}
			}
			res = truncateRat(res, resScale)
			if res.Sign() == 0 {
				resScale = 0
			}
		case TokEq, TokNe, TokLt, TokLe, TokGt, TokGe, TokAnd, TokOr:
			resScale = 0
			switch e.Op {
			case TokEq:
				if lhsVal.Rat.Cmp(rhsVal.Rat) == 0 {
					res.SetInt64(1)
				}
			case TokNe:
				if lhsVal.Rat.Cmp(rhsVal.Rat) != 0 {
					res.SetInt64(1)
				}
			case TokLt:
				if lhsVal.Rat.Cmp(rhsVal.Rat) < 0 {
					res.SetInt64(1)
				}
			case TokLe:
				if lhsVal.Rat.Cmp(rhsVal.Rat) <= 0 {
					res.SetInt64(1)
				}
			case TokGt:
				if lhsVal.Rat.Cmp(rhsVal.Rat) > 0 {
					res.SetInt64(1)
				}
			case TokGe:
				if lhsVal.Rat.Cmp(rhsVal.Rat) >= 0 {
					res.SetInt64(1)
				}
			case TokAnd:
				if lhsVal.IsTrue() && rhsVal.IsTrue() {
					res.SetInt64(1)
				}
			case TokOr:
				if lhsVal.IsTrue() || rhsVal.IsTrue() {
					res.SetInt64(1)
				}
			}
		default:
			return newValVoid(), fmt.Errorf("unrecognized binary operator")
		}
		return newValNum(res, resScale), nil

	case *UnaryExpr:
		if e.Op == TokInc || e.Op == TokDec {
			var varName string
			var isArray bool
			var arrayIdx string

			switch target := e.Expr.(type) {
			case *VarExpr:
				varName = target.Name
			case *ArrayAccessExpr:
				isArray = true
				varName = target.Name
				idxVal, err := ip.eval(target.Index)
				if err != nil {
					return newValVoid(), err
				}
				arrayIdx = idxString(idxVal.Rat)
			default:
				return newValVoid(), fmt.Errorf("invalid increment target")
			}

			// Handle special variables for ++/--
			switch varName {
			case "scale", "ibase", "obase", "last":
				curr, _ := ip.eval(&VarExpr{Name: varName})
				next := big.NewRat(0, 1)
				if e.Op == TokInc {
					next.Add(curr.Rat, big.NewRat(1, 1))
				} else {
					next.Sub(curr.Rat, big.NewRat(1, 1))
				}
				nextVal := newValNum(next, 0)
				switch varName {
				case "scale":
					ip.Scale = int(ratToInt64(next))
				case "ibase":
					v := int(ratToInt64(next))
					if v < 2 {
						v = 2
					}
					if v > 36 {
						v = 36
					}
					ip.Ibase = v
				case "obase":
					ip.Obase = int(ratToInt64(next))
				case "last":
					ip.Last = nextVal
				}
				if e.PostIncDec {
					return curr, nil
				}
				return nextVal, nil
			}

			var curr Val
			if isArray {
				curr = ip.Locals.GetArray(varName, arrayIdx)
			} else {
				curr = ip.Locals.Get(varName)
			}

			next := big.NewRat(0, 1)
			if e.Op == TokInc {
				next.Add(curr.Rat, big.NewRat(1, 1))
			} else {
				next.Sub(curr.Rat, big.NewRat(1, 1))
			}

			nextVal := newValNum(next, curr.Scale)
			if isArray {
				ip.Locals.SetArray(varName, arrayIdx, nextVal)
			} else {
				ip.Locals.Set(varName, nextVal)
			}

			if e.PostIncDec {
				return curr, nil
			}
			return nextVal, nil
		}

		val, err := ip.eval(e.Expr)
		if err != nil {
			return newValVoid(), err
		}
		if val.Type != ValNum {
			return newValVoid(), fmt.Errorf("unary operations only supported on numbers")
		}

		res := big.NewRat(0, 1)
		isNeg := false
		switch e.Op {
		case TokMinus:
			res.Neg(val.Rat)
			isNeg = !val.IsNeg
		case TokPlus:
			res.Set(val.Rat)
			isNeg = val.IsNeg
		case TokNot:
			if !val.IsTrue() {
				res.SetInt64(1)
			}
		default:
			return newValVoid(), fmt.Errorf("unrecognized unary operator")
		}
		return newValNumNeg(res, val.Scale, isNeg), nil

	case *CallExpr:
		if e.Name == "length" {
			if len(e.Args) != 1 {
				return newValVoid(), fmt.Errorf("length() expects exactly 1 argument")
			}
			argVal, err := ip.eval(e.Args[0])
			if err != nil {
				return newValVoid(), err
			}
			if argVal.Type == ValArrayRef {
				length := arrayLength(argVal.ArrayRef)
				return newValNum(big.NewRat(int64(length), 1), 0), nil
			}
			length := valLength(argVal.Rat, argVal.Scale)
			return newValNum(big.NewRat(int64(length), 1), 0), nil
		}

		if e.Name == "scale" {
			if len(e.Args) != 1 {
				return newValVoid(), fmt.Errorf("scale() expects exactly 1 argument")
			}
			argVal, err := ip.eval(e.Args[0])
			if err != nil {
				return newValVoid(), err
			}
			return newValNum(big.NewRat(int64(argVal.Scale), 1), 0), nil
		}

		if e.Name == "sqrt" {
			if len(e.Args) != 1 {
				return newValVoid(), fmt.Errorf("sqrt() expects exactly 1 argument")
			}
			argVal, err := ip.eval(e.Args[0])
			if err != nil {
				return newValVoid(), err
			}
			resRat, resScale, err := valSqrt(argVal.Rat, argVal.Scale, ip.Scale)
			if err != nil {
				return newValVoid(), err
			}
			return newValNum(resRat, resScale), nil
		}

		if e.Name == "read" {
			if len(e.Args) > 0 {
				return newValVoid(), fmt.Errorf("read() expects no arguments")
			}
			if ip.StdinReader == nil {
				return newValNum(big.NewRat(0, 1), 0), nil
			}
			line, err := ip.StdinReader.ReadString('\n')
			if err != nil && err != io.EOF {
				return newValVoid(), err
			}
			line = strings.TrimSpace(line)
			if line == "" && err == io.EOF {
				return newValNum(big.NewRat(0, 1), 0), nil
			}
			lex := NewLexer(line)
			parser := NewParser(lex)
			stmts, parseErr := parser.Parse()
			if parseErr != nil {
				return newValVoid(), parseErr
			}
			var finalVal Val = newValNum(big.NewRat(0, 1), 0)
			for _, stmt := range stmts {
				if exprSt, ok := stmt.(*ExprStmt); ok {
					var evalErr error
					finalVal, evalErr = ip.eval(exprSt.Expr)
					if evalErr != nil {
						return newValVoid(), evalErr
					}
				} else {
					_, evalErr := ip.execStmt(stmt)
					if evalErr != nil {
						return newValVoid(), evalErr
					}
				}
			}
			return finalVal, nil
		}

		decl, exists := ip.Functions[e.Name]
		if !exists {
			return newValVoid(), fmt.Errorf("undefined function %s", e.Name)
		}

		if len(e.Args) != len(decl.Params) {
			return newValVoid(), fmt.Errorf("function %s expects %d arguments, got %d", e.Name, len(decl.Params), len(e.Args))
		}

		var argVals []Val
		for _, argExpr := range e.Args {
			val, err := ip.eval(argExpr)
			if err != nil {
				return newValVoid(), err
			}
			argVals = append(argVals, val)
		}

		prevScope := ip.Locals
		newScope := NewScope(ip.Globals)

		for idx, paramName := range decl.Params {
			if idx < len(decl.ParamArrays) && decl.ParamArrays[idx] {
				argVal := argVals[idx]
				if argVal.Type == ValArrayRef {
					if idx < len(decl.ParamRefs) && decl.ParamRefs[idx] {
						newScope.Arrays[paramName] = argVal.ArrayRef
					} else {
						copiedArr := make(map[string]Val)
						for k, v := range argVal.ArrayRef {
							copiedArr[k] = v
						}
						newScope.Arrays[paramName] = copiedArr
					}
				} else {
					newScope.Arrays[paramName] = make(map[string]Val)
				}
			} else {
				newScope.Vars[paramName] = argVals[idx]
			}
		}
		for _, autoName := range decl.Autos {
			newScope.Vars[autoName] = newValNum(big.NewRat(0, 1), 0)
		}
		for _, autoArrName := range decl.AutoArrays {
			newScope.Arrays[autoArrName] = make(map[string]Val)
		}

		ip.Locals = newScope
		res, err := ip.execStmt(decl.Body)
		ip.Locals = prevScope

		if err != nil {
			return newValVoid(), err
		}

		if res.Flow == FlowReturn {
			if decl.Void {
				return newValVoid(), nil
			}
			if res.Val.Type == ValVoid {
				return newValNum(big.NewRat(0, 1), 0), nil
			}
			return res.Val, nil
		}
		if decl.Void {
			return newValVoid(), nil
		}
		return newValNum(big.NewRat(0, 1), 0), nil
	}

	return newValVoid(), nil
}

func ratToInt64(r *big.Rat) int64 {
	// Quo truncates toward zero, matching bc's integer truncation semantics
	intPart := big.NewInt(0).Quo(r.Num(), r.Denom())
	return intPart.Int64()
}

func ratPower(r *big.Rat, exponent int64) *big.Rat {
	if exponent == 0 {
		return big.NewRat(1, 1)
	}
	neg := exponent < 0
	if neg {
		// 0^(-n) is undefined, return 0 (matches BusyBox bc behavior)
		if r.Sign() == 0 {
			return big.NewRat(0, 1)
		}
		exponent = -exponent
	}

	base := big.NewRat(0, 1).Set(r)
	res := big.NewRat(1, 1)
	for exponent > 0 {
		if exponent%2 == 1 {
			res.Mul(res, base)
		}
		base.Mul(base, base)
		exponent /= 2
	}

	if neg {
		res.Inv(res)
	}
	return res
}

func parseNumberInBase(s string, ibase int) (*big.Rat, error) {
	s = strings.ToUpper(s)
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}

	parts := strings.Split(s, ".")
	intStr := parts[0]
	var fracStr string
	if len(parts) > 1 {
		fracStr = parts[1]
	}

	isSingleDigit := len(parts) == 1
	if isSingleDigit {
		trimmed := strings.TrimLeft(intStr, "0")
		isSingleDigit = len(trimmed) == 1 || (len(trimmed) == 0 && len(intStr) == 1)
	}

	// Integer part
	intVal := big.NewInt(0)
	base := big.NewInt(int64(ibase))
	for i := 0; i < len(intStr); i++ {
		d := digitVal(intStr[i])
		if d < 0 {
			continue // skip formatting characters if any
		}
		if !isSingleDigit && d >= ibase {
			d = ibase - 1
		}
		intVal.Mul(intVal, base)
		intVal.Add(intVal, big.NewInt(int64(d)))
	}

	// Fractional part
	fracVal := big.NewRat(0, 1)
	power := big.NewInt(1)
	for i := 0; i < len(fracStr); i++ {
		d := digitVal(fracStr[i])
		if d < 0 {
			continue
		}
		if d >= ibase {
			d = ibase - 1
		}
		power.Mul(power, base)
		term := big.NewRat(int64(d), 1)
		term.Quo(term, big.NewRat(1, 1).SetInt(power))
		fracVal.Add(fracVal, term)
	}

	res := big.NewRat(0, 1).SetInt(intVal)
	res.Add(res, fracVal)
	if neg {
		res.Neg(res)
	}
	return res, nil
}

func digitVal(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'A' && c <= 'Z' {
		return int(c - 'A' + 10)
	}
	if c >= 'a' && c <= 'z' {
		return int(c - 'a' + 10)
	}
	return -1
}

func printWrapped(w io.Writer, s string) {
	lineLength := 70
	if envVal := os.Getenv("BC_LINE_LENGTH"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil {
			lineLength = val
		}
	}
	col := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\n' {
			col = 0
			fmt.Fprint(w, "\n")
			continue
		}
		if lineLength > 1 && col == lineLength-2 {
			fmt.Fprint(w, "\\\n")
			col = 0
		}
		fmt.Fprintf(w, "%c", ch)
		col++
	}
}

func formatRat(r *big.Rat, obase int, scale int, forceNeg bool) string {
	if r.Sign() == 0 {
		if scale <= 0 {
			return "0"
		}
		if forceNeg {
			if obase > 16 {
				var zeroes []string
				for i := 0; i < scale; i++ {
					zeroes = append(zeroes, "00")
				}
				return "-." + strings.Join(zeroes, " ")
			}
			return "-." + strings.Repeat("0", scale)
		}
		return "0"
	}

	neg := r.Sign() < 0 || forceNeg
	val := big.NewRat(0, 1).Abs(r)

	num := val.Num()
	denom := val.Denom()

	intPart := big.NewInt(0).Div(num, denom)
	rem := big.NewInt(0).Mod(num, denom)
	fracPart := big.NewRat(0, 1).SetFrac(rem, denom)

	printScale := scale
	if obase != 10 && scale > 0 {
		factor := math.Log(10) / math.Log(float64(obase))
		printScale = int(math.Ceil(float64(scale) * factor))
	}

	var intDigits []string
	temp := big.NewInt(0).Set(intPart)
	base := big.NewInt(int64(obase))
	zero := big.NewInt(0)
	for temp.Cmp(zero) > 0 {
		remDigit := big.NewInt(0)
		temp.DivMod(temp, base, remDigit)
		intDigits = append(intDigits, formatDigit(remDigit.Int64(), obase))
	}
	if len(intDigits) == 0 {
		intDigits = append(intDigits, "0")
	}
	for i, j := 0, len(intDigits)-1; i < j; i, j = i+1, j-1 {
		intDigits[i], intDigits[j] = intDigits[j], intDigits[i]
	}
	var intStr string
	if intPart.Cmp(zero) == 0 && printScale > 0 {
		intStr = ""
	} else if obase > 16 {
		var spaced []string
		for _, d := range intDigits {
			spaced = append(spaced, " "+d)
		}
		intStr = strings.Join(spaced, "")
	} else {
		intStr = strings.Join(intDigits, "")
	}

	if printScale <= 0 {
		if neg {
			return "-" + intStr
		}
		return intStr
	}

	var fracDigits []string
	fracTemp := big.NewRat(0, 1).Set(fracPart)
	obaseRat := big.NewRat(int64(obase), 1)
	for s := 0; s < printScale; s++ {
		fracTemp.Mul(fracTemp, obaseRat)
		digitInt := big.NewInt(0).Div(fracTemp.Num(), fracTemp.Denom())
		fracDigits = append(fracDigits, formatDigit(digitInt.Int64(), obase))
		fracTemp.Sub(fracTemp, big.NewRat(0, 1).SetInt(digitInt))
	}
	var fracStr string
	if obase > 16 {
		fracStr = "." + strings.Join(fracDigits, " ")
	} else {
		fracStr = "." + strings.Join(fracDigits, "")
	}

	res := intStr + fracStr
	if neg {
		return "-" + res
	}
	return res
}

func formatDigit(val int64, obase int) string {
	if obase <= 16 {
		if val < 10 {
			return string(rune('0' + val))
		}
		return string(rune('A' + val - 10))
	}
	return fmt.Sprintf("%02d", val)
}

func valLength(r *big.Rat, scale int) int {
	if r.Sign() == 0 {
		if scale > 0 {
			return scale
		}
		return 1
	}
	s := formatRat(r, 10, scale, r.Sign() < 0)
	s = strings.ReplaceAll(s, "-", "")
	parts := strings.Split(s, ".")
	intStr := parts[0]
	var fracStr string
	if len(parts) > 1 {
		fracStr = parts[1]
	}
	intStr = strings.TrimLeft(intStr, "0")
	if len(intStr) > 0 {
		return len(intStr) + len(fracStr)
	} else {
		trimmed := strings.TrimLeft(fracStr, "0")
		return len(trimmed)
	}
}

func valSqrt(r *big.Rat, xValScale, globalScale int) (*big.Rat, int, error) {
	if r.Sign() < 0 {
		return nil, 0, fmt.Errorf("square root of negative number")
	}
	if r.Sign() == 0 {
		targetScale := xValScale
		if globalScale > targetScale {
			targetScale = globalScale
		}
		return big.NewRat(0, 1), targetScale, nil
	}

	targetScale := xValScale
	if globalScale > targetScale {
		targetScale = globalScale
	}

	prec := uint(targetScale)*4 + 128
	if prec < 256 {
		prec = 256
	}

	f := new(big.Float).SetPrec(prec).SetRat(r)
	sqrtF := new(big.Float).SetPrec(prec).Sqrt(f)

	resRat := new(big.Rat)
	sqrtF.Rat(resRat)

	return resRat, targetScale, nil
}

func unescapeBcString(s string) string {
	var sb strings.Builder
	runes := []rune(s)
	n := len(runes)
	for i := 0; i < n; i++ {
		if runes[i] == '\\' && i+1 < n {
			next := runes[i+1]
			switch next {
			case 'a':
				sb.WriteRune('\a')
				i++
			case 'b':
				sb.WriteRune('\b')
				i++
			case 'f':
				sb.WriteRune('\f')
				i++
			case 'n':
				sb.WriteRune('\n')
				i++
			case 'r':
				sb.WriteRune('\r')
				i++
			case 't':
				sb.WriteRune('\t')
				i++
			case '\\':
				sb.WriteRune('\\')
				i++
			case '"':
				sb.WriteRune('"')
				i++
			default:
				sb.WriteRune('\\')
				i++
			}
		} else {
			sb.WriteRune(runes[i])
		}
	}
	return sb.String()
}

func arrayLength(arr map[string]Val) int {
	maxIdx := -1
	for k := range arr {
		if idx, err := strconv.Atoi(k); err == nil {
			if idx > maxIdx {
				maxIdx = idx
			}
		}
	}
	return maxIdx + 1
}

func idxString(r *big.Rat) string {
	// Quo truncates toward zero for POSIX bc array index truncation
	idxInt := big.NewInt(0).Quo(r.Num(), r.Denom())
	return idxInt.String()
}

func truncateRat(r *big.Rat, scale int) *big.Rat {
	if scale < 0 {
		scale = 0
	}
	factor := big.NewInt(1)
	ten := big.NewInt(10)
	for i := 0; i < scale; i++ {
		factor.Mul(factor, ten)
	}

	temp := big.NewRat(0, 1).Mul(r, big.NewRat(0, 1).SetInt(factor))

	num := temp.Num()
	denom := temp.Denom()
	// Use Quo (truncation toward zero) not Div (floor toward -infinity)
	intPart := big.NewInt(0).Quo(num, denom)

	res := big.NewRat(0, 1).SetFrac(intPart, factor)
	return res
}

func Run(program io.Reader, stdin io.Reader, w io.Writer, mathLib bool) error {
	// Read entire input from program
	var buf bytes.Buffer
	_, err := io.Copy(&buf, program)
	if err != nil {
		return err
	}

	lex := NewLexer(buf.String())
	parser := NewParser(lex)
	stmts, err := parser.Parse()
	if err != nil {
		return err
	}

	interpreter := NewInterpreter(w, stdin, mathLib)
	return interpreter.Execute(stmts)
}

func bcRun(args []string, stdout, errOut io.Writer, stdin io.Reader, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "bc: %v\n", err)
		return 2
	}

	mathLib := flags.Has("l")
	jsonMode := flags.Has("json")

	var input io.Reader = stdin
	if len(flags.Positional) > 0 {
		file, err := os.Open(flags.Positional[0])
		if err != nil {
			fmt.Fprintf(errOut, "bc: %v\n", err)
			return 1
		}
		defer file.Close()
		input = file
	}

	if jsonMode {
		var buf bytes.Buffer
		err = Run(input, stdin, &buf, mathLib)
		if err != nil {
			common.RenderError("bc", 1, "ERR", err.Error(), true, stdout)
			return 1
		}
		lines := strings.Split(buf.String(), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		common.Render("bc", BcResult{Lines: lines}, true, stdout, func() {})
		return 0
	}

	err = Run(input, stdin, stdout, mathLib)
	if err != nil {
		fmt.Fprintf(errOut, "bc: %v\n", err)
		return 1
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "bc",
		Usage: "An arbitrary precision calculator language",
		Run: func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
			return bcRun(args, stdout, stderr, stdin, cwd)
		},
	})
}

const MathLibSource = `define e(x){
	auto b,s,n,r,d,i,p,f,v
	b=ibase
	ibase=A
	if(x<0){
		n=1
		x=-x
	}
	s=scale
	r=6+s+.44*x
	scale=scale(x)+1
	while(x>1){
		d+=1
		x/=2
		scale+=1
	}
	scale=r
	r=x+1
	p=x
	f=v=1
	for(i=2;v;++i){
		p*=x
		f*=i
		v=p/f
		r+=v
	}
	while(d--)r*=r
	scale=s
	ibase=b
	if(n)return(1/r)
	return(r/1)
}
define l(x){
	auto b,s,r,p,a,q,i,v
	if(x<=0)return((1-A^scale)/1)
	b=ibase
	ibase=A
	s=scale
	scale+=6
	p=2
	while(x>=2){
		p*=2
		x=sqrt(x)
	}
	while(x<=.5){
		p*=2
		x=sqrt(x)
	}
	r=a=(x-1)/(x+1)
	q=a*a
	v=1
	for(i=3;v;i+=2){
		a*=q
		v=a/i
		r+=v
	}
	r*=p
	scale=s
	ibase=b
	return(r/1)
}
define s(x){
	auto b,s,r,a,q,i
	if(x<0)return(-s(-x))
	b=ibase
	ibase=A
	s=scale
	scale=1.1*s+2
	a=a(1)
	scale=0
	q=(x/a+2)/4
	x-=4*q*a
	if(q%2)x=-x
	scale=s+2
	r=a=x
	q=-x*x
	for(i=3;a;i+=2){
		a*=q/(i*(i-1))
		r+=a
	}
	scale=s
	ibase=b
	return(r/1)
}
define c(x){
	auto b,s
	b=ibase
	ibase=A
	s=scale
	scale*=1.2
	x=s(2*a(1)+x)
	scale=s
	ibase=b
	return(x/1)
}
define a(x){
	auto b,s,r,n,a,m,t,f,i,u
	b=ibase
	ibase=A
	n=1
	if(x<0){
		n=-1
		x=-x
	}
	if(scale<65){
		if(x==1){
			r=.7853981633974483096156608458198757210492923498437764552437361480/n
			ibase=b
			return(r)
		}
		if(x==.2){
			r=.1973955598498807583700497651947902934475851037878521015176889402/n
			ibase=b
			return(r)
		}
	}
	s=scale
	if(x>.2){
		scale+=5
		a=a(.2)
	}
	scale=s+3
	while(x>.2){
		m+=1
		x=(x-.2)/(1+.2*x)
	}
	r=u=x
	f=-x*x
	t=1
	for(i=3;t;i+=2){
		u*=f
		t=u/i
		r+=t
	}
	scale=s
	ibase=b
	return((m*a+r)/n)
}
define j(n,x){
	auto b,s,o,a,i,r,v,f
	b=ibase
	ibase=A
	s=scale
	scale=0
	n/=1
	if(n<0){
		n=-n
		o=n%2
	}
	a=1
	for(i=2;i<=n;++i)a*=i
	scale=1.5*s
	a=(x^n)/2^n/a
	r=v=1
	f=-x*x/4
	scale+=length(a)-scale(a)
	for(i=1;v;++i){
		v=v*f/i/(n+i)
		r+=v
	}
	scale=s
	ibase=b
	if(o)a=-a
	return(a*r/1)
}
`
