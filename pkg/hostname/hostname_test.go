package hostname

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunReturnsHostname(t *testing.T) {
	result, err := Run(false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name == "" {
		t.Error("expected non-empty hostname")
	}
}

func TestRunCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRunCLIJSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Hostname_DomainFlag(t *testing.T) {
	t.Skip("environment-dependent: host may not have a DNS domain configured")

	// BusyBox: hostname -d returns the domain
	result, err := Run(false, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Domain may be "(none)" if not resolvable, or a real domain
	if result.Domain == "" {
		t.Error("expected non-empty domain (even if (none))")
	}
}

func TestBusyBox_Hostname_FQDNFlag(t *testing.T) {
	// BusyBox: hostname -f returns the FQDN
	result, err := Run(false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FQDN == "" {
		t.Error("expected non-empty fqdn")
	}
}

func TestBusyBox_Hostname_ShortFlag(t *testing.T) {
	// BusyBox: hostname -s returns the short hostname
	result, err := Run(true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name == "" {
		t.Error("expected non-empty hostname")
	}
}

func TestBusyBox_Hostname_DomainCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-d"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if out == "" {
		t.Error("expected domain output")
	}
}

func TestBusyBox_Hostname_FQDNCLI(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-f"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if out == "" {
		t.Error("expected fqdn output")
	}
}

func TestRunCLI_Domain(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-d"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected domain output")
	}
}

func TestRunCLI_IP(t *testing.T) {
	t.Skip("hostname -i uses DNS lookup not always available")
}

func TestRunCLI_JSON(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, nil, &buf, &buf, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(buf.String(), `"hostname"`) {
		t.Errorf("expected JSON with 'hostname', got %q", buf.String())
	}
}

func TestRunCLI_BadFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &buf, &buf, "")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestHostnameSetShort(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-s", "newhost"}, nil, &buf, &buf, "")
	// Will likely fail without root, but exercises the set-hostname code path
	if code == 0 {
		t.Log("set hostname succeeded (running as root?)")
	}
}
func TestHostnameSetFQDN(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-F", "/etc/hostname"}, nil, &buf, &buf, "")
	// Exercises file-based hostname setting
	_ = code
}
