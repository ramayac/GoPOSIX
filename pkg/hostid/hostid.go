// Package hostid implements the POSIX-compliant hostid utility.
package hostid

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"unsafe"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// HostidResult is the standard JSON schema for hostid output.
type HostidResult struct {
	Hostid string `json:"hostid"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Short: "V", Long: "version", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

// Global variables for mockability in tests
var (
	etcHostidPath      = "/etc/hostid"
	getHostnameFunc    = os.Hostname
	lookupIPFunc       = net.LookupIP
	interfaceAddrsFunc = net.InterfaceAddrs
	nativeEndian       binary.ByteOrder
)

func init() {
	// Detect native endianness
	i := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&i))
	if b[0] == 0 {
		nativeEndian = binary.BigEndian
	} else {
		nativeEndian = binary.LittleEndian
	}

	dispatch.Register(dispatch.Command{
		Name:  "hostid",
		Usage: "Print the numeric identifier (in hexadecimal) for the current host",
		Run:   run,
	})
}

// GetHostID retrieves the 32-bit numeric host ID.
func GetHostID() uint32 {
	// 1. Try /etc/hostid
	if data, err := os.ReadFile(etcHostidPath); err == nil && len(data) >= 4 {
		return nativeEndian.Uint32(data[:4])
	}

	// 2. Try resolving hostname to IPv4 address
	if hostname, err := getHostnameFunc(); err == nil && hostname != "" {
		if ips, err := lookupIPFunc(hostname); err == nil {
			for _, ip := range ips {
				if ip4 := ip.To4(); ip4 != nil {
					return constructHostID(ip4)
				}
			}
		}
	}

	// 3. Fallback: Find the first non-loopback local interface IPv4 address
	if addrs, err := interfaceAddrsFunc(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					return constructHostID(ip4)
				}
			}
		}
	}

	// 4. Ultimate fallback: Hash of the hostname, or a loopback constant
	if hostname, err := getHostnameFunc(); err == nil && hostname != "" {
		var hash uint32 = 5381
		for i := 0; i < len(hostname); i++ {
			hash = ((hash << 5) + hash) + uint32(hostname[i])
		}
		return hash
	}

	// Loopback 127.0.1.1 equivalent -> 007f0101
	return 0x007f0101
}

// constructHostID builds the host ID by swapping the first two bytes and last two bytes.
func constructHostID(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return uint32(ip4[1])<<24 | uint32(ip4[0])<<16 | uint32(ip4[3])<<8 | uint32(ip4[2])
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, cwd string) int {
	// Parse jsonMode early in case parsing flags fails
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
			common.RenderError("hostid", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "hostid: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: hostid [options]\n\n" +
			"Print the numeric identifier (in hexadecimal) for the current host\n\n" +
			"Options:\n" +
			"  -h, --help     Print help\n" +
			"  -V, --version  Print version"
		common.Render("hostid", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	if flags.Has("V") || flags.Has("version") {
		versionText := "hostid version v1.1.0-goposix"
		common.Render("hostid", struct {
			Version string `json:"version"`
		}{Version: versionText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, versionText)
		})
		return 0
	}

	hid := GetHostID()
	result := HostidResult{
		Hostid: fmt.Sprintf("%08x", hid),
	}

	common.Render("hostid", result, jsonMode, stdout, func() {
		fmt.Fprintln(stdout, result.Hostid)
	})

	return 0
}
