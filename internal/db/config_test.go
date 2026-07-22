package db

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

func newTestConfig(t *testing.T) *Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	return cfg
}

func reloadConfig(t *testing.T, cfg *Config) *Config {
	t.Helper()
	reloaded, err := NewConfig(cfg.path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	return reloaded
}

func TestConfigConnectionsCRUD(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Close(); err != nil {
		t.Fatal(err)
	}

	conn, err := cfg.AddConnection(types.Connection{
		Name:     "local",
		Host:     "localhost",
		Port:     9200,
		Password: "secret",
		APIKey:   "key123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if conn.ID != 1 || conn.Flavor != types.FlavorAuto {
		t.Fatalf("unexpected conn: %+v", conn)
	}

	list, err := cfg.ListConnections()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list len %d", len(list))
	}

	// Password stripped on disk
	reloaded := reloadConfig(t, cfg)
	list, err = reloaded.ListConnections()
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Password != "" || list[0].APIKey != "" {
		t.Fatal("password/api key should be stripped")
	}

	// Update preserves secrets when blank
	cfg2 := newTestConfig(t)
	c, err := cfg2.AddConnection(types.Connection{Name: "a", Host: "h", Port: 9200, Password: "p", APIKey: "k", Color: "pink", Group: "g"})
	if err != nil {
		t.Fatal(err)
	}
	c.Password = ""
	c.APIKey = ""
	c.Name = "b"
	updated, err := cfg2.UpdateConnection(c)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Password != "p" || updated.APIKey != "k" || updated.Name != "b" {
		t.Fatalf("update preserve failed: %+v", updated)
	}
	if updated.Color != "pink" || updated.Group != "g" {
		t.Fatalf("color/group not preserved: %+v", updated)
	}

	if err := cfg2.DeleteConnection(c.ID); err != nil {
		t.Fatal(err)
	}
	if err := cfg2.DeleteConnection(999); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist, got %v", err)
	}
	if _, err := cfg2.UpdateConnection(types.Connection{ID: 999}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist update: %v", err)
	}
}

func TestConfigFavoritesAndRecent(t *testing.T) {
	cfg := newTestConfig(t)
	c, err := cfg.AddConnection(types.Connection{Name: "x", Host: "h", Port: 1})
	if err != nil {
		t.Fatal(err)
	}

	fav, err := cfg.AddFavorite(c.ID, "products", "prod")
	if err != nil {
		t.Fatal(err)
	}
	if fav.Index != "products" {
		t.Fatal(fav)
	}
	// duplicate
	fav2, err := cfg.AddFavorite(c.ID, "products", "")
	if err != nil {
		t.Fatal(err)
	}
	if fav2.Index != "products" {
		t.Fatal(fav2)
	}
	if !cfg.IsFavorite(c.ID, "products") {
		t.Fatal("expected favorite")
	}
	if len(cfg.ListFavorites(c.ID)) != 1 {
		t.Fatal("fav count")
	}
	if err := cfg.RemoveFavorite(c.ID, "products"); err != nil {
		t.Fatal(err)
	}
	if cfg.IsFavorite(c.ID, "products") {
		t.Fatal("should not be favorite")
	}
	if err := cfg.RemoveFavorite(c.ID, "missing"); err != nil {
		t.Fatal(err)
	}

	cfg.AddRecentIndex(c.ID, "a")
	cfg.AddRecentIndex(c.ID, "b")
	cfg.AddRecentIndex(c.ID, "a") // move to front
	recent := cfg.ListRecentIndices(c.ID)
	if len(recent) != 2 || recent[0].Index != "a" {
		t.Fatalf("recent=%+v", recent)
	}
	// max recent
	cfg.MaxRecentIndices = 2
	cfg.AddRecentIndex(c.ID, "c")
	if len(cfg.ListRecentIndices(c.ID)) != 2 {
		t.Fatal("max recent not enforced")
	}
	cfg.ClearRecentIndices(c.ID)
	if len(cfg.ListRecentIndices(c.ID)) != 0 {
		t.Fatal("clear failed")
	}
}

func TestConfigValueHistory(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.MaxValueHistory = 2
	cfg.AddValueHistory("idx", "1", `{"a":1}`, "save")
	cfg.AddValueHistory("idx", "1", `{"a":2}`, "save")
	cfg.AddValueHistory("idx", "1", `{"a":3}`, "save")
	h := cfg.GetValueHistory("idx", "1")
	if len(h) != 2 {
		t.Fatalf("history len %d", len(h))
	}
	if h[0].Value != `{"a":3}` {
		t.Fatal(h[0])
	}
	cfg.ClearValueHistory()
	if len(cfg.GetValueHistory("idx", "1")) != 0 {
		t.Fatal("clear history failed")
	}
}

func TestConfigGroupsAndKeyBindings(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.AddGroup("prod", "pink"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddGroup("prod", "pink"); err != nil {
		t.Fatal(err)
	}
	c, err := cfg.AddConnection(types.Connection{Name: "n", Host: "h", Port: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddConnectionToGroup("prod", c.ID); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddConnectionToGroup("prod", c.ID); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddConnectionToGroup("missing", c.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	groups := cfg.ListGroups()
	if len(groups) != 1 || len(groups[0].Connections) != 1 {
		t.Fatalf("%+v", groups)
	}
	if err := cfg.RemoveConnectionFromGroup("prod", c.ID); err != nil {
		t.Fatal(err)
	}
	if err := cfg.RemoveConnectionFromGroup("prod", 999); err != nil {
		t.Fatal(err)
	}
	if err := cfg.RemoveConnectionFromGroup("nope", c.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

	kb := cfg.GetKeyBindings()
	if kb.Quit != "q" {
		t.Fatal(kb)
	}
	kb.Quit = "x"
	if err := cfg.SetKeyBindings(kb); err != nil {
		t.Fatal(err)
	}
	if cfg.GetKeyBindings().Quit != "x" {
		t.Fatal("set kb failed")
	}
	if err := cfg.ResetKeyBindings(); err != nil {
		t.Fatal(err)
	}
	if cfg.GetKeyBindings().Quit != "q" {
		t.Fatal("reset failed")
	}
}

func TestConfigLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewConfig(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestConfigNextIDFromExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := `{"connections":[{"id":5,"name":"a","host":"h","port":1,"created_at":"2020-01-01T00:00:00Z","updated_at":"2020-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	c, err := cfg.AddConnection(types.Connection{Name: "b", Host: "h", Port: 2})
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != 6 {
		t.Fatalf("id=%d", c.ID)
	}
}

func TestConfigSaveMarshalError(t *testing.T) {
	cfg := newTestConfig(t)
	old := jsonMarshalIndent
	jsonMarshalIndent = func(any, string, string) ([]byte, error) {
		return nil, errors.New("marshal fail")
	}
	t.Cleanup(func() { jsonMarshalIndent = old })
	if _, err := cfg.AddConnection(types.Connection{Name: "n", Host: "h", Port: 1}); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestConfigUpdateSaveError(t *testing.T) {
	cfg := newTestConfig(t)
	c, err := cfg.AddConnection(types.Connection{Name: "n", Host: "h", Port: 1, UseTLS: true, TLSConfig: &types.TLSConfig{ServerName: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	old := jsonMarshalIndent
	jsonMarshalIndent = func(any, string, string) ([]byte, error) {
		return nil, errors.New("fail")
	}
	t.Cleanup(func() { jsonMarshalIndent = old })
	c.Name = "changed"
	if _, err := cfg.UpdateConnection(c); err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigAddFavoriteSaveError(t *testing.T) {
	cfg := newTestConfig(t)
	c, err := cfg.AddConnection(types.Connection{Name: "n", Host: "h", Port: 1})
	if err != nil {
		t.Fatal(err)
	}
	old := jsonMarshalIndent
	jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
		// allow connection save, fail favorite
		if cfg, ok := v.(*Config); ok && len(cfg.Favorites) > 0 {
			return nil, errors.New("fail")
		}
		return json.MarshalIndent(v, prefix, indent)
	}
	t.Cleanup(func() { jsonMarshalIndent = old })
	if _, err := cfg.AddFavorite(c.ID, "idx", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigUpdateTLSPreserve(t *testing.T) {
	cfg := newTestConfig(t)
	c, err := cfg.AddConnection(types.Connection{
		Name: "n", Host: "h", Port: 1,
		UseTLS:    true,
		TLSConfig: &types.TLSConfig{CAFile: "/ca.pem"},
	})
	if err != nil {
		t.Fatal(err)
	}
	c.UseTLS = false
	c.TLSConfig = nil
	updated, err := cfg.UpdateConnection(c)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.UseTLS || updated.TLSConfig == nil || updated.TLSConfig.CAFile != "/ca.pem" {
		t.Fatalf("%+v", updated)
	}
	_ = time.Now()
}

func TestConfigSavedQueriesAndBearerStrip(t *testing.T) {
	cfg := newTestConfig(t)
	conn, err := cfg.AddConnection(types.Connection{
		Name: "local", Host: "localhost", Port: 9200,
		Password: "secret", APIKey: "key", BearerToken: "bearer",
	})
	if err != nil {
		t.Fatal(err)
	}
	reloaded := reloadConfig(t, cfg)
	list, err := reloaded.ListConnections()
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Password != "" || list[0].APIKey != "" || list[0].BearerToken != "" {
		t.Fatal("secrets should be stripped")
	}

	// Update preserves bearer when blank
	cfg2 := newTestConfig(t)
	c, err := cfg2.AddConnection(types.Connection{Name: "a", Host: "h", Port: 9200, BearerToken: "bt"})
	if err != nil {
		t.Fatal(err)
	}
	c.BearerToken = ""
	c.Name = "b"
	updated, err := cfg2.UpdateConnection(c)
	if err != nil {
		t.Fatal(err)
	}
	if updated.BearerToken != "bt" || updated.Name != "b" {
		t.Fatalf("%+v", updated)
	}
	_ = conn

	q, err := cfg2.AddSavedQuery(types.SavedQuery{Name: "q1", Index: "i", Query: "*"})
	if err != nil || q.Name != "q1" || q.Created.IsZero() {
		t.Fatalf("%+v %v", q, err)
	}
	if len(cfg2.ListSavedQueries()) != 1 {
		t.Fatal("list")
	}
	q2, err := cfg2.AddSavedQuery(types.SavedQuery{Name: "q1", Index: "j", Query: "x"})
	if err != nil || q2.Index != "j" {
		t.Fatal(err)
	}
	if len(cfg2.ListSavedQueries()) != 1 {
		t.Fatal("replace")
	}
	reloaded2 := reloadConfig(t, cfg2)
	if len(reloaded2.ListSavedQueries()) != 1 || reloaded2.ListSavedQueries()[0].Query != "x" {
		t.Fatal("persist")
	}
	if err := cfg2.DeleteSavedQuery("q1"); err != nil {
		t.Fatal(err)
	}
	if len(cfg2.ListSavedQueries()) != 0 {
		t.Fatal("deleted")
	}
	if err := cfg2.DeleteSavedQuery("missing"); err != nil {
		t.Fatal(err)
	}
}
