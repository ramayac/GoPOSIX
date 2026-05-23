// Package makedevs implements the POSIX-compliant makedevs utility.
package makedevs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"golang.org/x/sys/unix"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "d", Long: "table", Type: common.FlagValue},
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "makedevs",
		Usage: "Create device nodes from a device table",
		Run:   run,
	})
}

// Injectable mknod function for testing in non-privileged environments.
var mknodFn = func(path string, mode uint32, dev int) error {
	return unix.Mknod(path, mode, dev)
}

// Injectable chown function for testing.
var chownFn = func(path string, uid, gid int) error {
	return os.Chown(path, uid, gid)
}

// DeviceEntry represents a parsed device entry for JSON output.
type DeviceEntry struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Mode   string `json:"mode"`
	Uid    int    `json:"uid"`
	Gid    int    `json:"gid"`
	Major  int    `json:"major,omitempty"`
	Minor  int    `json:"minor,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// MakedevsResult represents the JSON response structure.
type MakedevsResult struct {
	Table       string        `json:"table"`
	Rootdir     string        `json:"rootdir"`
	Created     []DeviceEntry `json:"created"`
	FailedCount int           `json:"failedCount"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			break
		}
	}

	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		if jsonMode {
			common.RenderError("makedevs", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "makedevs: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: makedevs [-d TABLE] ROOTDIR\n\n" +
			"Create device nodes from a device table.\n\n" +
			"Options:\n" +
			"  -d, --table    Device table file (default: stdin)\n" +
			"  -h, --help     Print help"
		common.Render("makedevs", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	pos := flags.Positional
	if len(pos) == 0 {
		if jsonMode {
			common.RenderError("makedevs", 1, "MISSING_ARGUMENT", "missing ROOTDIR", true, stderr)
		} else {
			fmt.Fprintln(stderr, "makedevs: missing ROOTDIR")
		}
		return 1
	}

	rootDir := pos[0]
	absRootDir := rootDir
	if !filepath.IsAbs(absRootDir) {
		absRootDir = filepath.Join(cwd, rootDir)
	}

	tableFile := flags.Get("d")
	var tableReader io.Reader

	if tableFile == "" || tableFile == "-" {
		tableReader = stdin
	} else {
		absTablePath := tableFile
		if !filepath.IsAbs(absTablePath) {
			absTablePath = filepath.Join(cwd, tableFile)
		}
		file, err := os.Open(absTablePath)
		if err != nil {
			if jsonMode {
				common.RenderError("makedevs", 1, "OPEN_ERROR", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "makedevs: %v\n", err)
			}
			return 1
		}
		defer file.Close()
		tableReader = file
	}

	var created []DeviceEntry
	failedCount := 0

	// Print headers expected by BusyBox test suite in stdout when rootdir/table are parsed
	if !jsonMode {
		fmt.Fprintf(stdout, "rootdir=%s\n", rootDir)
		fmt.Fprintf(stdout, "table='%s'\n", tableFile)
	}

	scanner := bufio.NewScanner(tableReader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue // invalid line
		}

		name := fields[0]
		devType := fields[1]
		modeStr := fields[2]
		uidStr := fields[3]
		gidStr := fields[4]

		modeVal, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			continue
		}
		uid, _ := strconv.Atoi(uidStr)
		gid, _ := strconv.Atoi(gidStr)

		major := 0
		minor := 0
		start := 0
		inc := 1
		count := -1

		if len(fields) > 5 && fields[5] != "-" {
			major, _ = strconv.Atoi(fields[5])
		}
		if len(fields) > 6 && fields[6] != "-" {
			minor, _ = strconv.Atoi(fields[6])
		}
		if len(fields) > 7 && fields[7] != "-" {
			start, _ = strconv.Atoi(fields[7])
		}
		if len(fields) > 8 && fields[8] != "-" {
			inc, _ = strconv.Atoi(fields[8])
		}
		if len(fields) > 9 && fields[9] != "-" {
			count, _ = strconv.Atoi(fields[9])
		}

		err = createNode(absRootDir, name, devType, uint32(modeVal), uid, gid, major, minor, start, inc, count, &created)
		if err != nil {
			failedCount++
		}
	}

	if jsonMode {
		common.Render("makedevs", MakedevsResult{
			Table:       tableFile,
			Rootdir:     rootDir,
			Created:     created,
			FailedCount: failedCount,
		}, true, stdout, nil)
	}

	if failedCount > 0 {
		return 1
	}
	return 0
}

func createNode(rootDir, name, devType string, mode uint32, uid, gid, major, minor, start, inc, count int, created *[]DeviceEntry) error {
	relPath := strings.TrimPrefix(name, "/")

	if count < 0 {
		// Single node creation
		destPath := filepath.Clean(filepath.Join(rootDir, relPath))
		err := executeCreate(destPath, devType, mode, uid, gid, major, minor)
		errStr := ""
		status := "success"
		if err != nil {
			status = "failed"
			errStr = err.Error()
		}

		*created = append(*created, DeviceEntry{
			Name:   name,
			Type:   devType,
			Mode:   fmt.Sprintf("%o", mode),
			Uid:    uid,
			Gid:    gid,
			Major:  major,
			Minor:  minor,
			Status: status,
			Error:  errStr,
		})
		return err
	}

	// Multiple node creation
	var lastErr error
	for i := 0; i < count; i++ {
		suffixNum := start + i*inc
		nodeName := fmt.Sprintf("%s%d", name, suffixNum)
		nodeRelPath := fmt.Sprintf("%s%d", relPath, suffixNum)
		destPath := filepath.Clean(filepath.Join(rootDir, nodeRelPath))

		nodeMinor := minor + i*inc
		err := executeCreate(destPath, devType, mode, uid, gid, major, nodeMinor)
		errStr := ""
		status := "success"
		if err != nil {
			status = "failed"
			errStr = err.Error()
			lastErr = err
		}

		*created = append(*created, DeviceEntry{
			Name:   nodeName,
			Type:   devType,
			Mode:   fmt.Sprintf("%o", mode),
			Uid:    uid,
			Gid:    gid,
			Major:  major,
			Minor:  nodeMinor,
			Status: status,
			Error:  errStr,
		})
	}

	return lastErr
}

func executeCreate(path string, devType string, mode uint32, uid, gid, major, minor int) error {
	switch devType {
	case "d":
		if err := os.MkdirAll(path, os.FileMode(mode)); err != nil {
			return err
		}
		return chownFn(path, uid, gid)

	case "f":
		if _, err := os.Stat(path); err == nil {
			if err := os.Chmod(path, os.FileMode(mode)); err != nil {
				return err
			}
			return chownFn(path, uid, gid)
		}
		// Create empty file if nonexistent
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
		if err != nil {
			return err
		}
		file.Close()
		return chownFn(path, uid, gid)

	case "p":
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		_ = os.Remove(path)
		if err := syscall.Mkfifo(path, mode); err != nil {
			return err
		}
		return chownFn(path, uid, gid)

	case "c", "b":
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		_ = os.Remove(path)

		var mknodMode uint32
		if devType == "c" {
			mknodMode = mode | syscall.S_IFCHR
		} else {
			mknodMode = mode | syscall.S_IFBLK
		}

		dev := unix.Mkdev(uint32(major), uint32(minor))
		if err := mknodFn(path, mknodMode, int(dev)); err != nil {
			return err
		}
		return chownFn(path, uid, gid)

	default:
		return fmt.Errorf("unknown device type: %s", devType)
	}
}
