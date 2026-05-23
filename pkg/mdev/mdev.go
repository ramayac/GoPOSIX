// Package mdev implements a minimal mdev(8)-like device manager for GoPOSIX.
//
// mdev is BusyBox's minimal device manager. In containers, it is primarily
// used to populate /dev by scanning /sys/class and creating device nodes.
//
// Supported modes:
//
//	mdev -s              — scan /sys/class and populate /dev
//	mdev -d              — print devices (dry-run / discovery mode)
//	mdev                 — act as a uevent hotplug helper (reads env vars)
//
// The implementation uses linux syscalls to create device nodes via mknod.
// The actual mknod call is in mdev_linux.go for portability.
package mdev

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// DevNode describes a discovered or created device node.
type DevNode struct {
	Name  string `json:"name"`
	Type  string `json:"type"` // "char" or "block"
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Path  string `json:"path"`
}

// MdevResult is the JSON envelope.
type MdevResult struct {
	Devices []DevNode `json:"devices"`
}

var flagSpec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Long: "scan", Type: common.FlagBool},
		{Short: "d", Long: "dry-run", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "mdev",
		Usage: "Minimal device manager / hotplug helper",
		Run:   run,
	})
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return mdevRun(args, stdin, stdout, stderr, cwd)
}

// mdevRun is the injectable entry point for testing.
func mdevRun(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, flagSpec)
	if err != nil {
		fmt.Fprintf(stderr, "mdev: %v\n", err)
		return 1
	}

	scanMode := flags.Has("s")
	dryRun := flags.Has("d")
	jsonMode := flags.Has("json")

	// If neither -s nor -d, act as hotplug helper (reads from env)
	if !scanMode && !dryRun {
		return mdevHotplug(jsonMode, stdout, stderr)
	}

	devices, err := discoverDevices()
	if err != nil {
		fmt.Fprintf(stderr, "mdev: %v\n", err)
		return 1
	}

	if dryRun || jsonMode {
		if jsonMode {
			common.Render("mdev", MdevResult{Devices: devices}, true, stdout, nil)
		} else {
			for _, d := range devices {
				fmt.Fprintf(stdout, "%s %d:%d %s\n", d.Type, d.Major, d.Minor, d.Name)
			}
		}
		return 0
	}

	// Scan mode: create device nodes
	return mdevScan(devices, stdout, stderr)
}

// discoverDevices reads /sys/class to discover available device nodes.
func discoverDevices() ([]DevNode, error) {
	var devices []DevNode

	classDir := "/sys/class"
	classes, err := os.ReadDir(classDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", classDir, err)
	}

	for _, cls := range classes {
		clsPath := filepath.Join(classDir, cls.Name())
		devEntries, err := os.ReadDir(clsPath)
		if err != nil {
			continue
		}
		for _, dev := range devEntries {
			devPath := filepath.Join(clsPath, dev.Name())
			node, err := readDevNode(dev.Name(), devPath)
			if err != nil {
				continue
			}
			devices = append(devices, node)
		}
	}
	return devices, nil
}

// readDevNode reads the dev file for a sysfs device to get major:minor.
func readDevNode(name, sysfsPath string) (DevNode, error) {
	devFile := filepath.Join(sysfsPath, "dev")
	data, err := os.ReadFile(devFile)
	if err != nil {
		return DevNode{}, err
	}

	parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 2)
	if len(parts) != 2 {
		return DevNode{}, fmt.Errorf("invalid dev file: %s", string(data))
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return DevNode{}, err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return DevNode{}, err
	}

	// Determine type from uevent
	devType := "char"
	ueventFile := filepath.Join(sysfsPath, "uevent")
	if ueventData, err := os.ReadFile(ueventFile); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(ueventData)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "DEVTYPE=") {
				if strings.Contains(line, "disk") || strings.Contains(line, "partition") {
					devType = "block"
				}
			}
			if strings.HasPrefix(line, "MAJOR=") {
				// cross-check
			}
		}
	}

	// Also check if this is a block device via subsystem symlink
	subsystemLink := filepath.Join(sysfsPath, "subsystem")
	if target, err := os.Readlink(subsystemLink); err == nil {
		if strings.Contains(target, "block") {
			devType = "block"
		}
	}

	return DevNode{
		Name:  name,
		Type:  devType,
		Major: major,
		Minor: minor,
		Path:  filepath.Join("/dev", name),
	}, nil
}

// mdevScan creates device nodes in /dev for all discovered devices.
func mdevScan(devices []DevNode, stdout, stderr io.Writer) int {
	exitCode := 0
	for _, d := range devices {
		if err := createDevNode(d); err != nil {
			// Many device nodes already exist or require specific permissions;
			// report but continue
			fmt.Fprintf(stderr, "mdev: %s: %v\n", d.Name, err)
			exitCode = 1
		}
	}
	return exitCode
}

// mdevHotplug acts as a kernel hotplug helper.
// It reads DEVPATH, ACTION, SUBSYSTEM, MAJOR, MINOR from environment variables.
func mdevHotplug(jsonMode bool, stdout, stderr io.Writer) int {
	action := os.Getenv("ACTION")
	devPath := os.Getenv("DEVPATH")
	subsystem := os.Getenv("SUBSYSTEM")
	majorStr := os.Getenv("MAJOR")
	minorStr := os.Getenv("MINOR")
	devName := os.Getenv("DEVNAME")

	if devPath == "" && action == "" {
		// No env vars: print usage hint
		fmt.Fprintln(stderr, "mdev: use -s to scan /sys/class or -d for dry-run discovery")
		return 1
	}

	if devName == "" {
		devName = filepath.Base(devPath)
	}

	major, _ := strconv.Atoi(majorStr)
	minor, _ := strconv.Atoi(minorStr)

	devType := "char"
	if subsystem == "block" {
		devType = "block"
	}

	node := DevNode{
		Name:  devName,
		Type:  devType,
		Major: major,
		Minor: minor,
		Path:  filepath.Join("/dev", devName),
	}

	switch action {
	case "add", "":
		if err := createDevNode(node); err != nil {
			fmt.Fprintf(stderr, "mdev: add %s: %v\n", devName, err)
			return 1
		}
		if jsonMode {
			common.Render("mdev", MdevResult{Devices: []DevNode{node}}, true, stdout, nil)
		} else {
			fmt.Fprintf(stdout, "Added %s (%d:%d)\n", devName, major, minor)
		}
	case "remove":
		path := filepath.Join("/dev", devName)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "mdev: remove %s: %v\n", devName, err)
			return 1
		}
		if !jsonMode {
			fmt.Fprintf(stdout, "Removed %s\n", devName)
		}
	default:
		fmt.Fprintf(stderr, "mdev: unknown action: %s\n", action)
		return 1
	}
	return 0
}
