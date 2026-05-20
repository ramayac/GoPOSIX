package expr

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEvalArithmetic(t *testing.T) {
	tests := []struct {
		tokens []string
		want   string
		code   int
	}{
		{[]string{"3", "+", "5"}, "8", 0},
		{[]string{"10", "-", "3"}, "7", 0},
		{[]string{"4", "*", "5"}, "20", 0},
		{[]string{"20", "/", "4"}, "5", 0},
		{[]string{"17", "%", "5"}, "2", 0},
		{[]string{"0", "+", "0"}, "0", 1}, // result is zero → exit 1
		{[]string{"-3", "+", "1"}, "-2", 0},
	}
	for _, tc := range tests {
		result, code, err := Eval(tc.tokens)
		if err != nil {
			t.Errorf("Eval(%v) error: %v", tc.tokens, err)
			continue
		}
		if result != tc.want {
			t.Errorf("Eval(%v) = %q, want %q", tc.tokens, result, tc.want)
		}
		if code != tc.code {
			t.Errorf("Eval(%v) exit = %d, want %d", tc.tokens, code, tc.code)
		}
	}
}

func TestEvalPrecedence(t *testing.T) {
	// 2 + 3 * 4 = 14 (not 20)
	result, _, err := Eval([]string{"2", "+", "3", "*", "4"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "14" {
		t.Errorf("got %q, want %q", result, "14")
	}
}

func TestEvalParentheses(t *testing.T) {
	// (2 + 3) * 4 = 20
	result, _, err := Eval([]string{"(", "2", "+", "3", ")", "*", "4"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "20" {
		t.Errorf("got %q, want %q", result, "20")
	}
}

func TestEvalComparison(t *testing.T) {
	tests := []struct {
		tokens []string
		want   string
	}{
		{[]string{"5", "=", "5"}, "1"},
		{[]string{"5", "!=", "3"}, "1"},
		{[]string{"3", "<", "5"}, "1"},
		{[]string{"5", "<=", "5"}, "1"},
		{[]string{"5", ">", "3"}, "1"},
		{[]string{"5", ">=", "5"}, "1"},
		{[]string{"5", "=", "3"}, "0"},
		{[]string{"5", "<", "3"}, "0"},
	}
	for _, tc := range tests {
		result, _, err := Eval(tc.tokens)
		if err != nil {
			t.Errorf("Eval(%v) error: %v", tc.tokens, err)
			continue
		}
		if result != tc.want {
			t.Errorf("Eval(%v) = %q, want %q", tc.tokens, result, tc.want)
		}
	}
}

func TestEvalStringComparison(t *testing.T) {
	result, _, err := Eval([]string{"abc", "<", "def"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "1" {
		t.Errorf("got %q, want %q", result, "1")
	}
}

func TestEvalLogicalOr(t *testing.T) {
	// 0 | 5 → 5
	result, _, err := Eval([]string{"0", "|", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "5" {
		t.Errorf("got %q, want %q", result, "5")
	}

	// 3 | 5 → 3
	result, _, err = Eval([]string{"3", "|", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "3" {
		t.Errorf("got %q, want %q", result, "3")
	}
}

func TestEvalLogicalAnd(t *testing.T) {
	// 3 & 5 → 3 (both non-zero)
	result, _, err := Eval([]string{"3", "&", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "3" {
		t.Errorf("got %q, want %q", result, "3")
	}

	// 0 & 5 → 0 (left is zero)
	result, _, err = Eval([]string{"0", "&", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "0" {
		t.Errorf("got %q, want %q", result, "0")
	}
}

func TestEvalMatch(t *testing.T) {
	// match "hello123world" "[a-z]*\([0-9]*\)" → "123" (captured group)
	result, _, err := Eval([]string{"hello123world", ":", "[a-z]*([0-9]*)"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "123" {
		t.Errorf("got %q, want %q", result, "123")
	}
}

func TestEvalMatchNoGroup(t *testing.T) {
	// match without group returns length
	result, _, err := Eval([]string{"hello", ":", "hel"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "3" {
		t.Errorf("got %q, want %q", result, "3")
	}
}

func TestEvalMatchFail(t *testing.T) {
	result, code, err := Eval([]string{"hello", ":", "xyz"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "0" {
		t.Errorf("got %q, want %q", result, "0")
	}
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
}

func TestEvalSubstr(t *testing.T) {
	result, _, err := Eval([]string{"substr", "hello", "2", "3"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "ell" {
		t.Errorf("got %q, want %q", result, "ell")
	}
}

func TestEvalIndex(t *testing.T) {
	result, _, err := Eval([]string{"index", "hello", "lo"})
	if err != nil {
		t.Fatal(err)
	}
	// First occurrence of 'l' or 'o' in "hello" is at position 3 ('l')
	if result != "3" {
		t.Errorf("got %q, want %q", result, "3")
	}
}

func TestEvalLength(t *testing.T) {
	result, _, err := Eval([]string{"length", "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "5" {
		t.Errorf("got %q, want %q", result, "5")
	}
}

func TestEvalDivisionByZero(t *testing.T) {
	_, _, err := Eval([]string{"5", "/", "0"})
	if err == nil {
		t.Fatal("expected division by zero error")
	}
}

func TestEvalModuloByZero(t *testing.T) {
	_, _, err := Eval([]string{"5", "%", "0"})
	if err == nil {
		t.Fatal("expected division by zero error")
	}
}

func TestEvalEmpty(t *testing.T) {
	_, code, err := Eval([]string{})
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestEvalNonInteger(t *testing.T) {
	_, _, err := Eval([]string{"abc", "+", "3"})
	if err == nil {
		t.Fatal("expected non-integer error")
	}
}

func TestRunCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"3", "+", "5"}, nil, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	if buf.String() != "8\n" {
		t.Errorf("got %q, want %q", buf.String(), "8\n")
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "3", "+", "5"}, nil, &buf)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data := env["data"].(map[string]interface{})
	if data["result"] != "8" {
		t.Errorf("got %v, want %q", data["result"], "8")
	}
}

func TestRunCLIExitOne(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"5", "=", "3"}, nil, &buf)
	// Result is "0" → exit code 1
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestEvalMatchKeyword(t *testing.T) {
	// "match" keyword form — POSIX expr is anchored at start
	result, _, err := Eval([]string{"match", "123abc", "([0-9]+)"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "123" {
		t.Errorf("got %q, want %q", result, "123")
	}
}

func TestEvalComplexExpression(t *testing.T) {
	// (1 + 2) * (3 + 4) = 21
	result, _, err := Eval([]string{"(", "1", "+", "2", ")", "*", "(", "3", "+", "4", ")"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "21" {
		t.Errorf("got %q, want %q", result, "21")
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Expr_BigNumbers(t *testing.T) {
	// BusyBox: expr 0 '<' 3000000000 → 1 (handles >32-bit)
	result, _, err := Eval([]string{"0", "<", "3000000000"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "1" {
		t.Errorf("got %q, want 1", result)
	}
}

func TestBusyBox_Expr_BigNegativeNumbers(t *testing.T) {
	// BusyBox: expr -9223372036854775800 '<' 9223372036854775807 → 1
	result, _, err := Eval([]string{"-9223372036854775800", "<", "9223372036854775807"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "1" {
		t.Errorf("got %q, want 1", result)
	}
}

func TestBusyBox_Expr_NegativeArg(t *testing.T) {
	// Verify that -5 is treated as a positional arg, not a flag
	result, _, err := Eval([]string{"-5", "+", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "-2" {
		t.Errorf("got %q, want -2", result)
	}
}
