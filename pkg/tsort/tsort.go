// Package tsort implements the POSIX-aligned tsort utility.
package tsort

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// TsortResult represents the structured JSON output.
type TsortResult struct {
	Nodes []string `json:"nodes"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(stderr, "tsort: %v\n", err)
		return 1
	}

	jsonMode := flags.Has("json")

	var r io.Reader
	if len(flags.Positional) > 1 {
		fmt.Fprintf(stderr, "tsort: extra operand\n")
		return 1
	} else if len(flags.Positional) == 1 && flags.Positional[0] != "-" {
		f, err := os.Open(flags.Positional[0])
		if err != nil {
			fmt.Fprintf(stderr, "tsort: %v\n", err)
			return 1
		}
		defer f.Close()
		r = f
	} else {
		r = stdin
	}

	data, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(stderr, "tsort: read error: %v\n", err)
		return 1
	}

	words := strings.Fields(string(data))
	if len(words)%2 != 0 {
		fmt.Fprintf(stderr, "tsort: odd number of words in input\n")
		return 1
	}

	// Graph construction
	var uniqueNodes []string
	nodeMap := make(map[string]bool)
	addNode := func(node string) {
		if !nodeMap[node] {
			nodeMap[node] = true
			uniqueNodes = append(uniqueNodes, node)
		}
	}

	adj := make(map[string][]string)
	edgeMap := make(map[string]bool)
	inDegree := make(map[string]int)

	for i := 0; i < len(words); i += 2 {
		u := words[i]
		v := words[i+1]
		addNode(u)
		addNode(v)

		if u == v {
			// Self-loop is valid to specify singleton node
			continue
		}

		edgeKey := u + "->" + v
		if !edgeMap[edgeKey] {
			edgeMap[edgeKey] = true
			adj[u] = append(adj[u], v)
			inDegree[v]++
		}
	}

	// Kahn's algorithm
	var queue []string
	for _, n := range uniqueNodes {
		if inDegree[n] == 0 {
			queue = append(queue, n)
		}
	}

	var result []string
	for len(queue) > 0 {
		// pop
		u := queue[0]
		queue = queue[1:]
		result = append(result, u)

		for _, v := range adj[u] {
			inDegree[v]--
			if inDegree[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	// Cycle detection
	if len(result) < len(uniqueNodes) {
		fmt.Fprintf(stderr, "tsort: cycle detected\n")
		// Output the reachable nodes first
		common.Render("tsort", TsortResult{Nodes: result}, jsonMode, stdout, func() {
			for _, n := range result {
				fmt.Fprintln(stdout, n)
			}
		})
		return 1
	}

	common.Render("tsort", TsortResult{Nodes: result}, jsonMode, stdout, func() {
		for _, n := range result {
			fmt.Fprintln(stdout, n)
		}
	})

	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "tsort",
		Usage: "Topological sort of a directed graph",
		Run:   run,
	})
}
