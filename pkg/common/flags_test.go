package common

import (
	"testing"
)

// testSpec is a convenience spec used by most tests.
var testSpec = FlagSpec{
	Defs: []FlagDef{
		{Short: "l", Long: "list", Type: FlagBool},
		{Short: "a", Long: "all", Type: FlagBool},
		{Short: "R", Long: "recursive", Type: FlagBool},
		{Short: "v", Long: "verbose", Type: FlagBool},
		{Short: "n", Long: "no-newline", Type: FlagBool},
		{Short: "e", Long: "escape", Type: FlagBool},
		{Short: "P", Long: "physical", Type: FlagBool},
		{Short: "i", Long: "ignore-env", Type: FlagBool},
		{Short: "o", Long: "output", Type: FlagValue},
	},
}

func TestGroupedShortFlags(t *testing.T) {
	args := []string{"-laR", "/tmp"}
	result, err := ParseFlags(args, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Has("l") || !result.Has("a") || !result.Has("R") {
		t.Errorf("expected l, a, R flags to be set; got %+v", result.Bools)
	}
	if len(result.Positional) != 1 || result.Positional[0] != "/tmp" {
		t.Errorf("expected positional [/tmp], got %v", result.Positional)
	}
}

func TestLongFlagBool(t *testing.T) {
	result, err := ParseFlags([]string{"--all", "--recursive"}, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Has("all") || !result.Has("recursive") {
		t.Errorf("long flags not set: %+v", result.Bools)
	}
	if !result.Has("a") || !result.Has("R") {
		t.Errorf("short aliases not set: %+v", result.Bools)
	}
}

func TestLongFlagEqValue(t *testing.T) {
	result, err := ParseFlags([]string{"--output=foo.txt"}, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Get("output") != "foo.txt" {
		t.Errorf("expected output=foo.txt, got %q", result.Get("output"))
	}
}

func TestLongFlagSpaceValue(t *testing.T) {
	result, err := ParseFlags([]string{"--output", "bar.txt"}, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Get("output") != "bar.txt" {
		t.Errorf("expected output=bar.txt, got %q", result.Get("output"))
	}
}

func TestEndOfFlags(t *testing.T) {
	args := []string{"--", "-not-a-flag"}
	result, err := ParseFlags(args, FlagSpec{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Positional) != 1 || result.Positional[0] != "-not-a-flag" {
		t.Errorf("expected positional [-not-a-flag], got %v", result.Positional)
	}
}

func TestStdinMarker(t *testing.T) {
	result, err := ParseFlags([]string{"-"}, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Stdin {
		t.Error("expected Stdin=true")
	}
	if len(result.Positional) != 1 || result.Positional[0] != "-" {
		t.Errorf("expected '-' in Positional, got %v", result.Positional)
	}
}

func TestUnknownFlag(t *testing.T) {
	_, err := ParseFlags([]string{"-z"}, FlagSpec{})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	fe, ok := err.(*FlagError)
	if !ok {
		t.Fatalf("expected *FlagError, got %T", err)
	}
	if fe.ExitCode != 2 {
		t.Errorf("expected ExitCode 2, got %d", fe.ExitCode)
	}
}

func TestUnknownLongFlag(t *testing.T) {
	_, err := ParseFlags([]string{"--nope"}, FlagSpec{})
	if err == nil {
		t.Fatal("expected error for unknown long flag")
	}
	fe, _ := err.(*FlagError)
	if fe.ExitCode != 2 {
		t.Errorf("expected ExitCode 2, got %d", fe.ExitCode)
	}
}

func TestFlagRepetitionCounting(t *testing.T) {
	result, err := ParseFlags([]string{"-v", "-v", "-v"}, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count["v"] != 3 {
		t.Errorf("expected verbosity count 3, got %d", result.Count["v"])
	}
}

func TestFlagsMixedWithPositional(t *testing.T) {
	result, err := ParseFlags([]string{"-l", "/tmp", "-a", "/home"}, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Has("l") || !result.Has("a") {
		t.Errorf("expected l and a flags; got %+v", result.Bools)
	}
	if len(result.Positional) != 2 {
		t.Errorf("expected 2 positionals, got %v", result.Positional)
	}
}

func TestEmptyArgs(t *testing.T) {
	result, err := ParseFlags([]string{}, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Positional) != 0 {
		t.Errorf("expected no positionals, got %v", result.Positional)
	}
}

func TestShortValueInCluster(t *testing.T) {
	result, err := ParseFlags([]string{"-ofoo.txt"}, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Get("o") != "foo.txt" {
		t.Errorf("expected o=foo.txt, got %q", result.Get("o"))
	}
}

func TestOptionalValueFlag(t *testing.T) {
	spec := FlagSpec{
		Defs: []FlagDef{
			{Short: "e", Long: "eof", Type: FlagOptionalValue},
		},
	}
	result, err := ParseFlags([]string{"-e"}, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Has("e") {
		t.Error("expected -e to be set")
	}
	if result.Get("e") != "" {
		t.Errorf("expected empty value, got %q", result.Get("e"))
	}
}

func TestOptionalValueFlagWithValue(t *testing.T) {
	spec := FlagSpec{
		Defs: []FlagDef{
			{Short: "e", Long: "eof", Type: FlagOptionalValue},
		},
	}
	result, err := ParseFlags([]string{"-eEOFSTR"}, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Get("e") != "EOFSTR" {
		t.Errorf("expected EOFSTR, got %q", result.Get("e"))
	}
}

func TestLongOptionalValue(t *testing.T) {
	spec := FlagSpec{
		Defs: []FlagDef{
			{Long: "eof", Type: FlagOptionalValue},
		},
	}
	result, err := ParseFlags([]string{"--eof"}, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Has("eof") {
		t.Error("expected --eof to be set")
	}
	if result.Get("eof") != "" {
		t.Errorf("expected empty value, got %q", result.Get("eof"))
	}

	result2, err := ParseFlags([]string{"--eof=STR"}, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.Get("eof") != "STR" {
		t.Errorf("expected STR, got %q", result2.Get("eof"))
	}
}

func TestStopAtFirstNonFlag(t *testing.T) {
	spec := FlagSpec{
		Defs: []FlagDef{
			{Short: "n", Type: FlagBool},
			{Short: "e", Type: FlagBool},
		},
		StopAtFirstNonFlag: true,
	}
	result, err := ParseFlags([]string{"-n", "hello", "-e", "world"}, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Has("n") {
		t.Error("expected -n to be set")
	}
	if result.Has("e") {
		t.Error("-e should NOT be set (it's positional)")
	}
	if len(result.Positional) != 3 {
		t.Errorf("expected 3 positionals, got %d: %v", len(result.Positional), result.Positional)
	}
	if result.Positional[0] != "hello" || result.Positional[1] != "-e" || result.Positional[2] != "world" {
		t.Errorf("unexpected positionals: %v", result.Positional)
	}
}

// Benchmarks

var benchSpec = FlagSpec{
	Defs: []FlagDef{
		{Short: "i", Long: "ignore-case", Type: FlagBool},
		{Short: "v", Long: "invert-match", Type: FlagBool},
		{Short: "c", Long: "count", Type: FlagBool},
		{Short: "n", Long: "line-number", Type: FlagBool},
		{Short: "l", Long: "files-with-matches", Type: FlagBool},
		{Short: "o", Long: "only-matching", Type: FlagBool},
		{Short: "q", Long: "quiet", Type: FlagBool},
		{Short: "f", Long: "file", Type: FlagValue},
		{Short: "e", Long: "regexp", Type: FlagValue},
		{Long: "json", Type: FlagBool},
	},
}

var groupedBenchSpec = FlagSpec{
	Defs: []FlagDef{
		{Short: "l", Long: "list", Type: FlagBool},
		{Short: "a", Long: "all", Type: FlagBool},
		{Short: "R", Long: "recursive", Type: FlagBool},
	},
}

var longBenchSpec = FlagSpec{
	Defs: []FlagDef{
		{Short: "i", Long: "ignore-case", Type: FlagBool},
		{Short: "c", Long: "count", Type: FlagBool},
		{Short: "n", Long: "line-number", Type: FlagBool},
		{Short: "e", Long: "regexp", Type: FlagValue},
	},
}

func BenchmarkParseFlags_Typical(b *testing.B) {
	args := []string{"-i", "-v", "-n", "--regexp=foo", "-e", "bar", "file1.txt", "file2.txt"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseFlags(args, benchSpec)
	}
}

func BenchmarkParseFlags_Grouped(b *testing.B) {
	args := []string{"-laR", "/tmp", "/home"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseFlags(args, groupedBenchSpec)
	}
}

func BenchmarkParseFlags_LongOnly(b *testing.B) {
	args := []string{"--ignore-case", "--count", "--line-number", "--regexp=pat", "file.txt"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseFlags(args, longBenchSpec)
	}
}

func BenchmarkParseFlags_ShortOnly(b *testing.B) {
	args := []string{"-i", "-v", "-c", "-n", "-l", "-o", "-q", "file.txt"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseFlags(args, benchSpec)
	}
}
