package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// writeTarGz writes a gzip+tar archive holding one entry with the given name and payload.
func writeTarGz(t *testing.T, name string, payload []byte) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(payload)), Mode: 0o755}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestExtractBinaryRejectsUnsafeEntries checks traversal and absolute entries are refused.
func TestExtractBinaryRejectsUnsafeEntries(t *testing.T) {
	cases := map[string]string{
		"parent":   "../es-tui",
		"absolute": filepath.Join(string(filepath.Separator), "tmp", "es-tui"),
	}
	for label, entry := range cases {
		t.Run(label, func(t *testing.T) {
			archive := writeTarGz(t, entry, []byte("bin"))
			destDir := t.TempDir()
			if _, err := extractBinary(archive, destDir); err == nil {
				t.Fatalf("expected error for entry %q", entry)
			}
			if _, err := os.Stat(filepath.Join(destDir, "es-tui")); !os.IsNotExist(err) {
				t.Fatalf("es-tui written to destDir for entry %q: %v", entry, err)
			}
		})
	}
}

// TestExtractBinaryNestedEntry checks a nested entry extracts to destDir/es-tui.
func TestExtractBinaryNestedEntry(t *testing.T) {
	archive := writeTarGz(t, "subdir/es-tui", []byte("bin"))
	destDir := t.TempDir()
	out, err := extractBinary(archive, destDir)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(destDir, "es-tui")
	if out != want {
		t.Fatalf("path = %q, want %q", out, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "bin" {
		t.Fatalf("content = %q, want %q", data, "bin")
	}
}
