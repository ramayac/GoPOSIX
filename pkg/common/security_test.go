package common

import (
	"testing"
)

func TestSecurePath(t *testing.T) {
	tests := []struct {
		target  string
		baseDir string
		wantErr bool
	}{
		{"test.txt", "/app/sandbox", false},
		{"./test.txt", "/app/sandbox", false},
		{"sub/dir/test.txt", "/app/sandbox", false},
		{"../sandbox/test.txt", "/app/sandbox", false},
		{"../../app/sandbox/test.txt", "/app/sandbox", false},

		{"../outside.txt", "/app/sandbox", true},
		{"../../etc/shadow", "/app/sandbox", true},
		{"/etc/shadow", "/app/sandbox", true},
		{"/app/sandbox_other", "/app/sandbox", true},

		{"/etc/shadow", "/", false},
		{"../../etc/shadow", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.target+"_in_"+tt.baseDir, func(t *testing.T) {
			_, err := SecurePath(tt.target, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurePath(%q, %q) error = %v, wantErr %v", tt.target, tt.baseDir, err, tt.wantErr)
			}
		})
	}
}
