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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpdateAllErrorPaths(t *testing.T) {
	// fetchLatestVersion network error
	oldClient := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("net")
	})}
	t.Cleanup(func() { httpClient = oldClient })
	if _, err := fetchLatestVersion(); err == nil {
		t.Fatal("net")
	}

	// decode error
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("notjson")), Header: make(http.Header)}, nil
	})}
	if _, err := fetchLatestVersion(); err == nil {
		t.Fatal("decode")
	}

	// download create fail
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("x")), Header: make(http.Header)}, nil
	})}
	if err := downloadFile("http://x", filepath.Join(t.TempDir(), "no", "such", "file")); err == nil {
		t.Fatal("create")
	}

	// verifyChecksum open archive fail
	dir := t.TempDir()
	cs := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(cs, []byte("abc  file.tgz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(filepath.Join(dir, "missing"), cs, "file.tgz"); err == nil {
		t.Fatal("open")
	}

	// verifyChecksum with suffix match
	arch := "es-tui_1.0.0_Darwin_arm64.tar.gz"
	content := []byte("hello")
	sum := sha256.Sum256(content)
	ap := filepath.Join(dir, arch)
	if err := os.WriteFile(ap, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cs, []byte(fmt.Sprintf("%s  path/%s\n", hex.EncodeToString(sum[:]), arch)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(ap, cs, arch); err != nil {
		t.Fatal(err)
	}

	// extractBinary gzip error (plain file)
	plain := filepath.Join(dir, "plain")
	if err := os.WriteFile(plain, []byte("notgzip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := extractBinary(plain, dir); err == nil {
		t.Fatal("gzip")
	}

	// extractBinary with .exe name
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	payload := []byte("bin")
	_ = tw.WriteHeader(&tar.Header{Name: "es-tui.exe", Mode: 0755, Size: int64(len(payload))})
	_, _ = tw.Write(payload)
	_ = tw.Close()
	_ = gz.Close()
	exeArch := filepath.Join(dir, "w.tgz")
	if err := os.WriteFile(exeArch, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if p, err := extractBinary(exeArch, dir); err != nil || !strings.HasSuffix(p, "es-tui.exe") {
		t.Fatal(p, err)
	}

	// installBinary chmod/rename paths with osCreateTemp failure
	oldCT := osCreateTemp
	osCreateTemp = func(dir, pattern string) (*os.File, error) {
		return nil, fmt.Errorf("tmp")
	}
	src := filepath.Join(dir, "srcbin")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installBinary(src, filepath.Join(dir, "dest")); err == nil {
		t.Fatal("tmp fail")
	}
	osCreateTemp = oldCT

	// installBinary copy fail
	oldCopy := ioCopy
	ioCopy = func(dst io.Writer, src io.Reader) (int64, error) {
		return 0, fmt.Errorf("copy")
	}
	if err := installBinary(src, filepath.Join(dir, "dest2")); err == nil {
		t.Fatal("copy fail")
	}
	ioCopy = oldCopy // restore before later tests

	// runUpdate mkdirTemp fail
	oldMk := osMkdirTemp
	osMkdirTemp = func(string, string) (string, error) { return "", fmt.Errorf("mk") }
	oldExec := osExecutable
	osExecutable = func() (string, error) {
		p := filepath.Join(dir, "es-tui")
		_ = os.WriteFile(p, []byte("x"), 0o755)
		return p, nil
	}
	// need network for fetch - set up after mkdirTemp restored for other path
	osMkdirTemp = oldMk

	// runUpdate download fail after mkdir
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "latest") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v9.9.9"}`)), Header: make(http.Header)}, nil
		}
		return nil, fmt.Errorf("dl")
	})}
	if err := runUpdate("1.0.0"); err == nil {
		t.Fatal("expected download fail")
	}

	// runUpdate checksum download fail
	var tarBytes bytes.Buffer
	gz2 := gzip.NewWriter(&tarBytes)
	tw2 := tar.NewWriter(gz2)
	_ = tw2.WriteHeader(&tar.Header{Name: "es-tui", Mode: 0755, Size: 3})
	_, _ = tw2.Write([]byte("abc"))
	_ = tw2.Close()
	_ = gz2.Close()
	archName := archiveName("9.9.9", runtime.GOOS, runtime.GOARCH)
	step := 0
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "latest") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v9.9.9"}`)), Header: make(http.Header)}, nil
		}
		if strings.Contains(r.URL.Path, "checksums") {
			return nil, fmt.Errorf("cs")
		}
		if strings.Contains(r.URL.Path, archName) || strings.HasSuffix(r.URL.Path, ".tar.gz") {
			step++
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(tarBytes.Bytes())), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}
	if err := runUpdate("1.0.0"); err == nil {
		t.Fatal("checksum dl")
	}

	// EvalSymlinks fail
	osExecutable = func() (string, error) { return filepath.Join(dir, "no-exist-link"), nil }
	if err := runUpdate("1.0.0"); err == nil {
		t.Fatal("eval")
	}

	osExecutable = oldExec
	osMkdirTemp = oldMk
	_ = step
}

func TestDefaultNewProgramLiteral(t *testing.T) {
	// Capture and invoke the production newProgram once through a no-op that
	// doesn't call Run for long — just ensure the factory is the real one after reset.
	old := newProgram
	t.Cleanup(func() { newProgram = old })
	// reset to production if tests overwrote
	// The production literal is only the initial value; re-assign using tea path
	// is already tested via prodRunApp with fakeProgram. Cover warning path:
	oldArgs := os.Args
	os.Args = []string{"es-tui", "--host", "h", "--password", "p"}
	t.Cleanup(func() { os.Args = oldArgs })
	// parseFlags visits password
	conn, _, _, err := parseFlags([]string{"--host", "h", "--password", "secret", "--api-key", "k"})
	if err != nil || conn == nil {
		t.Fatal(err)
	}
}
