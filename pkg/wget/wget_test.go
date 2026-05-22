package wget

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramayac/goposix/pkg/common"
)

func TestWget(t *testing.T) {
	// Set up local mock HTTP server
	mockContent := "Hello, GoPOSIX wget!"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(mockContent)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockContent))
	}))
	defer server.Close()

	// Helper to create clean temp directories
	tempDir, err := os.MkdirTemp("", "wget_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name       string
		args       []string
		wantExit   int
		checkFile  string // file to check for mockContent
		checkOut   string // expected content in stdout
		checkJSON  bool
		checkQuiet bool
	}{
		{
			name:      "Simple download, default filename",
			args:      []string{server.URL + "/index.html"},
			wantExit:  0,
			checkFile: "index.html",
		},
		{
			name:      "Custom output file with -O",
			args:      []string{"-O", "custom.txt", server.URL + "/somefile.txt"},
			wantExit:  0,
			checkFile: "custom.txt",
		},
		{
			name:      "Quiet mode -q",
			args:      []string{"-q", "-O", "quiet.txt", server.URL + "/foo"},
			wantExit:  0,
			checkFile: "quiet.txt",
		},
		{
			name:     "Stdout output with -O -",
			args:     []string{"-O", "-", server.URL + "/stdout"},
			wantExit: 0,
			checkOut: mockContent,
		},
		{
			name:      "Prefix directory with -P",
			args:      []string{"-P", "downloads", server.URL + "/bar.txt"},
			wantExit:  0,
			checkFile: filepath.Join("downloads", "bar.txt"),
		},
		{
			name:      "-O overrides -P",
			args:      []string{"-O", "overridden.txt", "-P", "downloads", server.URL + "/overridden.txt"},
			wantExit:  0,
			checkFile: "overridden.txt",
		},
		{
			name:      "JSON mode",
			args:      []string{"--json", "-O", "json_out.txt", server.URL + "/jsonfile"},
			wantExit:  0,
			checkFile: "json_out.txt",
			checkJSON: true,
		},
		{
			name:     "HTTP Error response",
			args:     []string{server.URL + "/error"},
			wantExit: 1,
		},
		{
			name:     "Invalid URL scheme",
			args:     []string{"ftp://invalid.com/file"},
			wantExit: 1,
		},
		{
			name:     "Host connection failure",
			args:     []string{"http://localhost:59999/nonexistent"},
			wantExit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a specific subdirectory inside tempDir to isolate each test execution
			testDir, err := os.MkdirTemp(tempDir, "sub_*")
			if err != nil {
				t.Fatalf("failed to create isolated test dir: %v", err)
			}

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			exitCode := run(tt.args, nil, stdout, stderr, testDir)
			if exitCode != tt.wantExit {
				t.Errorf("expected exit code %d, got %d. stderr: %s", tt.wantExit, exitCode, stderr.String())
			}

			if tt.wantExit == 0 {
				if tt.checkFile != "" {
					filePath := filepath.Join(testDir, tt.checkFile)
					content, err := os.ReadFile(filePath)
					if err != nil {
						t.Errorf("expected downloaded file at %q, got error: %v", filePath, err)
					} else if string(content) != mockContent {
						t.Errorf("expected file content %q, got %q", mockContent, string(content))
					}
				}

				if tt.checkOut != "" {
					if stdout.String() != tt.checkOut {
						t.Errorf("expected stdout %q, got %q", tt.checkOut, stdout.String())
					}
				}

				if tt.checkJSON {
					var env common.JSONEnvelope
					if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
						t.Fatalf("failed to unmarshal JSON: %v. Output: %s", err, stdout.String())
					}
					if env.ExitCode != 0 {
						t.Errorf("expected env.ExitCode 0, got %d", env.ExitCode)
					}
					dataMap, ok := env.Data.(map[string]interface{})
					if !ok {
						t.Fatalf("expected env.Data to be a map, got %T", env.Data)
					}
					if !strings.Contains(dataMap["url"].(string), "/jsonfile") {
						t.Errorf("expected URL to contain /jsonfile, got %v", dataMap["url"])
					}
					if dataMap["output_file"].(string) != "json_out.txt" {
						t.Errorf("expected output_file 'json_out.txt', got %v", dataMap["output_file"])
					}
					if int(dataMap["bytes_downloaded"].(float64)) != len(mockContent) {
						t.Errorf("expected %d bytes downloaded, got %v", len(mockContent), dataMap["bytes_downloaded"])
					}
					if int(dataMap["status_code"].(float64)) != http.StatusOK {
						t.Errorf("expected status_code 200, got %v", dataMap["status_code"])
					}
				}
			}
		})
	}
}
