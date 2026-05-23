package goposix

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var doHTTPRequest = func(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

// Upgrade performs a self-upgrade: checks the latest GitHub release, compares
// it against the current version, and if a newer release exists, downloads and
// atomically replaces the running binary.
//
// Returns an error if the upgrade cannot proceed (network failure, permissions,
// already up-to-date, etc.). A nil return means the upgrade succeeded and the
// caller should exit immediately — the binary on disk has been replaced.
func Upgrade(currentVersion string) error {
	// Locate the running binary.
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate current binary: %w", err)
	}
	selfPath, err = filepath.EvalSymlinks(selfPath)
	if err != nil {
		return fmt.Errorf("cannot resolve binary path: %w", err)
	}

	// Fetch latest release metadata from GitHub.
	latestTag, assetURL, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("cannot check for updates: %w", err)
	}

	// Compare versions. Strip leading 'v' from tag if present.
	latestVersion := strings.TrimPrefix(latestTag, "v")
	if latestVersion == "" {
		return fmt.Errorf("empty version tag in GitHub release")
	}

	current := strings.TrimPrefix(currentVersion, "v")
	if current == latestVersion {
		fmt.Fprintf(os.Stderr, "goposix is already at the latest version (%s)\n", current)
		return nil
	}
	if current != "" && !isNewer(latestVersion, current) {
		fmt.Fprintf(os.Stderr, "goposix %s is up to date (latest: %s)\n", current, latestVersion)
		return nil
	}

	fmt.Fprintf(os.Stderr, "upgrading goposix from %s to %s...\n", current, latestVersion)

	// Download the new binary.
	tmpFile, err := downloadBinary(assetURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile) // clean up on failure

	// Make the downloaded binary executable.
	if err := os.Chmod(tmpFile, 0755); err != nil {
		return fmt.Errorf("cannot make binary executable: %w", err)
	}

	// Atomic replacement: rename into place (same filesystem, so it's atomic on Linux).
	if err := os.Rename(tmpFile, selfPath); err != nil {
		return fmt.Errorf("cannot replace binary (try running as root?): %w", err)
	}

	fmt.Fprintf(os.Stderr, "goposix upgraded to %s\n", latestVersion)
	return nil
}

// fetchLatestRelease queries the GitHub Releases API for the latest goposix
// release. Returns the tag name and the download URL of the asset matching the
// current platform.
func fetchLatestRelease() (tag string, assetURL string, err error) {
	url := "https://api.github.com/repos/ramayac/goposix/releases/latest"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "goposix-upgrade")

	resp, err := doHTTPRequest(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("invalid GitHub API response: %w", err)
	}

	if release.TagName == "" {
		return "", "", fmt.Errorf("no tag_name in release")
	}

	// Find the asset matching the current OS and architecture.
	// GoReleaser names archives as: goposix_<os>_<arch>.tar.gz
	wantSuffix := fmt.Sprintf("_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	for _, a := range release.Assets {
		if strings.Contains(a.Name, wantSuffix) {
			return release.TagName, a.BrowserDownloadURL, nil
		}
	}

	// Fallback: try plain binary name.
	wantBinary := fmt.Sprintf("goposix_%s_%s", runtime.GOOS, runtime.GOARCH)
	for _, a := range release.Assets {
		if strings.Contains(a.Name, wantBinary) && !strings.HasSuffix(a.Name, ".tar.gz") {
			return release.TagName, a.BrowserDownloadURL, nil
		}
	}

	return "", "", fmt.Errorf("no release asset for %s/%s", runtime.GOOS, runtime.GOARCH)
}

// downloadBinary downloads the release asset at assetURL to a temporary file
// and returns the path. Handles both .tar.gz archives (extracting the first
// regular file as the binary) and raw binaries.
func downloadBinary(assetURL string) (string, error) {
	req, err := http.NewRequest("GET", assetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "goposix-upgrade")

	resp, err := doHTTPRequest(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// Create temp file in the same directory as the binary so os.Rename is atomic.
	tmp, err := os.CreateTemp("", "goposix-upgrade-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()

	if strings.HasSuffix(assetURL, ".tar.gz") {
		if err := extractTarGzBinary(resp.Body, tmp); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("archive extraction failed: %w", err)
		}
	} else {
		if _, err := io.Copy(tmp, resp.Body); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return "", err
		}
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	return tmpPath, nil
}

// extractTarGzBinary reads a .tar.gz stream from r, finds the entry named
// "goposix" (the binary), and writes its content to w.
func extractTarGzBinary(r io.Reader, w io.Writer) error {
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gzReader.Close()

	tr := tar.NewReader(gzReader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("binary 'goposix' not found in archive")
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && (hdr.Name == "goposix" || filepath.Base(hdr.Name) == "goposix") {
			if _, err := io.Copy(w, tr); err != nil {
				return err
			}
			return nil
		}
	}
}

// isNewer returns true if version a is strictly greater than version b.
// Handles dotted versions like "1.2.3" and "0.1.0".
// Pre-release suffixes ("-rc1", "-beta") sort before stable releases.
// Git-derived versions like "abc1234" sort before numeric versions.
func isNewer(a, b string) bool {
	ap := parseVersion(a)
	bp := parseVersion(b)
	for i := 0; i < len(ap) && i < len(bp); i++ {
		if ap[i] > bp[i] {
			return true
		}
		if ap[i] < bp[i] {
			return false
		}
	}
	// Numeric segments equal up to the shorter one.
	if len(ap) != len(bp) {
		return len(ap) > len(bp)
	}
	// Same numeric segments. A stable release (no suffix) is newer than a
	// pre-release (has a suffix).
	return !hasSuffix(a) && hasSuffix(b)
}

// hasSuffix returns true if the version string has a pre-release suffix
// after the numeric dotted part (e.g., "-rc1", "-beta.2").
func hasSuffix(v string) bool {
	// A pre-release suffix follows a hyphen after the dotted version.
	// e.g., "0.1.0-rc1" -> has suffix, "0.1.0" -> no suffix.
	return strings.Contains(v, "-")
}

// parseVersion splits a dotted version string into integer components.
// Non-numeric segments (like "abc1234" or "-rc1") cause the segment to
// be treated as 0, making "1.0.0" > "1.0.0-rc1" and "abc1234" sort
// before any numeric version.
func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				break
			}
		}
		nums = append(nums, n)
	}
	return nums
}
