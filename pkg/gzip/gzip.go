package gzip

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// GzipStat holds the result of compressing or decompressing a single file.
type GzipStat struct {
	File         string  `json:"file"`
	OriginalSize int64   `json:"originalSize"`
	NewSize      int64   `json:"newSize"`
	Ratio        float64 `json:"ratio"`
	// Content is the base64-encoded processed payload. It is set only in
	// JSON stdout mode (-c or stdin): the payload cannot share stdout with
	// the envelope, so it travels inside the envelope instead (F16).
	Content string `json:"content,omitempty"`
}

// jsonStreamSink returns the writer for the processed payload. In JSON mode
// the payload is captured into buf so stdout carries only the envelope.
func jsonStreamSink(isJSON bool, stdout io.Writer, buf *bytes.Buffer) io.Writer {
	if isJSON {
		return &common.LimitWriter{W: buf, Limit: 50 * 1024 * 1024}
	}
	return stdout
}

// newWriterLevel is a seam for tests. getCompressionLevel only produces
// valid levels, so the real gzip.NewWriterLevel can never fail at the call
// sites; the seam makes the defensive branch testable.
var newWriterLevel = gzip.NewWriterLevel

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "d", Long: "decompress", Type: common.FlagBool},
		{Short: "c", Long: "stdout", Type: common.FlagBool},
		{Short: "k", Long: "keep", Type: common.FlagBool},
		{Short: "f", Long: "force", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
		// Numeric compression levels -1 through -9 (boolean-like for presence detection).
		{Short: "1", Long: "", Type: common.FlagBool},
		{Short: "2", Long: "", Type: common.FlagBool},
		{Short: "3", Long: "", Type: common.FlagBool},
		{Short: "4", Long: "", Type: common.FlagBool},
		{Short: "5", Long: "", Type: common.FlagBool},
		{Short: "6", Long: "", Type: common.FlagBool},
		{Short: "7", Long: "", Type: common.FlagBool},
		{Short: "8", Long: "", Type: common.FlagBool},
		{Short: "9", Long: "", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "gzip",
		Usage: "gzip [FILE]...",
		Run:   runGzip,
	})
	dispatch.Register(dispatch.Command{
		Name:  "gunzip",
		Usage: "gunzip [FILE]...",
		Run:   runGunzip,
	})
}

func runGunzip(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return gunzipRun(args, stdout, stderr, stdin, cwd)
}

func runGzip(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	return gzipRun(args, stdout, stderr, stdin, cwd)
}

func gunzipRun(args []string, stdout io.Writer, errOut io.Writer, stdin io.Reader, cwd string) int {
	return execute(args, stdout, errOut, stdin, true, "gunzip")
}

func gzipRun(args []string, stdout io.Writer, errOut io.Writer, stdin io.Reader, cwd string) int {
	return execute(args, stdout, errOut, stdin, false, "gzip")
}

// getCompressionLevel returns the compression level from flags, or flate.DefaultCompression.
func getCompressionLevel(flags *common.ParseResult) int {
	// Check for numeric compression level flags (highest priority last).
	for i := 9; i >= 1; i-- {
		if flags.Has(fmt.Sprintf("%d", i)) {
			return i
		}
	}
	return flate.DefaultCompression
}

func execute(args []string, stdout io.Writer, errOut io.Writer, stdin io.Reader, forceDecompress bool, cmdName string) int {
	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		return common.RenderFlagError(cmdName, args, err, errOut, 1)
	}

	isJSON := flags.Has("json")
	toStdout := flags.Has("c")
	keepOriginal := flags.Has("k") || toStdout
	force := flags.Has("f")
	decompress := forceDecompress || flags.Has("d")
	level := getCompressionLevel(flags)

	files := flags.Positional

	if len(files) == 0 {
		var payloadBuf bytes.Buffer
		target := jsonStreamSink(isJSON, stdout, &payloadBuf)
		if decompress {
			gr, err := gzip.NewReader(stdin)
			if err != nil {
				if !isJSON {
					fmt.Fprintf(errOut, cmdName+": stdin: %v\n", err)
				}
				return 1
			}
			io.Copy(target, gr)
			gr.Close()
		} else {
			gw, err := newWriterLevel(target, level)
			if err != nil {
				if !isJSON {
					fmt.Fprintf(errOut, cmdName+": %v\n", err)
				}
				return 1
			}
			io.Copy(gw, stdin)
			gw.Close()
		}
		if isJSON {
			common.Render(cmdName, []GzipStat{{
				File:    "-",
				NewSize: int64(payloadBuf.Len()),
				Content: base64.StdEncoding.EncodeToString(payloadBuf.Bytes()),
			}}, true, stdout, nil)
		}
		return 0
	}

	exitCode := 0
	var stats []GzipStat

	for _, file := range files {
		// Handle stdin/stdout via "-".
		if file == "-" {
			var payloadBuf bytes.Buffer
			target := jsonStreamSink(isJSON, stdout, &payloadBuf)
			if decompress {
				gr, err := gzip.NewReader(stdin)
				if err != nil {
					if !isJSON {
						fmt.Fprintf(errOut, cmdName+": stdin: %v\n", err)
					}
					return 1
				}
				io.Copy(target, gr)
				gr.Close()
			} else {
				gw, err := newWriterLevel(target, level)
				if err != nil {
					if !isJSON {
						fmt.Fprintf(errOut, cmdName+": %v\n", err)
					}
					return 1
				}
				io.Copy(gw, stdin)
				gw.Close()
			}
			if isJSON {
				stats = append(stats, GzipStat{
					File:    "-",
					NewSize: int64(payloadBuf.Len()),
					Content: base64.StdEncoding.EncodeToString(payloadBuf.Bytes()),
				})
			}
			continue
		}

		inInfo, err := os.Stat(file)
		if err != nil {
			if decompress {
				// gunzip: file doesn't exist
				if !isJSON {
					fmt.Fprintf(errOut, "%s: %s: No such file or directory\n", cmdName, file)
				}
				exitCode = 1
				continue
			}
			common.RenderError(cmdName, 1, "STAT_FAIL", err.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, "%s: %v\n", cmdName, err)
			}
			exitCode = 1
			continue
		}

		// gunzip: skip non-.gz files (check suffix before trying to open)
		if decompress && !strings.HasSuffix(file, ".gz") {
			if !isJSON {
				fmt.Fprintf(errOut, "%s: %s: unknown suffix - ignored\n", cmdName, file)
			}
			exitCode = 1
			continue
		}

		in, err := os.Open(file)
		if err != nil {
			common.RenderError("gzip", 1, "OPEN_FAIL", err.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, cmdName+": %v\n", err)
			}
			exitCode = 1
			continue
		}

		var targetWriter io.Writer
		var outFile *os.File
		var outName string
		var payloadBuf bytes.Buffer

		if toStdout {
			targetWriter = jsonStreamSink(isJSON, stdout, &payloadBuf)
		} else {
			if decompress {
				outName = strings.TrimSuffix(file, ".gz")
				if outName == file {
					outName = file + ".unpacked"
				}
			} else {
				outName = file + ".gz"
			}

			if !force {
				if _, err := os.Stat(outName); err == nil {
					if decompress {
						if !isJSON {
							fmt.Fprintf(errOut, "%s: can't open '%s': File exists\n", cmdName, outName)
						}
					} else {
						if !isJSON {
							fmt.Fprintf(errOut, "%s: %s already exists; use -f to overwrite\n", cmdName, outName)
						}
					}
					in.Close()
					exitCode = 1
					continue
				}
			}

			outFile, err = os.OpenFile(outName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, inInfo.Mode())
			if err != nil {
				common.RenderError("gzip", 1, "CREATE_FAIL", err.Error(), isJSON, stdout)
				if !isJSON {
					fmt.Fprintf(errOut, cmdName+": %v\n", err)
				}
				in.Close()
				exitCode = 1
				continue
			}
			targetWriter = outFile
		}

		var processErr error
		if decompress {
			gr, err := gzip.NewReader(in)
			if err != nil {
				processErr = err
			} else {
				_, processErr = io.Copy(targetWriter, gr)
				gr.Close()
			}
		} else {
			gw, err := newWriterLevel(targetWriter, level)
			if err != nil {
				processErr = err
			} else {
				_, processErr = io.Copy(gw, in)
				gw.Close()
			}
		}

		if outFile != nil {
			outFile.Close()
		}
		in.Close()

		if processErr != nil {
			common.RenderError("gzip", 1, "PROCESS_FAIL", processErr.Error(), isJSON, stdout)
			if !isJSON {
				fmt.Fprintf(errOut, cmdName+": %v\n", processErr)
			}
			if outFile != nil {
				os.Remove(outName) // remove incomplete output
			}
			exitCode = 1
			continue
		}

		var outSize int64
		if toStdout && isJSON {
			outSize = int64(payloadBuf.Len())
		} else if !toStdout && outFile != nil {
			if outInfo, err := os.Stat(outName); err == nil {
				outSize = outInfo.Size()
			}
			if !keepOriginal {
				os.Remove(file)
			}
		}

		inSize := inInfo.Size()
		var ratio float64
		if inSize > 0 && outSize > 0 {
			if decompress {
				ratio = float64(outSize) / float64(inSize)
			} else {
				ratio = float64(inSize) / float64(outSize)
			}
		}

		stat := GzipStat{
			File:         file,
			OriginalSize: inSize,
			NewSize:      outSize,
			Ratio:        ratio,
		}
		if toStdout && isJSON {
			stat.Content = base64.StdEncoding.EncodeToString(payloadBuf.Bytes())
		}
		stats = append(stats, stat)
	}

	if isJSON {
		common.Render("gzip", stats, true, stdout, func() {})
	}

	return exitCode
}
