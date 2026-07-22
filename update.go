package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var githubAPIBase = "https://api.github.com"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Overridable in tests.
var (
	osExecutable  = os.Executable
	osMkdirTemp   = os.MkdirTemp
	osUserHomeDir = os.UserHomeDir
	ioCopy        = io.Copy
	osOpenFile    = os.OpenFile
	osCreateTemp  = os.CreateTemp
)

const githubRepo = "davidbudnick/es-tui"

const maxDownloadSize = 256 << 20
const maxBinarySize = 128 << 20

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func runUpdate(currentVersion string) error {
	if currentVersion == "dev" || !isSemver(currentVersion) {
		return fmt.Errorf("cannot self-update a development build (version=%q); use the install script instead:\n  curl -fsSL https://raw.githubusercontent.com/davidbudnick/es-tui/main/install.sh | bash", currentVersion)
	}

	execPath, err := osExecutable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	if isHomebrew(execPath) {
		return fmt.Errorf("this binary was installed via Homebrew; please update with:\n  brew upgrade es-tui")
	}

	if err := checkWriteAccess(execPath); err != nil {
		home, homeErr := osUserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("cannot write to %s and could not determine home directory: %w", execPath, homeErr)
		}
		localBin := filepath.Join(home, ".local", "bin")
		if mkErr := os.MkdirAll(localBin, 0750); mkErr != nil {
			return fmt.Errorf("cannot write to %s and could not create %s: %w", execPath, localBin, mkErr)
		}
		execPath = filepath.Join(localBin, "es-tui")
		fmt.Printf("No write access to current location, installing to %s\n", execPath)
	}

	latest, err := fetchLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to fetch latest version: %w", err)
	}

	if strings.TrimPrefix(latest, "v") == strings.TrimPrefix(currentVersion, "v") {
		fmt.Printf("Already up to date (v%s).\n", strings.TrimPrefix(currentVersion, "v"))
		return nil
	}

	ver := strings.TrimPrefix(latest, "v")
	archive := archiveName(ver, runtime.GOOS, runtime.GOARCH)
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", githubRepo, latest)
	archiveURL := baseURL + "/" + archive
	checksumURL := baseURL + "/checksums.txt"

	tmpDir, err := osMkdirTemp("", "es-tui-update-*")
	if err != nil {
		return fmt.Errorf("could not create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archive)
	checksumPath := filepath.Join(tmpDir, "checksums.txt")

	if err := downloadFile(archiveURL, archivePath); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	if err := downloadFile(checksumURL, checksumPath); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	if err := verifyChecksum(archivePath, checksumPath, archive); err != nil {
		return err
	}

	binaryPath, err := extractBinary(archivePath, tmpDir)
	if err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}

	if err := installBinary(binaryPath, execPath); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}

	fmt.Printf("Updated to %s.\n", latest)
	return nil
}

func isSemver(v string) bool {
	v = strings.TrimPrefix(v, "v")
	re := regexp.MustCompile(`^\d+\.\d+\.\d+`)
	return re.MatchString(v)
}

func isHomebrew(path string) bool {
	return strings.Contains(path, "Homebrew") || strings.Contains(path, "homebrew") || strings.Contains(path, "Cellar")
}

func checkWriteAccess(path string) error {
	dir := filepath.Dir(path)
	f, err := osCreateTemp(dir, ".es-tui-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}

func fetchLatestVersion() (string, error) {
	url := githubAPIBase + "/repos/" + githubRepo + "/releases/latest"
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("empty tag name")
	}
	return rel.TagName, nil
}

func archiveName(version, goos, goarch string) string {
	osName := goos
	switch goos {
	case "darwin":
		osName = "Darwin"
	case "linux":
		osName = "Linux"
	case "windows":
		osName = "Windows"
	}
	arch := goarch
	if goarch == "amd64" {
		arch = "x86_64"
	}
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("es-tui_%s_%s_%s.%s", version, osName, arch, ext)
}

func downloadFile(url, dest string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = ioCopy(f, io.LimitReader(resp.Body, maxDownloadSize))
	return err
}

func verifyChecksum(archivePath, checksumPath, archiveName string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.HasSuffix(fields[1], archiveName) {
			expected = fields[0]
			break
		}
		if len(fields) >= 2 && fields[1] == archiveName {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum not found for %s", archiveName)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch: got %s want %s", actual, expected)
	}
	return nil
}

func extractBinary(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("gzip: %w (zip archives not supported by self-update on this platform)", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		base := filepath.Base(hdr.Name)
		if base != "es-tui" && base != "es-tui.exe" {
			continue
		}
		outPath := filepath.Join(destDir, base)
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o750)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(out, io.LimitReader(tr, maxBinarySize)); err != nil {
			_ = out.Close()
			return "", err
		}
		if err := out.Close(); err != nil {
			return "", err
		}
		return outPath, nil
	}
	return "", fmt.Errorf("binary not found in archive")
}

func installBinary(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := osCreateTemp(filepath.Dir(dest), ".es-tui-new-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := ioCopy(tmp, in); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, dest)
}
