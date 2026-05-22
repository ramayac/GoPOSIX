package date

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "u", Long: "utc", Type: common.FlagBool},
		{Short: "d", Long: "date", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

type DateInfo struct {
	ISO      string `json:"iso"`
	Unix     int64  `json:"unix"`
	UTC      string `json:"utc"`
	Timezone string `json:"timezone"`
}

func splitFormatAndDate(rawArgs []string) (args []string, format string) {
	format = ""
	for i := len(rawArgs) - 1; i >= 0; i-- {
		if len(rawArgs[i]) > 0 && rawArgs[i][0] == '+' {
			format = rawArgs[i][1:]
			rawArgs = append(rawArgs[:i], rawArgs[i+1:]...)
			break
		}
	}
	return rawArgs, format
}

func parseDateString(s string, loc *time.Location) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	if s[0] == '@' {
		sec, err := strconv.ParseInt(s[1:], 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid timestamp: %s", s[1:])
		}
		return time.Unix(sec, 0).In(loc), nil
	}

	rest := s
	if strings.HasSuffix(s, "Z") {
		loc = time.UTC
		rest = s[:len(s)-1]
	} else if len(s) >= 5 {
		sign := s[len(s)-5]
		if (sign == '+' || sign == '-') && isDigit(s[len(s)-4]) {
			h, _ := strconv.Atoi(s[len(s)-4 : len(s)-2])
			m, _ := strconv.Atoi(s[len(s)-2:])
			offset := h*3600 + m*60
			if sign == '-' {
				offset = -offset
			}
			loc = time.FixedZone("", offset)
			rest = strings.TrimSpace(s[:len(s)-5])
		}
	}

	// Flexible ISO: "1999-1-2 3:4:5" or "1999-1-2 3:4"
	t, err := time.ParseInLocation("2006-1-2 15:4:5", rest, loc)
	if err == nil {
		return t, nil
	}
	t, err = time.ParseInLocation("2006-1-2 15:4", rest, loc)
	if err == nil {
		return t, nil
	}

	// Compact: YYYYMMDDHHMM (14 digits + optional .SS)
	if len(rest) >= 12 && isAllDigits(rest[:12]) {
		year, _ := strconv.Atoi(rest[0:4])
		month, _ := strconv.Atoi(rest[4:6])
		day, _ := strconv.Atoi(rest[6:8])
		hour, _ := strconv.Atoi(rest[8:10])
		min, _ := strconv.Atoi(rest[10:12])
		sec := 0
		if len(rest) >= 14 && rest[12] == '.' && len(rest) >= 15 && isAllDigits(rest[13:15]) {
			sec, _ = strconv.Atoi(rest[13:15])
		}
		return time.Date(year, time.Month(month), day, hour, min, sec, 0, loc), nil
	}

	// Dotted: YYYY.M.D-HH:MM[:SS] or M.D-HH:MM[:SS]
	t, err = parseDottedDate(rest, loc)
	if err == nil {
		return t, nil
	}

	// Time only: HH:MM:SS or HH:MM
	t, err = parseTimeOnly(rest, loc)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

func parseDottedDate(s string, loc *time.Location) (time.Time, error) {
	dash := strings.LastIndex(s, "-")
	if dash < 0 {
		return time.Time{}, fmt.Errorf("no dash")
	}
	datePart := s[:dash]
	timePart := s[dash+1:]

	dotParts := strings.Split(datePart, ".")
	var year, month, day int
	if len(dotParts) == 3 {
		year, _ = strconv.Atoi(dotParts[0])
		month, _ = strconv.Atoi(dotParts[1])
		day, _ = strconv.Atoi(dotParts[2])
	} else if len(dotParts) == 2 {
		month, _ = strconv.Atoi(dotParts[0])
		day, _ = strconv.Atoi(dotParts[1])
		year = time.Now().In(loc).Year()
	} else {
		return time.Time{}, fmt.Errorf("invalid dotted date")
	}

	timeParts := strings.Split(timePart, ":")
	var hour, min, sec int
	if len(timeParts) >= 2 {
		hour, _ = strconv.Atoi(timeParts[0])
		min, _ = strconv.Atoi(timeParts[1])
	}
	if len(timeParts) >= 3 {
		sec, _ = strconv.Atoi(timeParts[2])
	}

	return time.Date(year, time.Month(month), day, hour, min, sec, 0, loc), nil
}

func parseTimeOnly(s string, loc *time.Location) (time.Time, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 || len(parts) > 3 {
		return time.Time{}, fmt.Errorf("invalid time")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	sVal := 0
	if len(parts) == 3 {
		sVal, err = strconv.Atoi(parts[2])
		if err != nil {
			return time.Time{}, err
		}
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), h, m, sVal, 0, loc), nil
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	rawArgs, format := splitFormatAndDate(args)

	flags, err := common.ParseFlags(rawArgs, spec)
	if err != nil {
		fmt.Fprintln(os.Stderr, "BusyBox v1.36.1-goposix multi-call binary")
		fmt.Fprintf(os.Stderr, "date: %v\n", err)
		return 1
	}

	utcMode := flags.Has("u")
	jsonMode := flags.Has("json")
	dateStr := flags.Get("d")

	// POSIX: reject unexpected positional arguments
	for _, p := range flags.Positional {
		fmt.Fprintln(os.Stderr, "BusyBox v1.36.1-goposix multi-call binary")
		fmt.Fprintf(os.Stderr, "date: invalid date '%s'\n", p)
		return 1
	}

	var now time.Time
	var loc *time.Location

	var tz *posixTZ
	var hasPOSIXTZ bool
	if !utcMode {
		if tzStr := os.Getenv("TZ"); tzStr != "" {
			if parsed, ok := parsePOSIXTZ(tzStr); ok {
				tz = parsed
				hasPOSIXTZ = true
			}
		}
	}

	if hasPOSIXTZ {
		loc = time.FixedZone(tz.stdName, tz.stdOffset)
	} else {
		if utcMode {
			loc = time.UTC
		} else {
			loc = time.Local
		}
	}

	if dateStr != "" {
		t, err := parseDateString(dateStr, loc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "date: invalid date '%s'\n", dateStr)
			return 1
		}
		now = t
	} else {
		now = time.Now()
		if utcMode {
			now = now.UTC()
		}
	}

	if hasPOSIXTZ {
		zoneName, zoneOffset := tz.eval(now.UTC())
		now = now.In(time.FixedZone(zoneName, zoneOffset))
	}

	zone, _ := now.Zone()
	info := DateInfo{
		ISO:      now.Format(time.RFC3339),
		Unix:     now.Unix(),
		UTC:      now.UTC().Format(time.RFC3339),
		Timezone: zone,
	}

	common.Render("date", info, jsonMode, stdout, func() {
		if format != "" {
			outStr := formatDate(now, format)
			fmt.Fprintln(stdout, outStr)
		} else {
			fmt.Fprintln(stdout, now.Format(time.UnixDate))
		}
	})

	return 0
}

func formatDate(t time.Time, f string) string {
	var b strings.Builder
	i := 0
	for i < len(f) {
		if f[i] == '%' && i+1 < len(f) {
			i++
			switch f[i] {
			case '%':
				b.WriteByte('%')
			case 'a':
				b.WriteString(t.Format("Mon"))
			case 'A':
				b.WriteString(t.Format("Monday"))
			case 'b':
				b.WriteString(t.Format("Jan"))
			case 'B':
				b.WriteString(t.Format("January"))
			case 'c':
				// POSIX locale date/time: "Sun Jan 23 11:33:00 2000"
				// (no timezone, unlike UnixDate which includes TZ)
				b.WriteString(t.Format("Mon Jan _2 15:04:05 2006"))
			case 'd':
				b.WriteString(t.Format("02"))
			case 'e':
				b.WriteString(fmt.Sprintf("%2d", t.Day()))
			case 'H':
				b.WriteString(t.Format("15"))
			case 'I':
				b.WriteString(t.Format("03"))
			case 'm':
				b.WriteString(t.Format("01"))
			case 'M':
				b.WriteString(t.Format("04"))
			case 'S':
				b.WriteString(t.Format("05"))
			case 'T':
				b.WriteString(t.Format("15:04:05"))
			case 'y':
				b.WriteString(t.Format("06"))
			case 'Y':
				b.WriteString(t.Format("2006"))
			case 'Z':
				zone, _ := t.Zone()
				b.WriteString(zone)
			case 's':
				b.WriteString(strconv.FormatInt(t.Unix(), 10))
			case 'j':
				b.WriteString(fmt.Sprintf("%03d", t.YearDay()))
			case 'p':
				b.WriteString(t.Format("PM"))
			case 'r':
				b.WriteString(t.Format("03:04:05 PM"))
			case 'u':
				w := int(t.Weekday())
				if w == 0 {
					w = 7
				}
				b.WriteString(strconv.Itoa(w))
			case 'V':
				_, weekV := t.ISOWeek()
				b.WriteString(fmt.Sprintf("%02d", weekV))
			case 'W':
				yearStart := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
				startWD := int(yearStart.Weekday())
				daysBeforeFirstMonday := (8 - startWD) % 7
				daysSinceYearStart := t.YearDay() - 1
				var weekW int
				if daysSinceYearStart < daysBeforeFirstMonday {
					weekW = 0
				} else {
					weekW = 1 + (daysSinceYearStart-daysBeforeFirstMonday)/7
				}
				b.WriteString(fmt.Sprintf("%02d", weekW))
			case 'U':
				yearStart := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
				startWD := int(yearStart.Weekday())
				daysBeforeFirstSunday := (7 - startWD) % 7
				daysSinceYearStart := t.YearDay() - 1
				var weekU int
				if daysSinceYearStart < daysBeforeFirstSunday {
					weekU = 0
				} else {
					weekU = 1 + (daysSinceYearStart-daysBeforeFirstSunday)/7
				}
				b.WriteString(fmt.Sprintf("%02d", weekU))
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'D':
				b.WriteString(t.Format("01/02/06"))
			case 'F':
				b.WriteString(t.Format("2006-01-02"))
			case 'R':
				b.WriteString(t.Format("15:04"))
			case 'w':
				b.WriteString(strconv.Itoa(int(t.Weekday())))
			case 'k':
				b.WriteString(fmt.Sprintf("%2d", t.Hour()))
			case 'l':
				hl := t.Hour() % 12
				if hl == 0 {
					hl = 12
				}
				b.WriteString(fmt.Sprintf("%2d", hl))
			default:
				b.WriteByte('%')
				b.WriteByte(f[i])
			}
		} else {
			b.WriteByte(f[i])
		}
		i++
	}
	return b.String()
}

func init() {
	dispatch.Register(dispatch.Command{Name: "date", Usage: "Print or set the system date and time", Run: run})
}

type posixTZRule struct {
	isJulianNoLeap   bool
	isJulianLeap     bool
	isMonthWeekDay   bool
	julianDay        int
	month            int
	week             int
	weekday          int
	timeOfTransition int // seconds since midnight
	isStart          bool
}

type posixTZ struct {
	stdName   string
	stdOffset int
	dstName   string
	dstOffset int
	hasDST    bool
	start     posixTZRule
	end       posixTZRule
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func getDayOfMonth(year, m, w, d int) int {
	tFirst := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
	firstWD := int(tFirst.Weekday())
	day := 1 + (d-firstWD+7)%7
	if w >= 1 && w <= 4 {
		day += (w - 1) * 7
	} else if w == 5 {
		daysInMonth := time.Date(year, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
		if day+28 <= daysInMonth {
			day += 28
		} else {
			day += 21
		}
	}
	return day
}

func (r *posixTZRule) eval(year int, stdOffset, dstOffset int) time.Time {
	var day int
	var offset int
	if r.isStart {
		offset = stdOffset
	} else {
		offset = dstOffset
	}

	if r.isMonthWeekDay {
		day = getDayOfMonth(year, r.month, r.week, r.weekday)
		tLocal := time.Date(year, time.Month(r.month), day, 0, 0, r.timeOfTransition, 0, time.FixedZone("", offset))
		return tLocal.UTC()
	}

	tUTC := time.Date(year, 1, 1, 0, 0, r.timeOfTransition, 0, time.UTC)
	if r.isJulianNoLeap {
		if isLeapYear(year) && r.julianDay >= 60 {
			tUTC = tUTC.AddDate(0, 0, r.julianDay)
		} else {
			tUTC = tUTC.AddDate(0, 0, r.julianDay-1)
		}
	} else {
		tUTC = tUTC.AddDate(0, 0, r.julianDay)
	}

	tLocal := time.Date(year, tUTC.Month(), tUTC.Day(), tUTC.Hour(), tUTC.Minute(), tUTC.Second(), 0, time.FixedZone("", offset))
	return tLocal.UTC()
}

func (tz *posixTZ) eval(t time.Time) (string, int) {
	if !tz.hasDST {
		return tz.stdName, tz.stdOffset
	}
	year := t.Year()
	startUTC := tz.start.eval(year, tz.stdOffset, tz.dstOffset)
	endUTC := tz.end.eval(year, tz.stdOffset, tz.dstOffset)

	inDST := false
	if startUTC.Before(endUTC) {
		inDST = (t.Equal(startUTC) || t.After(startUTC)) && t.Before(endUTC)
	} else {
		inDST = t.Equal(startUTC) || t.After(startUTC) || t.Before(endUTC)
	}

	if inDST {
		return tz.dstName, tz.dstOffset
	}
	return tz.stdName, tz.stdOffset
}

func parsePOSIXTZ(tz string) (*posixTZ, bool) {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return nil, false
	}

	parseName := func(s string) (string, string, bool) {
		if len(s) == 0 {
			return "", "", false
		}
		if s[0] == '<' {
			idx := strings.Index(s, ">")
			if idx < 0 {
				return "", "", false
			}
			return s[1:idx], s[idx+1:], true
		}
		i := 0
		for i < len(s) {
			c := s[i]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				i++
			} else {
				break
			}
		}
		if i == 0 {
			return "", "", false
		}
		return s[:i], s[i:], true
	}

	parseOffset := func(s string) (int, string, bool) {
		if len(s) == 0 {
			return 0, "", false
		}
		sign := 1
		if s[0] == '+' {
			sign = 1
			s = s[1:]
		} else if s[0] == '-' {
			sign = -1
			s = s[1:]
		}

		i := 0
		for i < len(s) {
			c := s[i]
			if (c >= '0' && c <= '9') || c == ':' {
				i++
			} else {
				break
			}
		}
		if i == 0 {
			return 0, "", false
		}

		offsetStr := s[:i]
		rem := s[i:]

		parts := strings.Split(offsetStr, ":")
		if len(parts) < 1 || len(parts) > 3 {
			return 0, "", false
		}

		h, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, "", false
		}

		m := 0
		if len(parts) >= 2 {
			m, err = strconv.Atoi(parts[1])
			if err != nil {
				return 0, "", false
			}
		}

		sec := 0
		if len(parts) == 3 {
			sec, err = strconv.Atoi(parts[2])
			if err != nil {
				return 0, "", false
			}
		}

		totalSecs := sign * (h*3600 + m*60 + sec)
		return totalSecs, rem, true
	}

	stdName, rest, ok := parseName(tz)
	if !ok {
		return nil, false
	}

	posixStdOffset, rest, ok := parseOffset(rest)
	if !ok {
		return nil, false
	}
	stdOffset := -posixStdOffset

	res := &posixTZ{
		stdName:   stdName,
		stdOffset: stdOffset,
	}

	if len(rest) == 0 {
		return res, true
	}

	dstName, rest, ok := parseName(rest)
	if !ok {
		return res, true
	}
	res.dstName = dstName
	res.hasDST = true

	var dstOffset int
	if len(rest) > 0 && (rest[0] == '+' || rest[0] == '-' || (rest[0] >= '0' && rest[0] <= '9')) {
		posixDstOffset, r, ok := parseOffset(rest)
		if !ok {
			return nil, false
		}
		dstOffset = -posixDstOffset
		rest = r
	} else {
		dstOffset = stdOffset + 3600
	}
	res.dstOffset = dstOffset

	if len(rest) == 0 {
		res.start = posixTZRule{
			isMonthWeekDay:   true,
			month:            3,
			week:             2,
			weekday:          0,
			timeOfTransition: 2 * 3600,
			isStart:          true,
		}
		res.end = posixTZRule{
			isMonthWeekDay:   true,
			month:            11,
			week:             1,
			weekday:          0,
			timeOfTransition: 2 * 3600,
			isStart:          false,
		}
		return res, true
	}

	if rest[0] != ',' {
		return nil, false
	}
	rest = rest[1:]

	parseRule := func(s string, isStart bool) (posixTZRule, string, bool) {
		var rule posixTZRule
		rule.isStart = isStart
		if len(s) == 0 {
			return rule, "", false
		}

		if s[0] == 'M' {
			rule.isMonthWeekDay = true
			idx := 1
			for idx < len(s) && s[idx] != ',' && s[idx] != '/' {
				idx++
			}
			ruleStr := s[1:idx]
			rem := s[idx:]

			parts := strings.Split(ruleStr, ".")
			if len(parts) != 3 {
				return rule, "", false
			}
			m, err1 := strconv.Atoi(parts[0])
			w, err2 := strconv.Atoi(parts[1])
			d, err3 := strconv.Atoi(parts[2])
			if err1 != nil || err2 != nil || err3 != nil {
				return rule, "", false
			}
			rule.month = m
			rule.week = w
			rule.weekday = d
			s = rem
		} else if s[0] == 'J' {
			rule.isJulianNoLeap = true
			idx := 1
			for idx < len(s) && s[idx] != ',' && s[idx] != '/' {
				idx++
			}
			day, err := strconv.Atoi(s[1:idx])
			if err != nil {
				return rule, "", false
			}
			rule.julianDay = day
			s = s[idx:]
		} else if s[0] >= '0' && s[0] <= '9' {
			rule.isJulianLeap = true
			idx := 0
			for idx < len(s) && s[idx] != ',' && s[idx] != '/' {
				idx++
			}
			day, err := strconv.Atoi(s[:idx])
			if err != nil {
				return rule, "", false
			}
			rule.julianDay = day
			s = s[idx:]
		} else {
			return rule, "", false
		}

		rule.timeOfTransition = 2 * 3600
		if len(s) > 0 && s[0] == '/' {
			s = s[1:]
			idx := 0
			for idx < len(s) && s[idx] != ',' {
				idx++
			}
			timeStr := s[:idx]
			s = s[idx:]

			parts := strings.Split(timeStr, ":")
			if len(parts) >= 1 && len(parts) <= 3 {
				h, err := strconv.Atoi(parts[0])
				if err == nil {
					m := 0
					if len(parts) >= 2 {
						m, _ = strconv.Atoi(parts[1])
					}
					sec := 0
					if len(parts) == 3 {
						sec, _ = strconv.Atoi(parts[2])
					}
					rule.timeOfTransition = h*3600 + m*60 + sec
				}
			}
		}

		return rule, s, true
	}

	startRule, rest, ok := parseRule(rest, true)
	if !ok {
		return nil, false
	}
	res.start = startRule

	if len(rest) == 0 || rest[0] != ',' {
		return nil, false
	}
	rest = rest[1:]

	endRule, rest, ok := parseRule(rest, false)
	if !ok {
		return nil, false
	}
	res.end = endRule

	return res, true
}
