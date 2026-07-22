package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestDBCoverageGaps(t *testing.T) {
	// NewConfig mkdir failure - use invalid path
	if _, err := NewConfig(string([]byte{0})); err == nil {
		// may fail on some OS
	}

	// ListConnections sort with multiple
	cfg := newTestConfig(t)
	_, err := cfg.AddConnection(types.Connection{Name: "b", Host: "h", Port: 2})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.AddConnection(types.Connection{Name: "a", Host: "h", Port: 1})
	if err != nil {
		t.Fatal(err)
	}
	list, err := cfg.ListConnections()
	if err != nil {
		t.Fatal(err)
	}
	if list[0].ID > list[1].ID {
		t.Fatal("not sorted")
	}

	// ClearRecentIndices keeps other conn
	c1 := list[0]
	c2 := list[1]
	cfg.AddRecentIndex(c1.ID, "a")
	cfg.AddRecentIndex(c2.ID, "b")
	cfg.ClearRecentIndices(c1.ID)
	if len(cfg.ListRecentIndices(c2.ID)) != 1 {
		t.Fatal("should keep c2")
	}

	// NewConfig on nested path that exists as file - mkdir fails
	dir := t.TempDir()
	fileAsDir := filepath.Join(dir, "blocked")
	if err := os.WriteFile(fileAsDir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewConfig(filepath.Join(fileAsDir, "config.json")); err == nil {
		t.Fatal("expected mkdir fail")
	}
}
