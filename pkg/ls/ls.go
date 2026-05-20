// Package ls implements the POSIX ls utility.
package ls

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// FileInfo is the structured representation of a single directory entry.
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	ModTime time.Time `json:"modTime"`
	IsDir   bool      `json:"isDir"`
	Owner   string    `json:"owner"`
	Group   string    `json:"group"`
	Inode   uint64    `json:"inode"`
	Links   uint64    `json:"links"`
	Target  string    `json:"target,omitempty"` // symlink target
	Blocks  int64     `json:"blocks"`
}

// LsResult is the --json envelope data for ls.
type LsResult struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
	Total int        `json:"total"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "a", Long: "all", Type: common.FlagBool},
		{Short: "A", Long: "almost-all", Type: common.FlagBool},
		{Short: "l", Long: "long", Type: common.FlagBool},
		{Short: "R", Long: "recursive", Type: common.FlagBool},
		{Short: "h", Long: "human-readable", Type: common.FlagBool},
		{Short: "t", Long: "sort-time", Type: common.FlagBool},
		{Short: "r", Long: "reverse", Type: common.FlagBool},
		{Short: "S", Long: "sort-size", Type: common.FlagBool},
		{Short: "1", Long: "one-per-line", Type: common.FlagBool},
		{Short: "d", Long: "directory", Type: common.FlagBool},
		{Short: "i", Long: "inode", Type: common.FlagBool},
		{Short: "s", Long: "size", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// statInfo extracts syscall-level fields from an fs.FileInfo.
func statInfo(info fs.FileInfo) (inode, links uint64, blocks int64, uid, gid uint32) {
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return sys.Ino, uint64(sys.Nlink), sys.Blocks, sys.Uid, sys.Gid
	}
	return 0, 1, 0, 0, 0
}

func ownerName(uid uint32) string {
	u, err := user.LookupId(strconv.Itoa(int(uid)))
	if err != nil {
		return strconv.Itoa(int(uid))
	}
	return u.Username
}

func groupName(gid uint32) string {
	g, err := user.LookupGroupId(strconv.Itoa(int(gid)))
	if err != nil {
		return strconv.Itoa(int(gid))
	}
	return g.Name
}

func buildFileInfo(path string, info fs.FileInfo) FileInfo {
	inode, links, blocks, uid, gid := statInfo(info)
	fi := FileInfo{
		Name:    info.Name(),
		Path:    path,
		Size:    info.Size(),
		Mode:    info.Mode().String(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
		Owner:   ownerName(uid),
		Group:   groupName(gid),
		Inode:   inode,
		Links:   links,
		Blocks:  blocks,
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		if target, err := os.Readlink(path); err == nil {
			fi.Target = target
		}
	}
	return fi
}

// humanSize formats a byte count in human-readable form (e.g. 1.5K).
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for n := n / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(n)/float64(div), "KMGTPE"[exp])
}

// Run performs the ls operation and returns the result.
func Run(paths []string, showAll, almostAll, recursive, directoryMode bool) ([]LsResult, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var results []LsResult
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			return nil, err
		}

		if directoryMode || !info.IsDir() {
			fi := buildFileInfo(p, info)
			results = append(results, LsResult{
				Path:  p,
				Files: []FileInfo{fi},
				Total: 1,
			})
			continue
		}

		entries, err := os.ReadDir(p)
		if err != nil {
			return nil, err
		}

		var files []FileInfo
		// Synthetic . and ..
		if showAll {
			for _, dot := range []string{".", ".."} {
				di, err := os.Lstat(filepath.Join(p, dot))
				if err == nil {
					fi := buildFileInfo(filepath.Join(p, dot), di)
					fi.Name = dot
					files = append(files, fi)
				}
			}
		}

		for _, e := range entries {
			name := e.Name()
			if !showAll && !almostAll && strings.HasPrefix(name, ".") {
				continue
			}
			fullPath := filepath.Join(p, name)
			info, err := os.Lstat(fullPath)
			if err != nil {
				continue
			}
			files = append(files, buildFileInfo(fullPath, info))
		}
		results = append(results, LsResult{Path: p, Files: files, Total: len(files)})

		if recursive {
			for _, e := range entries {
				if e.IsDir() && e.Name() != "." && e.Name() != ".." {
					sub, err := Run([]string{filepath.Join(p, e.Name())}, showAll, almostAll, true, false)
					if err == nil {
						results = append(results, sub...)
					}
				}
			}
		}
	}
	return results, nil
}

func sortFiles(files []FileInfo, byTime, bySize, reverse bool) []FileInfo {
	sort.Slice(files, func(i, j int) bool {
		var less bool
		switch {
		case byTime:
			less = files[i].ModTime.After(files[j].ModTime)
		case bySize:
			less = files[i].Size > files[j].Size
		default:
			// Byte-order comparison (LC_ALL=C / POSIX default).
			// Dotfiles sort first.
			a, b := files[i].Name, files[j].Name
			adot, bdot := strings.HasPrefix(a, "."), strings.HasPrefix(b, ".")
			if adot != bdot {
				less = adot
			} else {
				less = a < b
			}
		}
		if reverse {
			return !less
		}
		return less
	})
	return files
}

func printLong(out io.Writer, fi FileInfo, showInode, showBlocks, humanReadable bool) {
	prefix := ""
	if showInode {
		prefix = fmt.Sprintf("%7d ", fi.Inode)
	}
	if showBlocks {
		prefix += fmt.Sprintf("%4d ", fi.Blocks/2)
	}
	sizeStr := fmt.Sprintf("%8d", fi.Size)
	if humanReadable {
		sizeStr = fmt.Sprintf("%8s", humanSize(fi.Size))
	}
	name := fi.Name
	if fi.Target != "" {
		name = fmt.Sprintf("%s -> %s", fi.Name, fi.Target)
	}
	fmt.Fprintf(out, "%s%s %3d %-8s %-8s %s %s %s\n",
		prefix, fi.Mode, fi.Links, fi.Owner, fi.Group,
		sizeStr, fi.ModTime.Format("Jan _2 15:04"), name)
}

func run(args []string, out io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ls: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	showAll := flags.Has("a")
	almostAll := flags.Has("A")
	longFmt := flags.Has("l")
	recursive := flags.Has("R")
	humanReadable := flags.Has("h")
	byTime := flags.Has("t")
	reverse := flags.Has("r")
	bySize := flags.Has("S")
	onePer := flags.Has("1")
	showInode := flags.Has("i")
	showBlocks := flags.Has("s")
	directoryMode := flags.Has("d")

	paths := flags.Positional
	results, err := Run(paths, showAll, almostAll, recursive, directoryMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ls: %v\n", err)
		common.RenderError("ls", 2, "ENOENT", err.Error(), jsonMode, out)
		return 2
	}

	if jsonMode {
		// Flatten to first result for single-path case.
		if len(results) == 1 {
			common.Render("ls", results[0], true, out, func() {})
		} else {
			common.Render("ls", results, true, out, func() {})
		}
		return 0
	}

	multiPath := len(results) > 1
	for _, res := range results {
		files := sortFiles(res.Files, byTime, bySize, reverse)
		// Only show path header for directories, not individual files.
		// System ls: "ls -l file1 dir1" shows "dir1:" header but not "file1:".
		showHeader := multiPath && !isSingleFile(files, res.Path, directoryMode)
		if showHeader {
			fmt.Fprintf(out, "%s:\n", res.Path)
		}
		// Emit "total NNN" line when -l or -s is active and we're
		// listing a directory or multiple items (not a single file).
		if (longFmt || (showBlocks && !longFmt)) && !isSingleFile(files, res.Path, directoryMode) {
			var totalBlocks int64
			for _, fi := range files {
				totalBlocks += fi.Blocks / 2
			}
			fmt.Fprintf(out, "total %d\n", totalBlocks)
		}
		for _, fi := range files {
			name := fi.Name
			// For single-file arguments, use the original path (GNU ls
			// preserves the path specified by the user).
			if isSingleFile(files, res.Path, directoryMode) && res.Path != "" {
				name = res.Path
				fi.Name = res.Path // update for printLong too
			}
			if fi.Target != "" {
				name = fmt.Sprintf("%s -> %s", fi.Name, fi.Target)
			}

			switch {
			case longFmt:
				printLong(out, fi, showInode, showBlocks, humanReadable)
			case onePer:
				if showInode {
					fmt.Fprintf(out, "%7d ", fi.Inode)
				}
				if showBlocks {
					fmt.Fprintf(out, "%4d ", fi.Blocks/2)
				}
				fmt.Fprintln(out, name)
			default:
				// Default mode: one-per-line when not a terminal (piped).
				// Multi-column would require terminal width; for busytest
				// and pipes, one-per-line is the expected behavior.
				if showInode {
					fmt.Fprintf(out, "%7d ", fi.Inode)
				}
				if showBlocks {
					fmt.Fprintf(out, "%4d ", fi.Blocks/2)
				}
				fmt.Fprintln(out, name)
			}
		}
		if showHeader {
			fmt.Fprintln(out)
		}
	}
	return 0
}

// isSingleFile returns true if the listing is for a single regular file
// (not a directory or multiple files). Used to suppress "total" header.
func isSingleFile(files []FileInfo, path string, directoryMode bool) bool {
	if len(files) != 1 {
		return false
	}
	if directoryMode {
		return true
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return !fi.IsDir()
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "ls",
		Usage: "List directory contents",
		Run:   run,
	})
}
