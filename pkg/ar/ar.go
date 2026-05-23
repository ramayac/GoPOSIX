// Package ar implements the ar(1) utility for GoPOSIX.
//
// ar creates, modifies, and extracts from archives. It supports the classic
// BSD/GNU .a format (used by linkers and package managers).
//
// Supported operations:
//
//	ar r archive file...   — replace (add/update) members
//	ar c archive file...   — create archive (same as r, suppresses warning)
//	ar t archive           — list table of contents
//	ar x archive [file...] — extract members
//	ar p archive [file...] — print members to stdout
//	ar d archive file...   — delete members
package ar

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	bsar "github.com/blakesmith/ar"
	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// ArMember holds metadata for a single archive member (for JSON output).
type ArMember struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
	Mode    uint32 `json:"mode"`
	UID     int    `json:"uid"`
	GID     int    `json:"gid"`
}

// ArResult is the JSON output envelope.
type ArResult struct {
	Archive string     `json:"archive"`
	Members []ArMember `json:"members"`
}

// arEntry holds an in-memory archive member.
type arEntry struct {
	hdr  bsar.Header
	data []byte
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "ar",
		Usage: "Create, modify, and extract from archives",
		Run:   run,
	})
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return arRun(args, stdin, stdout, stderr, cwd)
}

// arRun is the injectable entry point for testing.
func arRun(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "ar: usage: ar [dprqtxco][vV] archive files...")
		return 1
	}

	jsonMode := false
	// Check for --json anywhere in args
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) == 0 {
		fmt.Fprintln(stderr, "ar: usage: ar [dprqtxco][vV] archive files...")
		return 1
	}

	// Parse operation flags (first arg may be flags without '-' like BusyBox)
	opStr := args[0]
	if len(opStr) > 0 && opStr[0] == '-' {
		opStr = opStr[1:]
	}
	args = args[1:]

	if len(args) == 0 {
		fmt.Fprintln(stderr, "ar: missing archive argument")
		return 1
	}

	archive := args[0]
	if !filepath.IsAbs(archive) {
		archive = filepath.Join(cwd, archive)
	}
	members := args[1:]

	// Parse operation character
	op := byte(0)
	verbose := false
	for i := 0; i < len(opStr); i++ {
		switch opStr[i] {
		case 'r', 'q': // replace/quick-append (treat same)
			op = 'r'
		case 'c': // create — same as r
			if op == 0 {
				op = 'r'
			}
		case 'd': // delete
			op = 'd'
		case 't': // list table of contents
			op = 't'
		case 'x': // extract
			op = 'x'
		case 'p': // print
			op = 'p'
		case 'v', 'V': // verbose
			verbose = true
		}
	}

	if op == 0 {
		fmt.Fprintln(stderr, "ar: no operation specified")
		return 1
	}

	switch op {
	case 'r':
		return arReplace(archive, members, verbose, jsonMode, stdout, stderr, cwd)
	case 't':
		return arList(archive, members, verbose, jsonMode, stdout, stderr)
	case 'x':
		return arExtract(archive, members, verbose, jsonMode, stdout, stderr, cwd)
	case 'p':
		return arPrint(archive, members, verbose, stdout, stderr)
	case 'd':
		return arDelete(archive, members, verbose, jsonMode, stdout, stderr)
	}

	fmt.Fprintln(stderr, "ar: unknown operation")
	return 1
}

// readArchive reads all entries from an ar archive file.
func readArchive(archive string) ([]arEntry, error) {
	f, err := os.Open(archive)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // new archive — empty
		}
		return nil, err
	}
	defer f.Close()

	reader := bsar.NewReader(f)
	var entries []arEntry
	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		entries = append(entries, arEntry{hdr: *hdr, data: data})
	}
	return entries, nil
}

// flushArchive writes entries back to an archive file atomically.
func flushArchive(archive string, entries []arEntry, stderr io.Writer) int {
	tmp := archive + ".tmp"
	outf, err := os.Create(tmp)
	if err != nil {
		fmt.Fprintf(stderr, "ar: %v\n", err)
		return 1
	}

	writer := bsar.NewWriter(outf)
	// Write the global header first
	if err := writer.WriteGlobalHeader(); err != nil {
		outf.Close()
		os.Remove(tmp)
		fmt.Fprintf(stderr, "ar: %v\n", err)
		return 1
	}
	for i := range entries {
		e := &entries[i]
		e.hdr.Size = int64(len(e.data))
		if err := writer.WriteHeader(&e.hdr); err != nil {
			outf.Close()
			os.Remove(tmp)
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}
		if _, err := writer.Write(e.data); err != nil {
			outf.Close()
			os.Remove(tmp)
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}
	}
	outf.Close()

	if err := os.Rename(tmp, archive); err != nil {
		fmt.Fprintf(stderr, "ar: %v\n", err)
		return 1
	}
	return 0
}

// arList lists the table of contents of an archive.
func arList(archive string, filter []string, verbose, jsonMode bool, stdout, stderr io.Writer) int {
	f, err := os.Open(archive)
	if err != nil {
		fmt.Fprintf(stderr, "ar: %v\n", err)
		return 1
	}
	defer f.Close()

	reader := bsar.NewReader(f)
	filterSet := makeSet(filter)

	var jsonMembers []ArMember
	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}
		if len(filterSet) > 0 && !filterSet[hdr.Name] {
			continue
		}
		if jsonMode {
			jsonMembers = append(jsonMembers, ArMember{
				Name:    hdr.Name,
				Size:    hdr.Size,
				ModTime: hdr.ModTime.Unix(),
				Mode:    uint32(hdr.Mode),
				UID:     hdr.Uid,
				GID:     hdr.Gid,
			})
		} else if verbose {
			fmt.Fprintf(stdout, "%s %d/%d %8d %s %s\n",
				modeString(hdr.Mode),
				hdr.Uid, hdr.Gid,
				hdr.Size,
				hdr.ModTime.Format("Jan  2 15:04 2006"),
				hdr.Name,
			)
		} else {
			fmt.Fprintln(stdout, hdr.Name)
		}
	}

	if jsonMode {
		common.Render("ar", ArResult{Archive: archive, Members: jsonMembers}, true, stdout, nil)
	}
	return 0
}

// arExtract extracts members from an archive into the current directory.
func arExtract(archive string, filter []string, verbose, jsonMode bool, stdout, stderr io.Writer, cwd string) int {
	f, err := os.Open(archive)
	if err != nil {
		fmt.Fprintf(stderr, "ar: %v\n", err)
		return 1
	}
	defer f.Close()

	reader := bsar.NewReader(f)
	filterSet := makeSet(filter)

	var jsonMembers []ArMember
	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}

		if len(filterSet) > 0 && !filterSet[hdr.Name] {
			// consume data to advance reader
			if _, err = io.Copy(io.Discard, reader); err != nil {
				fmt.Fprintf(stderr, "ar: %v\n", err)
				return 1
			}
			continue
		}

		dest := filepath.Join(cwd, filepath.Base(hdr.Name))
		if verbose {
			fmt.Fprintf(stdout, "x - %s\n", hdr.Name)
		}
		outFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
		if err != nil {
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}
		_, copyErr := io.Copy(outFile, reader)
		outFile.Close()
		if copyErr != nil {
			fmt.Fprintf(stderr, "ar: %v\n", copyErr)
			return 1
		}
		if jsonMode {
			jsonMembers = append(jsonMembers, ArMember{
				Name:    hdr.Name,
				Size:    hdr.Size,
				ModTime: hdr.ModTime.Unix(),
				Mode:    uint32(hdr.Mode),
				UID:     hdr.Uid,
				GID:     hdr.Gid,
			})
		}
	}

	if jsonMode {
		common.Render("ar", ArResult{Archive: archive, Members: jsonMembers}, true, stdout, nil)
	}
	return 0
}

// arPrint prints members to stdout.
func arPrint(archive string, filter []string, verbose bool, stdout, stderr io.Writer) int {
	f, err := os.Open(archive)
	if err != nil {
		fmt.Fprintf(stderr, "ar: %v\n", err)
		return 1
	}
	defer f.Close()

	reader := bsar.NewReader(f)
	filterSet := makeSet(filter)

	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}
		if len(filterSet) > 0 && !filterSet[hdr.Name] {
			if _, err = io.Copy(io.Discard, reader); err != nil {
				fmt.Fprintf(stderr, "ar: %v\n", err)
				return 1
			}
			continue
		}
		if verbose {
			fmt.Fprintf(stdout, "\n<%s>\n\n", hdr.Name)
		}
		if _, err = io.Copy(stdout, reader); err != nil {
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}
	}
	return 0
}

// arDelete removes members from an archive (rebuilds it without them).
func arDelete(archive string, toDelete []string, verbose, jsonMode bool, stdout, stderr io.Writer) int {
	if len(toDelete) == 0 {
		fmt.Fprintln(stderr, "ar: no files to delete specified")
		return 1
	}
	deleteSet := makeSet(toDelete)

	entries, err := readArchive(archive)
	if err != nil {
		fmt.Fprintf(stderr, "ar: %v\n", err)
		return 1
	}

	kept := entries[:0]
	for _, e := range entries {
		if deleteSet[e.hdr.Name] {
			if verbose {
				fmt.Fprintf(stdout, "d - %s\n", e.hdr.Name)
			}
			continue
		}
		kept = append(kept, e)
	}

	return flushArchive(archive, kept, stderr)
}

// arReplace adds or replaces members in an archive.
func arReplace(archive string, files []string, verbose, jsonMode bool, stdout, stderr io.Writer, cwd string) int {
	entries, err := readArchive(archive)
	if err != nil {
		fmt.Fprintf(stderr, "ar: %v\n", err)
		return 1
	}

	// Build index for fast lookup
	index := map[string]int{}
	for i, e := range entries {
		index[e.hdr.Name] = i
	}

	for _, file := range files {
		absFile := file
		if !filepath.IsAbs(absFile) {
			absFile = filepath.Join(cwd, file)
		}
		data, err := os.ReadFile(absFile)
		if err != nil {
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}
		info, err := os.Stat(absFile)
		if err != nil {
			fmt.Fprintf(stderr, "ar: %v\n", err)
			return 1
		}
		baseName := filepath.Base(file)
		hdr := bsar.Header{
			Name:    baseName,
			ModTime: info.ModTime(),
			Mode:    int64(info.Mode()),
			Size:    int64(len(data)),
		}
		e := arEntry{hdr: hdr, data: data}
		if idx, exists := index[baseName]; exists {
			if verbose {
				fmt.Fprintf(stdout, "r - %s\n", baseName)
			}
			entries[idx] = e
		} else {
			if verbose {
				fmt.Fprintf(stdout, "a - %s\n", baseName)
			}
			index[baseName] = len(entries)
			entries = append(entries, e)
		}
	}

	return flushArchive(archive, entries, stderr)
}

// makeSet converts a string slice to a lookup set (by basename too).
func makeSet(s []string) map[string]bool {
	m := map[string]bool{}
	for _, v := range s {
		m[filepath.Base(v)] = true
		m[v] = true
	}
	return m
}

// modeString returns a simple -rwxrwxrwx string for a file mode.
func modeString(mode int64) string {
	const chars = "rwxrwxrwx"
	result := make([]byte, 10)
	result[0] = '-'
	for i := 0; i < 9; i++ {
		bit := uint(8 - i)
		if mode&(1<<bit) != 0 {
			result[i+1] = chars[i]
		} else {
			result[i+1] = '-'
		}
	}
	return string(result)
}
