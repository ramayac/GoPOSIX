package hostid

import (
	"bytes"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetHostID_EtcHostid(t *testing.T) {
	// Setup temporary directory for mock /etc/hostid
	tmpDir, err := os.MkdirTemp("", "hostid-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockFile := filepath.Join(tmpDir, "hostid")
	expectedVal := uint32(0x12345678)
	buf := make([]byte, 4)
	nativeEndian.PutUint32(buf, expectedVal)

	if err := os.WriteFile(mockFile, buf, 0644); err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	// Backup and mock global etcHostidPath
	oldPath := etcHostidPath
	etcHostidPath = mockFile
	defer func() { etcHostidPath = oldPath }()

	val := GetHostID()
	if val != expectedVal {
		t.Errorf("expected 0x%08x from /etc/hostid, got 0x%08x", expectedVal, val)
	}
}

func TestGetHostID_HostnameResolution(t *testing.T) {
	// Backup mocks
	oldPath := etcHostidPath
	oldGetHostname := getHostnameFunc
	oldLookupIP := lookupIPFunc
	defer func() {
		etcHostidPath = oldPath
		getHostnameFunc = oldGetHostname
		lookupIPFunc = oldLookupIP
	}()

	// Point /etc/hostid to non-existent file
	etcHostidPath = "/non-existent/file/path"

	// Mock hostname to return "testhost"
	getHostnameFunc = func() (string, error) {
		return "testhost", nil
	}

	// Mock LookupIP to return 127.0.1.1
	lookupIPFunc = func(host string) ([]net.IP, error) {
		if host == "testhost" {
			return []net.IP{net.ParseIP("127.0.1.1")}, nil
		}
		return nil, errors.New("unknown host")
	}

	val := GetHostID()
	// ConstructHostID swaps: ip[1]<<24 | ip[0]<<16 | ip[3]<<8 | ip[2]
	// for [127, 0, 1, 1]: 0<<24 | 127<<16 | 1<<8 | 1 = 0x007f0101
	expectedVal := uint32(0x007f0101)
	if val != expectedVal {
		t.Errorf("expected 0x%08x, got 0x%08x", expectedVal, val)
	}
}

func TestGetHostID_InterfaceFallback(t *testing.T) {
	// Backup mocks
	oldPath := etcHostidPath
	oldGetHostname := getHostnameFunc
	oldLookupIP := lookupIPFunc
	oldInterfaceAddrs := interfaceAddrsFunc
	defer func() {
		etcHostidPath = oldPath
		getHostnameFunc = oldGetHostname
		lookupIPFunc = oldLookupIP
		interfaceAddrsFunc = oldInterfaceAddrs
	}()

	etcHostidPath = "/non-existent/file/path"
	getHostnameFunc = func() (string, error) {
		return "", errors.New("hostname failed")
	}

	// Mock interfaceAddrs to return a non-loopback address
	_, ipnet, _ := net.ParseCIDR("192.168.1.50/24")
	ipnet.IP = net.ParseIP("192.168.1.50")
	interfaceAddrsFunc = func() ([]net.Addr, error) {
		return []net.Addr{
			&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)}, // loopback
			ipnet, // 192.168.1.50
		}, nil
	}

	val := GetHostID()
	// For 192.168.1.50: ip[1]<<24 | ip[0]<<16 | ip[3]<<8 | ip[2]
	// = 168<<24 | 192<<16 | 50<<8 | 1 = 0xa8c03201
	expectedVal := uint32(0xa8c03201)
	if val != expectedVal {
		t.Errorf("expected 0xa8c03201, got 0x%08x", val)
	}
}

func TestGetHostID_HashFallback(t *testing.T) {
	// Backup mocks
	oldPath := etcHostidPath
	oldGetHostname := getHostnameFunc
	oldLookupIP := lookupIPFunc
	oldInterfaceAddrs := interfaceAddrsFunc
	defer func() {
		etcHostidPath = oldPath
		getHostnameFunc = oldGetHostname
		lookupIPFunc = oldLookupIP
		interfaceAddrsFunc = oldInterfaceAddrs
	}()

	etcHostidPath = "/non-existent/file/path"
	getHostnameFunc = func() (string, error) {
		return "fallback-host", nil
	}
	lookupIPFunc = func(host string) ([]net.IP, error) {
		return nil, errors.New("lookup failed")
	}
	interfaceAddrsFunc = func() ([]net.Addr, error) {
		return nil, errors.New("interface lookup failed")
	}

	// Hash function DJB2:
	var expectedHash uint32 = 5381
	hostStr := "fallback-host"
	for i := 0; i < len(hostStr); i++ {
		expectedHash = ((expectedHash << 5) + expectedHash) + uint32(hostStr[i])
	}

	val := GetHostID()
	if val != expectedHash {
		t.Errorf("expected hash 0x%08x, got 0x%08x", expectedHash, val)
	}
}

func TestGetHostID_AbsoluteFallback(t *testing.T) {
	// Backup mocks
	oldPath := etcHostidPath
	oldGetHostname := getHostnameFunc
	oldLookupIP := lookupIPFunc
	oldInterfaceAddrs := interfaceAddrsFunc
	defer func() {
		etcHostidPath = oldPath
		getHostnameFunc = oldGetHostname
		lookupIPFunc = oldLookupIP
		interfaceAddrsFunc = oldInterfaceAddrs
	}()

	etcHostidPath = "/non-existent/file/path"
	getHostnameFunc = func() (string, error) {
		return "", errors.New("failed")
	}
	interfaceAddrsFunc = func() ([]net.Addr, error) {
		return nil, errors.New("failed")
	}

	val := GetHostID()
	expectedVal := uint32(0x007f0101)
	if val != expectedVal {
		t.Errorf("expected absolute fallback 0x007f0101, got 0x%08x", val)
	}
}

func TestConstructHostID_NilIP(t *testing.T) {
	val := constructHostID(nil)
	if val != 0 {
		t.Errorf("expected 0 for nil IP, got %d", val)
	}
}

func TestHostidRun_Standard(t *testing.T) {
	// Backup mocks
	oldPath := etcHostidPath
	defer func() { etcHostidPath = oldPath }()

	// Write mock file to force consistent output
	tmpDir, err := os.MkdirTemp("", "hostid-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockFile := filepath.Join(tmpDir, "hostid")
	buf := make([]byte, 4)
	nativeEndian.PutUint32(buf, 0xaabbccdd)
	if err := os.WriteFile(mockFile, buf, 0644); err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}
	etcHostidPath = mockFile

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{}, nil, stdout, stderr, "")

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	expectedOut := "aabbccdd\n"
	if stdout.String() != expectedOut {
		t.Errorf("expected output %q, got %q", expectedOut, stdout.String())
	}
}

func TestHostidRun_Json(t *testing.T) {
	// Backup mocks
	oldPath := etcHostidPath
	defer func() { etcHostidPath = oldPath }()

	// Write mock file to force consistent output
	tmpDir, err := os.MkdirTemp("", "hostid-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockFile := filepath.Join(tmpDir, "hostid")
	buf := make([]byte, 4)
	nativeEndian.PutUint32(buf, 0x007f0101)
	if err := os.WriteFile(mockFile, buf, 0644); err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}
	etcHostidPath = mockFile

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"--json"}, nil, stdout, stderr, "")

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, `"command":"hostid"`) || !strings.Contains(outStr, `"hostid":"007f0101"`) {
		t.Errorf("JSON output did not contain expected elements. Got: %s", outStr)
	}
}

func TestHostidRun_HelpAndVersion(t *testing.T) {
	// Test --help
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"--help"}, nil, stdout, stderr, "")
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for help, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected help text, got %q", stdout.String())
	}

	// Test --version
	stdout.Reset()
	stderr.Reset()
	exitCode = run([]string{"--version"}, nil, stdout, stderr, "")
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for version, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "version") {
		t.Errorf("expected version text, got %q", stdout.String())
	}

	// Test invalid flags
	stdout.Reset()
	stderr.Reset()
	exitCode = run([]string{"--invalid-flag"}, nil, stdout, stderr, "")
	if exitCode == 0 {
		t.Error("expected non-zero exit code for invalid flags")
	}
}
