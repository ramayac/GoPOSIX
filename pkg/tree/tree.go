// Package tree implements the POSIX-aligned tree utility.
package tree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// Node represents a node in the directory tree for internal building.
type Node struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`             // "directory" or "file"
	Target   string  `json:"target,omitempty"` // symlink target if any
	Error    bool    `json:"error,omitempty"`  // true if error opening dir
	Children []*Node `json:"contents,omitempty"`
}

// TreeReport holds summary numbers.
type TreeReport struct {
	Directories int `json:"directories"`
	Files       int `json:"files"`
}

// TreeResult wraps the trees and report for JSON rendering.
type TreeResult struct {
	Trees  []*Node    `json:"trees"`
	Report TreeReport `json:"report"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "a", Long: "all", Type: common.FlagBool},
		{Short: "d", Type: common.FlagBool},
		{Short: "L", Type: common.FlagValue},
		{Long: "json", Type: common.FlagBool},
	},
}

type treeContext struct {
	dirsCount  int
	filesCount int
	all        bool
	dirsOnly   bool
	maxDepth   int
	hadError   bool
}

func (ctx *treeContext) buildTree(path string, nodeName string, depth int) (*Node, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		ctx.hadError = true
		return &Node{
			Name:  nodeName,
			Type:  "directory",
			Error: true,
		}, nil
	}

	isDir := fi.IsDir()
	isSymlink := (fi.Mode() & os.ModeSymlink) != 0

	var target string
	if isSymlink {
		if t, err := os.Readlink(path); err == nil {
			target = t
		}
	}

	node := &Node{
		Name: nodeName,
	}

	if isDir && !isSymlink {
		node.Type = "directory"
	} else {
		node.Type = "file"
		if isSymlink {
			node.Target = target
		}
	}

	// Limit depth (maxDepth of 0 means only list root, etc)
	if ctx.maxDepth >= 0 && depth >= ctx.maxDepth {
		return node, nil
	}

	if isDir && !isSymlink {
		entries, err := os.ReadDir(path)
		if err != nil {
			node.Error = true
			ctx.hadError = true
			return node, nil
		}

		// Sort entries alphabetically
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			childName := entry.Name()
			if !ctx.all && strings.HasPrefix(childName, ".") {
				continue
			}

			childPath := filepath.Join(path, childName)
			childFi, err := os.Lstat(childPath)
			if err != nil {
				continue
			}

			childIsDir := childFi.IsDir()
			childIsSymlink := (childFi.Mode() & os.ModeSymlink) != 0

			if ctx.dirsOnly && !(childIsDir && !childIsSymlink) {
				continue
			}

			if childIsDir && !childIsSymlink {
				ctx.dirsCount++
			} else {
				ctx.filesCount++
			}

			childNode, _ := ctx.buildTree(childPath, childName, depth+1)
			if childNode != nil {
				node.Children = append(node.Children, childNode)
			}
		}
	}

	return node, nil
}

func printTextTree(stdout io.Writer, node *Node, indent string, isLast bool, isRoot bool) {
	if isRoot {
		if node.Error {
			fmt.Fprintf(stdout, "%s [error opening dir]\n", node.Name)
			return
		}
		fmt.Fprintf(stdout, "%s\n", node.Name)
	} else {
		prefix := "├── "
		if isLast {
			prefix = "└── "
		}
		if node.Error {
			fmt.Fprintf(stdout, "%s%s%s [error opening dir]\n", indent, prefix, node.Name)
			return
		}
		if node.Type == "file" && node.Target != "" {
			fmt.Fprintf(stdout, "%s%s%s -> %s\n", indent, prefix, node.Name, node.Target)
		} else {
			fmt.Fprintf(stdout, "%s%s%s\n", indent, prefix, node.Name)
		}
	}

	var childIndent string
	if !isRoot {
		if isLast {
			childIndent = indent + "    "
		} else {
			childIndent = indent + "│   "
		}
	}

	for i, child := range node.Children {
		printTextTree(stdout, child, childIndent, i == len(node.Children)-1, false)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "tree: %v\n", err)
		return 1
	}

	jsonMode := flags.Has("json")
	all := flags.Has("all")
	dirsOnly := flags.Has("d")
	maxDepth := -1

	if flags.Has("L") {
		lVal := flags.Get("L")
		if depth, err := strconv.Atoi(lVal); err == nil {
			maxDepth = depth
		} else {
			fmt.Fprintf(stderr, "tree: invalid depth: %s\n", lVal)
			return 1
		}
	}

	ctx := &treeContext{
		all:      all,
		dirsOnly: dirsOnly,
		maxDepth: maxDepth,
	}

	roots := flags.Positional
	if len(roots) == 0 {
		roots = []string{"."}
	}

	var builtTrees []*Node
	for _, root := range roots {
		node, _ := ctx.buildTree(root, root, 0)
		if node != nil {
			builtTrees = append(builtTrees, node)
		}
	}

	result := TreeResult{
		Trees: builtTrees,
		Report: TreeReport{
			Directories: ctx.dirsCount,
			Files:       ctx.filesCount,
		},
	}

	common.Render("tree", result, jsonMode, stdout, func() {
		for _, tNode := range builtTrees {
			printTextTree(stdout, tNode, "", true, true)
		}
		fmt.Fprintf(stdout, "\n%d directories, %d files\n", ctx.dirsCount, ctx.filesCount)
	})

	if ctx.hadError {
		return 1
	}
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "tree",
		Usage: "List contents of directories in a tree-like format",
		Run:   run,
	})
}
