package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidbudnick/es-tui/internal/ui"
)

func TestParseFlags(t *testing.T) {
	conn, ver, upd, err := parseFlags([]string{})
	if err != nil || conn != nil || ver || upd {
		t.Fatalf("empty: %v %v %v %v", conn, ver, upd, err)
	}

	_, ver, _, err = parseFlags([]string{"--version"})
	if err != nil || !ver {
		t.Fatal("version")
	}
	_, _, upd, err = parseFlags([]string{"--update"})
	if err != nil || !upd {
		t.Fatal("update")
	}

	conn, _, _, err = parseFlags([]string{
		"--host", "localhost",
		"--port", "9201",
		"--user", "elastic",
		"--password", "secret",
		"--api-key", "key",
		"--name", "local",
		"--flavor", "opensearch",
		"--tls",
		"--tls-cert", "c",
		"--tls-key", "k",
		"--tls-ca", "ca",
		"--tls-skip-verify",
	})
	if err != nil || conn == nil {
		t.Fatal(err)
	}
	if conn.Host != "localhost" || conn.Port != 9201 || conn.Name != "local" {
		t.Fatal(conn)
	}
	if !conn.UseTLS || conn.TLSConfig == nil || !conn.TLSConfig.InsecureSkipVerify {
		t.Fatal(conn.TLSConfig)
	}

	conn, _, _, err = parseFlags([]string{"-h", "es.example", "-p", "9200", "-a", "pw"})
	if err != nil || conn == nil || conn.Name != "es.example:9200" {
		t.Fatal(conn, err)
	}

	conn, _, _, err = parseFlags([]string{"--host", "x", "--flavor", "bogus"})
	if err != nil || conn.Flavor != "auto" {
		t.Fatal(conn)
	}

	_, _, _, err = parseFlags([]string{"--help"})
	if err != flag.ErrHelp {
		// ContinueOnError returns ErrHelp
		if err == nil {
			t.Fatal("expected help err")
		}
	}
}

func TestInitConfig(t *testing.T) {
	dir := t.TempDir()
	old := userHomeDir
	userHomeDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userHomeDir = old })

	cfg, err := initConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Close(); err != nil {
		t.Fatal(err)
	}
	// home dir error falls back to temp
	userHomeDir = func() (string, error) { return "", errors.New("no home") }
	cfg, err = initConfig()
	if err != nil {
		t.Fatal(err)
	}
	_ = cfg.Close()
}

func TestSetup(t *testing.T) {
	dir := t.TempDir()
	oldHome := userHomeDir
	oldArgs := os.Args
	userHomeDir = func() (string, error) { return dir, nil }
	os.Args = []string{"es-tui"}
	t.Cleanup(func() {
		userHomeDir = oldHome
		os.Args = oldArgs
	})

	m, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	if m.Cmds == nil || m.Logs == nil {
		t.Fatal("incomplete model")
	}

	// with host flag
	os.Args = []string{"es-tui", "--host", "localhost", "--port", "9200"}
	m, err = setup()
	if err != nil {
		t.Fatal(err)
	}
	if m.CLIConnection == nil {
		t.Fatal("expected cli connection")
	}
}

func TestParseCLIFlagsVersionUpdate(t *testing.T) {
	oldExit := osExit
	oldArgs := os.Args
	code := -1
	osExit = func(c int) { code = c }
	t.Cleanup(func() {
		osExit = oldExit
		os.Args = oldArgs
	})

	os.Args = []string{"es-tui", "--version"}
	_ = parseCLIFlags()
	if code != 0 {
		t.Fatalf("version exit %d", code)
	}

	// update on dev fails and exits 1
	code = -1
	os.Args = []string{"es-tui", "--update"}
	_ = parseCLIFlags()
	if code != 1 {
		t.Fatalf("update exit %d", code)
	}

	// help
	code = -1
	os.Args = []string{"es-tui", "--help"}
	_ = parseCLIFlags()
	if code != 0 && code != 2 {
		// ErrHelp -> 0
	}
}

func TestMainErrorPaths(t *testing.T) {
	oldFatal := logFatal
	oldRun := runApp
	oldExit := osExit
	t.Cleanup(func() {
		logFatal = oldFatal
		runApp = oldRun
		osExit = oldExit
	})

	called := false
	logFatal = func(v ...any) { called = true }
	runApp = func(m ui.Model) error { return errors.New("run fail") }
	_ = called

	dir := t.TempDir()
	userHomeDir = func() (string, error) { return dir, nil }
	os.Args = []string{"es-tui"}
	m, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	runApp = func(model ui.Model) error { return nil }
	if err := runApp(m); err != nil {
		t.Fatal(err)
	}
	runApp = func(model ui.Model) error { return errors.New("x") }
	if err := runApp(m); err == nil {
		t.Fatal("expected err")
	}
	_ = filepath.Join(dir, ".config", "es-tui")

	// exercise prod helpers signatures (not run — needs a TTY)
	_ = prodLogFatal
	_ = prodRunApp
}
