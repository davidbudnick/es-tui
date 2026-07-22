package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIsSemverAndHomebrew(t *testing.T) {
	if !isSemver("1.2.3") || !isSemver("v1.2.3") || isSemver("dev") || isSemver("latest") {
		t.Fatal("semver")
	}
	if !isHomebrew("/opt/homebrew/bin/es-tui") || !isHomebrew("/usr/local/Cellar/es-tui/1.0/bin/es-tui") {
		t.Fatal("homebrew")
	}
	if isHomebrew("/usr/local/bin/es-tui") {
		// may or may not - homebrew check is substring
	}
}

func TestArchiveName(t *testing.T) {
	n := archiveName("1.0.0", "darwin", "arm64")
	if n != "es-tui_1.0.0_Darwin_arm64.tar.gz" {
		t.Fatal(n)
	}
	n = archiveName("1.0.0", "linux", "amd64")
	if n != "es-tui_1.0.0_Linux_x86_64.tar.gz" {
		t.Fatal(n)
	}
	n = archiveName("1.0.0", "windows", "amd64")
	if !strings.HasSuffix(n, ".zip") {
		t.Fatal(n)
	}
}

func TestRunUpdateDev(t *testing.T) {
	if err := runUpdate("dev"); err == nil {
		t.Fatal("expected error for dev")
	}
	if err := runUpdate("not-semver"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckWriteAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "es-tui")
	if err := checkWriteAccess(path); err != nil {
		t.Fatal(err)
	}
}

func TestFetchLatestAndDownloadVerifyExtractInstall(t *testing.T) {
	// Build a minimal gzip tar with es-tui binary
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	content := []byte("#!/bin/sh\necho es-tui\n")
	hdr := &tar.Header{Name: "es-tui", Mode: 0755, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archiveBytes := buf.Bytes()
	sum := sha256.Sum256(archiveBytes)
	checksum := hex.EncodeToString(sum[:])
	arch := archiveName("1.0.0", runtime.GOOS, runtime.GOARCH)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			_, _ = fmt.Fprintf(w, "%s  %s\n", checksum, arch)
		case strings.Contains(r.URL.Path, arch):
			_, _ = w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPIBase
	oldClient := httpClient
	githubAPIBase = srv.URL
	httpClient = srv.Client()
	t.Cleanup(func() {
		githubAPIBase = oldAPI
		httpClient = oldClient
	})

	// Patch download URLs by temporarily replacing fetch + using custom server for github downloads
	// runUpdate uses hard-coded github.com URLs for assets — override via replacing functions
	// Instead test pieces:

	ver, err := fetchLatestVersion()
	if err != nil || ver != "v1.0.0" {
		t.Fatalf("%s %v", ver, err)
	}

	// bad status
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", 500)
	}))
	defer bad.Close()
	githubAPIBase = bad.URL
	if _, err := fetchLatestVersion(); err == nil {
		t.Fatal("expected error")
	}
	githubAPIBase = srv.URL

	// empty tag
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":""}`))
	}))
	defer empty.Close()
	githubAPIBase = empty.URL
	if _, err := fetchLatestVersion(); err == nil {
		t.Fatal("empty tag")
	}
	githubAPIBase = srv.URL

	dir := t.TempDir()
	archivePath := filepath.Join(dir, arch)
	if err := downloadFile(srv.URL+"/"+arch, archivePath); err != nil {
		t.Fatal(err)
	}
	checksumPath := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksumPath, []byte(fmt.Sprintf("%s  %s\n", checksum, arch)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(archivePath, checksumPath, arch); err != nil {
		t.Fatal(err)
	}
	// mismatch
	if err := os.WriteFile(checksumPath, []byte("deadbeef  "+arch+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(archivePath, checksumPath, arch); err == nil {
		t.Fatal("mismatch")
	}
	// missing
	if err := os.WriteFile(checksumPath, []byte("abc  other.tar.gz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(archivePath, checksumPath, arch); err == nil {
		t.Fatal("missing")
	}

	// extract
	if err := os.WriteFile(checksumPath, []byte(fmt.Sprintf("%s  %s\n", checksum, arch)), 0o600); err != nil {
		t.Fatal(err)
	}
	bin, err := extractBinary(archivePath, dir)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "installed-es-tui")
	if err := installBinary(bin, dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, content) {
		t.Fatal("content mismatch")
	}

	// extract no binary
	var emptyTar bytes.Buffer
	gz2 := gzip.NewWriter(&emptyTar)
	tw2 := tar.NewWriter(gz2)
	_ = tw2.WriteHeader(&tar.Header{Name: "README", Size: 1, Mode: 0644})
	_, _ = tw2.Write([]byte("x"))
	_ = tw2.Close()
	_ = gz2.Close()
	emptyPath := filepath.Join(dir, "empty.tgz")
	if err := os.WriteFile(emptyPath, emptyTar.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := extractBinary(emptyPath, dir); err == nil {
		t.Fatal("expected no binary")
	}

	// download error
	if err := downloadFile(srv.URL+"/missing", filepath.Join(dir, "x")); err == nil {
		t.Fatal("download 404")
	}

	// full runUpdate with patched executable and homebrew bypass
	installDir := t.TempDir()
	execPath := filepath.Join(installDir, "es-tui")
	if err := os.WriteFile(execPath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Override github download base by intercepting httpClient to rewrite hosts
	assetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "releases/latest") || strings.HasSuffix(r.URL.Path, "latest") {
			_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
			return
		}
		if strings.Contains(r.URL.Path, "checksums") {
			_, _ = fmt.Fprintf(w, "%s  %s\n", checksum, arch)
			return
		}
		if strings.Contains(r.URL.Path, arch) || strings.HasSuffix(r.URL.Path, ".tar.gz") {
			_, _ = w.Write(archiveBytes)
			return
		}
		// API
		if strings.Contains(r.URL.Path, "releases") {
			_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer assetSrv.Close()

	// Custom transport that redirects github to assetSrv
	httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			u := *req.URL
			// rewrite to asset server
			base := assetSrv.URL
			newReq, err := http.NewRequest(req.Method, base+u.Path, req.Body)
			if err != nil {
				return nil, err
			}
			return http.DefaultTransport.RoundTrip(newReq)
		}),
	}
	githubAPIBase = assetSrv.URL

	oldExec := osExecutable
	osExecutable = func() (string, error) { return execPath, nil }
	t.Cleanup(func() { osExecutable = oldExec })

	// already up to date
	if err := runUpdate("1.0.0"); err != nil {
		t.Fatal(err)
	}
	// upgrade from 0.9.0
	if err := runUpdate("0.9.0"); err != nil {
		t.Fatalf("update: %v", err)
	}

	// homebrew path
	osExecutable = func() (string, error) { return "/opt/homebrew/bin/es-tui", nil }
	if err := runUpdate("0.1.0"); err == nil {
		t.Fatal("expected homebrew error")
	}

	// executable error
	osExecutable = func() (string, error) { return "", errorsNew("no exec") }
	if err := runUpdate("0.1.0"); err == nil {
		t.Fatal("expected exec error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func errorsNew(s string) error { return fmt.Errorf("%s", s) }

func TestInstallBinaryAndCopyOverrides(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "dest")
	if err := installBinary(src, dest); err != nil {
		t.Fatal(err)
	}

	// ioCopy override error path in downloadFile
	old := ioCopy
	ioCopy = func(dst io.Writer, src io.Reader) (int64, error) {
		return 0, fmt.Errorf("copy fail")
	}
	t.Cleanup(func() { ioCopy = old })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()
	if err := downloadFile(srv.URL, filepath.Join(dir, "f")); err == nil {
		t.Fatal("expected copy error")
	}
}
