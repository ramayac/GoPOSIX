// Package cpio implements the cpio(1) utility for GoPOSIX.
//
// cpio copies files to/from archives in the cpio format. It supports the
// three traditional modes:
//
//	cpio -o  (--create)  — create: read file names from stdin, write archive to stdout
//	cpio -i  (--extract) — extract: read archive from stdin, extract files to disk
//	cpio -p  (--pass-through) — copy files to dest directory
//
// Only the newc (SVR4 ASCII) format is used for creation; all formats supported
// by the cavaliergopher/cpio library can be read for extraction.
package cpio

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cavaliergopher/cpio"
	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// CpioMember describes a single file entry for JSON output.
type CpioMember struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    uint32 `json:"mode"`
	ModTime int64  `json:"mod_time"`
	UID     int    `json:"uid"`
	GID     int    `json:"gid"`
}

// CpioResult is the JSON output envelope.
type CpioResult struct {
	Members []CpioMember `json:"members"`
}

var flagSpec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "o", Long: "create", Type: common.FlagBool},
		{Short: "i", Long: "extract", Type: common.FlagBool},
		{Short: "p", Long: "pass-through", Type: common.FlagBool},
		{Short: "t", Long: "list", Type: common.FlagBool},
		{Short: "v", Long: "verbose", Type: common.FlagBool},
		{Short: "d", Long: "make-directories", Type: common.FlagBool},
		{Short: "u", Long: "unconditional", Type: common.FlagBool},
		{Short: "F", Long: "file", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "cpio",
		Usage: "Copy files to/from archives",
		Run:   run,
	})
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return cpioRun(args, stdin, stdout, stderr, cwd)
}

// cpioRun is the injectable entry point for testing.
func cpioRun(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, flagSpec)
	if err != nil {
		fmt.Fprintf(stderr, "cpio: %v\n", err)
		return 1
	}

	pos := flags.Positional

	createMode := flags.Has("o")
	extractMode := flags.Has("i")
	passMode := flags.Has("p")
	listMode := flags.Has("t")
	verbose := flags.Has("v")
	makeDirs := flags.Has("d")
	jsonMode := flags.Has("json")
	archiveFile := flags.Get("F")

	// Exactly one mode must be selected
	modeCount := 0
	if createMode {
		modeCount++
	}
	if extractMode {
		modeCount++
	}
	if passMode {
		modeCount++
	}
	if listMode {
		modeCount++
		extractMode = true // list is a subset of extract
	}

	if modeCount == 0 {
		fmt.Fprintln(stderr, "cpio: you must specify one of -oip or -t")
		return 1
	}
	if modeCount > 1 && !(listMode && extractMode) {
		fmt.Fprintln(stderr, "cpio: only one of -oip may be specified")
		return 1
	}

	// Resolve the archive I/O stream
	var archiveIn io.Reader = stdin
	var archiveOut io.Writer = stdout

	if archiveFile != "" {
		if !filepath.IsAbs(archiveFile) {
			archiveFile = filepath.Join(cwd, archiveFile)
		}
		if createMode {
			f, err := os.Create(archiveFile)
			if err != nil {
				fmt.Fprintf(stderr, "cpio: %v\n", err)
				return 1
			}
			defer f.Close()
			archiveOut = f
		} else {
			f, err := os.Open(archiveFile)
			if err != nil {
				fmt.Fprintf(stderr, "cpio: %v\n", err)
				return 1
			}
			defer f.Close()
			archiveIn = f
		}
	}

	switch {
	case createMode:
		return cpioCreate(archiveOut, stdin, verbose, jsonMode, stdout, stderr, cwd)
	case passMode:
		if len(pos) == 0 {
			fmt.Fprintln(stderr, "cpio: destination directory required for -p")
			return 1
		}
		dest := pos[0]
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(cwd, dest)
		}
		return cpioPass(dest, stdin, verbose, jsonMode, stdout, stderr, cwd)
	default: // extract or list
		return cpioExtract(archiveIn, listMode, makeDirs, verbose, jsonMode, pos, stdout, stderr, cwd)
	}
}

// cpioCreate reads file names from stdin and writes a cpio archive.
func cpioCreate(out io.Writer, nameReader io.Reader, verbose, jsonMode bool, stdout, stderr io.Writer, cwd string) int {
	w := cpio.NewWriter(out)
	var members []CpioMember

	scanner := bufio.NewScanner(nameReader)
	for scanner.Scan() {
		name := scanner.Text()
		if name == "" {
			continue
		}
		absPath := name
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(cwd, name)
		}

		info, err := os.Lstat(absPath)
		if err != nil {
			fmt.Fprintf(stderr, "cpio: %v\n", err)
			continue
		}

		hdr := &cpio.Header{
			Name:    name,
			Size:    info.Size(),
			Mode:    cpio.FileMode(info.Mode()),
			ModTime: info.ModTime(),
		}

		if err := w.WriteHeader(hdr); err != nil {
			fmt.Fprintf(stderr, "cpio: %v\n", err)
			return 1
		}

		if info.Mode().IsRegular() {
			f, err := os.Open(absPath)
			if err != nil {
				fmt.Fprintf(stderr, "cpio: %v\n", err)
				return 1
			}
			if _, err = io.Copy(w, f); err != nil {
				f.Close()
				fmt.Fprintf(stderr, "cpio: %v\n", err)
				return 1
			}
			f.Close()
		}

		if verbose {
			fmt.Fprintln(stderr, name)
		}
		if jsonMode {
			members = append(members, CpioMember{
				Name:    name,
				Size:    info.Size(),
				Mode:    uint32(info.Mode()),
				ModTime: info.ModTime().Unix(),
			})
		}
	}

	if err := w.Close(); err != nil {
		fmt.Fprintf(stderr, "cpio: %v\n", err)
		return 1
	}

	if jsonMode {
		common.Render("cpio", CpioResult{Members: members}, true, stdout, nil)
	}
	return 0
}

// cpioExtract reads a cpio archive from in and extracts (or lists) its contents.
func cpioExtract(in io.Reader, listOnly, makeDirs, verbose, jsonMode bool, filter []string, stdout, stderr io.Writer, cwd string) int {
	r := cpio.NewReader(in)
	filterSet := makeSet(filter)
	var members []CpioMember

	for {
		hdr, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(stderr, "cpio: %v\n", err)
			return 1
		}

		name := hdr.Name
		if name == "TRAILER!!!" {
			break
		}

		if len(filterSet) > 0 && !filterSet[name] {
			continue
		}

		if (verbose || listOnly) && !jsonMode {
			fmt.Fprintln(stdout, name)
		}

		if jsonMode {
			members = append(members, CpioMember{
				Name:    name,
				Size:    hdr.Size,
				Mode:    uint32(hdr.Mode),
				ModTime: hdr.ModTime.Unix(),
			})
		}

		if listOnly {
			continue
		}

		dest := filepath.Join(cwd, filepath.FromSlash(name))
		mode := os.FileMode(hdr.Mode & 0xFFF)

		// Determine type using the standard file type mask (octal 0170000)
		const modeTypeMask = cpio.FileMode(0170000)
		switch hdr.Mode & modeTypeMask {
		case cpio.TypeDir:
			if makeDirs {
				if err := os.MkdirAll(dest, mode|0700); err != nil {
					fmt.Fprintf(stderr, "cpio: mkdir %s: %v\n", dest, err)
				}
			}
		case cpio.TypeSymlink:
			// read symlink target from data
			data, _ := io.ReadAll(r)
			target := strings.TrimRight(string(data), "\x00")
			if makeDirs {
				os.MkdirAll(filepath.Dir(dest), 0755)
			}
			os.Remove(dest)
			os.Symlink(target, dest)
		default:
			// regular file
			if makeDirs {
				os.MkdirAll(filepath.Dir(dest), 0755)
			}
			f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				fmt.Fprintf(stderr, "cpio: %v\n", err)
				return 1
			}
			_, copyErr := io.Copy(f, r)
			f.Close()
			if copyErr != nil {
				fmt.Fprintf(stderr, "cpio: %v\n", copyErr)
				return 1
			}
		}
	}

	if jsonMode {
		common.Render("cpio", CpioResult{Members: members}, true, stdout, nil)
	}
	return 0
}

// cpioPass copies files named on stdin to a destination directory.
func cpioPass(dest string, nameReader io.Reader, verbose, jsonMode bool, stdout, stderr io.Writer, cwd string) int {
	var members []CpioMember
	scanner := bufio.NewScanner(nameReader)

	for scanner.Scan() {
		name := scanner.Text()
		if name == "" {
			continue
		}
		srcPath := name
		if !filepath.IsAbs(srcPath) {
			srcPath = filepath.Join(cwd, name)
		}

		info, err := os.Lstat(srcPath)
		if err != nil {
			fmt.Fprintf(stderr, "cpio: %v\n", err)
			continue
		}

		destPath := filepath.Join(dest, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(destPath), 0755)

		if verbose {
			fmt.Fprintln(stderr, name)
		}

		if info.Mode().IsRegular() {
			src, err := os.Open(srcPath)
			if err != nil {
				fmt.Fprintf(stderr, "cpio: %v\n", err)
				continue
			}
			dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
			if err != nil {
				src.Close()
				fmt.Fprintf(stderr, "cpio: %v\n", err)
				continue
			}
			_, err = io.Copy(dst, src)
			dst.Close()
			src.Close()
			if err != nil {
				fmt.Fprintf(stderr, "cpio: %v\n", err)
				continue
			}
		} else if info.IsDir() {
			os.MkdirAll(destPath, info.Mode())
		}

		if jsonMode {
			members = append(members, CpioMember{
				Name:    name,
				Size:    info.Size(),
				Mode:    uint32(info.Mode()),
				ModTime: info.ModTime().Unix(),
			})
		}
	}

	if jsonMode {
		common.Render("cpio", CpioResult{Members: members}, true, stdout, nil)
	}
	return 0
}

// makeSet converts a string slice into a fast lookup set.
func makeSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

// Ensure time is used (it's used in CpioMember.ModTime via Unix()).
var _ = time.Now
