// Package cksum implements the POSIX cksum utility — CRC-32 checksum and byte count.
package cksum

import (
	"fmt"
	"io"
	"os"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// CksumResult is the --json output.
type CksumResult struct {
	Files []CksumFileResult `json:"files"`
}

// CksumFileResult holds checksum info for one file.
type CksumFileResult struct {
	Name     string `json:"name,omitempty"`
	Checksum uint32 `json:"checksum"`
	Bytes    int64  `json:"bytes"`
}

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Long: "json", Type: common.FlagBool},
	},
}

// posixCRC computes the POSIX-standard CRC-32 for given data.
//
// Algorithm from IEEE Std 1003.1-2017:
// Polynomial: G(x) = x^32 + x^26 + x^23 + x^22 + x^16 + x^12 + x^11 +
//
//	x^10 + x^8 + x^7 + x^5 + x^4 + x^2 + x + 1
//
// (0x04C11DB7, non-reflected).
//
// 1. CRC of data bytes using bit-by-bit polynomial division
// 2. CRC of file length octets (LSB first, variable-length until zero)
// 3. Final XOR with 0xFFFFFFFF
func posixCRC(data []byte) uint32 {
	var crc uint32

	// Process data bytes
	for _, b := range data {
		crc ^= uint32(b) << 24
		for i := 0; i < 8; i++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ 0x04C11DB7
			} else {
				crc <<= 1
			}
		}
	}

	// Process length octets, LSB first, until zero
	n := len(data)
	for n != 0 {
		octet := uint32(n & 0xFF)
		crc ^= octet << 24
		for i := 0; i < 8; i++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ 0x04C11DB7
			} else {
				crc <<= 1
			}
		}
		n >>= 8
	}

	return crc ^ 0xFFFFFFFF
}

// computeCRC reads an entire reader and returns the POSIX CRC and byte count.
func computeCRC(r io.Reader) (uint32, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, 0, err
	}
	return posixCRC(data), int64(len(data)), nil
}

// Run computes POSIX CRC-32 checksums for the given files (or stdin).
func Run(files []string) (CksumResult, error) {
	var result CksumResult

	if len(files) == 0 {
		// Read from stdin
		crc, n, err := computeCRC(os.Stdin)
		if err != nil {
			return result, err
		}
		result.Files = append(result.Files, CksumFileResult{
			Checksum: crc,
			Bytes:    n,
		})
		return result, nil
	}

	for _, filename := range files {
		f, err := os.Open(filename)
		if err != nil {
			return result, err
		}
		crc, n, err := computeCRC(f)
		f.Close()
		if err != nil {
			return result, err
		}
		result.Files = append(result.Files, CksumFileResult{
			Name:     filename,
			Checksum: crc,
			Bytes:    n,
		})
	}

	return result, nil
}

func run(args []string, stdin io.Reader, stdout io.Writer) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cksum: %v\n", err)
		return 2
	}
	jsonMode := flags.Has("json")

	result, err := Run(flags.Positional)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cksum: %v\n", err)
		common.RenderError("cksum", 1, "ECKSUM", err.Error(), jsonMode, stdout)
		return 1
	}

	common.Render("cksum", result, jsonMode, stdout, func() {
		for _, f := range result.Files {
			if f.Name != "" {
				fmt.Fprintf(stdout, "%d %d %s\n", f.Checksum, f.Bytes, f.Name)
			} else {
				fmt.Fprintf(stdout, "%d %d\n", f.Checksum, f.Bytes)
			}
		}
	})
	return 0
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "cksum",
		Usage: "Print CRC checksum and byte count of files",
		Run:   run,
	})
}
