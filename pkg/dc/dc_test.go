package dc

import (
	"math/big"
	"strings"
	"testing"
)

// ----- Helper test function -----

func testDC(t *testing.T, name, input string, expected string) {
	t.Helper()
	state := &dcState{regs: make(map[string][]dcValue)}
	var output []string
	err := evalDC(state, input, nil, &output)
	if err != nil {
		t.Fatalf("%s: evalDC error: %v", name, err)
	}
	got := strings.Join(output, "\n")
	if got != "" {
		got += "\n"
	}
	if got != expected {
		t.Errorf("%s: got %q, want %q", name, got, expected)
	}
}

func testDCFail(t *testing.T, name, input string, wantErr string) {
	t.Helper()
	state := &dcState{regs: make(map[string][]dcValue)}
	var output []string
	err := evalDC(state, input, nil, &output)
	if err == nil {
		t.Fatalf("%s: expected error, got none", name)
	}
	if !strings.Contains(err.Error(), wantErr) {
		t.Errorf("%s: got error %q, want containing %q", name, err.Error(), wantErr)
	}
}

func TestDcArithmetic(t *testing.T) {
	tests := []struct{ name, input, expected string }{
		{"add", "10 20+p", "30\n"},
		{"sub", "10 3-p", "7\n"},
		{"mul", "8 8*p", "64\n"},
		{"div", "8 2/p", "4\n"},
		{"mod", "15 4%p", "3\n"},
		{"mod neg a", "_15 4%p", "-3\n"},
		{"mod neg b", "15 _4%p", "3\n"},
		{"pow 2^10", "2 10^p", "1024\n"},
		{"pow 0^0", "0 0^p", "1\n"},
		{"pow 0^neg", "0 _1^p", "0\n"},
		{"sqrt 16", "16vp", "4\n"},
		{"sqrt 0", "0vp", "0\n"},
		{"divmod", "15 4~pRpR", "3\n3\n"},
		{"modexp", "2 2 3|pR", "1\n"},
		{"modexp exp0", "5 0 7|pR", "1\n"},
		{"complex", "8 8*2 2+/p", "16\n"},
		{"neg number", "_5p", "-5\n"},
		{"decimal", "3.14p", "3.14\n"},
		{"decimal start", ".5p", ".5\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDC(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestDcStackOps(t *testing.T) {
	tests := []struct{ name, input, expected string }{
		{"dup", "1dpR", "1\n"},
		{"swap", "1 2rpRpR", "1\n2\n"},
		{"discard", "1 2RpR", "1\n"},
		{"depth 0", "zpR", "0\n"},
		{"depth 2", "1 2zpR", "2\n"},
		{"clear", "1 2cf", ""},
		{"print stack", "1 2f", "2\n1\n"},
		{"length int", "123ZpR", "3\n"},
		{"length str", "[hello]ZpR", "5\n"},
		{"length neg", "_4567ZpR", "4\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDC(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestDcPrintOps(t *testing.T) {
	// n: pop and print without newline
	t.Run("n print", func(t *testing.T) {
		testDC(t, "n", "42np", "42\n")
	})
	// P: pop string and print with escapes processed
	t.Run("P string", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		pushStrDC(state, "foo")
		if err := evalDC(state, "P", nil, &output); err != nil {
			t.Fatal(err)
		}
		got := strings.Join(output, "")
		if got != "foo" {
			t.Errorf("P: got %q, want %q", got, "foo")
		}
	})
	// p on empty stack - no output
	t.Run("p empty", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		if err := evalDC(state, "p", nil, &output); err != nil {
			t.Fatal(err)
		}
		if len(output) != 0 {
			t.Errorf("expected empty output, got %v", output)
		}
	})
}

func TestDcScale(t *testing.T) {
	t.Run("scale 2 div", func(t *testing.T) {
		testDC(t, "scale2", "2k 7 3/p", "2.33\n")
	})
	t.Run("scale push", func(t *testing.T) {
		// K pushes scale (5), p prints with current scale → "5.00000"
		testDC(t, "K", "5k KpR", "5.00000\n")
	})
	t.Run("scale neg", func(t *testing.T) {
		testDC(t, "neg k", "_1k KpR", "0\n")
	})
	t.Run("scale 20 integer", func(t *testing.T) {
		testDC(t, "int", "20k 1 1/p", "1.00000000000000000000\n")
	})
	t.Run("scale 20 zero", func(t *testing.T) {
		testDC(t, "zero", "20k 0 1/p", "0\n")
	})
	t.Run("scale 20 small", func(t *testing.T) {
		testDC(t, "small", "20k 1 2/p", ".50000000000000000000\n")
	})
}

func TestDcBoolean(t *testing.T) {
	tests := []struct{ name, input, expected string }{
		{"gt t", "2 1(pR", "1\n"},
		{"gt f", "0 1(pR", "0\n"},
		{"gte t", "1 1{pR", "1\n"},
		{"gte f", "0 1{pR", "0\n"},
		{"eq t", "1 1GpR", "1\n"},
		{"eq f", "0 1GpR", "0\n"},
		{"not f", "1NpR", "0\n"},
		{"not t", "0NpR", "1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDC(t, tt.name, tt.input, tt.expected)
		})
	}
}

func TestDcRegisters(t *testing.T) {
	t.Run("store and load", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "[Hello]sa", nil, &output)
		output = nil
		testEval(t, state, "lapR", "Hello\n")
	})
	t.Run("S push and L pop", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "1Sx 2Sx", nil, &output)
		output = nil
		evalDC(state, "LxpR", nil, &output)
		got := strings.Join(output, "\n")
		if !strings.Contains(got, "2") {
			t.Errorf("L: got %q, want containing 2", got)
		}
	})
	t.Run("load undefined", func(t *testing.T) {
		testDC(t, "undef", "lzpR", "0\n")
	})
	t.Run("s empty stack", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		if err := evalDC(state, "sa", nil, &output); err != nil {
			t.Fatal(err)
		}
		// Store empty string on empty stack
		if vals, ok := state.regs["a"]; !ok || len(vals) == 0 {
			t.Error("expected register 'a' to exist")
		} else if !vals[0].isStr || vals[0].str != "" {
			t.Errorf("expected empty string, got %v", vals[0])
		}
	})
}

func testEval(t *testing.T, state *dcState, input, expected string) {
	t.Helper()
	var output []string
	if err := evalDC(state, input, nil, &output); err != nil {
		t.Fatalf("evalDC error: %v", err)
	}
	got := strings.Join(output, "\n")
	if got != "" {
		got += "\n"
	}
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestDcConditionals(t *testing.T) {
	t.Run("> true", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "[1p]sa", nil, &output)
		output = nil
		testEval(t, state, "1 2>a 9p", "1\n9\n")
	})
	t.Run("< true", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "[2p]sb", nil, &output)
		output = nil
		testEval(t, state, "2 1<b 9p", "2\n9\n")
	})
	t.Run("= true", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "[5p]sc", nil, &output)
		output = nil
		testEval(t, state, "3 3=c 9p", "5\n9\n")
	})
	t.Run("> false", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "[1p]sa", nil, &output)
		output = nil
		testEval(t, state, "2 1>a 9p", "9\n")
	})
	t.Run("else clause", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "[1p]sa [2p]sb", nil, &output)
		output = nil
		testEval(t, state, "2 1>aeb 9p", "2\n9\n")
	})
	t.Run("! negated", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "[1p]sa", nil, &output)
		output = nil
		testEval(t, state, "2 1!>a 9p", "1\n9\n")
	})
}

func TestDcMacros(t *testing.T) {
	t.Run("x execute", func(t *testing.T) {
		// x executes macro which prints 3, then outer p prints remaining 3
		testDC(t, "x", "[1 2 + p]xpR", "3\n3\n")
	})
	t.Run("x non-string", func(t *testing.T) {
		state := &dcState{regs: make(map[string][]dcValue)}
		var output []string
		evalDC(state, "42 x", nil, &output)
		if len(output) != 0 {
			t.Errorf("x on non-string should do nothing, got %v", output)
		}
	})
}

func TestDcAscii(t *testing.T) {
	t.Run("a number", func(t *testing.T) {
		testDC(t, "a", "65apR", "A\n")
	})
}

func TestDcComments(t *testing.T) {
	t.Run("comment", func(t *testing.T) {
		testDC(t, "#", "# this is a comment\n1 2+p", "3\n")
	})
}

func TestDcErrors(t *testing.T) {
	tests := []struct{ name, input, wantErr string }{
		{"add empty", "+", "stack empty"},
		{"sub empty", "-", "stack empty"},
		{"mul empty", "*", "stack empty"},
		{"div empty", "/", "stack empty"},
		{"div zero", "1 0/p", "divide by zero"},
		{"mod empty", "%", "stack empty"},
		{"mod zero", "1 0%p", "remainder by zero"},
		{"divmod empty", "~", "stack empty"},
		{"divmod zero", "1 0~p", "divide by zero"},
		{"pow empty", "^", "stack empty"},
		{"sqrt empty", "v", "stack empty"},
		{"sqrt neg", "_1vp", "square root of negative number"},
		{"k empty", "k", "stack empty"},
		{"modexp empty1", "|", "stack empty"},
		{"modexp empty2", "1|", "stack empty"},
		{"modexp empty3", "1 2|", "stack empty"},
		{"gt empty", ">a", "stack empty"},
		{"a empty", "a", "stack empty"},
		{"N empty", "N", "stack empty"},
		{"( empty", "(", "stack empty"},
		{"{ empty", "{", "stack empty"},
		{"G empty", "G", "stack empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDCFail(t, tt.name, tt.input, tt.wantErr)
		})
	}
}

func TestDcJSONMode(t *testing.T) {
	var stdout, stderr strings.Builder
	rc := run([]string{"--json", "-e", "10 20+p"}, nil, &stdout, &stderr, "")
	if rc != 0 {
		t.Fatalf("run returned %d: %s", rc, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"output"`) {
		t.Errorf("JSON output missing output key: %s", out)
	}
	if !strings.Contains(out, `"30"`) {
		t.Errorf("JSON output missing 30: %s", out)
	}
}

func TestDcFileInput(t *testing.T) {
	var stdout, stderr strings.Builder
	rc := run([]string{"-e", "10 20+p"}, nil, &stdout, &stderr, "")
	if rc != 0 {
		t.Fatalf("run -e returned %d: %s", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "30") {
		t.Errorf("expected 30 in output, got %q", stdout.String())
	}
}

func TestDcPositionalArgs(t *testing.T) {
	var stdout, stderr strings.Builder
	rc := run([]string{"1", "2", "+", "p"}, strings.NewReader(""), &stdout, &stderr, "")
	if rc != 0 {
		t.Fatalf("run returned %d: %s", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "3") {
		t.Errorf("expected 3 in output, got %q", stdout.String())
	}
}

func TestDcStdinInput(t *testing.T) {
	var stdout, stderr strings.Builder
	rc := run(nil, strings.NewReader("5 6+p\n"), &stdout, &stderr, "")
	if rc != 0 {
		t.Fatalf("run stdin returned %d: %s", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "11") {
		t.Errorf("expected 11 in output, got %q", stdout.String())
	}
}

func TestDcWrapOutput(t *testing.T) {
	long := strings.Repeat("1234567890", 20) // 200 chars
	result := wrapOutput([]string{long})
	// Should produce multiple lines of max 69 chars + backslash
	for _, line := range result {
		if strings.HasSuffix(line, "\\") {
			if len(line) > 70 {
				t.Errorf("wrapped line too long: %d chars: %q", len(line), line)
			}
		}
	}
}

func TestDcNewNumStr(t *testing.T) {
	tests := []struct {
		input  string
		expect string
		isErr  bool
	}{
		{"0", "0", false},
		{"_5", "-5", false},
		{"-5", "-5", false},
		{"1.5", "3/2", false},
		{"0.5", "1/2", false},
		{"_0.5", "-1/2", false},
		{"", "0", false},
		{"007", "7", false},
		{"abc", "", true},
		{".5", "1/2", false},
		{"_123", "-123", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, _, err := newNumStr(tt.input)
			if tt.isErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			got := r.RatString()
			if got != tt.expect {
				t.Errorf("newNumStr(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestDcFormatRat(t *testing.T) {
	tests := []struct {
		rat    *big.Rat
		scale  int
		expect string
	}{
		{new(big.Rat).SetInt64(0), 0, "0"},
		{new(big.Rat).SetInt64(42), 0, "42"},
		{new(big.Rat).SetInt64(42), 5, "42.00000"},
		{ratFromStr("1/2"), 0, ".5"},
		{ratFromStr("1/2"), 5, ".50000"},
		{ratFromStr("-1/2"), 5, "-.50000"},
		{ratFromStr("1/3"), 2, ".33"},
		{ratFromStr("7/3"), 2, "2.33"},
		{ratFromStr("-1/1"), 0, "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			got := formatRat(tt.rat, tt.scale, false)
			if got != tt.expect {
				t.Errorf("formatRat(%s, %d) = %q, want %q",
					tt.rat.RatString(), tt.scale, got, tt.expect)
			}
		})
	}
}

func ratFromStr(s string) *big.Rat {
	r := new(big.Rat)
	r.SetString(s)
	return r
}

func TestDcRatModExp(t *testing.T) {
	r := ratModExpVal(
		new(big.Rat).SetInt64(2),
		new(big.Rat).SetInt64(10),
		new(big.Rat).SetInt64(1000),
	)
	if r.Num().Int64() != 24 {
		t.Errorf("2^10 mod 1000 should be 24, got %s", r.RatString())
	}

	// Zero result
	r = ratModExpVal(
		new(big.Rat).SetInt64(10),
		new(big.Rat).SetInt64(1),
		new(big.Rat).SetInt64(5),
	)
	if r.Num().Int64() != 0 {
		t.Errorf("10^1 mod 5 should be 0, got %s", r.RatString())
	}
}

func TestDcRatSqrt(t *testing.T) {
	tests := []struct {
		input  string
		scale  int
		expect string
	}{
		{"0", 0, "0"},
		{"4", 0, "2"},
		{"2", 5, "1.41421"},
		{"9", 0, "3"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, _ := new(big.Rat).SetString(tt.input)
			got := ratSqrtNewton(r, tt.scale)
			s := formatRat(got, tt.scale, false)
			if s != tt.expect {
				t.Errorf("sqrt(%s, %d) = %q, want %q",
					tt.input, tt.scale, s, tt.expect)
			}
		})
	}
}

func TestDcParseStringEscapes(t *testing.T) {
	tests := []struct{ input, expect string }{
		{`hello`, "hello"},
		{`hello\nworld`, "hello\nworld"},
		{`tab\there`, "tab\there"},
		{`slash\\here`, "slash\\here"},
		{`\q`, "\\q"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseStringEscapes(tt.input)
			if got != tt.expect {
				t.Errorf("parseStringEscapes(%q) = %q, want %q",
					tt.input, got, tt.expect)
			}
		})
	}
}

func TestDcReadline(t *testing.T) {
	// readLine reads until \n
	r := strings.NewReader("hello\nworld")
	got, err := readLine(r)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("readLine got %q, want %q", got, "hello")
	}
}

func TestDcRunError(t *testing.T) {
	var stdout, stderr strings.Builder
	rc := run([]string{"-f", "/nonexistent/file"}, nil, &stdout, &stderr, "")
	if rc == 0 {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestDcPopTwoNumVal(t *testing.T) {
	state := &dcState{regs: make(map[string][]dcValue)}
	state.stack = append(state.stack, dcValue{rat: new(big.Rat).SetInt64(42)})
	a, b, ok := popTwoNumVal(state)
	if ok {
		t.Errorf("popTwoNumVal with one element should fail, got a=%v b=%v", a, b)
	}
}

func TestDcExtendedRegisters(t *testing.T) {
	state := &dcState{
		regs:        make(map[string][]dcValue),
		extendedReg: true,
	}
	var output []string
	// Test storing and loading multi-char register
	err := evalDC(state, "42S xotj l xotj p", nil, &output)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(output, "\n")
	if got != "42" {
		t.Errorf("expected 42, got %q", got)
	}

	// Test conditionals with extended register name
	output = nil
	err = evalDC(state, "[1p]S cond 1 2> cond", nil, &output)
	if err != nil {
		t.Fatal(err)
	}
	got = strings.Join(output, "\n")
	if got != "1" {
		t.Errorf("expected 1, got %q", got)
	}
}

func TestDcRunExtendedMode(t *testing.T) {
	var stdout, stderr strings.Builder
	// Test running run function with -x flag
	rc := run([]string{"-x", "42S reg l reg p"}, nil, &stdout, &stderr, "")
	if rc != 0 {
		t.Fatalf("run failed: %s", stderr.String())
	}
	got := stdout.String()
	if got != "42\n" {
		t.Errorf("expected 42, got %q", got)
	}
}
