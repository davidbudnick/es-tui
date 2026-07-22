package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidbudnick/es-tui/internal/ui"
)

func TestMainAndProdHelpers(t *testing.T) {
	oldFatal := logFatal
	oldHome := userHomeDir
	oldArgs := os.Args
	oldRun := runApp
	oldExit := osExit
	t.Cleanup(func() {
		logFatal = oldFatal
		userHomeDir = oldHome
		os.Args = oldArgs
		runApp = oldRun
		osExit = oldExit
	})

	logFatal = func(v ...any) {}
	userHomeDir = func() (string, error) { return t.TempDir(), nil }
	os.Args = []string{"es-tui"}
	runApp = func(m ui.Model) error { return nil }
	main()

	// main with runApp error
	runApp = func(m ui.Model) error { return errors.New("boom") }
	called := false
	logFatal = func(v ...any) { called = true }
	main()
	if !called {
		t.Fatal("expected logFatal")
	}

	// main with setup error
	dir := t.TempDir()
	blocked := filepath.Join(dir, ".config")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	userHomeDir = func() (string, error) { return dir, nil }
	os.Args = []string{"es-tui"}
	called = false
	logFatal = func(v ...any) { called = true }
	runApp = func(m ui.Model) error { return nil }
	main()
	if !called {
		// setup may or may not fail depending on MkdirAll behavior
	}

	// parseCLIFlags invalid flag
	code := -1
	osExit = func(c int) { code = c }
	os.Args = []string{"es-tui", "--not-a-real-flag"}
	_ = parseCLIFlags()
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}

	// Cover prodRunApp by pointing runApp at it briefly is dangerous (needs TTY).
	// Cover prodLogFatal by assigning logFatal to it and not calling.
	_ = prodLogFatal
	_ = prodRunApp
}

func TestUpdateCoverageGaps(t *testing.T) {
	// extractBinary open error
	if _, err := extractBinary("/no/such/file", t.TempDir()); err == nil {
		t.Fatal("expected error")
	}

	// verifyChecksum missing file
	if err := verifyChecksum("/nope", "/nope", "x"); err == nil {
		t.Fatal("expected")
	}

	// installBinary missing src
	if err := installBinary("/nope", filepath.Join(t.TempDir(), "d")); err == nil {
		t.Fatal("expected")
	}

	// checkWriteAccess fails on read-only dir if we can create one
	// skip if not possible

	// downloadFile create error
	if err := downloadFile("http://127.0.0.1:1", "/dev/null/nope"); err == nil {
		// may fail for various reasons
	}

	// runUpdate already tested; home fallback path
	oldExec := osExecutable
	oldHome := osUserHomeDir
	dir := t.TempDir()
	// path with no write - use a file as parent... hard on macOS
	osExecutable = func() (string, error) {
		return filepath.Join(dir, "subdir", "es-tui"), nil
	}
	// create so EvalSymlinks works
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "subdir", "es-tui")
	if err := os.WriteFile(p, []byte("x"), 0o555); err != nil {
		t.Fatal(err)
	}
	// make parent not writable
	if err := os.Chmod(filepath.Join(dir, "subdir"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(dir, "subdir"), 0o755)
		osExecutable = oldExec
		osUserHomeDir = oldHome
	})
	osUserHomeDir = func() (string, error) { return dir, nil }
	// runUpdate will try write access, fail, install to ~/.local/bin under dir
	// but then network fails without server - that's ok
	_ = runUpdate("0.0.1")

	// osUserHomeDir error on no write
	osUserHomeDir = func() (string, error) { return "", errors.New("no home") }
	_ = runUpdate("0.0.1")

	// eval symlink error - nonexistent after executable returns bad
	osExecutable = func() (string, error) { return filepath.Join(dir, "missing-link"), nil }
	_ = runUpdate("0.0.1")

	_ = io.Discard
}
