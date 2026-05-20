package sed

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type engineState struct {
	patSpace    string
	holdSpace   string
	lineNum     int
	isEOF       bool
	substituted bool
	skipRead    bool

	out         io.Writer
	suppress    bool
	files       []string
	fileIdx     int
	inPlace     bool
	currentFile string
	tmpFile     *os.File
	outChanged  bool

	lastRegex *regexp.Regexp

	pendingAppend []string
	pendingRead   []string

	addrState        map[*Instruction]bool
	matchedLineAddrs map[*Address]bool

	reader                 *bufio.Reader
	hasTrailingNewline     bool
	nextHasTrailingNewline bool
	nextLine               string
	hasNext                bool
	lastLacked             map[string]bool
}

func (e *engineState) printStream(w io.Writer, s string, streamID string) {
	if e.lastLacked[streamID] {
		fmt.Fprint(w, "\n")
	}
	fmt.Fprint(w, s)
	if e.hasTrailingNewline {
		fmt.Fprint(w, "\n")
		e.lastLacked[streamID] = false
	} else {
		e.lastLacked[streamID] = true
	}
}

func (e *engineState) printLine(s string) {
	streamID := "stdout"
	if e.inPlace {
		streamID = e.currentFile
	}
	e.printStream(e.out, s, streamID)
}

func (e *engineState) printText(s string) {
	streamID := "stdout"
	if e.inPlace {
		streamID = e.currentFile
	}
	if e.lastLacked[streamID] {
		fmt.Fprint(e.out, "\n")
	}
	fmt.Fprint(e.out, s, "\n")
	e.lastLacked[streamID] = false
}

func (e *engineState) printLineRaw(s string) {
	fmt.Fprint(e.out, s)
}

func runEngine(insts []*Instruction, readers []string, suppress bool, inPlace bool, globalOut io.Writer) int {
	return runEngineInternal(insts, readers, suppress, inPlace, globalOut, os.Stderr, os.Stdin)
}

func runEngineInternal(insts []*Instruction, readers []string, suppress bool, inPlace bool, globalOut io.Writer, errOut io.Writer, stdin io.Reader) int {
	var err error
	insts, err = compileAst(insts)
	if err != nil {
		fmt.Fprintf(errOut, "sed: %v\n", err)
		return 1
	}
	truncateWFiles(insts)

	if len(readers) == 0 {
		readers = []string{"-"}
	}

	e := &engineState{
		files:            readers,
		suppress:         suppress,
		inPlace:          inPlace,
		addrState:        make(map[*Instruction]bool),
		matchedLineAddrs: make(map[*Address]bool),
		lastLacked:       make(map[string]bool),
		out:              globalOut,
	}

	exitCode := 0

	for i := 0; i < len(e.files); i++ {
		e.fileIdx = i
		path := e.files[i]
		e.currentFile = path
		if e.inPlace {
			e.lineNum = 0
			e.addrState = make(map[*Instruction]bool)
			e.matchedLineAddrs = make(map[*Address]bool)
		}

		var r io.Reader
		if path == "-" {
			r = stdin
		} else {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(errOut, "sed: %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			r = f
			defer f.Close()
		}

		if e.inPlace && path != "-" {
			tmpPath := path + ".tmp"
			tmpFile, err := os.Create(tmpPath)
			if err != nil {
				fmt.Fprintf(errOut, "sed: %v\n", err)
				exitCode = 1
				continue
			}
			e.tmpFile = tmpFile
			e.out = tmpFile
		}

		e.reader = bufio.NewReader(r)

		lineStr, _ := e.reader.ReadString('\n')
		if len(lineStr) > 0 {
			if strings.HasSuffix(lineStr, "\n") {
				e.nextHasTrailingNewline = true
				e.nextLine = lineStr[:len(lineStr)-1]
				if strings.HasSuffix(e.nextLine, "\r") {
					e.nextLine = e.nextLine[:len(e.nextLine)-1]
				}
			} else {
				e.nextHasTrailingNewline = false
				e.nextLine = lineStr
			}
			e.hasNext = true
		} else {
			e.hasNext = false
		}
		e.isEOF = !e.hasNext && (e.inPlace || e.fileIdx == len(e.files)-1)

		if !e.hasNext {
			// empty file
			if e.inPlace && e.tmpFile != nil {
				e.tmpFile.Close()
				os.Rename(e.tmpFile.Name(), path)
			}
			continue
		}

		for e.hasNext {
			if !e.skipRead {
				e.patSpace = e.nextLine
				e.hasTrailingNewline = e.nextHasTrailingNewline
				e.lineNum++
				e.substituted = false
			}
			e.skipRead = false

			lineStr, _ = e.reader.ReadString('\n')
			if len(lineStr) > 0 {
				if strings.HasSuffix(lineStr, "\n") {
					e.nextHasTrailingNewline = true
					e.nextLine = lineStr[:len(lineStr)-1]
					if strings.HasSuffix(e.nextLine, "\r") {
						e.nextLine = e.nextLine[:len(e.nextLine)-1]
					}
				} else {
					e.nextHasTrailingNewline = false
					e.nextLine = lineStr
				}
				e.hasNext = true
			} else {
				e.hasNext = false
				e.nextLine = ""
			}
			e.isEOF = !e.hasNext && (e.inPlace || e.fileIdx == len(e.files)-1)

			// execute instructions
			if err := e.execFlat(insts); err != nil {
				if err.Error() == "quit" {
					if !e.suppress && !e.skipRead {
						e.printLine(e.patSpace)
					}
					return exitCode
				} else if err.Error() == "next" {
					// d or similar
				} else if err.Error() == "quit_no_print" {
					return exitCode
				} else {
					fmt.Fprintf(errOut, "sed: %v\n", err)
					exitCode = 1
					break
				}
			} else {
				if !e.suppress {
					e.printLine(e.patSpace)
				}
			}

			// flush appends
			for _, text := range e.pendingAppend {
				e.printText(text)
			}
			e.pendingAppend = nil
		}

		if e.inPlace && path != "-" && e.tmpFile != nil {
			e.tmpFile.Close()
			os.Rename(e.tmpFile.Name(), path)
		}
	}

	return exitCode
}

func (e *engineState) matchAddress(addr *Address) bool {
	if addr == nil {
		return false
	}
	if addr.Type == AddrLine {
		return e.lineNum == addr.Line
	}
	if addr.Type == AddrLast {
		return e.isEOF
	}
	if addr.Type == AddrRegexp {
		re := addr.Regexp
		if re == nil {
			re = e.lastRegex
		}
		if re != nil {
			e.lastRegex = re
			return re.MatchString(e.patSpace)
		}
	}
	return false
}

func (e *engineState) shouldRun(inst *Instruction) bool {
	if inst.Addr1 == nil {
		return true != inst.AddressInvert
	}
	if inst.Addr2 == nil {
		return e.matchAddress(inst.Addr1) != inst.AddressInvert
	}

	// Address ranges
	active := e.addrState[inst]
	if !active {
		matchStart := false
		if inst.Addr1.Type == AddrLine {
			if e.lineNum >= inst.Addr1.Line {
				if !e.matchedLineAddrs[inst.Addr1] {
					e.matchedLineAddrs[inst.Addr1] = true
					matchStart = true
				}
			}
		} else {
			matchStart = e.matchAddress(inst.Addr1)
		}

		if matchStart {
			active = true
			e.addrState[inst] = true
			// If addr2 matches right away (and it's not a +N address)
			// we don't deactivate it here because POSIX says address ranges are inclusive
			// but if it matches the SAME line, it closes immediately.
			// Actually POSIX says if addr2 matches on the SAME line, it still spans at least one line.
			if inst.Addr2.Type == AddrLine && inst.Addr2.Step > 0 {
				// GNU extension +N
				// we change Addr2.Line to be e.lineNum + step, preserving Step
				inst.Addr2.Line = e.lineNum + inst.Addr2.Step
			}
		}
	}

	result := active

	if active {
		addr2Matched := false
		if inst.Addr2.Type == AddrLine {
			addr2Matched = e.lineNum >= inst.Addr2.Line
		} else {
			addr2Matched = e.matchAddress(inst.Addr2)
		}
		if addr2Matched {
			e.addrState[inst] = false // deactivate for next line
		}
	}

	return result != inst.AddressInvert
}

func (e *engineState) execFlat(insts []*Instruction) error {
	for ip := 0; ip < len(insts); ip++ {
		inst := insts[ip]
		if !e.shouldRun(inst) {
			if inst.Cmd == '{' {
				ip = inst.JumpTarget // skip block
			}
			continue
		}

		switch inst.Cmd {
		case '{', '}', ':':
			// structure only
		case 's':
			re := inst.Regexp
			if re == nil {
				re = e.lastRegex
			}
			if re != nil {
				e.lastRegex = re
				if re.MatchString(e.patSpace) {
					e.substituted = true

					// Replacement logic
					repl := inst.Repl

					if inst.Global {
						e.patSpace = re.ReplaceAllString(e.patSpace, repl)
					} else if inst.SubNum > 0 {
						// replace Nth occurence
						matches := re.FindAllStringIndex(e.patSpace, -1)
						if len(matches) >= inst.SubNum {
							loc := matches[inst.SubNum-1]
							replced := re.ReplaceAllString(e.patSpace[loc[0]:loc[1]], repl)
							e.patSpace = e.patSpace[:loc[0]] + replced + e.patSpace[loc[1]:]
						}
					} else {
						// replace first
						loc := re.FindStringIndex(e.patSpace)
						if loc != nil {
							replced := re.ReplaceAllString(e.patSpace[loc[0]:loc[1]], repl)
							e.patSpace = e.patSpace[:loc[0]] + replced + e.patSpace[loc[1]:]
						}
					}

					if inst.Print {
						e.printLine(e.patSpace)
					}
					if inst.WFile != "" {
						f, err := os.OpenFile(inst.WFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
						if err == nil {
							e.printStream(f, e.patSpace, inst.WFile)
							f.Close()
						}
					}
				}
			}
		case 'p':
			e.printLine(e.patSpace)
		case 'P':
			idx := strings.Index(e.patSpace, "\n")
			if idx >= 0 {
				e.printLineRaw(e.patSpace[:idx+1])
			} else {
				e.printLine(e.patSpace)
			}
		case 'd':
			return fmt.Errorf("next") // skip rest of commands and don't print
		case 'D':
			idx := strings.Index(e.patSpace, "\n")
			if idx >= 0 {
				e.patSpace = e.patSpace[idx+1:]
				// POSIX says if pattern space is not empty, start next cycle without reading a new line.
				// We can achieve this by rewinding to the start of the block and setting skipRead = true.
				e.skipRead = true
				ip = -1 // restart script
				continue
			} else {
				return fmt.Errorf("next")
			}
		case 'n':
			if !e.suppress {
				e.printLine(e.patSpace)
			}

			if e.isEOF {
				return fmt.Errorf("quit_no_print")
			}

			e.patSpace = e.nextLine
			e.hasTrailingNewline = e.nextHasTrailingNewline
			e.lineNum++

			lineStr, _ := e.reader.ReadString('\n')
			if len(lineStr) > 0 {
				if strings.HasSuffix(lineStr, "\n") {
					e.nextHasTrailingNewline = true
					e.nextLine = lineStr[:len(lineStr)-1]
					if strings.HasSuffix(e.nextLine, "\r") {
						e.nextLine = e.nextLine[:len(e.nextLine)-1]
					}
				} else {
					e.nextHasTrailingNewline = false
					e.nextLine = lineStr
				}
				e.hasNext = true
			} else {
				e.hasNext = false
				e.nextLine = ""
			}
			e.isEOF = !e.hasNext && (e.inPlace || e.fileIdx == len(e.files)-1)
		case 'N':
			if e.isEOF {
				// GNU sed prints pattern space if N is at EOF
				return fmt.Errorf("quit")
			}

			e.patSpace += "\n" + e.nextLine
			e.hasTrailingNewline = e.nextHasTrailingNewline
			e.lineNum++

			lineStr, _ := e.reader.ReadString('\n')
			if len(lineStr) > 0 {
				if strings.HasSuffix(lineStr, "\n") {
					e.nextHasTrailingNewline = true
					e.nextLine = lineStr[:len(lineStr)-1]
					if strings.HasSuffix(e.nextLine, "\r") {
						e.nextLine = e.nextLine[:len(e.nextLine)-1]
					}
				} else {
					e.nextHasTrailingNewline = false
					e.nextLine = lineStr
				}
				e.hasNext = true
			} else {
				e.hasNext = false
				e.nextLine = ""
			}
			e.isEOF = !e.hasNext && (e.inPlace || e.fileIdx == len(e.files)-1)
		case 'a':
			e.pendingAppend = append(e.pendingAppend, inst.Text)
		case 'i':
			e.printText(inst.Text)
		case 'c':
			// delete pattern space, print text, skip rest of script
			e.printText(inst.Text)
			return fmt.Errorf("next")
		case 'q':
			return fmt.Errorf("quit")
		case '=':
			fmt.Fprintln(e.out, e.lineNum)
		case 'h':
			e.holdSpace = e.patSpace
		case 'H':
			e.holdSpace += "\n" + e.patSpace
		case 'g':
			e.patSpace = e.holdSpace
		case 'G':
			e.patSpace += "\n" + e.holdSpace
		case 'x':
			e.patSpace, e.holdSpace = e.holdSpace, e.patSpace
		case 'b':
			ip = inst.JumpTarget - 1
		case 't':
			cond := e.substituted
			e.substituted = false
			if cond {
				ip = inst.JumpTarget - 1
			}
		case 'T':
			cond := e.substituted
			e.substituted = false
			if !cond {
				ip = inst.JumpTarget - 1
			}
		case 'w':
			if inst.File != "" {
				f, err := os.OpenFile(inst.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					e.printStream(f, e.patSpace, inst.File)
					f.Close()
				}
			}
		}
	}
	return nil
}

func truncateWFiles(insts []*Instruction) {
	for _, inst := range insts {
		var file string
		if inst.Cmd == 'w' {
			file = inst.File
		} else if inst.Cmd == 's' && inst.WFile != "" {
			file = inst.WFile
		}
		if file != "" {
			f, _ := os.Create(file)
			f.Close()
		}
		if inst.Block != nil {
			truncateWFiles(inst.Block)
		}
	}
}

func compileAst(insts []*Instruction) ([]*Instruction, error) {
	var flat []*Instruction
	var flatten func([]*Instruction)
	flatten = func(in []*Instruction) {
		for _, inst := range in {
			if inst.Cmd == '{' {
				start := &Instruction{Cmd: '{', Addr1: inst.Addr1, Addr2: inst.Addr2, AddressInvert: inst.AddressInvert}
				flat = append(flat, start)
				flatten(inst.Block)
				end := &Instruction{Cmd: '}'}
				flat = append(flat, end)
				start.JumpTarget = len(flat) - 1
			} else {
				flat = append(flat, inst)
			}
		}
	}
	flatten(insts)

	labels := make(map[string]int)
	for i, inst := range flat {
		if inst.Cmd == ':' {
			labels[inst.Label] = i
		}
	}

	for _, inst := range flat {
		if inst.Cmd == 'b' || inst.Cmd == 't' || inst.Cmd == 'T' {
			if inst.Label == "" {
				inst.JumpTarget = len(flat)
			} else {
				if target, ok := labels[inst.Label]; ok {
					inst.JumpTarget = target
				} else {
					return nil, fmt.Errorf("can't find label for jump to '%s'", inst.Label)
				}
			}
		}
	}

	return flat, nil
}
