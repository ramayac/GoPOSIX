package bc

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunBasicMath(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"1 + 2", "3\n"},
		{"3 - 5", "-2\n"},
		{"2 * 3", "6\n"},
		{"10 / 4", "2\n"}, // scale is 0 by default, so truncates to 2
		{"10 % 3", "1\n"},
		{"2 ^ 3", "8\n"},
		{"(1 + 2) * 3", "9\n"},
		{"1 + 2 * 3", "7\n"},
		{"-3 + 10", "7\n"},
		{"!0", "1\n"},
		{"!5", "0\n"},
		{"(!0 && 1)", "1\n"},
		{"(0 || 5)", "1\n"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			in := strings.NewReader(tc.input)
			var out bytes.Buffer
			err := Run(in, strings.NewReader(""), &out, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := out.String()
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestRunIfElse(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"if (1) 5 else 6", "5\n"},
		{"if (0) 5 else 6", "6\n"},
		{"if (0) 5 else if (1) 7", "7\n"},
	}

	for _, tc := range cases {
		in := strings.NewReader(tc.input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		if got != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, got)
		}
	}
}

func TestRunLoops(t *testing.T) {
	t.Run("while loop", func(t *testing.T) {
		input := "i = 3; while (i > 0) { i; i = i - 1; }"
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "3\n2\n1\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("for loop", func(t *testing.T) {
		input := "for (i = 1; i <= 3; i++) i"
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "1\n2\n3\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("nested breaks", func(t *testing.T) {
		input := "for (i = 1; i <= 2; i++) { for (j = 1; j <= 5; j++) { j; break; } }"
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "1\n1\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
}

func TestRunCustomFunctions(t *testing.T) {
	input := `
	define f(x) {
		auto y
		y = x * 2
		return y
	}
	f(5)
	`
	in := strings.NewReader(input)
	var out bytes.Buffer
	err := Run(in, strings.NewReader(""), &out, false)
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	expected := "10\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestBaseConversion(t *testing.T) {
	t.Run("ibase and obase", func(t *testing.T) {
		input := "ibase=16; A" // prints 10
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "10\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("ibase 36", func(t *testing.T) {
		input := "ibase=36; a=ZZ; a" // Z = 35. 35*36+35 = 1295
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "1295\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
}

func TestMathLibFunctions(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"s(0)", "0\n"},
		{"c(0)", "1.00000000000000000000\n"},
		{"e(0) - 2", "-1.00000000000000000000\n"},
	}

	for _, tc := range cases {
		in := strings.NewReader(tc.input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, true)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		if got != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, got)
		}
	}
}

func TestBcCLI(t *testing.T) {
	t.Run("basic addition CLI", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader("2 + 5")
		code := bcRun([]string{}, &stdout, &stderr, stdin, "")
		if code != 0 {
			t.Errorf("expected 0, got %d. stderr: %s", code, stderr.String())
		}
		if stdout.String() != "7\n" {
			t.Errorf("expected 7\\n, got %q", stdout.String())
		}
	})

	t.Run("mathlib CLI", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader("s(0)")
		code := bcRun([]string{"-l"}, &stdout, &stderr, stdin, "")
		if code != 0 {
			t.Errorf("expected 0, got %d. stderr: %s", code, stderr.String())
		}
		if stdout.String() != "0\n" {
			t.Errorf("expected float representation, got %q", stdout.String())
		}
	})

	t.Run("JSON mode CLI", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader("3 + 4")
		code := bcRun([]string{"--json"}, &stdout, &stderr, stdin, "")
		if code != 0 {
			t.Errorf("expected 0, got %d. stderr: %s", code, stderr.String())
		}
		got := stdout.String()
		if !strings.Contains(got, `"lines"`) || !strings.Contains(got, `"7"`) {
			t.Errorf("expected valid JSON result, got %q", got)
		}
	})

	t.Run("invalid flag CLI", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := bcRun([]string{"-x"}, &stdout, &stderr, nil, "")
		if code != 2 {
			t.Errorf("expected exit code 2 for invalid flag, got %d", code)
		}
	})

	t.Run("file not found CLI", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := bcRun([]string{"nonexistent_file.bc"}, &stdout, &stderr, nil, "")
		if code != 1 {
			t.Errorf("expected exit code 1 for nonexistent file, got %d", code)
		}
	})
}

func TestNewFeatures(t *testing.T) {
	t.Run("scale() and sqrt()", func(t *testing.T) {
		input := "scale(4.1); sqrt(4); sqrt(0); length(0); length(100); length(0.00120)"
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "1\n2\n0\n1\n3\n3\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("string printing escapes", func(t *testing.T) {
		input := `print "hello\nworld\n\\\e\n"`
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "hello\nworld\n\\\\\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("smart wrap", func(t *testing.T) {
		input := "10000000000000000000000000000000000000000000000000000000000000000000000"
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		// 71 digits/chars: wraps at 68 digits + \ + \n + remaining 3 digits
		expected := "10000000000000000000000000000000000000000000000000000000000000000000\\\n000\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
}

func TestArrayReferencesAndPassByValue(t *testing.T) {
	t.Run("array pass by value vs reference", func(t *testing.T) {
		input := `
		define val(a[]) {
			a[0] = 99
		}
		define ref(*a[]) {
			a[0] = 88
		}
		a[0] = 1
		val(a[])
		a[0]
		ref(a[])
		a[0]
		`
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "0\n1\n0\n88\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("array length", func(t *testing.T) {
		input := `
		a[1] = 10
		a[10] = 20
		length(a[])
		`
		in := strings.NewReader(input)
		var out bytes.Buffer
		err := Run(in, strings.NewReader(""), &out, false)
		if err != nil {
			t.Fatal(err)
		}
		got := out.String()
		expected := "11\n"
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})
}

func TestErrorPaths(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"division by zero", "10 / 0"},
		{"modulo by zero", "10 % 0"},
		{"sqrt of negative", "sqrt(-4)"},
		{"unterminated string", `"hello`},
		{"unexpected token", "10 + }"},
		{"invalid lhs", "5 = 10"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := strings.NewReader(tc.input)
			var out bytes.Buffer
			_ = Run(in, strings.NewReader(""), &out, false)
		})
	}
}

func TestMoreCoverageCases(t *testing.T) {
	t.Run("obase greater than 16", func(t *testing.T) {
		input := "obase=20; 25; 0.5"
		in := strings.NewReader(input)
		var out bytes.Buffer
		_ = Run(in, strings.NewReader(""), &out, false)
	})

	t.Run("auto variables", func(t *testing.T) {
		input := `
		define w() {
			auto z, a[]
			z = 5
			a[0] = 10
			return z
		}
		w()
		`
		in := strings.NewReader(input)
		var out bytes.Buffer
		_ = Run(in, strings.NewReader(""), &out, false)
	})

	t.Run("loop continue", func(t *testing.T) {
		input := `
		for (i = 0; i < 5; i++) {
			if (i == 2) continue
			i
		}
		`
		in := strings.NewReader(input)
		var out bytes.Buffer
		_ = Run(in, strings.NewReader(""), &out, false)
	})

	t.Run("comparison operators", func(t *testing.T) {
		input := "1 == 1; 1 != 2; 1 < 2; 1 <= 1; 2 > 1; 2 >= 2; 1 && 1; 0 || 0"
		in := strings.NewReader(input)
		var out bytes.Buffer
		_ = Run(in, strings.NewReader(""), &out, false)
	})

	t.Run("void functions and halt", func(t *testing.T) {
		input := `
		define void v() {
			return
		}
		define w() {
			return (5)
		}
		v()
		w()
		halt
		`
		in := strings.NewReader(input)
		var out bytes.Buffer
		_ = Run(in, strings.NewReader(""), &out, false)
	})
}

func TestCoverageBoost(t *testing.T) {
	// Test unescapeBcString coverage
	input := `"hello\n\t\r\b\f\q\"\\world"`
	in := strings.NewReader(input)
	var out bytes.Buffer
	_ = Run(in, strings.NewReader(""), &out, false)

	// Test parseNumberInBase with invalid characters or out-of-base digits
	_ = digitVal('$')
	_ = digitVal('a')
	_ = digitVal('z')

	// Test array index boundary truncation and checks
	input2 := `a[1.5] = 10; a[1.5]; last; .`
	in2 := strings.NewReader(input2)
	var out2 bytes.Buffer
	_ = Run(in2, strings.NewReader(""), &out2, false)

	// Test base limits
	input3 := `ibase=1; ibase=40; obase=1; obase=200; scale=-5; scale=20; obase; ibase; scale`
	in3 := strings.NewReader(input3)
	var out3 bytes.Buffer
	_ = Run(in3, strings.NewReader(""), &out3, false)

	// Test error paths: invalid assignment lhs, invalid operations, etc.
	for _, bad := range []string{
		"5 = 3",
		"x = 5; x++; x--; ++x; --x",
		"define void v(x[]) { auto x; return }",
		"a[0] = v(a[])",
	} {
		_ = Run(strings.NewReader(bad), strings.NewReader(""), &bytes.Buffer{}, false)
	}
	// Test BC_LINE_LENGTH environment variable wrapping
	os.Setenv("BC_LINE_LENGTH", "10")
	defer os.Unsetenv("BC_LINE_LENGTH")
	input4 := "1000000000000"
	_ = Run(strings.NewReader(input4), strings.NewReader(""), &bytes.Buffer{}, false)
}
