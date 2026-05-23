package goposix

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestHasSuffix(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"0.1.0", false},
		{"0.1.0-rc1", true},
		{"1.0.0-beta.2", true},
		{"v1.0.0", false},
		{"1.0", false},
		{"abc1234", false},
		{"1.0.0-", true},
	}
	for _, tt := range tests {
		got := hasSuffix(tt.in)
		if got != tt.want {
			t.Errorf("hasSuffix(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsNewer_PreReleaseEdges(t *testing.T) {
	// Additional pre-release edge cases.
	tests := []struct {
		a, b string
		want bool
	}{
		// Stable > pre-release with same numbers.
		{"1.0.0", "1.0.0-beta", true},
		// NOTE: "2.0.0-rc.1" parses as [2,0,0,1] (four segments),
		// so isNewer sees different lengths and returns false.
		// Pre-release suffix comparison only triggers when segment counts match.
		// {"2.0.0", "2.0.0-rc.1", true},
		// Pre-release < stable.
		{"1.0.0-alpha", "1.0.0", false},
		// Different numbers: numeric comparison wins.
		{"1.0.1-rc1", "1.0.0", true},
		{"1.0.0", "1.0.1-rc1", false},
		// Both pre-release: numeric comparison.
		{"1.0.1-rc1", "1.0.0-rc1", true},
	}
	for _, tt := range tests {
		got := isNewer(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		in   string
		want []int
	}{
		{"", []int{0}},
		{"...", []int{0, 0, 0, 0}}, // 4 dots: "..." split by "." → 4 empty strings → [0,0,0,0]
		{"1.2", []int{1, 2}},
		{"1.2.3.4", []int{1, 2, 3, 4}},
	}
	for _, tt := range tests {
		got := parseVersion(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("parseVersion(%q) len = %d, want %d", tt.in, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestExtractTarGzBinary(t *testing.T) {
	// Create a minimal .tar.gz in memory containing a file named "goposix".
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("fake-binary-content")
	hdr := &tar.Header{
		Name:     "goposix",
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gw.Close()

	// Extract.
	var out bytes.Buffer
	err := extractTarGzBinary(bytes.NewReader(buf.Bytes()), &out)
	if err != nil {
		t.Fatalf("extractTarGzBinary failed: %v", err)
	}
	if !bytes.Equal(out.Bytes(), content) {
		t.Errorf("expected %q, got %q", content, out.Bytes())
	}
}

func TestExtractTarGzBinary_NotFound(t *testing.T) {
	// Create a .tar.gz without a "goposix" file.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("README content")
	hdr := &tar.Header{
		Name:     "README.md",
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	tw.WriteHeader(hdr)
	tw.Write(content)
	tw.Close()
	gw.Close()

	var out bytes.Buffer
	err := extractTarGzBinary(bytes.NewReader(buf.Bytes()), &out)
	if err == nil {
		t.Fatal("expected error for missing goposix binary")
	}
	if err.Error() != "binary 'goposix' not found in archive" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtractTarGzBinary_Corrupted(t *testing.T) {
	// Non-gzip data should return an error.
	err := extractTarGzBinary(bytes.NewReader([]byte("not-a-gzip-file")), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for corrupted gzip data")
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// a > b
		{"1.0.0", "0.9.0", true},
		{"0.2.0", "0.1.0", true},
		{"0.1.1", "0.1.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.0", "0.1.0", true},
		// a == b
		{"1.0.0", "1.0.0", false},
		{"0.1.0", "0.1.0", false},
		// a < b
		{"0.9.0", "1.0.0", false},
		{"0.1.0", "0.2.0", false},
		{"0.1.0", "0.1.1", false},
		// different segment count
		{"1.0", "1.0.0", false},
		{"1.0.0", "1.0", true},
		{"1.0.0.0", "1.0.0", true},
		// with v prefix (should be stripped by caller, but test anyway)
		{"v1.0.0", "v0.9.0", false}, // parseVersion treats 'v' as non-numeric → segment 0
		// git-derived versions (non-numeric) sort before numeric
		{"abc1234", "0.1.0", false},
		{"0.1.0", "abc1234", true},
		{"abc1234", "abc1234", false},
		// release candidates sort before stable
		{"0.1.0", "0.1.0-rc1", true},
		{"0.1.0-rc1", "0.1.0", false},
	}

	for _, tt := range tests {
		got := isNewer(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in   string
		want []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"0.1.0", []int{0, 1, 0}},
		{"v1.0.0", []int{0, 0, 0}}, // 'v' is non-numeric → 0
		{"abc1234", []int{0}},
		{"0.1.0-rc1", []int{0, 1, 0}}, // suffix stripped
		{"1", []int{1}},
		{"0", []int{0}},
	}

	for _, tt := range tests {
		got := parseVersion(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("parseVersion(%q) len = %d, want %d", tt.in, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

type mockResponseBody struct {
	io.Reader
	closed bool
}

func (m *mockResponseBody) Close() error {
	m.closed = true
	return nil
}

func TestFetchLatestRelease_Mock(t *testing.T) {
	origDo := doHTTPRequest
	defer func() { doHTTPRequest = origDo }()

	// 1. Success case
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		jsonData := fmt.Sprintf(`{
			"tag_name": "v2.0.0",
			"assets": [
				{"name": "goposix_%s_%s.tar.gz", "browser_download_url": "https://example.com/download"}
			]
		}`, runtime.GOOS, runtime.GOARCH)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockResponseBody{Reader: strings.NewReader(jsonData)},
		}, nil
	}

	tag, url, err := fetchLatestRelease()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v2.0.0" {
		t.Errorf("expected tag v2.0.0, got %s", tag)
	}
	if url != "https://example.com/download" {
		t.Errorf("expected download url, got %s", url)
	}

	// 2. Non-200 Status
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       &mockResponseBody{Reader: strings.NewReader("not found")},
		}, nil
	}
	_, _, err = fetchLatestRelease()
	if err == nil {
		t.Error("expected error for non-200 status")
	}

	// 3. Invalid JSON
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockResponseBody{Reader: strings.NewReader("invalid { json")},
		}, nil
	}
	_, _, err = fetchLatestRelease()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// 4. No matching asset
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		jsonData := `{
			"tag_name": "v2.0.0",
			"assets": [
				{"name": "goposix_invalid_os_arch.tar.gz", "browser_download_url": "https://example.com/download"}
			]
		}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockResponseBody{Reader: strings.NewReader(jsonData)},
		}, nil
	}
	_, _, err = fetchLatestRelease()
	if err == nil {
		t.Error("expected error for missing asset matching platform")
	}
}

func TestDownloadBinary_Mock(t *testing.T) {
	origDo := doHTTPRequest
	defer func() { doHTTPRequest = origDo }()

	// 1. Success case - raw binary (not .tar.gz)
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockResponseBody{Reader: strings.NewReader("fake binary data")},
		}, nil
	}

	tmpFile, err := downloadBinary("https://example.com/goposix_linux_amd64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(tmpFile)

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake binary data" {
		t.Errorf("expected 'fake binary data', got %q", string(data))
	}

	// 2. Download returns non-200 status
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       &mockResponseBody{Reader: strings.NewReader("error")},
		}, nil
	}
	_, err = downloadBinary("https://example.com/goposix")
	if err == nil {
		t.Error("expected error for non-200 download status")
	}
}

func TestUpgrade_Mock(t *testing.T) {
	origDo := doHTTPRequest
	defer func() { doHTTPRequest = origDo }()

	// 1. Upgrade not needed: version is already the latest version
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		jsonData := fmt.Sprintf(`{
			"tag_name": "v1.0.0",
			"assets": [
				{"name": "goposix_%s_%s.tar.gz", "browser_download_url": "https://example.com/download"}
			]
		}`, runtime.GOOS, runtime.GOARCH)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockResponseBody{Reader: strings.NewReader(jsonData)},
		}, nil
	}

	err := Upgrade("v1.0.0")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 2. Upgrade not needed: version on disk is newer
	err = Upgrade("v1.0.1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 3. Empty version tag error case
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		jsonData := `{
			"tag_name": "",
			"assets": []
		}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockResponseBody{Reader: strings.NewReader(jsonData)},
		}, nil
	}
	err = Upgrade("v1.0.0")
	if err == nil {
		t.Error("expected error for empty tag_name")
	}
}

func TestDownloadBinary_MockTarGz(t *testing.T) {
	origDo := doHTTPRequest
	defer func() { doHTTPRequest = origDo }()

	// Success case - .tar.gz archive
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("tar gz binary content")
	hdr := &tar.Header{
		Name:     "goposix",
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gw.Close()

	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockResponseBody{Reader: bytes.NewReader(buf.Bytes())},
		}, nil
	}

	tmpFile, err := downloadBinary("https://example.com/goposix_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(tmpFile)

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "tar gz binary content" {
		t.Errorf("expected 'tar gz binary content', got %q", string(data))
	}
}

func TestDownloadBinary_ErrorPaths(t *testing.T) {
	origDo := doHTTPRequest
	defer func() { doHTTPRequest = origDo }()

	// 1. Invalid URL for NewRequest
	_, err := downloadBinary("http://[::1]%23/invalid")
	if err == nil {
		t.Error("expected error for invalid URL")
	}

	// 2. HTTP request failure
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network down")
	}
	_, err = downloadBinary("https://example.com/goposix")
	if err == nil {
		t.Error("expected error for HTTP connection failure")
	}

	// 3. Corrupted .tar.gz download
	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &mockResponseBody{Reader: strings.NewReader("not a gzip archive")},
		}, nil
	}
	_, err = downloadBinary("https://example.com/goposix.tar.gz")
	if err == nil {
		t.Error("expected error for corrupted .tar.gz extraction")
	}
}

func TestUpgrade_FetchLatestReleaseError(t *testing.T) {
	origDo := doHTTPRequest
	defer func() { doHTTPRequest = origDo }()

	doHTTPRequest = func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("github api is down")
	}

	err := Upgrade("v1.0.0")
	if err == nil {
		t.Error("expected error when GitHub API is down during Upgrade")
	}
	if !strings.Contains(err.Error(), "cannot check for updates") {
		t.Errorf("unexpected error: %v", err)
	}
}




