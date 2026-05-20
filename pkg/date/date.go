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
			rest = s[:len(s)-5]
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

func run(args []string, out io.Writer) int {
	rawArgs, format := splitFormatAndDate(args)

	flags, err := common.ParseFlags(rawArgs, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "date: %v\n", err)
		return 1
	}

	utcMode := flags.Has("u")
	jsonMode := flags.Has("json")
	dateStr := flags.Get("d")

	// POSIX: reject unexpected positional arguments
	for _, p := range flags.Positional {
		fmt.Fprintf(os.Stderr, "date: invalid date '%s'\n", p)
		return 1
	}

	var now time.Time
	var loc *time.Location

	if utcMode {
		loc = time.UTC
	} else {
		loc = time.Local
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

	zone, _ := now.Zone()
	info := DateInfo{
		ISO:      now.Format(time.RFC3339),
		Unix:     now.Unix(),
		UTC:      now.UTC().Format(time.RFC3339),
		Timezone: zone,
	}

	common.Render("date", info, jsonMode, out, func() {
		if format != "" {
			outStr := formatDate(now, format)
			fmt.Fprintln(out, outStr)
		} else {
			fmt.Fprintln(out, now.Format(time.UnixDate))
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
