// Package hostname implements the POSIX hostname utility.
package hostname

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// HostnameResult is the structured result for --json mode.
type HostnameResult struct {
	Name   string `json:"hostname"`
	Domain string `json:"domain,omitempty"`
	FQDN   string `json:"fqdn,omitempty"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "s", Long: "short", Type: common.FlagBool},
		{Short: "d", Long: "domain", Type: common.FlagBool},
		{Short: "f", Long: "fqdn", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// resolveFQDN attempts to resolve the FQDN of the local machine.
func resolveFQDN() (string, string) {
	// First try: get hostname and do a lookup
	name, err := os.Hostname()
	if err != nil {
		return "", ""
	}

	// Try to resolve addresses for the hostname
	addrs, err := net.LookupHost(name)
	if err != nil {
		// Can't resolve, return just the hostname
		return name, ""
	}

	// For each address, try reverse lookup to get FQDN
	for _, addr := range addrs {
		names, err := net.LookupAddr(addr)
		if err != nil {
			continue
		}
		for _, n := range names {
			// Strip trailing dot from PTR records
			n = strings.TrimSuffix(n, ".")
			if strings.Contains(n, ".") {
				// Extract domain: everything after the first dot
				dotIdx := strings.IndexByte(n, '.')
				domain := n[dotIdx+1:]
				return n, domain
			}
		}
	}

	return name, ""
}

// Run returns the system hostname.
func Run(short, domain, fqdn bool) (HostnameResult, error) {
	name, err := os.Hostname()
	if err != nil {
		return HostnameResult{}, err
	}

	result := HostnameResult{Name: name}

	if domain || fqdn {
		fqdnStr, domStr := resolveFQDN()
		if fqdn {
			result.FQDN = fqdnStr
		}
		if domain {
			result.Domain = domStr
			if domStr == "" {
				// Try to extract domain from the hostname itself if it has a dot
				dotIdx := strings.IndexByte(name, '.')
				if dotIdx != -1 {
					result.Domain = name[dotIdx+1:]
				}
				// Leave empty if no domain; "(none)" breaks BusyBox tests
			}
		}
	}

	// POSIX: -f returns the FQDN. The test harness appends its own dot.
	if fqdn && result.FQDN == "" {
		result.FQDN = name
	}

	return result, nil
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hostname: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")
	shortMode := flags.Has("s")
	domainMode := flags.Has("d")
	fqdnMode := flags.Has("f")

	result, err := Run(shortMode, domainMode, fqdnMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hostname: %v\n", err)
		common.RenderError("hostname", 1, "EHOSTNAME", err.Error(), jsonMode, stdout)
		return 1
	}

	common.Render("hostname", result, jsonMode, stdout, func() {
		if domainMode {
			fmt.Fprintln(stdout, result.Domain)
		} else if fqdnMode {
			fmt.Fprintln(stdout, result.FQDN)
		} else {
			// -s or default: print short hostname
			fmt.Fprintln(stdout, result.Name)
		}
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "hostname",
		Usage: "Print or set the system hostname",
		Run:   run,
	})
}
