// Package uuencode implements the POSIX-compliant uuencode utility.
package uuencode

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "m", Long: "base64", Type: common.FlagBool},
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "uuencode",
		Usage: "Encode a binary file",
		Run:   run,
	})
}

// UuencodeResult represents the JSON response structure.
type UuencodeResult struct {
	Source      string `json:"source"`
	RemoteName  string `json:"remoteName"`
	EncodedData string `json:"encodedData"`
	Format      string `json:"format"`
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
			common.RenderError("uuencode", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "uuencode: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: uuencode [-m] [INFILE] REMOTE_FILE\n\n" +
			"Encode INFILE (or stdin) to stdout.\n\n" +
			"Options:\n" +
			"  -m             Use Base64 encoding\n" +
			"  -h             Print help"
		common.Render("uuencode", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	pos := flags.Positional
	if len(pos) == 0 {
		if jsonMode {
			common.RenderError("uuencode", 1, "MISSING_ARGUMENT", "missing remote file name", true, stderr)
		} else {
			fmt.Fprintln(stderr, "uuencode: missing remote file name")
		}
		return 1
	}

	var infile string
	var remotename string
	var reader io.Reader
	mode := "666"

	if len(pos) == 1 {
		infile = "-"
		remotename = pos[0]
		reader = stdin
	} else {
		infile = pos[0]
		remotename = pos[1]
		if infile == "-" {
			reader = stdin
		} else {
			absInPath := infile
			if !filepath.IsAbs(absInPath) {
				absInPath = filepath.Join(cwd, infile)
			}
			file, err := os.Open(absInPath)
			if err != nil {
				if jsonMode {
					common.RenderError("uuencode", 1, "OPEN_ERROR", err.Error(), true, stderr)
				} else {
					fmt.Fprintf(stderr, "uuencode: %v\n", err)
				}
				return 1
			}
			defer file.Close()
			reader = file

			if info, err := file.Stat(); err == nil {
				mode = fmt.Sprintf("%o", info.Mode()&0777)
			}
		}
	}

	useBase64 := flags.Has("m") || flags.Has("base64")

	if jsonMode {
		// Read all data to encode into JSON
		data, err := io.ReadAll(reader)
		if err != nil {
			common.RenderError("uuencode", 1, "READ_ERROR", err.Error(), true, stderr)
			return 1
		}

		var encoded string
		var fmtStr string
		if useBase64 {
			encoded = base64.StdEncoding.EncodeToString(data)
			fmtStr = "base64"
		} else {
			encoded = encodeTraditionalString(data)
			fmtStr = "uuencode"
		}

		common.Render("uuencode", UuencodeResult{
			Source:      infile,
			RemoteName:  remotename,
			EncodedData: encoded,
			Format:      fmtStr,
		}, true, stdout, nil)
		return 0
	}

	// Text mode streaming
	if useBase64 {
		fmt.Fprintf(stdout, "begin-base64 %s %s\n", mode, remotename)
		// Standard Base64 streaming with 60 char lines
		buf := make([]byte, 45) // 45 bytes becomes 60 base64 chars
		for {
			n, err := io.ReadFull(reader, buf)
			if n > 0 {
				encoded := base64.StdEncoding.EncodeToString(buf[:n])
				fmt.Fprintln(stdout, encoded)
			}
			if err != nil {
				break
			}
		}
		fmt.Fprintln(stdout, "====")
		fmt.Fprintln(stdout, "end")
	} else {
		fmt.Fprintf(stdout, "begin %s %s\n", mode, remotename)
		buf := make([]byte, 45)
		for {
			n, err := io.ReadFull(reader, buf)
			if n > 0 {
				line := encodeTraditionalLine(buf[:n])
				fmt.Fprintln(stdout, line)
			}
			if err != nil {
				break
			}
		}
		fmt.Fprintln(stdout, "`")
		fmt.Fprintln(stdout, "end")
	}

	return 0
}

func uuenc(val byte) byte {
	v := (val & 63) + 32
	if v == 32 {
		return 96 // grave accent
	}
	return v
}

func encodeTraditionalLine(data []byte) string {
	n := len(data)
	res := make([]byte, 0, 1+((n+2)/3)*4)
	res = append(res, uuenc(byte(n)))

	for i := 0; i < n; i += 3 {
		var b1, b2, b3 byte
		b1 = data[i]
		if i+1 < n {
			b2 = data[i+1]
		}
		if i+2 < n {
			b3 = data[i+2]
		}

		c1 := b1 >> 2
		c2 := ((b1 & 3) << 4) | (b2 >> 4)
		c3 := ((b2 & 15) << 2) | (b3 >> 6)
		c4 := b3 & 63

		res = append(res, uuenc(c1), uuenc(c2), uuenc(c3), uuenc(c4))
	}
	return string(res)
}

func encodeTraditionalString(data []byte) string {
	var out string
	n := len(data)
	for i := 0; i < n; i += 45 {
		end := i + 45
		if end > n {
			end = n
		}
		out += encodeTraditionalLine(data[i:end]) + "\n"
	}
	out += "`\n"
	return out
}
