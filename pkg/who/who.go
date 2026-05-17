// Package who implements the POSIX who utility — display who is logged on.
package who

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// WhoResult is the --json output.
type WhoResult struct {
	Users []WhoUser `json:"users"`
	Count int       `json:"count,omitempty"`
}

// WhoUser represents a logged-in user.
type WhoUser struct {
	Name     string `json:"name"`
	Terminal string `json:"terminal"`
	Time     string `json:"time,omitempty"`
	Host     string `json:"host,omitempty"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "q", Long: "quick", Type: common.FlagBool},
		{Short: "s", Long: "", Type: common.FlagBool},
		{Short: "H", Long: "heading", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// Utmp entry format (Linux x86_64):
// Type (int32), PID (int32), Line (32 bytes), ID (4 bytes),
// User (32 bytes), Host (256 bytes), Exit, Session, Time, AddrV6, Reserved
// See: man 5 utmp

const (
	utmpSize    = 384
	utmpLineLen = 32
	utmpUserLen = 32
	utmpHostLen = 256
	utmpTimeOff = 4 + 4 + utmpLineLen + 4 + utmpUserLen + utmpHostLen
)

// fixedString extracts a null-terminated string from a byte slice.
func fixedString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// parseUtmpEntry parses a single utmp entry.
func parseUtmpEntry(data []byte) (WhoUser, error) {
	if len(data) < utmpSize {
		return WhoUser{}, fmt.Errorf("utmp entry too short: %d bytes", len(data))
	}

	entryType := int32(binary.LittleEndian.Uint32(data[0:4]))

	// EMPTY (0), RUN_LVL (1), BOOT_TIME (2), NEW_TIME (3), OLD_TIME (4)
	// INIT_PROCESS (5), LOGIN_PROCESS (6), USER_PROCESS (7), DEAD_PROCESS (8)
	if entryType != 7 { // USER_PROCESS
		return WhoUser{}, nil
	}

	user := fixedString(data[4+4+utmpLineLen+4 : 4+4+utmpLineLen+4+utmpUserLen])
	if user == "" {
		return WhoUser{}, nil
	}

	line := fixedString(data[4+4 : 4+4+utmpLineLen])

	// Parse timestamp (seconds + microseconds at offset utmpTimeOff)
	sec := int64(binary.LittleEndian.Uint32(data[utmpTimeOff : utmpTimeOff+4]))
	t := time.Unix(sec, 0)

	host := fixedString(data[4+4+utmpLineLen+4+utmpUserLen : 4+4+utmpLineLen+4+utmpUserLen+utmpHostLen])

	return WhoUser{
		Name:     user,
		Terminal: line,
		Time:     t.Format("2006-01-02 15:04"),
		Host:     host,
	}, nil
}

// readUtmp reads the utmp file and returns logged-in users.
func readUtmp(path string) ([]WhoUser, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var users []WhoUser
	for offset := 0; offset+utmpSize <= len(data); offset += utmpSize {
		u, err := parseUtmpEntry(data[offset : offset+utmpSize])
		if err != nil {
			continue
		}
		if u.Name != "" {
			users = append(users, u)
		}
	}
	return users, nil
}

// Run reads utmp and returns logged-in users.
func Run() (WhoResult, error) {
	var utmpPaths = []string{
		"/var/run/utmp",
		"/run/utmp",
	}

	for _, p := range utmpPaths {
		users, err := readUtmp(p)
		if err == nil {
			return WhoResult{Users: users, Count: len(users)}, nil
		}
	}

	// If no utmp file found, return empty result
	return WhoResult{Users: nil, Count: 0}, nil
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "who: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	quick := flags.Has("q")
	heading := flags.Has("H")

	result, err := Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "who: %v\n", err)
		common.RenderError("who", 1, "EWHO", err.Error(), jsonMode, out)
		return 1
	}

	if quick {
		// Quick mode: names only + count
		if jsonMode {
			common.Render("who", result, true, out, func() {})
		} else {
			names := make([]string, len(result.Users))
			for i, u := range result.Users {
				names[i] = u.Name
			}
			sort.Strings(names)
			fmt.Fprintln(out, strings.Join(names, " "))
			fmt.Fprintf(out, "# users=%d\n", len(names))
		}
		return 0
	}

	common.Render("who", result, jsonMode, out, func() {
		if heading {
			fmt.Fprintf(out, "%-8s %-12s %-16s %s\n", "NAME", "LINE", "TIME", "COMMENT")
		}
		for _, u := range result.Users {
			comment := u.Host
			if comment == "" {
				comment = ""
			}
			fmt.Fprintf(out, "%-8s %-12s %-16s %s\n", u.Name, u.Terminal, u.Time, comment)
		}
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "who",
		Usage: "Display who is logged on",
		Run:   run,
	})
}
