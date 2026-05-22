package factor

import (
	"bytes"
	"strings"
	"testing"
)

func TestFactorizeDirect(t *testing.T) {
	tests := []struct {
		n        uint64
		expected []uint64
	}{
		{0, nil},
		{1, nil},
		{2, []uint64{2}},
		{3, []uint64{3}},
		{4, []uint64{2, 2}},
		{10, []uint64{2, 5}},
		{1024, []uint64{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}},
		{2305843009213693951, []uint64{2305843009213693951}}, // M61 prime
		{6144867742934288163, []uint64{3, 37831, 37831, 37831, 37831}},
	}

	for _, tt := range tests {
		var factors []uint64
		factorize(tt.n, &factors)
		if len(factors) != len(tt.expected) {
			t.Errorf("factorize(%d): expected %v, got %v", tt.n, tt.expected, factors)
			continue
		}
		// Sort just in case
		for i, v := range factors {
			if v != tt.expected[i] {
				t.Errorf("factorize(%d): expected %v, got %v", tt.n, tt.expected, factors)
				break
			}
		}
	}
}

func TestFactorRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      string
		wantStdout string
		wantStderr string
		wantExit   int
	}{
		{
			name:       "Basic CLI arguments",
			args:       []string{"1024", "10"},
			wantStdout: "1024: 2 2 2 2 2 2 2 2 2 2\n10: 2 5\n",
			wantExit:   0,
		},
		{
			name:       "Leading space and plus sign",
			args:       []string{"  0", "+1", " +2"},
			wantStdout: "0:\n1:\n2: 2\n",
			wantExit:   0,
		},
		{
			name:       "Input via stdin",
			args:       []string{},
			stdin:      "1024\t10\n15   7 ",
			wantStdout: "1024: 2 2 2 2 2 2 2 2 2 2\n10: 2 5\n15: 3 5\n7: 7\n",
			wantExit:   0,
		},
		{
			name:       "Invalid number",
			args:       []string{"abc"},
			wantStderr: "factor: abc: invalid number\n",
			wantExit:   1,
		},
		{
			name:       "Negative number (treated as positional)",
			args:       []string{"-10"},
			wantStderr: "factor: -10: invalid number\n",
			wantExit:   1,
		},
		{
			name:       "Mixed valid and invalid",
			args:       []string{"10", "abc", "15"},
			wantStdout: "10: 2 5\n15: 3 5\n",
			wantStderr: "factor: abc: invalid number\n",
			wantExit:   1,
		},
		{
			name:       "Empty string arg",
			args:       []string{""},
			wantStderr: "factor: : invalid number\n",
			wantExit:   1,
		},
		{
			name:       "Overflow uint64",
			args:       []string{"18446744073709551616"},
			wantStderr: "factor: 18446744073709551616: invalid number\n",
			wantExit:   1,
		},
		{
			name:       "Help option",
			args:       []string{"-h"},
			wantStdout: "Usage: factor",
			wantExit:   0,
		},
		{
			name:       "Version option",
			args:       []string{"--version"},
			wantStdout: "factor version",
			wantExit:   0,
		},
		{
			name:       "JSON Mode valid",
			args:       []string{"--json", "10", "1"},
			wantStdout: `"input":"10"`,
			wantExit:   0,
		},
		{
			name:       "JSON Mode invalid",
			args:       []string{"--json", "abc"},
			wantStdout: `"error":"invalid number"`,
			wantExit:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tt.stdin)
			gotExit := run(tt.args, stdin, &stdout, &stderr, "")
			if gotExit != tt.wantExit {
				t.Errorf("run(%v) exit code: got %d, want %d", tt.args, gotExit, tt.wantExit)
			}
			gotStdout := stdout.String()
			gotStderr := stderr.String()

			if tt.wantStdout != "" && !strings.Contains(gotStdout, tt.wantStdout) {
				t.Errorf("run(%v) stdout: got %q, want containing %q", tt.args, gotStdout, tt.wantStdout)
			}
			if tt.wantStderr != "" && !strings.Contains(gotStderr, tt.wantStderr) {
				t.Errorf("run(%v) stderr: got %q, want containing %q", tt.args, gotStderr, tt.wantStderr)
			}
		})
	}
}

func TestPreProcessArgsError(t *testing.T) {
	// Simple test to exercise preProcessFactorArgs completely
	args := []string{"-h", "--json", "10", "--", "-5"}
	processed, err := preProcessFactorArgs(args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Check that "--" was preserved or inserted appropriately
	foundDashDash := false
	for _, arg := range processed {
		if arg == "--" {
			foundDashDash = true
		}
	}
	if !foundDashDash {
		t.Errorf("expected '--' in processed args: %v", processed)
	}
}
