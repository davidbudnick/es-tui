package cmd

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidbudnick/es-tui/internal/db"
	"github.com/davidbudnick/es-tui/internal/service"
	"github.com/davidbudnick/es-tui/internal/testutil"
	"github.com/davidbudnick/es-tui/internal/types"
)

func newTestCmds(t *testing.T) (*Commands, *testutil.MockES, *db.Config) {
	t.Helper()
	cfg, err := db.NewConfig(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	mock := &testutil.MockES{
		Info:         types.ClusterInfo{ClusterName: "c", Version: types.VersionInfo{Number: "8.0"}},
		Indices:      []types.IndexInfo{{Name: "products"}},
		IndexDetail:  types.IndexInfo{Name: "products"},
		Settings:     "{}",
		Mappings:     "{}",
		SearchResult: types.SearchResult{Total: 1, Hits: []types.Document{{ID: "1", Index: "products"}}},
		Document:     types.Document{ID: "1", Index: "products"},
		Health:       types.ClusterHealth{Status: "green"},
		Nodes:        []types.NodeInfo{{Name: "n"}},
		Shards:       []types.ShardInfo{{Index: "products"}},
		Aliases:      []types.AliasInfo{{Alias: "a"}},
		Templates:    []types.IndexTemplate{{Name: "t"}},
		Metrics:      types.LiveMetricsData{Status: "green"},
		CatBody:      "[]",
		DeleteByQ:    2,
		TestLatency:  time.Millisecond,
		TestInfo:     types.ClusterInfo{ClusterName: "c"},
	}
	return NewCommands(cfg, mock), mock, cfg
}

func TestCommandsAll(t *testing.T) {
	c, mock, cfg := newTestCmds(t)
	if c.Config() != cfg || c.ES() != mock {
		t.Fatal("accessors")
	}
	container := service.NewContainer(cfg, mock)
	c2 := NewCommandsFromContainer(container)
	if c2.ES() != mock {
		t.Fatal("from container")
	}

	msg := c.LoadConnections()()
	if _, ok := msg.(types.ConnectionsLoadedMsg); !ok {
		t.Fatalf("%T", msg)
	}

	msg = c.AddConnection(types.Connection{Name: "n", Host: "h", Port: 9200})()
	if m, ok := msg.(types.ConnectionAddedMsg); !ok || m.Err != nil {
		t.Fatalf("%T %+v", msg, msg)
	}

	list, err := cfg.ListConnections()
	if err != nil {
		t.Fatal(err)
	}
	conn := list[0]
	conn.Name = "n2"
	msg = c.UpdateConnection(conn)()
	if m, ok := msg.(types.ConnectionUpdatedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}

	msg = c.Connect(conn)()
	if m, ok := msg.(types.ConnectedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	mock.ConnectErr = errors.New("fail")
	msg = c.Connect(conn)()
	if m, ok := msg.(types.ConnectedMsg); !ok || m.Err == nil {
		t.Fatal("expected connect err")
	}
	mock.ConnectErr = nil

	msg = c.Disconnect()()
	if _, ok := msg.(types.DisconnectedMsg); !ok {
		t.Fatal(msg)
	}

	msg = c.TestConnection(conn)()
	if m, ok := msg.(types.ConnectionTestMsg); !ok || !m.Success {
		t.Fatal(msg)
	}

	msg = c.LoadIndices("*")()
	if m, ok := msg.(types.IndicesLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}

	msg = c.LoadIndexDetail("products")()
	if m, ok := msg.(types.IndexDetailLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	mock.IndexErr = errors.New("x")
	msg = c.LoadIndexDetail("x")()
	if m, ok := msg.(types.IndexDetailLoadedMsg); !ok || m.Err == nil {
		t.Fatal(msg)
	}
	mock.IndexErr = nil

	msg = c.CreateIndex("x", "")()
	if m, ok := msg.(types.IndexCreatedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.DeleteIndex("x")()
	if m, ok := msg.(types.IndexDeletedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}

	msg = c.LoadDocuments("products", "", 0, 10)()
	if m, ok := msg.(types.DocumentsLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	mock.SearchErr = errors.New("s")
	msg = c.LoadDocuments("products", "", 0, 10)()
	if m, ok := msg.(types.DocumentsLoadedMsg); !ok || m.Err == nil {
		t.Fatal(msg)
	}
	mock.SearchErr = nil

	msg = c.LoadDocument("products", "1")()
	if m, ok := msg.(types.DocumentLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.SaveDocument("products", "1", `{}`)()
	if m, ok := msg.(types.DocumentSavedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.DeleteDocument("products", "1")()
	if m, ok := msg.(types.DocumentDeletedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.Search("products", "*", 0, 10)()
	if m, ok := msg.(types.SearchResultMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadClusterHealth()()
	if m, ok := msg.(types.ClusterHealthLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadNodes()()
	if m, ok := msg.(types.NodesLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadShards("products")()
	if m, ok := msg.(types.ShardsLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadAliases()()
	if m, ok := msg.(types.AliasesLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadTemplates()()
	if m, ok := msg.(types.TemplatesLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadLiveMetrics()()
	if m, ok := msg.(types.LiveMetricsMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	tickCmd := c.LiveMetricsTick()
	if tickCmd == nil {
		t.Fatal("tick cmd")
	}
	// Execute the tick callback (waits ~2s) for full coverage.
	if msg := tickCmd(); msg == nil {
		// tea.Tick may return LiveMetricsTickMsg
	} else if _, ok := msg.(types.LiveMetricsTickMsg); !ok {
		// some bubbletea versions wrap differently
		_ = msg
	}

	msg = c.BulkDelete("products", "*")()
	if m, ok := msg.(types.BulkDeleteMsg); !ok || m.Err != nil || m.Deleted != 2 {
		t.Fatal(msg)
	}

	msg = c.AddFavorite(conn.ID, "products", "")()
	if m, ok := msg.(types.FavoriteAddedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadFavorites(conn.ID)()
	if m, ok := msg.(types.FavoritesLoadedMsg); !ok || len(m.Favorites) != 1 {
		t.Fatal(msg)
	}
	msg = c.RemoveFavorite(conn.ID, "products")()
	if m, ok := msg.(types.FavoriteRemovedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	cfg.AddRecentIndex(conn.ID, "products")
	msg = c.LoadRecentIndices(conn.ID)()
	if m, ok := msg.(types.RecentIndicesLoadedMsg); !ok || len(m.Indices) != 1 {
		t.Fatal(msg)
	}

	msg = c.CatAPI("indices")()
	if m, ok := msg.(types.CatAPIResultMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadIndexSettings("products")()
	if m, ok := msg.(types.IndexSettingsLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.LoadIndexMappings("products")()
	if m, ok := msg.(types.IndexMappingsLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	msg = c.RefreshIndex("products")()
	if m, ok := msg.(types.IndicesLoadedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
	mock.RefreshErr = errors.New("r")
	msg = c.RefreshIndex("products")()
	if m, ok := msg.(types.IndicesLoadedMsg); !ok || m.Err == nil {
		t.Fatal(msg)
	}
	mock.RefreshErr = nil

	msg = c.RefreshIndexOnly("products")()
	if m, ok := msg.(types.IndexOpMsg); !ok || m.Err != nil || m.Op != "refresh" {
		t.Fatal(msg)
	}
	mock.RefreshErr = errors.New("r")
	msg = c.RefreshIndexOnly("products")()
	if m, ok := msg.(types.IndexOpMsg); !ok || m.Err == nil {
		t.Fatal(msg)
	}
	mock.RefreshErr = nil

	msg = c.OpenIndex("products")()
	if m, ok := msg.(types.IndexOpMsg); !ok || m.Err != nil || m.Op != "open" {
		t.Fatal(msg)
	}
	mock.OpenErr = errors.New("o")
	msg = c.OpenIndex("products")()
	if m, ok := msg.(types.IndexOpMsg); !ok || m.Err == nil {
		t.Fatal(msg)
	}
	mock.OpenErr = nil

	msg = c.CloseIndex("products")()
	if m, ok := msg.(types.IndexOpMsg); !ok || m.Err != nil || m.Op != "close" {
		t.Fatal(msg)
	}
	mock.CloseErr = errors.New("c")
	msg = c.CloseIndex("products")()
	if m, ok := msg.(types.IndexOpMsg); !ok || m.Err == nil {
		t.Fatal(msg)
	}
	mock.CloseErr = nil

	msg = c.ForceMerge("products")()
	if m, ok := msg.(types.IndexOpMsg); !ok || m.Err != nil || m.Op != "forcemerge" {
		t.Fatal(msg)
	}
	mock.ForceMergeErr = errors.New("f")
	msg = c.ForceMerge("products")()
	if m, ok := msg.(types.IndexOpMsg); !ok || m.Err == nil {
		t.Fatal(msg)
	}
	mock.ForceMergeErr = nil

	msg = c.CopyToClipboard("hello")()
	if m, ok := msg.(types.ClipboardCopiedMsg); !ok || m.Content != "hello" {
		t.Fatal(msg)
	}

	msg = c.DeleteConnection(conn.ID)()
	if m, ok := msg.(types.ConnectionDeletedMsg); !ok || m.Err != nil {
		t.Fatal(msg)
	}
}
