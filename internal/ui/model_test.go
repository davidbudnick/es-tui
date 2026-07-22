package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/cmd"
	"github.com/davidbudnick/es-tui/internal/db"
	"github.com/davidbudnick/es-tui/internal/testutil"
	"github.com/davidbudnick/es-tui/internal/types"
)

func testModel(t *testing.T) (Model, *testutil.MockES) {
	t.Helper()
	cfg, err := db.NewConfig(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	mock := &testutil.MockES{
		Info: types.ClusterInfo{
			ClusterName: "test",
			Version:     types.VersionInfo{Number: "8.17.0"},
			Flavor:      types.FlavorElasticsearch,
			Tagline:     "You Know, for Search",
		},
		Indices: []types.IndexInfo{
			{Name: "products", Health: "green", Status: "open", DocsCount: 4, StoreSize: "1kb", PrimaryShards: 1},
			{Name: "orders", Health: "yellow", Status: "open", DocsCount: 3, StoreSize: "2kb"},
		},
		IndexDetail: types.IndexInfo{Name: "products", Health: "green", Status: "open", UUID: "u"},
		Settings:    `{"products":{"settings":{}}}`,
		Mappings:    `{"products":{"mappings":{}}}`,
		SearchResult: types.SearchResult{
			Total: 1,
			Hits:  []types.Document{{Index: "products", ID: "1", Score: 1, Raw: `{"name":"w"}`}},
		},
		Document:  types.Document{Index: "products", ID: "1", Raw: `{"name":"w"}`, Source: map[string]any{"name": "w"}},
		Health:    types.ClusterHealth{ClusterName: "test", Status: "green", NumberOfNodes: 1, ActiveShards: 1},
		Nodes:     []types.NodeInfo{{Name: "n1", IP: "127.0.0.1", NodeRole: "dimr", Master: "*"}},
		Shards:    []types.ShardInfo{{Index: "products", Shard: "0", State: "STARTED"}},
		Aliases:   []types.AliasInfo{{Alias: "p", Index: "products"}},
		Templates: []types.IndexTemplate{{Name: "t", IndexPatterns: []string{"logs-*"}}},
		Metrics:   types.LiveMetricsData{Status: "green", Nodes: 1, DocsCount: 10, QueryTotal: 5},
		CatBody:   `[{"index":"products"}]`,
		DeleteByQ: 2,
	}
	m := NewModel()
	m.Width = 100
	m.Height = 40
	m.Version = "test"
	m.Cmds = cmd.NewCommands(cfg, mock)
	m.Logs = types.NewLogWriter()
	send := func(tea.Msg) {}
	m.SendFunc = &send
	return m, mock
}

func TestNewModelInit(t *testing.T) {
	m, _ := testModel(t)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected init cmd")
	}
	// with CLI connection
	m2, _ := testModel(t)
	m2.CLIConnection = &types.Connection{Host: "localhost", Port: 9200}
	if m2.Init() == nil {
		t.Fatal("expected batch")
	}
}

func TestViewsRender(t *testing.T) {
	m, _ := testModel(t)
	m.Connections = []types.Connection{{Name: "local", Host: "localhost", Port: 9200, Flavor: types.FlavorAuto}}
	screens := []types.Screen{
		types.ScreenConnections,
		types.ScreenAddConnection,
		types.ScreenIndices,
		types.ScreenIndexDetail,
		types.ScreenDocuments,
		types.ScreenDocumentDetail,
		types.ScreenSearch,
		types.ScreenHelp,
		types.ScreenConfirmDelete,
		types.ScreenClusterHealth,
		types.ScreenNodes,
		types.ScreenIndexCreate,
		types.ScreenIndexSettings,
		types.ScreenIndexMappings,
		types.ScreenAliases,
		types.ScreenShards,
		types.ScreenLiveMetrics,
		types.ScreenTestConnection,
		types.ScreenLogs,
		types.ScreenFavorites,
		types.ScreenRecentIndices,
		types.ScreenBulkDelete,
		types.ScreenEditDocument,
		types.ScreenIndexTemplates,
		types.ScreenCatAPI,
	}
	idx := types.IndexInfo{Name: "products", Health: "green", Status: "open"}
	m.CurrentIndex = &idx
	m.CurrentDocument = &types.Document{Index: "products", ID: "1", Raw: `{"a":1}`}
	m.Indices = []types.IndexInfo{idx}
	m.Documents = []types.Document{{ID: "1", Index: "products", Score: 1}}
	m.SearchResult = &types.SearchResult{Total: 1, Hits: m.Documents}
	m.ClusterHealth = types.ClusterHealth{Status: "green", ClusterName: "c"}
	m.Nodes = []types.NodeInfo{{Name: "n"}}
	m.Shards = []types.ShardInfo{{Index: "products"}}
	m.Aliases = []types.AliasInfo{{Alias: "a", Index: "products"}}
	m.Templates = []types.IndexTemplate{{Name: "t"}}
	m.LiveMetrics = &types.LiveMetrics{Latest: types.LiveMetricsData{Status: "green", QueryTotal: 1}, History: []types.LiveMetricsData{{QueryTotal: 1}, {QueryTotal: 2}}}
	m.Favorites = []types.Favorite{{Index: "products"}}
	m.RecentIndices = []types.RecentIndex{{Index: "products"}}
	m.IndexSettings = `{"a":1}`
	m.IndexMappings = `{"b":2}`
	m.CatResult = "[]"
	m.TestConnResult = "ok"
	m.ConfirmType = "index"
	m.ConfirmData = "products"
	m.CurrentConn = &types.Connection{Name: "local", Host: "h", Port: 9200}
	m.Flavor = types.FlavorElasticsearch
	m.ClusterInfo = types.ClusterInfo{Version: types.VersionInfo{Number: "8"}, Tagline: "x"}

	for _, s := range screens {
		m.Screen = s
		view := m.View()
		if view.Content == "" && m.render() == "" {
			// View wraps render
		}
		out := m.render()
		if out == "" {
			t.Fatalf("empty view for %s", s)
		}
	}

	// empty states
	m.Screen = types.ScreenConnections
	m.Connections = nil
	m.ConnectionError = "boom"
	if m.render() == "" {
		t.Fatal("empty conn")
	}
	m.Screen = types.ScreenIndices
	m.Indices = nil
	if m.render() == "" {
		t.Fatal("empty indices")
	}
	m.Screen = types.ScreenDocuments
	m.Documents = nil
	if m.render() == "" {
		t.Fatal("empty docs")
	}
	m.Screen = types.ScreenDocumentDetail
	m.CurrentDocument = nil
	if m.render() == "" {
		t.Fatal("no doc")
	}
	m.Screen = types.ScreenIndexDetail
	m.CurrentIndex = nil
	if m.render() == "" {
		t.Fatal("no index")
	}
	m.Screen = types.ScreenLiveMetrics
	m.LiveMetrics = nil
	if m.render() == "" {
		t.Fatal("metrics nil")
	}
	m.Screen = types.ScreenLogs
	m.Logs = nil
	if m.render() == "" {
		t.Fatal("logs nil")
	}
	m.Logs = types.NewLogWriter()
	_, _ = m.Logs.Write([]byte("line"))
	m.Screen = types.ScreenLogs
	if m.render() == "" {
		t.Fatal("logs")
	}
	m.Screen = types.ScreenFavorites
	m.Favorites = nil
	_ = m.render()
	m.Screen = types.ScreenRecentIndices
	m.RecentIndices = nil
	_ = m.render()
	m.Screen = types.ScreenAliases
	m.Aliases = nil
	_ = m.render()
	m.Screen = types.ScreenIndexTemplates
	m.Templates = nil
	_ = m.render()
	m.Screen = types.ScreenIndexSettings
	m.IndexSettings = ""
	_ = m.render()

	// too small
	m.Width = 10
	m.Height = 5
	if m.render() == "" {
		t.Fatal("small")
	}

	// status bar variants
	m.Width = 100
	m.Height = 40
	m.Loading = true
	_ = m.getStatusBar()
	m.Loading = false
	m.StatusMsg = "hi"
	_ = m.getStatusBar()
	m.StatusMsg = ""
	m.Err = errString("e")
	_ = m.getStatusBar()
	m.Err = nil
	m.UpdateAvailable = "v2"
	_ = m.getStatusBar()
}

type errString string

func (e errString) Error() string { return string(e) }
