package uptime

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/ramayac/goposix/pkg/common"
)

func TestUptime(t *testing.T) {
	// Backup original readers
	origUptime := readProcUptime
	origLoadavg := readProcLoadavg
	defer func() {
		readProcUptime = origUptime
		readProcLoadavg = origLoadavg
	}()

	tests := []struct {
		name        string
		uptimeData  string
		loadavgData string
		mockUsers   int
		args        []string
		wantExit    int
		wantStdout  string
		checkJSON   bool
		jsonUptime  float64
		jsonLoad1m  float64
		jsonLoad5m  float64
		jsonLoad15m float64
		jsonUsers   int
	}{
		{
			name:        "Uptime in minutes, singular user",
			uptimeData:  "32.45 120.30",
			loadavgData: "0.46 0.61 0.61 1/141 12345",
			mockUsers:   1,
			args:        []string{},
			wantExit:    0,
			wantStdout:  "up 0 min,  1 user,  load average: 0.46, 0.61, 0.61",
		},
		{
			name:        "Uptime in hours, plural users",
			uptimeData:  "7300.00 24000.00", // 2 hours, 1 minute, 40 seconds
			loadavgData: "0.12 0.25 0.18 2/150 12346",
			mockUsers:   3,
			args:        []string{},
			wantExit:    0,
			wantStdout:  "up  2:01,  3 users,  load average: 0.12, 0.25, 0.18",
		},
		{
			name:        "Uptime in days (singular)",
			uptimeData:  "90000.00 300000.00", // 1 day, 1 hour, 0 minutes
			loadavgData: "0.00 0.01 0.05 1/120 12347",
			mockUsers:   0,
			args:        []string{},
			wantExit:    0,
			wantStdout:  "up 1 day,  1:00,  0 users,  load average: 0.00, 0.01, 0.05",
		},
		{
			name:        "Uptime in days (plural)",
			uptimeData:  "262800.00 500000.00", // 3 days, 1 hour, 0 minutes
			loadavgData: "1.23 2.34 3.45 4/200 12348",
			mockUsers:   5,
			args:        []string{},
			wantExit:    0,
			wantStdout:  "up 3 days,  1:00,  5 users,  load average: 1.23, 2.34, 3.45",
		},
		{
			name:        "JSON mode",
			uptimeData:  "7300.00 24000.00",
			loadavgData: "0.12 0.25 0.18 2/150 12346",
			mockUsers:   3,
			args:        []string{"--json"},
			wantExit:    0,
			checkJSON:   true,
			jsonUptime:  7300.0,
			jsonLoad1m:  0.12,
			jsonLoad5m:  0.25,
			jsonLoad15m: 0.18,
			jsonUsers:   3,
		},
		{
			name:        "Missing proc files fallback gracefully",
			uptimeData:  "",
			loadavgData: "",
			mockUsers:   0,
			args:        []string{},
			wantExit:    0, // Usually exit 0 even with missing files as standard uptime prints defaults or zeroes
			wantStdout:  "up 0 min,  0 users,  load average: 0.00, 0.00, 0.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock readers
			if tt.uptimeData != "" {
				readProcUptime = func() ([]byte, error) {
					return []byte(tt.uptimeData), nil
				}
			} else {
				readProcUptime = func() ([]byte, error) {
					return nil, errors.New("file not found") // mock missing file
				}
			}

			if tt.loadavgData != "" {
				readProcLoadavg = func() ([]byte, error) {
					return []byte(tt.loadavgData), nil
				}
			} else {
				readProcLoadavg = func() ([]byte, error) {
					return nil, errors.New("file not found") // mock missing file
				}
			}

			// Mock who.Run user count
			mockUserCount = tt.mockUsers

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			exitCode := run(tt.args, nil, stdout, stderr, "")
			if exitCode != tt.wantExit {
				t.Errorf("expected exit code %d, got %d. stderr: %s", tt.wantExit, exitCode, stderr.String())
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
				if dataMap["uptime"].(float64) != tt.jsonUptime {
					t.Errorf("expected uptime %f, got %f", tt.jsonUptime, dataMap["uptime"])
				}
				if dataMap["load_1m"].(float64) != tt.jsonLoad1m {
					t.Errorf("expected load_1m %f, got %f", tt.jsonLoad1m, dataMap["load_1m"])
				}
				if dataMap["load_5m"].(float64) != tt.jsonLoad5m {
					t.Errorf("expected load_5m %f, got %f", tt.jsonLoad5m, dataMap["load_5m"])
				}
				if dataMap["load_15m"].(float64) != tt.jsonLoad15m {
					t.Errorf("expected load_15m %f, got %f", tt.jsonLoad15m, dataMap["load_15m"])
				}
				if int(dataMap["users"].(float64)) != tt.jsonUsers {
					t.Errorf("expected users %d, got %v", tt.jsonUsers, dataMap["users"])
				}
			} else {
				outStr := stdout.String()
				// Should contain hh:mm:ss up ... load average: ...
				// Since hh:mm:ss changes depending on the execution time, we check if stdout has the expected suffix
				if !strings.Contains(outStr, tt.wantStdout) {
					t.Errorf("expected stdout to contain %q, got %q", tt.wantStdout, outStr)
				}
			}
		})
	}
}
