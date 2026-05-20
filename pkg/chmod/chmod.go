package chmod

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "R", Long: "recursive", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

type ChmodResult struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type ChmodResp struct {
	Changed []ChmodResult `json:"changed"`
}

// applySymbolicMode applies a symbolic mode string to a file's current mode.
// Supports POSIX symbolic modes: [ugoa]*[-+=][rwxst]*  (e.g., "a-r", "u+x", "go-w")
func applySymbolicMode(modeStr string, current os.FileMode) (os.FileMode, error) {
	modeStr = strings.TrimSpace(modeStr)
	if len(modeStr) < 3 {
		return 0, fmt.Errorf("invalid mode: %s", modeStr)
	}

	opIdx := strings.IndexAny(modeStr, "-+=")
	if opIdx < 0 || opIdx == 0 {
		return 0, fmt.Errorf("invalid mode: %s", modeStr)
	}

	who := modeStr[:opIdx]
	op := modeStr[opIdx]
	perms := modeStr[opIdx+1:]

	whoMask := os.FileMode(0)
	if who == "" || strings.Contains(who, "a") {
		whoMask = os.FileMode(0777)
	} else {
		for _, c := range who {
			switch c {
			case 'u':
				whoMask |= os.FileMode(0700)
			case 'g':
				whoMask |= os.FileMode(0070)
			case 'o':
				whoMask |= os.FileMode(0007)
			}
		}
	}
	if whoMask == 0 {
		whoMask = os.FileMode(0777)
	}

	permBits := os.FileMode(0)
	for _, c := range perms {
		switch c {
		case 'r':
			permBits |= os.FileMode(0444)
		case 'w':
			permBits |= os.FileMode(0222)
		case 'x':
			permBits |= os.FileMode(0111)
		case 's':
			permBits |= os.FileMode(os.ModeSetuid | os.ModeSetgid)
		case 't':
			permBits |= os.FileMode(os.ModeSticky)
		}
	}

	maskedPerms := permBits & whoMask

	switch op {
	case '+':
		return current | maskedPerms, nil
	case '-':
		return current &^ maskedPerms, nil
	case '=':
		return (current &^ whoMask) | maskedPerms, nil
	}
	return current, nil
}

func isSymbolicMode(modeStr string) bool {
	if _, err := strconv.ParseUint(modeStr, 8, 32); err == nil {
		return false
	}
	return strings.ContainsAny(modeStr, "-+=")
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chmod: %v\n", err)
		return 1
	}

	if len(flags.Positional) < 2 {
		fmt.Fprintln(os.Stderr, "chmod: missing operand")
		return 1
	}

	modeStr := flags.Positional[0]
	paths := flags.Positional[1:]

	if isSymbolicMode(modeStr) {
		exitCode := 0
		var res []ChmodResult
		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "chmod: %v\n", err)
				exitCode = 1
				continue
			}
			newMode, err := applySymbolicMode(modeStr, info.Mode())
			if err != nil {
				fmt.Fprintf(os.Stderr, "chmod: %v\n", err)
				exitCode = 1
				continue
			}
			if err := os.Chmod(path, newMode); err != nil {
				fmt.Fprintf(os.Stderr, "chmod: %v\n", err)
				exitCode = 1
			} else {
				res = append(res, ChmodResult{Path: path, Mode: fmt.Sprintf("%04o", newMode.Perm())})
			}
		}
		if flags.Has("json") {
			common.Render("chmod", ChmodResp{Changed: res}, true, stdout, func() {})
		}
		return exitCode
	}

	// Numeric octal mode.
	modeNum, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chmod: invalid mode: %s\n", modeStr)
		return 1
	}
	mode := os.FileMode(modeNum)

	exitCode := 0
	var res []ChmodResult
	for _, path := range paths {
		if err := os.Chmod(path, mode); err != nil {
			fmt.Fprintf(os.Stderr, "chmod: %v\n", err)
			exitCode = 1
		} else {
			res = append(res, ChmodResult{Path: path, Mode: fmt.Sprintf("%04o", mode)})
		}
	}
	if flags.Has("json") {
		common.Render("chmod", ChmodResp{Changed: res}, true, stdout, func() {})
	}
	return exitCode
}

func init() {
	dispatch.Register(dispatch.Command{Name: "chmod", Usage: "Change file mode bits", Run: run})
}
