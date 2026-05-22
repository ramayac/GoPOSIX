package sed

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type AddressType int

const (
	AddrNone AddressType = iota
	AddrLine
	AddrLast
	AddrRegexp
)

type Address struct {
	Type   AddressType
	Line   int
	Regexp *regexp.Regexp
	Step   int // +N extension
}

type Instruction struct {
	Addr1         *Address
	Addr2         *Address
	AddressInvert bool

	Cmd byte

	// For s
	Regexp *regexp.Regexp
	Repl   string
	Global bool
	Print  bool
	SubNum int
	WFile  string

	// For a, i, c
	Text string

	// For b, t, T, :
	Label string

	// For w, r
	File string

	// For block {}
	Block []*Instruction

	// Execution state
	JumpTarget int
}

type Parser struct {
	text string
	pos  int
}

func (p *Parser) peek() int {
	if p.pos >= len(p.text) {
		return -1
	}
	return int(p.text[p.pos])
}

func (p *Parser) next() int {
	if p.pos >= len(p.text) {
		return -1
	}
	c := int(p.text[p.pos])
	p.pos++
	return c
}

func (p *Parser) skipWhitespace() {
	for {
		c := p.peek()
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ';' {
			p.next()
		} else {
			break
		}
	}
}

func Parse(script string) ([]*Instruction, error) {
	p := &Parser{text: script}
	return p.parseBlock(false)
}

func (p *Parser) parseBlock(inBlock bool) ([]*Instruction, error) {
	var insts []*Instruction
	for {
		p.skipWhitespace()
		if p.peek() == -1 {
			if inBlock {
				return nil, fmt.Errorf("unclosed {")
			}
			break
		}
		if p.peek() == '}' {
			if inBlock {
				p.next() // consume }
				break
			} else {
				return nil, fmt.Errorf("unexpected }")
			}
		}

		if p.peek() == '#' {
			// comment
			for p.peek() != '\n' && p.peek() != -1 {
				p.next()
			}
			continue
		}

		inst, err := p.parseInstruction()
		if err != nil {
			return nil, err
		}
		if inst != nil {
			insts = append(insts, inst)
		}
	}
	return insts, nil
}

func (p *Parser) parseAddress() (*Address, error) {
	c := p.peek()
	if c >= '0' && c <= '9' {
		start := p.pos
		for p.peek() >= '0' && p.peek() <= '9' {
			p.next()
		}
		line, _ := strconv.Atoi(p.text[start:p.pos])
		return &Address{Type: AddrLine, Line: line}, nil
	} else if c == '$' {
		p.next()
		return &Address{Type: AddrLast}, nil
	} else if c == '/' || c == '\\' {
		delim := p.next()
		if delim == '\\' {
			delim = p.next()
		}
		start := p.pos
		inBracket := false
		for {
			c := p.peek()
			if c == -1 || c == '\n' {
				return nil, fmt.Errorf("unterminated regex")
			}
			if c == '\\' {
				p.next()
				if p.peek() != -1 {
					p.next()
				}
				continue
			}
			if c == '[' && !inBracket {
				inBracket = true
			} else if c == ']' && inBracket {
				inBracket = false
			} else if c == delim && !inBracket {
				break
			}
			p.next()
		}
		expr := p.text[start:p.pos]
		p.next() // consume delim

		// regex fixups
		if expr == "" {
			// use previous regex? Let's just store nil
			return &Address{Type: AddrRegexp, Regexp: nil}, nil
		}

		re, err := compileBRE(expr, byte(delim))
		if err != nil {
			return nil, err
		}
		return &Address{Type: AddrRegexp, Regexp: re}, nil
	}
	return nil, nil
}

func (p *Parser) parseInstruction() (*Instruction, error) {
	inst := &Instruction{}

	// parse address 1
	addr1, err := p.parseAddress()
	if err != nil {
		return nil, err
	}
	inst.Addr1 = addr1

	if p.peek() == ',' {
		p.next() // consume ,
		if p.peek() == '+' {
			p.next()
			// parse step
			start := p.pos
			for p.peek() >= '0' && p.peek() <= '9' {
				p.next()
			}
			step, _ := strconv.Atoi(p.text[start:p.pos])
			inst.Addr2 = &Address{Type: AddrLine, Step: step} // store step here temporarily
		} else {
			addr2, err := p.parseAddress()
			if err != nil {
				return nil, err
			}
			inst.Addr2 = addr2
		}
	}

	p.skipWhitespace()
	if p.peek() == '!' {
		p.next()
		inst.AddressInvert = true
		p.skipWhitespace()
	}

	cmd := p.next()
	if cmd == -1 {
		return nil, nil // end of script
	}
	inst.Cmd = byte(cmd)

	switch cmd {
	case '{':
		block, err := p.parseBlock(true)
		if err != nil {
			return nil, err
		}
		inst.Block = block
	case 's':
		delim := p.next()
		if delim == -1 || delim == '\n' {
			return nil, fmt.Errorf("invalid s command")
		}
		// parse pattern
		start := p.pos
		inBracket := false
		for {
			c := p.peek()
			if c == -1 || c == '\n' {
				return nil, fmt.Errorf("unterminated s command pattern")
			}
			if c == '\\' {
				p.next()
				if p.peek() != -1 {
					p.next()
				}
				continue
			}
			if c == '[' && !inBracket {
				inBracket = true
			} else if c == ']' && inBracket {
				inBracket = false
			} else if c == delim && !inBracket {
				break
			}
			p.next()
		}
		pat := p.text[start:p.pos]
		p.next() // consume delim

		// parse replacement
		start = p.pos
		for {
			c := p.peek()
			if c == -1 || c == '\n' {
				return nil, fmt.Errorf("unterminated s command replacement")
			}
			if c == '\\' {
				p.next()
				if p.peek() != -1 {
					p.next()
				}
				continue
			}
			if c == delim {
				break
			}
			p.next()
		}

		// Unescape replacement string: convert \& to &, \1 to $1, etc.
		// Go's regexp uses $1 instead of \1 for backrefs.
		// Wait, we need to do this properly.
		replRaw := p.text[start:p.pos]
		repl := ""
		for i := 0; i < len(replRaw); i++ {
			if replRaw[i] == '\\' && i+1 < len(replRaw) {
				next := replRaw[i+1]
				if next == byte(delim) {
					repl += string(byte(delim))
				} else if next >= '0' && next <= '9' {
					repl += "$" + string(next)
				} else if next == '&' {
					repl += "&" // escaped & becomes literal &
				} else if next == '\\' {
					repl += "\\"
				} else if next == 'n' {
					repl += "\n"
				} else if next == 't' {
					repl += "\t"
				} else if next == 'r' {
					repl += "\r"
				} else {
					repl += string(next)
				}
				i++
			} else if replRaw[i] == '&' {
				repl += "$0"
			} else {
				repl += string(replRaw[i])
			}
		}

		inst.Repl = repl
		p.next() // consume delim

		// flags
		for {
			c := p.peek()
			if c == 'g' {
				inst.Global = true
				p.next()
			} else if c == 'p' {
				inst.Print = true
				p.next()
			} else if c == 'I' || c == 'i' {
				// case insensitive
				pat = "(?i)" + pat
				p.next()
			} else if c >= '1' && c <= '9' {
				inst.SubNum = int(c - '0')
				p.next()
			} else if c == 'w' {
				p.next()
				// read filename
				for p.peek() == ' ' || p.peek() == '\t' {
					p.next()
				}
				wstart := p.pos
				for p.peek() != ' ' && p.peek() != '\t' && p.peek() != '\n' && p.peek() != ';' && p.peek() != -1 {
					p.next()
				}
				inst.WFile = p.text[wstart:p.pos]
			} else {
				break
			}
		}

		if pat == "" {
			inst.Regexp = nil
		} else {
			re, err := compileBRE(pat, byte(delim))
			if err != nil {
				return nil, err
			}
			inst.Regexp = re
		}

	case 'a', 'i', 'c':
		for p.peek() == ' ' || p.peek() == '\t' {
			p.next()
		}
		if p.peek() == '\\' {
			p.next()
		}
		if p.peek() == '\n' {
			p.next()
		}
		start := p.pos
		for {
			c := p.peek()
			if c == -1 {
				break
			}
			if c == '\\' {
				p.next()
				if p.peek() == -1 {
					break
				}
				if p.peek() == '\n' {
					p.next()
					continue
				}
				p.next() // consume escaped char
				continue
			}
			if c == '\n' {
				// end of text
				p.next()
				break
			}
			p.next()
		}
		rawText := p.text[start:p.pos]
		// unescape \n \t \r
		text := ""
		for i := 0; i < len(rawText); i++ {
			if rawText[i] == '\\' && i+1 < len(rawText) {
				if rawText[i+1] == 'n' {
					text += "\n"
					i++
				} else if rawText[i+1] == 't' {
					text += "\t"
					i++
				} else if rawText[i+1] == 'r' {
					text += "\r"
					i++
				} else if rawText[i+1] == '\n' {
					i++
				} else // escaped newline removed
				{
					text += string(rawText[i+1])
					i++
				}
			} else {
				text += string(rawText[i])
			}
		}
		text = strings.TrimSuffix(text, "\n")
		inst.Text = text

	case 'b', 't', 'T', ':', 'r', 'w':
		for p.peek() == ' ' || p.peek() == '\t' {
			p.next()
		}
		start := p.pos
		for p.peek() != ' ' && p.peek() != '\t' && p.peek() != '\n' && p.peek() != ';' && p.peek() != -1 {
			p.next()
		}
		if cmd == 'r' || cmd == 'w' {
			inst.File = p.text[start:p.pos]
		} else {
			inst.Label = p.text[start:p.pos]
		}

	case 'd', 'D', 'p', 'P', 'n', 'N', 'g', 'G', 'h', 'H', 'x', 'q', '=', 'l':
		// No arguments
	default:
		return nil, fmt.Errorf("unknown command '%c'", cmd)
	}

	return inst, nil
}

func compileBRE(expr string, delim byte) (*regexp.Regexp, error) {
	var ere strings.Builder
	escaped := false
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if escaped {
			if c == delim {
				switch c {
				case '.', '+', '*', '?', '(', ')', '[', ']', '{', '}', '|', '^', '$', '\\':
					ere.WriteByte('\\')
					ere.WriteByte(c)
				default:
					ere.WriteByte(c)
				}
			} else {
				switch c {
				case '|', '+', '?', '(', ')', '{', '}':
					ere.WriteByte(c)
				default:
					ere.WriteByte('\\')
					ere.WriteByte(c)
				}
			}
			escaped = false
		} else {
			if c == '\\' {
				escaped = true
			} else {
				switch c {
				case '|', '+', '?', '(', ')', '{', '}':
					ere.WriteByte('\\')
					ere.WriteByte(c)
				default:
					ere.WriteByte(c)
				}
			}
		}
	}
	if escaped {
		ere.WriteByte('\\')
	}
	return regexp.Compile(ere.String())
}
