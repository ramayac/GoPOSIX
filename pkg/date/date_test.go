package date

import (
	"bytes"
	"strings"
	"testing"
)

func TestDateRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-u"}, &out)
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if out.String() == "" {
		t.Error("expected output")
	}
}

func TestDateJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json"}, &out)
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
	rc := run([]string{"-d", "@1288486801"}, &out)
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
	rc := run([]string{"-d", "1999.01.01-11:22:33", "+%d/%m/%y"}, &out)
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
	rc := run([]string{"-d", "1999-1-2 3:4:5"}, &out)
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
	rc := run([]string{"-d", "200001231133"}, &out)
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
	rc := run([]string{"-d", "200001231133.30"}, &out)
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
	rc := run([]string{"-d", "1:2"}, &out)
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
	rc := run([]string{"-d", "1:2:3"}, &out)
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
	rc := run([]string{"-d", "1.2-3:4"}, &out)
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
	rc := run([]string{"-d", "1.2-3:4:5"}, &out)
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
	rc := run([]string{"-u", "-d", "2000.01.01-11:22:33"}, &out)
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
	rc := run([]string{"-d", "1:2:3", "+%T"}, &out)
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
	rc := run([]string{"-d", "200001231133", "+%c"}, &out)
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
	rc := run([]string{"-d", "1999.01.01-11:22:33", "+%d"}, &out)
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if strings.TrimSpace(out.String()) != "01" {
		t.Errorf("got %q, want '01'", strings.TrimSpace(out.String()))
	}
}

func TestBusyBox_Date_FormatPercentY(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-d", "200001231133", "+%Y"}, &out)
	if rc != 0 {
		t.Fatalf("exit code %d, want 0", rc)
	}
	if strings.TrimSpace(out.String()) != "2000" {
		t.Errorf("got %q, want '2000'", strings.TrimSpace(out.String()))
	}
}

func TestBusyBox_Date_FormatPercentYPercentm(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-d", "1999.01.01-11:22:33", "+%Y%m"}, &out)
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
	code := run([]string{"-d", "012311332000.30", "%+c"}, &buf)
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
		rc := run([]string{"-d", "2026.05.19-21:47:58", tc.format}, &out)
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
