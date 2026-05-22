// Package dd implements the POSIX dd utility — convert and copy a file.
package dd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
)

// Run is the library-layer entry point for dd. It reads from in, writes to stdout,
// and writes status information to statusW. args contains key=value operands.
// Returns a POSIX exit code (0 on success, 1 on write error, 2 on usage error).
func Run(args []string, in io.Reader, stdout io.Writer, statusW io.Writer) int {
	opts := parseOperands(args)

	// Resolve input
	var input io.Reader
	if opts.ifile != "" {
		f, err := os.Open(opts.ifile)
		if err != nil {
			fmt.Fprintf(statusW, "dd: %s: %v\n", opts.ifile, err)
			return 1
		}
		defer f.Close()
		input = f
	} else {
		input = in
	}

	// Resolve output
	var output io.Writer
	var outFile *os.File
	if opts.ofile != "" {
		flag := os.O_WRONLY | os.O_CREATE
		if opts.convNotrunc {
			flag |= os.O_APPEND
		} else {
			flag |= os.O_TRUNC
		}
		f, err := os.OpenFile(opts.ofile, flag, 0666)
		if err != nil {
			fmt.Fprintf(statusW, "dd: %s: %v\n", opts.ofile, err)
			return 1
		}
		defer f.Close()
		outFile = f
		output = f
	} else {
		output = stdout
	}

	// Determine block sizes
	ibs := opts.ibs
	obs := opts.obs
	if opts.bs > 0 {
		ibs = opts.bs
		obs = opts.bs
	}
	if ibs <= 0 {
		ibs = 512
	}
	if obs <= 0 {
		obs = 512
	}

	// Skip input blocks
	if opts.skip > 0 {
		skipBuf := make([]byte, ibs)
		for i := int64(0); i < opts.skip; i++ {
			_, err := io.ReadFull(input, skipBuf)
			if err != nil {
				break
			}
		}
	}

	// Seek output blocks (only for seekable files)
	if opts.seek > 0 && outFile != nil {
		outFile.Seek(opts.seek*obs, io.SeekStart)
	}

	buf := make([]byte, ibs)
	var inFull, inPart, outFull, outPart int64
	var totalIn, totalOut int64
	writeErr := false

	countRemaining := opts.count
	countBytes := opts.iflagCountBytes
	maxBytes := int64(-1)
	if countBytes && countRemaining > 0 {
		maxBytes = countRemaining
		countRemaining = 0
	}

	for {
		// Determine how many bytes to read
		readLen := int64(ibs)
		if maxBytes >= 0 {
			remaining := maxBytes - totalIn
			if remaining <= 0 {
				break
			}
			if remaining < readLen {
				readLen = remaining
			}
		}

		n, err := io.ReadAtLeast(input, buf[:readLen], 1)
		if err == io.ErrUnexpectedEOF {
			// Short read — handle normally
			err = nil
		}
		if n == 0 && err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			if opts.convNoerror {
				// Write what we have, continue
				if n > 0 {
					// Treat as full block with sync padding
					inFull++
				}
			} else {
				fmt.Fprintf(statusW, "dd: read error: %v\n", err)
				return 1
			}
		}

		if n > 0 {
			totalIn += int64(n)
			if int64(n) == ibs {
				inFull++
			} else {
				inPart++
			}

			// Apply conv=sync — pad partial block to obs with NULs
			outBlock := buf[:n]
			if opts.convSync && int64(n) < obs {
				padded := make([]byte, obs)
				copy(padded, buf[:n])
				outBlock = padded
			}

			writtenTotal := 0
			for writtenTotal < len(outBlock) {
				chunk := obs
				if chunk > int64(len(outBlock)-writtenTotal) {
					chunk = int64(len(outBlock) - writtenTotal)
				}
				wn, werr := output.Write(outBlock[writtenTotal : writtenTotal+int(chunk)])
				if wn > 0 {
					writtenTotal += wn
					totalOut += int64(wn)
				}
				if werr != nil {
					writeErr = true
					break
				}
			}

			if writeErr {
				break
			}

			if int64(len(outBlock)) == obs {
				outFull++
			} else {
				outPart++
			}

			// Decrement block count
			if !countBytes && countRemaining > 0 {
				countRemaining--
				if countRemaining <= 0 {
					break
				}
			}
		}

		if err == io.EOF {
			break
		}
	}

	// conv=fsync
	if opts.convFsync && outFile != nil {
		outFile.Sync()
	}

	// Status report to stderr
	if opts.status != "none" {
		fmt.Fprintf(statusW, "%d+%d records in\n", inFull, inPart)
		fmt.Fprintf(statusW, "%d+%d records stdout\n", outFull, outPart)
		if opts.status != "noxfer" && totalOut > 0 {
			fmt.Fprintf(statusW, "%d bytes copied\n", totalOut)
		}
	}

	if writeErr {
		return 1
	}
	return 0
}

// ddOpts holds parsed dd operands.
type ddOpts struct {
	ifile  string
	ofile  string
	bs     int64
	ibs    int64
	obs    int64
	count  int64
	skip   int64
	seek   int64
	status string // none, noxfer, or empty

	convSync        bool
	convNoerror     bool
	convNotrunc     bool
	convFsync       bool
	iflagCountBytes bool
}

func parseOperands(args []string) ddOpts {
	var opts ddOpts
	for _, a := range args {
		eq := strings.IndexByte(a, '=')
		if eq < 0 {
			continue
		}
		key := a[:eq]
		val := a[eq+1:]

		switch key {
		case "if":
			opts.ifile = val
		case "of":
			opts.ofile = val
		case "bs":
			opts.bs, _ = parseInt64(val)
		case "ibs":
			opts.ibs, _ = parseInt64(val)
		case "obs":
			opts.obs, _ = parseInt64(val)
		case "count":
			opts.count, _ = parseInt64(val)
		case "skip":
			opts.skip, _ = parseInt64(val)
		case "seek":
			opts.seek, _ = parseInt64(val)
		case "status":
			opts.status = val
		case "conv":
			for _, c := range strings.Split(val, ",") {
				switch strings.TrimSpace(c) {
				case "sync":
					opts.convSync = true
				case "noerror":
					opts.convNoerror = true
				case "notrunc":
					opts.convNotrunc = true
				case "fsync":
					opts.convFsync = true
				}
			}
		case "iflag":
			for _, c := range strings.Split(val, ",") {
				switch strings.TrimSpace(c) {
				case "count_bytes":
					opts.iflagCountBytes = true
				}
			}
		}
	}
	return opts
}

func parseInt64(s string) (int64, error) {
	// Handle hex and octal suffixes
	s = strings.TrimSpace(s)
	mult := int64(1)
	if len(s) > 1 {
		switch s[len(s)-1] {
		case 'b', 'B':
			// 512-byte blocks (POSIX cbs)
			mult = 512
			s = s[:len(s)-1]
		case 'k', 'K':
			mult = 1024
			s = s[:len(s)-1]
		case 'm', 'M':
			mult = 1024 * 1024
			s = s[:len(s)-1]
		case 'w', 'W':
			mult = 2
			s = s[:len(s)-1]
		case 'x':
			// hex number without the 0x prefix
			s = s[:len(s)-1]
			v, err := strconv.ParseInt(s, 16, 64)
			return v, err
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	return v * mult, err
}

// ---------------------------------------------------------------------------
// CLI glue
// ---------------------------------------------------------------------------

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return ddRun(args, stdin, stdout, stderr, cwd)
}

// ddRun is the testable entry point for the dd CLI.
func ddRun(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return Run(args, stdin, stdout, stderr)
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "dd",
		Usage: "dd [OPERAND]... — convert and copy a file",
		Run:   run,
	})
}
