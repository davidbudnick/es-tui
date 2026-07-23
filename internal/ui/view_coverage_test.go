package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestFullscreenViewsCoverage(t *testing.T) {
	m, _ := testModel(t)
	m.Width, m.Height = 140, 50
	m.CurrentConn = &types.Connection{Name: "local", Host: "localhost", Port: 9200}
	m.Flavor = types.FlavorElasticsearch
	m.ClusterInfo = types.ClusterInfo{Version: types.VersionInfo{Number: "8.17.0"}, Tagline: "You Know, for Search"}
	m.ClusterHealth = types.ClusterHealth{
		ClusterName: "es", Status: "yellow", NumberOfNodes: 1, NumberOfDataNodes: 1,
		ActiveShards: 10, ActivePrimaryShards: 5, UnassignedShards: 2,
		ActiveShardsPercentAsNumber: 80,
	}

	// Nodes: wide split + narrow list-only
	m.Nodes = []types.NodeInfo{
		{Name: "node-1", IP: "10.0.0.1", NodeRole: "dimr", Roles: []string{"d", "i"}, HeapPercent: 90, RamPercent: 50, CPU: 85, Master: "*", Version: "8.17", Host: "n1", Load1m: "0.5", DiskUsedPercent: "40", DiskUsed: "100gb", DiskTotal: "250gb", DiskAvail: "150gb"},
		{Name: "node-2", IP: "10.0.0.2", NodeRole: "d", HeapPercent: 40, RamPercent: 30, CPU: 10, Master: "-"},
	}
	m.SelectedNode = 1
	m.Screen = types.ScreenNodes
	if out := m.render(); out == "" {
		t.Fatal("nodes wide")
	}
	m.Width = 80
	if out := m.render(); out == "" {
		t.Fatal("nodes narrow")
	}
	m.Width = 140
	m.Nodes = nil
	_ = m.viewNodes()
	m.Nodes = []types.NodeInfo{{Name: "only", HeapPercent: 75, CPU: 60}}
	_ = m.buildNodePreview(40)

	// Shards with states
	m.Shards = []types.ShardInfo{
		{Index: "a", Shard: "0", Prirep: "p", State: "STARTED", Docs: "1", Store: "1kb", Node: "n1"},
		{Index: "b", Shard: "0", Prirep: "r", State: "RELOCATING", Docs: "0", Store: "0", Node: "n2"},
		{Index: "c", Shard: "1", Prirep: "p", State: "UNASSIGNED", Docs: "0", Store: "0", Node: ""},
		{Index: "d", Shard: "0", Prirep: "p", State: "INITIALIZING", Docs: "0", Store: "0", Node: "n1"},
	}
	m.DetailScroll = 3
	m.Screen = types.ScreenShards
	_ = m.render()
	m.Shards = nil
	_ = m.viewShards()

	// Health / metrics / aliases / templates
	m.Screen = types.ScreenClusterHealth
	_ = m.render()
	m.Screen = types.ScreenLiveMetrics
	m.LiveMetrics = &types.LiveMetrics{
		Latest:  types.LiveMetricsData{Status: "green", Nodes: 1, DataNodes: 1, ActiveShards: 5, UnassignedShards: 0, DocsCount: 100, StoreSizeBytes: 1024, QueryTotal: 50, SearchLatencyMs: 1.2, IndexingTotal: 10, JVMHeapUsedPct: 90, CPUPercent: 80},
		History: []types.LiveMetricsData{{QueryTotal: 1}, {QueryTotal: 50}, {QueryTotal: 0}},
	}
	_ = m.render()
	m.LiveMetrics = nil
	_ = m.viewLiveMetrics()

	m.Aliases = []types.AliasInfo{{Alias: "a", Index: "i"}, {Alias: "b", Index: "j"}}
	m.DetailScroll = 1
	m.Screen = types.ScreenAliases
	_ = m.render()
	m.Aliases = nil
	_ = m.viewAliases()

	m.Templates = []types.IndexTemplate{{Name: "t1", IndexPatterns: []string{"logs-*"}}, {Name: "t2", IndexPatterns: []string{"*"}}}
	m.DetailScroll = 1
	m.Screen = types.ScreenIndexTemplates
	_ = m.render()
	m.Templates = nil
	_ = m.viewTemplates()

	// Favorites / recent / logs / cat
	m.Favorites = []types.Favorite{{Index: "a", Label: "A"}, {Index: "b"}}
	m.SelectedFavIdx = 1
	m.Screen = types.ScreenFavorites
	_ = m.render()
	m.Favorites = nil
	_ = m.viewFavorites()

	m.RecentIndices = []types.RecentIndex{{Index: "a", AccessedAt: time.Now()}, {Index: "b", AccessedAt: time.Now()}}
	m.SelectedRecentIdx = 1
	m.Screen = types.ScreenRecentIndices
	_ = m.render()
	m.RecentIndices = nil
	_ = m.viewRecentIndices()

	m.Logs = types.NewLogWriter()
	_, _ = m.Logs.Write([]byte("line one\n"))
	m.Screen = types.ScreenLogs
	_ = m.render()
	m.Logs = nil
	_ = m.viewLogs()

	m.Inputs.CatInput.SetValue("indices")
	m.Inputs.CatInput.Focus()
	m.CatResult = strings.Repeat("row\n", 40)
	m.Screen = types.ScreenCatAPI
	_ = m.render()
	m.Inputs.CatInput.Blur()
	m.Inputs.CatInput.SetValue("")
	m.CatEndpoint = ""
	m.CatResult = ""
	_ = m.viewCatAPI()

	// JSON panel long body
	m.IndexSettings = "{\n  \"a\": 1\n}\n" + strings.Repeat("x\n", 50)
	m.DetailScroll = 2
	_ = m.viewJSONPanel("Settings", m.IndexSettings)
	_ = m.viewJSONPanel("Empty", "")

	// Admin tables
	m.Allocation = []types.AllocationInfo{
		{Node: "n1", IP: "1.1.1.1", DiskUsed: "10gb", DiskAvail: "90gb", DiskPercent: "95%", Shards: "5"},
		{Node: "n2", IP: "1.1.1.2", DiskUsed: "10gb", DiskAvail: "90gb", DiskPercent: "85%", Shards: "3"},
		{Node: "n3", IP: "1.1.1.3", DiskUsed: "1gb", DiskAvail: "99gb", DiskPercent: "10%", Shards: "1"},
	}
	m.Screen = types.ScreenAllocation
	_ = m.render()
	m.Allocation = nil
	_ = m.viewAllocation()

	m.Tasks = []types.TaskInfo{
		{ID: "1", Action: "indices:data/write/reindex", RunningTime: "1s", Node: "n1"},
		{ID: "2", Action: "cluster:monitor/tasks", RunningTime: "2s", Node: "n2"},
	}
	m.SelectedTaskIdx = 1
	m.Screen = types.ScreenTasks
	_ = m.render()
	m.Tasks = nil
	_ = m.viewTasks()

	m.Plugins = []types.PluginInfo{{Name: "analysis-icu", Component: "analysis-icu", Version: "8"}, {Name: "p2", Component: "c", Version: "1"}}
	m.Screen = types.ScreenPlugins
	_ = m.render()
	m.Plugins = nil
	_ = m.viewPlugins()

	m.DataStreams = []types.DataStreamInfo{
		{Name: "logs", Status: "GREEN", Generation: "3", Template: "logs-template"},
		{Name: "metrics", Status: "YELLOW", Generation: "1", Template: "m"},
		{Name: "bad", Status: "RED", Generation: "0", Template: "b"},
	}
	m.Screen = types.ScreenDataStreams
	_ = m.render()
	m.DataStreams = nil
	_ = m.viewDataStreams()

	m.Inputs.SnapshotRepo.SetValue("backup")
	m.Inputs.SnapshotRepo.Focus()
	m.Snapshots = []types.SnapshotInfo{
		{Snapshot: "snap1", State: "SUCCESS", Repository: "backup", StartTime: "now"},
		{Snapshot: "snap2", State: "FAILED", Repository: "backup", StartTime: "then"},
		{Snapshot: "snap3", State: "IN_PROGRESS", Repository: "backup", StartTime: "soon"},
	}
	m.Screen = types.ScreenSnapshots
	_ = m.render()
	m.Inputs.SnapshotRepo.Blur()
	m.Inputs.SnapshotRepo.SetValue("")
	m.Snapshots = nil
	_ = m.viewSnapshots()

	m.SavedQueries = []types.SavedQuery{{Name: "q1", Index: "a", Query: "foo"}, {Name: "q2", Index: "b", Query: "bar"}}
	m.SelectedSQIdx = 1
	m.Screen = types.ScreenSavedQueries
	_ = m.render()
	m.SavedQueries = nil
	_ = m.viewSavedQueries()

	m.ExplainResult = &types.ExplainResult{Matched: true, Explanation: strings.Repeat("line\n", 30)}
	m.Screen = types.ScreenExplain
	_ = m.render()
	m.ExplainResult = &types.ExplainResult{Matched: false, Raw: "raw"}
	_ = m.viewExplain()
	m.ExplainResult = nil
	_ = m.viewExplain()

	m.Screen = types.ScreenReindex
	m.ReadOnly = true
	_ = m.render()
	m.ReadOnly = false
	m.ReindexFocus = 1
	_ = m.viewReindex()

	m.Screen = types.ScreenExport
	m.SearchIndex = "products"
	m.SearchQuery = "status:active"
	_ = m.render()
	m.SearchIndex = ""
	m.SearchQuery = ""
	m.DocQuery = ""
	m.CurrentIndex = nil
	_ = m.viewExport()

	m.Screen = types.ScreenCommandPalette
	m.PaletteItems = defaultPaletteItems()
	m.Inputs.PaletteInput.SetValue("health")
	m.PaletteIdx = 0
	_ = m.render()
	m.Inputs.PaletteInput.SetValue("zzzz-no-match")
	_ = m.viewCommandPalette()
	m.Inputs.PaletteInput.SetValue("")
	m.PaletteItems = nil
	_ = m.filteredPalette()

	m.Screen = types.ScreenClusterSettings
	m.ClusterSettings = `{"persistent":{}}`
	_ = m.render()
}

func TestIndicesDocumentsSearchCoverage(t *testing.T) {
	m, _ := testModel(t)
	m.Width, m.Height = 140, 50
	m.CurrentConn = &types.Connection{Name: "local"}

	// Indices wide + narrow
	m.Indices = []types.IndexInfo{
		{Name: "products", Health: "green", Status: "open", DocsCount: 10, StoreSize: "1mb", PrimaryShards: 1, ReplicaShards: 1, UUID: "u1", IsFavorite: true},
		{Name: "orders", Health: "yellow", Status: "open", DocsCount: 5, StoreSize: "2mb"},
	}
	m.SelectedIndexIdx = 1
	m.Screen = types.ScreenIndices
	_ = m.render()
	m.Width = 80
	_ = m.viewIndices()
	_ = m.viewIndicesListOnly()
	m.Width = 140
	m.Indices = nil
	_ = m.buildIndicesListPanel(60)
	m.Indices = []types.IndexInfo{{Name: "x", Health: "red", Status: "close", PriStoreSize: "1kb", UUID: "abc", IsFavorite: true}}
	m.SelectedIndexIdx = 0
	_ = m.buildIndexPreviewPanel(40)
	m.CurrentIndex = nil
	_ = m.viewIndexDetail()
	idx := m.Indices[0]
	m.CurrentIndex = &idx
	m.Width = 30
	_ = m.viewIndexDetail()
	m.Width = 140

	// Documents with rich source for columns
	m.Documents = []types.Document{
		{ID: "1", Index: "products", Score: 1.2, Source: map[string]any{"name": "Widget", "email": "a@b.c", "status": "active", "price": float64(10), "tags": []any{"a", "b", "c", "d"}, "active": true, "nested": map[string]any{"x": 1}}, Raw: `{"name":"Widget"}`},
		{ID: "2", Index: "products", Score: 0.5, Source: map[string]any{"name": "Gadget", "email": "x@y.z", "status": "pending", "price": 3.14, "active": false}, Raw: ""},
		{ID: "3", Index: "products", Source: nil, Raw: "raw-only\nline"},
		{ID: "4", Index: "products", Source: map[string]any{"zzz": "only"}},
	}
	m.SelectedDocIdx = 0
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Screen = types.ScreenDocuments
	_ = m.render()
	m.Width = 80
	_ = m.viewDocuments()
	_ = m.viewDocumentsListOnly()
	m.Width = 140
	_ = m.buildDocumentPreviewPanel(40)
	m.Documents = nil
	_ = m.buildDocumentPreviewPanel(40)
	m.Documents = []types.Document{{ID: "z", Source: map[string]any{}}}
	_ = docSummary(m.Documents[0])
	_ = docSummary(types.Document{ID: "id", Source: nil, Raw: ""})
	_ = docSummary(types.Document{ID: "id", Source: nil, Raw: "raw"})
	_ = pickDocumentListColumns(nil, 80)
	_ = pickDocumentListColumns([]types.Document{
		{ID: "1", Source: map[string]any{"name": "Widget", "category": "hardware", "brand": "elastic", "price": 10.0}},
		{ID: "2", Source: map[string]any{"name": "Mug", "category": "merch", "brand": "kibana", "price": 14.0}},
	}, 100)
	_ = docsHaveUsefulScores([]types.Document{{Score: 1}, {Score: 1}})
	_ = docsHaveUsefulScores([]types.Document{{Score: 0.2}, {Score: 3.5}})
	_ = humanizeField("customer_id")
	_ = mostCommonField(map[string]int{"a": 3, "b": 1}, 2, map[string]bool{}, false)
	_ = fieldString(nil, "x")
	_ = fieldString(map[string]any{}, "x")
	_ = fieldString(map[string]any{"k": nil}, "k")
	_ = fieldString(map[string]any{"k": "s"}, "k")
	_ = fieldString(map[string]any{"k": float64(3)}, "k")
	_ = fieldString(map[string]any{"k": 3.14}, "k")
	_ = fieldString(map[string]any{"k": true}, "k")
	_ = fieldString(map[string]any{"k": false}, "k")
	_ = fieldString(map[string]any{"k": []any{"a", "b", "c", "d"}}, "k")
	_ = fieldString(map[string]any{"k": map[string]any{"long": strings.Repeat("x", 50)}}, "k")
	_ = isScalarish("s")
	_ = isScalarish(float64(1))
	_ = isScalarish(true)
	_ = isScalarish(1)
	_ = isScalarish(int64(1))
	_ = isScalarish([]any{})
	_ = isScalarish(map[string]any{})
	_ = titleCase("")
	_ = titleCase("hello")

	m.CurrentDocument = &types.Document{ID: "1", Index: "p", Raw: `{"a":1}`, Source: map[string]any{"a": 1}}
	m.Screen = types.ScreenDocumentDetail
	_ = m.render()
	m.Width = 20
	_ = m.viewDocumentDetail()
	_ = detailBoxWidth(20)
	_ = detailBoxWidth(80)
	_ = detailBoxWidth(200)
	m.Width = 140
	m.CurrentDocument = nil
	_ = m.viewDocumentDetail()

	// Search wide + compact
	m.SearchResult = &types.SearchResult{
		Total: 2,
		Hits: []types.Document{
			{ID: "1", Index: "products", Score: 1, Source: map[string]any{"name": "A"}},
			{ID: "2", Index: "products", Score: 0.5, Source: map[string]any{"name": "B"}},
		},
	}
	m.SelectedDocIdx = 1
	m.SearchIndex = "products"
	m.SearchQuery = "name:A"
	m.Screen = types.ScreenSearch
	_ = m.render()
	m.Width = 80
	_ = m.viewSearch()
	_ = m.viewSearchCompact()
	m.Width = 140
	m.SearchResult = nil
	_ = m.buildSearchResultsPanel(60)
	_ = m.buildSearchPreviewPanel(40)
	m.SearchResult = &types.SearchResult{Total: 0, Hits: nil}
	_ = m.buildSearchResultsPanel(60)
}

func TestLayoutHelpersCoverage(t *testing.T) {
	m, _ := testModel(t)
	m.Width, m.Height = 120, 40

	_ = m.fullScreenFrame("body")
	_ = m.fullScreenFrame("body", keyDesc{"help", ""})
	// short body pads
	m.Height = 30
	_ = m.fullScreenFrame("x", keyDesc{"h", ""})

	_ = m.splitBrowse(0, "L", "R")
	_ = m.splitBrowse(60, "L", "R")
	m.Width = 50
	_ = m.splitBrowse(60, "L", "R")
	m.Width = 120

	_ = m.listHeader("Title")
	_ = m.tableSep(10)
	_ = m.tableSep(1000)

	_ = heapColor(90)
	_ = heapColor(75)
	_ = heapColor(10)
	_ = cpuColor(90)
	_ = cpuColor(60)
	_ = cpuColor(10)
	_ = masterBadge("*")
	_ = masterBadge("-")
	_ = padRight("hi", 5)
	_ = padRight("hello-world", 3)
	_ = fmtInt(42)

	_ = parsePct("95%")
	_ = parsePct("not-a-number")
	_ = parsePct(" 12.5 ")
}
