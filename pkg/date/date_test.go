package date

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestIsDigit(t *testing.T) {
	if !isDigit('5') {
		t.Error("expected '5' to be digit")
	}
	if !isDigit('0') {
		t.Error("expected '0' to be digit")
	}
	if !isDigit('9') {
		t.Error("expected '9' to be digit")
	}
	if isDigit('a') {
		t.Error("expected 'a' not to be digit")
	}
	if isDigit('-') {
		t.Error("expected '-' not to be digit")
	}
}

func TestIsLeapYear(t *testing.T) {
	// Typical leap years.
	for _, y := range []int{2000, 2004, 2008, 2012, 2016, 2020, 2024, 2400} {
		if !isLeapYear(y) {
			t.Errorf("expected %d to be leap year", y)
		}
	}
	// Non-leap years.
	for _, y := range []int{1900, 2001, 2002, 2003, 2005, 2006, 2007, 2100} {
		if isLeapYear(y) {
			t.Errorf("expected %d NOT to be leap year", y)
		}
	}
}

func TestDateRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-u"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if out.String() == "" {
		t.Error("expected output")
	}
}

func TestDateJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "jsonrpc") && !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Date_DashD_UnixTimestamp(t *testing.T) {
	// date -d @1288486801 — Unix timestamp via @ prefix
	var out bytes.Buffer
	rc := run([]string{"-d", "@1288486801"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if !strings.Contains(out.String(), "2010") {
		t.Errorf("expected year 2010 in output: %q", out.String())
	}
}

func TestBusyBox_Date_DashD_DottedFormat(t *testing.T) {
	// date -d 1999.01.01-11:22:33 '+%d/%m/%y'
	var out bytes.Buffer
	rc := run([]string{"-d", "1999.01.01-11:22:33", "+%d/%m/%y"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if strings.TrimSpace(out.String()) != "01/01/99" {
		t.Errorf("got %q, want %q", strings.TrimSpace(out.String()), "01/01/99")
	}
}

func TestBusyBox_Date_DashD_YMD_HMS(t *testing.T) {
	// date -d '1999-1-2 3:4:5'
	var out bytes.Buffer
	rc := run([]string{"-d", "1999-1-2 3:4:5"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	s := strings.TrimSpace(out.String())
	if !strings.Contains(s, "Sat Jan  2 03:04:05") {
		t.Errorf("got %q, want to contain 'Sat Jan  2 03:04:05'", s)
	}
}

func TestBusyBox_Date_DashD_Compact(t *testing.T) {
	// date -d 200001231133
	var out bytes.Buffer
	rc := run([]string{"-d", "200001231133"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	s := strings.TrimSpace(out.String())
	if !strings.Contains(s, "Jan 23 11:33:00") {
		t.Errorf("got %q, want to contain 'Jan 23 11:33:00'", s)
	}
}

func TestBusyBox_Date_DashD_CompactWithSeconds(t *testing.T) {
	// date -d 200001231133.30
	var out bytes.Buffer
	rc := run([]string{"-d", "200001231133.30"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	s := strings.TrimSpace(out.String())
	if !strings.Contains(s, "Jan 23 11:33:30") {
		t.Errorf("got %q, want to contain 'Jan 23 11:33:30'", s)
	}
}

func TestBusyBox_Date_DashD_TimeOnly(t *testing.T) {
	// date -d 1:2 should parse as today at 01:02
	var out bytes.Buffer
	rc := run([]string{"-d", "1:2"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %v, want 0 (failed to parse '1:2')", rc)
	}
	s := strings.TrimSpace(out.String())
	// Should contain 01:02:00 somewhere
	if !strings.Contains(s, "01:02") {
		t.Errorf("got %q, want to contain '01:02'", s)
	}
}

func TestBusyBox_Date_DashD_TimeOnlySeconds(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-d", "1:2:3"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	s := strings.TrimSpace(out.String())
	if !strings.Contains(s, "01:02:03") {
		t.Errorf("got %q, want to contain '01:02:03'", s)
	}
}

func TestBusyBox_Date_DashD_MonthDay_Time(t *testing.T) {
	// date -d 1.2-3:4 → today's year, Jan 2, 03:04
	var out bytes.Buffer
	rc := run([]string{"-d", "1.2-3:4"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %v, want 0 (failed to parse '1.2-3:4')", rc)
	}
	s := strings.TrimSpace(out.String())
	if !strings.Contains(s, "Jan  2 03:04:00") {
		t.Errorf("got %q, want to contain 'Jan  2 03:04:00'", s)
	}
}

func TestBusyBox_Date_DashD_MonthDay_TimeSeconds(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-d", "1.2-3:4:5"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %v, want 0", rc)
	}
	s := strings.TrimSpace(out.String())
	if !strings.Contains(s, "Jan  2 03:04:05") {
		t.Errorf("got %q, want to contain 'Jan  2 03:04:05'", s)
	}
}

func TestBusyBox_Date_DashU(t *testing.T) {
	// date -u -d 2000.01.01-11:22:33 → UTC output
	var out bytes.Buffer
	rc := run([]string{"-u", "-d", "2000.01.01-11:22:33"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	s := strings.TrimSpace(out.String())
	if !strings.Contains(s, "UTC") {
		t.Errorf("got %q, want UTC in output", s)
	}
	if !strings.Contains(s, "11:22:33") {
		t.Errorf("got %q, want 11:22:33 in output", s)
	}
}

func TestBusyBox_Date_FormatStrftime(t *testing.T) {
	// +%T format
	var out bytes.Buffer
	rc := run([]string{"-d", "1:2:3", "+%T"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if strings.TrimSpace(out.String()) != "01:02:03" {
		t.Errorf("got %q, want %q", strings.TrimSpace(out.String()), "01:02:03")
	}
}

func TestBusyBox_Date_FormatC(t *testing.T) {
	// +%c format (locale date/time)
	var out bytes.Buffer
	rc := run([]string{"-d", "200001231133", "+%c"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	s := strings.TrimSpace(out.String())
	if !strings.Contains(s, "Jan 23 11:33:00") {
		t.Errorf("got %q, want to contain 'Jan 23 11:33:00'", s)
	}
}

func TestBusyBox_Date_FormatPercentD(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-d", "1999.01.01-11:22:33", "+%d"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if strings.TrimSpace(out.String()) != "01" {
		t.Errorf("got %q, want '01'", strings.TrimSpace(out.String()))
	}
}

func TestBusyBox_Date_FormatPercentY(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-d", "200001231133", "+%Y"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if strings.TrimSpace(out.String()) != "2000" {
		t.Errorf("got %q, want '2000'", strings.TrimSpace(out.String()))
	}
}

func TestBusyBox_Date_FormatPercentYPercentm(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-d", "1999.01.01-11:22:33", "+%Y%m"}, nil, &out, &out, "")
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if strings.TrimSpace(out.String()) != "199901" {
		t.Errorf("got %q, want '199901'", strings.TrimSpace(out.String()))
	}
}

func TestBusyBox_Date_RejectsExtraArgs(t *testing.T) {
	// date -d 012311332000.30 %+c → should reject extra non-format arg
	var buf bytes.Buffer
	code := run([]string{"-d", "012311332000.30", "%+c"}, nil, &buf, &buf, "")
	if code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
}

func TestPOSIXDateSpecifiers(t *testing.T) {
	// We want to test a specific date to verify all format specifiers:
	// Date: 2026-05-19 21:47:58 (Tuesday, 139th day of year)
	// We can use the "-d" parsing. Let's construct a dotted compact string: 202605192147.58
	tests := []struct {
		format   string
		expected string
	}{
		{"+%j", "139"},         // day of year
		{"+%p", "PM"},          // AM/PM
		{"+%r", "09:47:58 PM"}, // 12-hour clock
		{"+%u", "2"},           // weekday [1-7], Tuesday = 2
		{"+%V", "21"},          // ISO 8601 week number (2026-05-19 is in week 21)
		{"+%W", "20"},          // Week of year, Monday first
		{"+%U", "20"},          // Week of year, Sunday first
		{"+%n", "\n"},          // newline
		{"+%t", "\t"},          // tab
		{"+%D", "05/19/26"},    // %m/%d/%y
		{"+%F", "2026-05-19"},  // ISO date
		{"+%R", "21:47"},       // %H:%M
		{"+%w", "2"},           // weekday [0-6], Sunday = 0
		{"+%k", "21"},          // hour 0-23 space padded
		{"+%l", " 9"},          // hour 1-12 space padded
	}

	for _, tc := range tests {
		var out bytes.Buffer
		rc := run([]string{"-d", "2026.05.19-21:47:58", tc.format}, nil, &out, &out, "")
		if rc != 0 {
			t.Fatalf("failed to run date for format %s: exit code %d", tc.format, rc)
		}
		got := out.String()
		// strip trailing newline which run() appends
		if len(got) > 0 && got[len(got)-1] == '\n' {
			got = got[:len(got)-1]
		}
		if got != tc.expected {
			t.Errorf("format %s: got %q, want %q", tc.format, got, tc.expected)
		}
	}
}

func TestPOSIXTZ_ParsingAndEvaluation(t *testing.T) {
	// Test parsePOSIXTZ
	tzStr := "EET-2EEST,M3.5.0/3,M10.5.0/4"
	tz, ok := parsePOSIXTZ(tzStr)
	if !ok {
		t.Fatalf("failed to parse POSIX TZ spec: %q", tzStr)
	}

	if tz.stdName != "EET" || tz.stdOffset != 7200 {
		t.Errorf("std zone mismatch: got %s offset %d", tz.stdName, tz.stdOffset)
	}
	if tz.dstName != "EEST" || tz.dstOffset != 10800 {
		t.Errorf("dst zone mismatch: got %s offset %d", tz.dstName, tz.dstOffset)
	}

	// Oct 31, 2010 at 00:59:59 UTC
	// Standard transition starts M3.5.0/3, ends M10.5.0/4.
	// Since 00:59:59 UTC is 03:59:59 EEST (1 sec before 04:00:00 EEST transition/01:00:00 UTC),
	// it must evaluate as DST!
	t1 := time.Date(2010, 10, 31, 0, 59, 59, 0, time.UTC)
	name, offset := tz.eval(t1)
	if name != "EEST" || offset != 10800 {
		t.Errorf("expected EEST/10800 at %v, got %s/%d", t1, name, offset)
	}

	// Oct 31, 2010 at 01:00:00 UTC
	// At exactly transition boundary, it must evaluate to standard time EET!
	t2 := time.Date(2010, 10, 31, 1, 0, 0, 0, time.UTC)
	name, offset = tz.eval(t2)
	if name != "EET" || offset != 7200 {
		t.Errorf("expected EET/7200 at %v, got %s/%d", t2, name, offset)
	}
}

func TestDateRun_InvalidDate_MulticallHeader(t *testing.T) {
	// Rejects invalid positional argument
	rc := run([]string{"-d", "012311332000.30", "%+c"}, nil, nil, nil, "")
	if rc != 1 {
		t.Errorf("expected exit code 1, got %d", rc)
	}
}

func TestParsePOSIXTZ_AngleBrackets(t *testing.T) {
	// TZ with angle-bracket names: "<UTC+5>-5"
	tz, ok := parsePOSIXTZ("<UTC+5>-5")
	if !ok {
		t.Fatal("expected valid TZ")
	}
	if tz.stdName != "UTC+5" {
		t.Errorf("expected stdName UTC+5, got %q", tz.stdName)
	}
	if tz.stdOffset != 5*3600 {
		t.Errorf("expected stdOffset 18000 (POSIX -5 = UTC+5 = east), got %d", tz.stdOffset)
	}
}

func TestParsePOSIXTZ_Empty(t *testing.T) {
	_, ok := parsePOSIXTZ("")
	if ok {
		t.Error("empty TZ should not be valid")
	}
}

func TestParsePOSIXTZ_DST(t *testing.T) {
	// Standard POSIX TZ with DST: "EST5EDT,M3.2.0/2,M11.1.0/2"
	tz, ok := parsePOSIXTZ("EST5EDT,M3.2.0/2,M11.1.0/2")
	if ok {
		if tz.stdName != "EST" {
			t.Errorf("expected stdName EST, got %q", tz.stdName)
		}
		if !tz.hasDST {
			t.Error("expected DST")
		}
	}
}

func TestParsePOSIXTZ_NoDST(t *testing.T) {
	// Simple offset: "UTC-5"
	tz, ok := parsePOSIXTZ("UTC-5")
	if ok {
		if tz.hasDST {
			t.Error("UTC-5 should not have DST")
		}
	}
}

func TestPosixTZ_Eval_JulianNoLeap(t *testing.T) {
	// Rule with isJulianNoLeap, testing leap year handling.
	r := &posixTZRule{
		isStart:       false,
		isJulianNoLeap: true,
		julianDay:     60,
		timeOfTransition: 7200,
	}
	// In a leap year (2024), day >= 60 subtracts 0, in non-leap subtracts 1.
	_ = r.eval(2024, -18000, -14400)
	_ = r.eval(2023, -18000, -14400)
}

func TestPosixTZ_Eval_MonthWeekDay(t *testing.T) {
	r := &posixTZRule{
		isStart:          true,
		isMonthWeekDay:   true,
		month:            3,
		week:             2,
		weekday:          0,
		timeOfTransition: 7200,
	}
	_ = r.eval(2024, -18000, -14400)
}

func TestPosixTZ_Eval_LastWeek(t *testing.T) {
	// M3.5.0 = last Sunday in March (week=5 means last)
	r := &posixTZRule{
		isStart:          true,
		isMonthWeekDay:   true,
		month:            3,
		week:             5,
		weekday:          0,
		timeOfTransition: 7200,
	}
	_ = r.eval(2024, -18000, -14400)
}

func TestPosixTZ_Eval_NonLeapJulian(t *testing.T) {
	r := &posixTZRule{
		isStart:          false,
		isJulianNoLeap:   true,
		julianDay:        59,
		timeOfTransition: 7200,
	}
	_ = r.eval(2023, -18000, -14400)
}

func TestParsePOSIXTZ_StandardOnly(t *testing.T) {
	// Simple TZ with just name and offset: "EST5"
	tz, ok := parsePOSIXTZ("EST5")
	if ok {
		if tz.stdName != "EST" {
			t.Errorf("expected EST, got %q", tz.stdName)
		}
	}
}

func TestParsePOSIXTZ_Complex(t *testing.T) {
	// Full TZ: "NZST-12:00:00NZDT-13:00:00,M10.1.0,M3.3.0"
	tz, ok := parsePOSIXTZ("NZST-12:00:00NZDT-13:00:00,M10.1.0,M3.3.0")
	if ok {
		if !tz.hasDST {
			t.Error("expected DST for complex TZ")
		}
	}
}
