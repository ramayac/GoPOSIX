package bc

import (
	"bytes"
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
}
