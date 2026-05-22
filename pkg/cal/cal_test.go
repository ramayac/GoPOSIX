package cal

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCalMonthGregorian(t *testing.T) {
	// Replicate BusyBox's "cal 1 2000" test
	got := Run(2000, 1, false, false).Calendar
	want := strings.TrimPrefix(`
    January 2000
Su Mo Tu We Th Fr Sa
                   1
 2  3  4  5  6  7  8
 9 10 11 12 13 14 15
16 17 18 19 20 21 22
23 24 25 26 27 28 29
30 31
`, "\n")

	if got != want {
		t.Errorf("cal 1 2000 mismatch.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestCalMonthJulian(t *testing.T) {
	// Test Julian month output
	got := Run(2000, 1, true, false).Calendar
	want := strings.TrimPrefix(`
       January 2000
 Su  Mo  Tu  We  Th  Fr  Sa
                          1
  2   3   4   5   6   7   8
  9  10  11  12  13  14  15
 16  17  18  19  20  21  22
 23  24  25  26  27  28  29
 30  31
`, "\n")

	if got != want {
		t.Errorf("cal -j 1 2000 mismatch.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestCalMonthMondayStart(t *testing.T) {
	// Test Monday start month output
	got := Run(2000, 1, false, true).Calendar
	want := strings.TrimPrefix(`
    January 2000
Mo Tu We Th Fr Sa Su
                1  2
 3  4  5  6  7  8  9
10 11 12 13 14 15 16
17 18 19 20 21 22 23
24 25 26 27 28 29 30
31
`, "\n")

	if got != want {
		t.Errorf("cal -m 1 2000 mismatch.\nGot:\n%q\nWant:\n%q", got, want)
	}
}

func TestCalYearGregorian(t *testing.T) {
	// Test Year output
	got := Run(2000, 0, false, false).Calendar
	if !strings.Contains(got, "2000") {
		t.Errorf("expected year 2000 calendar to contain title '2000'")
	}
	if !strings.Contains(got, "January") || !strings.Contains(got, "December") {
		t.Errorf("expected year calendar to contain all months")
	}
}

func TestCalCLI(t *testing.T) {
	// Test standard CLI execution
	var stdout, stderr bytes.Buffer
	code := run([]string{"1", "2000"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("run(1, 2000) exited with %d, stderr: %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "January 2000") {
		t.Errorf("stdout doesn't contain standard output: %q", stdout.String())
	}
}

func TestCalCLIJSON(t *testing.T) {
	// Test CLI JSON output
	var stdout, stderr bytes.Buffer
	code := run([]string{"--json", "1", "2000"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("run(--json) exited with %d, stderr: %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"calendar"`) {
		t.Errorf("JSON output expected: %q", stdout.String())
	}
}

func TestCalCLINoArgs(t *testing.T) {
	// Test CLI with no arguments (current month/year)
	var stdout, stderr bytes.Buffer
	code := run([]string{}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("run() with no args failed: %d, stderr: %q", code, stderr.String())
	}
	now := time.Now()
	monthName := now.Month().String()
	if !strings.Contains(stdout.String(), monthName) {
		t.Errorf("expected current month %q in stdout, got: %q", monthName, stdout.String())
	}
}

func TestCalCLIYearOnly(t *testing.T) {
	// Test CLI with year only
	var stdout, stderr bytes.Buffer
	code := run([]string{"2000"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("run(2000) failed: %d, stderr: %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "2000") {
		t.Errorf("expected '2000' in year output")
	}
}

func TestCalCLIInvalidArgs(t *testing.T) {
	// Test invalid inputs
	cases := [][]string{
		{"13", "2000"}, // Invalid month
		{"0", "2000"},  // Invalid month
		{"1", "0"},     // Invalid year
		{"1", "10000"}, // Invalid year
		{"foo", "2000"}, // Non-numeric month string not matching month names
	}
	for _, c := range cases {
		var stdout, stderr bytes.Buffer
		code := run(c, nil, &stdout, &stderr, "")
		if code == 0 {
			t.Errorf("expected failure for args %v, but got exit 0", c)
		}
	}
}
