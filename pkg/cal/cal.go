// Package cal implements the POSIX-compliant cal utility.
package cal

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// CalResult contains the rendered calendar output and query details.
type CalResult struct {
	Month       int    `json:"month,omitempty"`
	Year        int    `json:"year"`
	Julian      bool   `json:"julian"`
	MondayStart bool   `json:"monday_start"`
	Calendar    string `json:"calendar"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "j", Long: "julian", Type: common.FlagBool},
		{Short: "m", Long: "monday", Type: common.FlagBool},
		{Short: "y", Long: "year", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// parseMonth parses a month from a string (either numeric 1-12 or month name prefixes).
func parseMonth(mStr string) (int, error) {
	var m int
	_, err := fmt.Sscan(mStr, &m)
	if err == nil {
		if m >= 1 && m <= 12 {
			return m, nil
		}
		return 0, fmt.Errorf("invalid month %s (1-12)", mStr)
	}

	mStrLower := strings.ToLower(mStr)
	months := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	var found []int
	for idx, name := range months {
		if strings.HasPrefix(name, mStrLower) {
			found = append(found, idx+1)
		}
	}

	if len(found) == 1 {
		return found[0], nil
	}
	if len(found) > 1 {
		return 0, fmt.Errorf("ambiguous month: %s", mStr)
	}
	return 0, fmt.Errorf("invalid month: %s", mStr)
}

// Run renders the calendar based on year, month, and formatting flags.
// If month is 0, the calendar for the entire year is rendered.
func Run(year int, month int, julian bool, mondayStart bool) CalResult {
	var calendar string
	if month == 0 {
		calendar = RenderYear(year, julian, mondayStart)
	} else {
		calendar = RenderMonthString(year, month, julian, mondayStart)
	}
	return CalResult{
		Month:       month,
		Year:        year,
		Julian:      julian,
		MondayStart: mondayStart,
		Calendar:    calendar,
	}
}

// RenderMonthString renders a single month calendar as a trimmed string.
func RenderMonthString(year int, month int, julian bool, mondayStart bool) string {
	lines := RenderMonth(year, month, julian, mondayStart, true)
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(strings.TrimRight(l, " "))
		sb.WriteString("\n")
	}
	return sb.String()
}

// RenderMonth returns a slice of exactly 8 strings (header1, header2, and 6 weeks)
// representing the month calendar. Each line is padded to the full month width (20 or 27).
func RenderMonth(year int, month int, julian bool, mondayStart bool, includeYear bool) []string {
	var W int
	if julian {
		W = 27
	} else {
		W = 20
	}

	monthNames := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	mName := monthNames[month-1]

	var title string
	if includeYear {
		title = fmt.Sprintf("%s %d", mName, year)
	} else {
		title = mName
	}

	// Center title
	leftPad := (W - len(title)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	header1 := strings.Repeat(" ", leftPad) + title
	if len(header1) < W {
		header1 = header1 + strings.Repeat(" ", W-len(header1))
	} else if len(header1) > W {
		header1 = header1[:W]
	}

	// Weekday headers
	var header2 string
	if julian {
		if mondayStart {
			header2 = " Mo  Tu  We  Th  Fr  Sa  Su"
		} else {
			header2 = " Su  Mo  Tu  We  Th  Fr  Sa"
		}
	} else {
		if mondayStart {
			header2 = "Mo Tu We Th Fr Sa Su"
		} else {
			header2 = "Su Mo Tu We Th Fr Sa"
		}
	}
	if len(header2) < W {
		header2 = header2 + strings.Repeat(" ", W-len(header2))
	} else if len(header2) > W {
		header2 = header2[:W]
	}

	// Grid calculations
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	weekday := firstDay.Weekday()

	var startSlot int
	if mondayStart {
		startSlot = int(weekday) - 1
		if startSlot < 0 {
			startSlot = 6
		}
	} else {
		startSlot = int(weekday)
	}

	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()

	var grid [6][7]string
	for d := 1; d <= daysInMonth; d++ {
		s := startSlot + d - 1
		r := s / 7
		c := s % 7
		if r < 6 {
			if julian {
				julianDay := time.Date(year, time.Month(month), d, 0, 0, 0, 0, time.UTC).YearDay()
				grid[r][c] = fmt.Sprintf("%3d", julianDay)
			} else {
				grid[r][c] = fmt.Sprintf("%2d", d)
			}
		}
	}

	var dateLines []string
	for r := 0; r < 6; r++ {
		var weekLine strings.Builder
		for c := 0; c < 7; c++ {
			cell := grid[r][c]
			if cell == "" {
				if julian {
					weekLine.WriteString("   ")
				} else {
					weekLine.WriteString("  ")
				}
			} else {
				weekLine.WriteString(cell)
			}
			if c < 6 {
				weekLine.WriteString(" ")
			}
		}
		lineStr := weekLine.String()
		if len(lineStr) < W {
			lineStr = lineStr + strings.Repeat(" ", W-len(lineStr))
		} else if len(lineStr) > W {
			lineStr = lineStr[:W]
		}
		dateLines = append(dateLines, lineStr)
	}

	res := []string{header1, header2}
	res = append(res, dateLines...)
	return res
}

// RenderYear renders the 12-month grid for a full year calendar.
func RenderYear(year int, julian bool, mondayStart bool) string {
	var totalW int
	var cols int
	var sep string
	var rows [][]int

	if julian {
		cols = 2
		sep = "   "
		totalW = 27*2 + 3
		rows = [][]int{
			{1, 2},
			{3, 4},
			{5, 6},
			{7, 8},
			{9, 10},
			{11, 12},
		}
	} else {
		cols = 3
		sep = "  "
		totalW = 20*3 + 2*2
		rows = [][]int{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9},
			{10, 11, 12},
		}
	}

	yearStr := fmt.Sprintf("%d", year)
	leftPad := (totalW - len(yearStr)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	yearHeader := strings.Repeat(" ", leftPad) + yearStr

	var sb strings.Builder
	sb.WriteString(strings.TrimRight(yearHeader, " "))
	sb.WriteString("\n\n")

	for rIdx, rMonths := range rows {
		var renderedMonths [][]string
		for _, m := range rMonths {
			renderedMonths = append(renderedMonths, RenderMonth(year, m, julian, mondayStart, false))
		}

		for l := 0; l < 8; l++ {
			var lineSb strings.Builder
			for c := 0; c < cols; c++ {
				lineSb.WriteString(renderedMonths[c][l])
				if c < cols-1 {
					lineSb.WriteString(sep)
				}
			}
			sb.WriteString(strings.TrimRight(lineSb.String(), " "))
			sb.WriteString("\n")
		}

		if rIdx < len(rows)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "cal: %v\n", err)
		return 2
	}

	julian := flags.Has("julian")
	mondayStart := flags.Has("monday")
	jsonMode := flags.Has("json")
	forceYear := flags.Has("year")

	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if forceYear {
		month = 0
	}

	posArgs := flags.Positional
	if len(posArgs) == 1 {
		yearVal, err := strconv.Atoi(posArgs[0])
		if err != nil || yearVal < 1 || yearVal > 9999 {
			fmt.Fprintf(stderr, "cal: invalid year %s (1-9999)\n", posArgs[0])
			return 1
		}
		year = yearVal
		month = 0 // Full year view
	} else if len(posArgs) == 2 {
		mVal, err := parseMonth(posArgs[0])
		if err != nil {
			fmt.Fprintf(stderr, "cal: %v\n", err)
			return 1
		}
		month = mVal

		yearVal, err := strconv.Atoi(posArgs[1])
		if err != nil || yearVal < 1 || yearVal > 9999 {
			fmt.Fprintf(stderr, "cal: invalid year %s (1-9999)\n", posArgs[1])
			return 1
		}
		year = yearVal

		if forceYear {
			month = 0 // Still full year view if forced by -y
		}
	} else if len(posArgs) > 2 {
		fmt.Fprintf(stderr, "cal: too many arguments\n")
		return 1
	}

	result := Run(year, month, julian, mondayStart)

	common.Render("cal", result, jsonMode, stdout, func() {
		fmt.Fprint(stdout, result.Calendar)
	})

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "cal",
		Usage: "Display a calendar",
		Run:   run,
	})
}
