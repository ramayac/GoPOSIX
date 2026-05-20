package tar

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// TarFileStat holds metadata for a single file in the archive for JSON output.
type TarFileStat struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Mode string `json:"mode"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "c", Long: "create", Type: common.FlagBool},
		{Short: "x", Long: "extract", Type: common.FlagBool},
		{Short: "t", Long: "list", Type: common.FlagBool},
		{Short: "z", Long: "gzip", Type: common.FlagBool},
		{Short: "v", Long: "verbose", Type: common.FlagBool},
		{Short: "O", Long: "to-stdout", Type: common.FlagBool},
		{Short: "overwrite", Long: "overwrite", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
		{Short: "f", Long: "file", Type: common.FlagValue},
		{Short: "C", Long: "directory", Type: common.FlagValue},
		{Short: "X", Long: "exclude-from", Type: common.FlagValue},
	},
	PreProcess: preprocessOldStyleFlags,
}

// preprocessOldStyleFlags expands traditional tar flag bundles like "xvf" → "-x -v -f".
func preprocessOldStyleFlags(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	first := args[0]
	if first == "" || first[0] == '-' {
		return args, nil
	}

	// Check if first arg is a bundle of valid tar single-char flags.
	validChars := map[byte]bool{
		'c': true, 'x': true, 't': true, 'r': true, 'u': true,
		'z': true, 'v': true, 'O': true, 'j': true, 'J': true,
		'f': true, 'C': true, 'X': true,
	}
	isOldStyle := true
	hasModeChar := false
	for i := 0; i < len(first); i++ {
		if !validChars[first[i]] {
			isOldStyle = false
			break
		}
		if first[i] == 'c' || first[i] == 'x' || first[i] == 't' || first[i] == 'r' || first[i] == 'u' {
			hasModeChar = true
		}
	}
	if !isOldStyle || !hasModeChar {
		return args, nil
	}

	var expanded []string
	rest := args[1:]
	for i := 0; i < len(first); i++ {
		ch := first[i]
		expanded = append(expanded, "-"+string(ch))
		if (ch == 'f' || ch == 'C' || ch == 'X') && len(rest) > 0 {
			expanded = append(expanded, rest[0])
			rest = rest[1:]
		}
	}
	expanded = append(expanded, rest...)
	return expanded, nil
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	return tarRun(args, stdout, os.Stderr, os.Stdin)
}

func tarRun(args []string, stdout io.Writer, errOut io.Writer, stdin io.Reader) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(errOut, "tar: %v\n", err)
		return 1
	}

	create := flags.Has("c")
	extract := flags.Has("x")
	list := flags.Has("t")
	useGzip := flags.Has("z")
	verbose := flags.Has("v")
	toStdout := flags.Has("O")
	overwrite := flags.Has("overwrite")
	isJSON := flags.Has("json")
	file := flags.Get("f")
	dir := flags.Get("C")

	// Resolve -X exclude files: collect all values from repeatable -X flags.
	var excludePatterns []string
	for _, xf := range flags.GetAll("X") {
		if xf != "" {
			patterns, err := readExcludeFile(xf)
			if err != nil {
				common.RenderError("tar", 1, "IO", fmt.Sprintf("%s: %v", xf, err), isJSON, stdout)
				if !isJSON {
					fmt.Fprintf(errOut, "tar: %s: %v\n", xf, err)
				}
				return 1
			}
			excludePatterns = append(excludePatterns, patterns...)
		}
	}

	if file == "" && !create {
		// No -f specified: default to stdin for extract/list.
		// Also check if there's a positional "-" meaning stdin.
		file = "-"
		for _, p := range flags.Positional {
			if p == "-" {
				file = "-"
				break
			}
		}
	}

	modeCount := 0
	if create {
		modeCount++
	}
	if extract {
		modeCount++
	}
	if list {
		modeCount++
	}
	if modeCount != 1 {
		common.RenderError("tar", 1, "USAGE", "must specify exactly one of -c, -x, or -t", isJSON, stdout)
		if !isJSON {
			fmt.Fprintln(errOut, "tar: must specify exactly one of -c, -x, or -t")
		}
		return 1
	}

	// Resolve archive path before chdir (relative paths break after chdir).
	if file != "" && file != "-" {
		if abs, err := filepath.Abs(file); err == nil {
			file = abs
		}
	}

	var curDir string
	if dir != "" {
		curDir, err = os.Getwd()
		if err != nil {
			common.RenderError("tar", 1, "DIR", err.Error(), isJSON, stdout)
			return 1
		}
		if err := os.Chdir(dir); err != nil {
			common.RenderError("tar", 1, "DIR", err.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, "tar: %v\n", err)
			}
			return 1
		}
		defer os.Chdir(curDir)
	}

	if create {
		return doCreate(file, useGzip, verbose, isJSON, flags.Positional, stdout, errOut)
	} else if extract {
		return doExtract(file, useGzip, verbose, toStdout, overwrite, isJSON, excludePatterns, flags.Positional, stdout, errOut, stdin)
	} else if list {
		return doList(file, useGzip, verbose, isJSON, excludePatterns, stdout, errOut, stdin)
	}

	return 1
}

// resolveTarPath resolves /../ and /./ in a tar target path for member name normalization.
// It walks components left to right, maintaining a stack. ".." pops from stack;
// if stack is empty, the first such ".." forward-cancels the next regular component.
// Returns the resolved path and the stripped prefix (with trailing /).
func resolveTarPath(target string) (resolved string, strippedPrefix string) {
	parts := strings.Split(target, "/")
	var stack []string
	var stripParts []string
	firstEmptyPop := true // first time .. sees an empty stack
	skipNext := false

	for _, p := range parts {
		if p == "" || p == "." {
			stripParts = append(stripParts, p)
			continue
		}
		if p == ".." {
			if len(stack) > 0 {
				popped := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				stripParts = append(stripParts, popped)
			} else if firstEmptyPop {
				skipNext = true
				firstEmptyPop = false
			}
			stripParts = append(stripParts, p)
			continue
		}
		// Regular component
		if skipNext {
			stripParts = append(stripParts, p)
			skipNext = false
			continue
		}
		stack = append(stack, p)
	}

	resolved = strings.Join(stack, "/")
	// Build strip prefix from strip parts.
	if len(stripParts) > 0 {
		for _, s := range stripParts {
			strippedPrefix += s + "/"
		}
		// Preserve trailing / only if the original target had one.
		if !strings.HasSuffix(target, "/") && len(target) > 0 {
			strippedPrefix = strings.TrimRight(strippedPrefix, "/")
			if strippedPrefix != "" {
				strippedPrefix += "/"
			}
		}
	}
	return
}

// createArchiveStream writes a tar (optionally gzipped) to w from the given
// filesystem targets. This is the testable core, separated from file I/O.
func createArchiveStream(w io.Writer, targets []string, archiveAbsPath string, verbose bool, logOut io.Writer) ([]TarFileStat, error) {
	tw := tar.NewWriter(w)
	defer tw.Close()

	var stats []TarFileStat
	for _, target := range targets {
		resolved, strippedPrefix := resolveTarPath(target)
		if resolved == "" {
			resolved = "."
		}

		err := filepath.Walk(resolved, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Do not archive the output file itself
			if archiveAbsPath != "" {
				absFile, _ := filepath.Abs(file)
				if archiveAbsPath == absFile {
					return nil
				}
			}

			header, err := tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return err
			}

			memberName := filepath.ToSlash(file)
			header.Name = memberName

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			stats = append(stats, TarFileStat{
				Name: header.Name,
				Size: header.Size,
				Mode: fi.Mode().String(),
			})

			if verbose {
				name := header.Name
				if fi.IsDir() && !strings.HasSuffix(name, "/") {
					name += "/"
				}
				fmt.Fprintln(logOut, name)
			}

			if !fi.Mode().IsRegular() {
				return nil
			}

			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()
			_, err = io.Copy(tw, data)
			return err
		})

		if err != nil {
			return stats, fmt.Errorf("%s: %w", target, err)
		}

		// Emit message about stripped prefix (always to stderr, per POSIX).
		if strippedPrefix != "" {
			fmt.Fprintf(os.Stderr, "tar: removing leading '%s' from member names\n", strippedPrefix)
		}
	}
	return stats, nil
}

func doCreate(archive string, useGzip, verbose, isJSON bool, targets []string, stdout io.Writer, errOut io.Writer) int {
	var w io.Writer
	if archive == "-" {
		w = stdout
	} else {
		f, err := os.Create(archive)
		if err != nil {
			common.RenderError("tar", 1, "IO", err.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, "tar: %v\n", err)
			}
			return 1
		}
		defer f.Close()
		w = f
	}

	if useGzip {
		gw := gzip.NewWriter(w)
		defer gw.Close()
		w = gw
	}

	archiveAbsPath := ""
	if archive != "-" {
		archiveAbsPath, _ = filepath.Abs(archive)
	}

	logOut := stdout
	if !verbose || isJSON {
		logOut = io.Discard
	}

	stats, err := createArchiveStream(w, targets, archiveAbsPath, verbose && !isJSON, logOut)
	if err != nil {
		common.RenderError("tar", 1, "IO", err.Error(), isJSON, stdout)
		if !isJSON {
			fmt.Fprintf(errOut, "tar: %v\n", err)
		}
		return 1
	}

	if isJSON {
		common.Render("tar", stats, true, stdout, func() {})
	}
	return 0
}

// readExcludeFile reads a list of exclude patterns from a file, one per line.
func readExcludeFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var patterns []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			patterns = append(patterns, line)
		}
	}
	return patterns, sc.Err()
}

// isExcluded checks if name matches any exclude pattern (path prefix match for dirs).
func isExcluded(name string, excludePatterns []string) bool {
	for _, p := range excludePatterns {
		if p == name {
			return true
		}
		// Check if name is under an excluded directory.
		if strings.HasPrefix(name, p+"/") {
			return true
		}
	}
	return false
}

// buildIncludeSet from positional args for extract/include mode.
func buildIncludeSet(positional []string) map[string]bool {
	if len(positional) == 0 {
		return nil
	}
	set := make(map[string]bool)
	for _, p := range positional {
		// Normalize: strip leading ./ and trailing /
		p = strings.TrimPrefix(p, "./")
		p = strings.TrimSuffix(p, "/")
		set[p] = true
	}
	return set
}

// extractArchiveStream reads a tar (optionally gzipped) from r and extracts
// entries to the current directory. This is the testable core.
func extractArchiveStream(r io.Reader, verbose, toStdout, overwrite bool, excludePatterns []string, includeSet map[string]bool, logOut, stdOut io.Writer) ([]TarFileStat, bool, error) {
	hasIncludeList := includeSet != nil
	matchedAny := false

	br := bufio.NewReader(r)
	if _, err := br.Peek(1); err == io.EOF {
		return nil, false, fmt.Errorf("short read")
	}

	tr := tar.NewReader(br)
	var stats []TarFileStat

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, matchedAny, err
		}

		if isExcluded(header.Name, excludePatterns) {
			continue
		}
		if includeSet != nil && !includeSet[header.Name] {
			continue
		}
		matchedAny = true

		target := filepath.Clean(header.Name)
		if strings.HasPrefix(target, "..") || strings.HasPrefix(target, "/") {
			continue
		}

		if verbose {
			name := header.Name
			if header.Typeflag == tar.TypeDir && !strings.HasSuffix(name, "/") {
				name += "/"
			}
			fmt.Fprintln(logOut, name)
		}

		stats = append(stats, TarFileStat{
			Name: header.Name,
			Size: header.Size,
			Mode: fmt.Sprintf("%04o", header.Mode),
		})

		switch header.Typeflag {
		case tar.TypeDir:
			if toStdout {
				continue
			}
			if err := os.MkdirAll(target, os.FileMode(header.Mode)|0300); err != nil {
				return stats, matchedAny, err
			}
			defer os.Chmod(target, os.FileMode(header.Mode))
		case tar.TypeReg:
			if toStdout {
				if _, err := io.Copy(stdOut, tr); err != nil {
					return stats, matchedAny, err
				}
				continue
			}
			dir := filepath.Dir(target)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return stats, matchedAny, err
			}
			var flag int
			if overwrite {
				flag = os.O_WRONLY | os.O_TRUNC
			} else {
				flag = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
			}
			f, err := os.OpenFile(target, flag, os.FileMode(header.Mode))
			if err != nil {
				return stats, matchedAny, err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return stats, matchedAny, err
			}
			f.Close()
			os.Chtimes(target, header.AccessTime, header.ModTime)
		case tar.TypeSymlink:
			if toStdout {
				continue
			}
			dir := filepath.Dir(target)
			os.MkdirAll(dir, 0755)
			os.Symlink(header.Linkname, target)
		}
	}

	if hasIncludeList && !matchedAny {
		return stats, false, fmt.Errorf("file not found in archive")
	}
	return stats, matchedAny, nil
}

func doExtract(archive string, useGzip, verbose, toStdout, overwrite, isJSON bool, excludePatterns, positional []string, stdout io.Writer, errOut io.Writer, stdin io.Reader) int {
	includeSet := buildIncludeSet(positional)

	var r io.Reader
	if archive == "-" {
		r = stdin
	} else {
		f, err := os.Open(archive)
		if err != nil {
			common.RenderError("tar", 1, "IO", err.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, "tar: %v\n", err)
			}
			return 1
		}
		defer f.Close()
		r = f
	}

	if useGzip {
		gr, err := gzip.NewReader(r)
		if err != nil {
			common.RenderError("tar", 1, "GZIP", err.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, "tar: %v\n", err)
			}
			return 1
		}
		defer gr.Close()
		r = gr
	}

	logOut := stdout
	if !verbose || isJSON {
		logOut = io.Discard
	}

	stats, matchedAny, err := extractArchiveStream(r, verbose && !isJSON, toStdout, overwrite, excludePatterns, includeSet, logOut, stdout)
	if err != nil {
		common.RenderError("tar", 1, "IO", err.Error(), isJSON, stdout)
		if !isJSON {
			fmt.Fprintf(errOut, "tar: %v\n", err)
		}
		return 1
	}

	if includeSet != nil && !matchedAny {
		common.RenderError("tar", 1, "NOT_FOUND", "file not found in archive", isJSON, stdout)
		if !isJSON {
			for _, p := range positional {
				fmt.Fprintf(errOut, "tar: %s: Not found in archive\n", p)
			}
		}
		return 1
	}

	if isJSON {
		common.Render("tar", stats, true, stdout, func() {})
	}
	return 0
}

// localTime returns t adjusted for the TZ environment variable.
// POSIX TZ convention: "UTC-2" means 2 hours EAST of UTC (UTC+2).
func localTime(t time.Time) time.Time {
	tz := os.Getenv("TZ")
	if tz == "" {
		return t
	}
	// Handle POSIX "UTC±N" format.
	if strings.HasPrefix(tz, "UTC") {
		rest := tz[3:]
		if rest == "" {
			return t
		}
		sign := 1
		start := 0
		if rest[0] == '-' {
			sign = 1 // POSIX: - means east of UTC
			start = 1
		} else if rest[0] == '+' {
			sign = -1 // POSIX: + means west of UTC
			start = 1
		}
		hours, err := strconv.Atoi(rest[start:])
		if err != nil {
			return t
		}
		offset := time.Duration(sign*hours) * time.Hour
		loc := time.FixedZone(tz, int(offset.Seconds()))
		return t.In(loc)
	}
	return t
}

// formatTarMode returns a BusyBox-style mode string like "drwxr-xr-x" from the tar header.
func formatTarMode(header *tar.Header) string {
	var tc byte
	switch header.Typeflag {
	case tar.TypeDir:
		tc = 'd'
	case tar.TypeSymlink:
		tc = 'l'
	case tar.TypeLink:
		tc = 'h'
	case tar.TypeChar:
		tc = 'c'
	case tar.TypeBlock:
		tc = 'b'
	case tar.TypeFifo:
		tc = 'p'
	default:
		tc = '-'
	}
	mode := int64(header.Mode) & 0777
	rwx := [9]byte{'-', '-', '-', '-', '-', '-', '-', '-', '-'}
	if mode&0400 != 0 {
		rwx[0] = 'r'
	}
	if mode&0200 != 0 {
		rwx[1] = 'w'
	}
	if mode&0100 != 0 {
		rwx[2] = 'x'
	}
	if mode&0040 != 0 {
		rwx[3] = 'r'
	}
	if mode&0020 != 0 {
		rwx[4] = 'w'
	}
	if mode&0010 != 0 {
		rwx[5] = 'x'
	}
	if mode&0004 != 0 {
		rwx[6] = 'r'
	}
	if mode&0002 != 0 {
		rwx[7] = 'w'
	}
	if mode&0001 != 0 {
		rwx[8] = 'x'
	}
	return string(tc) + string(rwx[:])
}

// listArchiveStream reads a tar (optionally gzipped) from r and writes a
// listing to listOut. This is the testable core, separated from file I/O.
func listArchiveStream(r io.Reader, verbose bool, excludePatterns []string, listOut io.Writer) ([]TarFileStat, error) {
	br := bufio.NewReader(r)
	if _, err := br.Peek(1); err == io.EOF {
		return nil, fmt.Errorf("short read")
	}

	tr := tar.NewReader(br)
	var stats []TarFileStat

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, err
		}

		if isExcluded(header.Name, excludePatterns) {
			continue
		}

		stats = append(stats, TarFileStat{
			Name: header.Name,
			Size: header.Size,
			Mode: fmt.Sprintf("%04o", header.Mode),
		})

		name := header.Name
		if header.Typeflag == tar.TypeDir && !strings.HasSuffix(name, "/") {
			name += "/"
		}
		if verbose {
			mode := formatTarMode(header)
			size := header.Size
			if header.Typeflag == tar.TypeSymlink {
				size = 0
			}
			t := localTime(header.ModTime)
			line := fmt.Sprintf("%s %s/%s%10d %04d-%02d-%02d %02d:%02d:%02d %s",
				mode,
				header.Uname, header.Gname,
				size,
				t.Year(), t.Month(), t.Day(),
				t.Hour(), t.Minute(), t.Second(),
				name,
			)
			if header.Typeflag == tar.TypeSymlink {
				line += " -> " + header.Linkname
			}
			fmt.Fprintln(listOut, line)
		} else {
			fmt.Fprintln(listOut, name)
		}
	}
	return stats, nil
}

func doList(archive string, useGzip, verbose, isJSON bool, excludePatterns []string, stdout io.Writer, errOut io.Writer, stdin io.Reader) int {
	var r io.Reader
	if archive == "-" {
		r = stdin
	} else {
		f, err := os.Open(archive)
		if err != nil {
			common.RenderError("tar", 1, "IO", err.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, "tar: %v\n", err)
			}
			return 1
		}
		defer f.Close()
		r = f
	}

	if useGzip {
		gr, err := gzip.NewReader(r)
		if err != nil {
			common.RenderError("tar", 1, "GZIP", err.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, "tar: %v\n", err)
			}
			return 1
		}
		defer gr.Close()
		r = gr
	}

	listOut := stdout
	if isJSON {
		listOut = io.Discard
	}

	stats, err := listArchiveStream(r, verbose, excludePatterns, listOut)
	if err != nil {
		common.RenderError("tar", 1, "IO", err.Error(), isJSON, stdout)
		if !isJSON {
			fmt.Fprintf(errOut, "tar: %v\n", err)
		}
		return 1
	}

	if isJSON {
		common.Render("tar", stats, true, stdout, func() {})
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{Name: "tar", Usage: "tar archive utility", Run: run})
}
