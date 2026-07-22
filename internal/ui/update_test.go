package ui

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func key(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 0, Text: s}
}

// tea.KeyPressMsg String() - need proper construction
// In bubbletea v2, use tea.KeyMsg helpers. Let's use type that works with handleKeyPress.

func press(m Model, k string) Model {
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = nm.(Model)
	nm, _ = m.handleKeyPress(makeKey(k))
	return nm.(Model)
}

func makeKey(name string) tea.KeyPressMsg {
	// Map common names to KeyPressMsg
	switch name {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	default:
		if len(name) == 1 {
			return tea.KeyPressMsg{Text: name, Code: []rune(name)[0]}
		}
		return tea.KeyPressMsg{Text: name}
	}
}

func TestUpdateMessages(t *testing.T) {
	m, mock := testModel(t)

	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = nm.(Model)
	if m.Width != 120 {
		t.Fatal(m.Width)
	}

	// Connections loaded
	nm, _ = m.Update(types.ConnectionsLoadedMsg{Connections: []types.Connection{{ID: 1, Name: "a", Host: "h", Port: 9200}}})
	m = nm.(Model)
	if len(m.Connections) != 1 {
		t.Fatal("conns")
	}
	nm, _ = m.Update(types.ConnectionsLoadedMsg{Err: errors.New("x")})
	m = nm.(Model)
	if m.Err == nil {
		t.Fatal("err")
	}

	// Connection added/updated/deleted
	nm, _ = m.Update(types.ConnectionAddedMsg{Connection: types.Connection{ID: 2, Name: "b", Host: "h", Port: 1}})
	m = nm.(Model)
	nm, _ = m.Update(types.ConnectionAddedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, _ = m.Update(types.ConnectionUpdatedMsg{Connection: types.Connection{ID: 1, Name: "aa", Host: "h", Port: 9200}})
	m = nm.(Model)
	nm, _ = m.Update(types.ConnectionUpdatedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, _ = m.Update(types.ConnectionDeletedMsg{ID: 2})
	m = nm.(Model)
	nm, _ = m.Update(types.ConnectionDeletedMsg{Err: errors.New("e")})
	m = nm.(Model)

	// Auto connect + connected
	nm, cmd := m.Update(types.AutoConnectMsg{Connection: types.Connection{Host: "h", Port: 9200}})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("connect cmd")
	}
	nm, cmd = m.Update(types.ConnectedMsg{Info: mock.Info})
	m = nm.(Model)
	if m.Screen != types.ScreenIndices || cmd == nil {
		t.Fatalf("screen=%v", m.Screen)
	}
	nm, _ = m.Update(types.ConnectedMsg{Err: errors.New("fail")})
	m = nm.(Model)
	if m.ConnectionError == "" {
		t.Fatal("conn err")
	}

	// Indices
	m.Screen = types.ScreenIndices
	m.CurrentConn = &types.Connection{ID: 1, Name: "a", Host: "h", Port: 9200}
	// add favorite for mark
	_, _ = m.Cmds.Config().AddFavorite(1, "products", "")
	nm, _ = m.Update(types.IndicesLoadedMsg{Indices: mock.Indices})
	m = nm.(Model)
	if len(m.Indices) != 2 {
		t.Fatal(m.Indices)
	}
	nm, _ = m.Update(types.IndicesLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)

	// Index detail
	nm, _ = m.Update(types.IndexDetailLoadedMsg{Index: mock.IndexDetail, Settings: "{}", Mappings: "{}"})
	m = nm.(Model)
	if m.Screen != types.ScreenIndexDetail {
		t.Fatal(m.Screen)
	}
	nm, _ = m.Update(types.IndexDetailLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)

	// Index created/deleted
	nm, cmd = m.Update(types.IndexCreatedMsg{Name: "x"})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("reload")
	}
	nm, _ = m.Update(types.IndexCreatedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, cmd = m.Update(types.IndexDeletedMsg{Name: "x"})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("reload del")
	}
	nm, _ = m.Update(types.IndexDeletedMsg{Err: errors.New("e")})
	m = nm.(Model)

	// Documents
	nm, _ = m.Update(types.DocumentsLoadedMsg{Index: "products", Documents: mock.SearchResult.Hits, Total: 1})
	m = nm.(Model)
	if m.Screen != types.ScreenDocuments {
		t.Fatal(m.Screen)
	}
	nm, _ = m.Update(types.DocumentsLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)

	nm, _ = m.Update(types.DocumentLoadedMsg{Document: mock.Document})
	m = nm.(Model)
	if m.Screen != types.ScreenDocumentDetail {
		t.Fatal(m.Screen)
	}
	nm, _ = m.Update(types.DocumentLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)

	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	nm, cmd = m.Update(types.DocumentSavedMsg{Index: "products", ID: "1"})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("reload docs")
	}
	nm, _ = m.Update(types.DocumentSavedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, cmd = m.Update(types.DocumentDeletedMsg{Index: "products", ID: "1"})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("reload after del doc")
	}
	nm, _ = m.Update(types.DocumentDeletedMsg{Err: errors.New("e")})
	m = nm.(Model)

	// Search
	nm, _ = m.Update(types.SearchResultMsg{Result: mock.SearchResult})
	m = nm.(Model)
	nm, _ = m.Update(types.SearchResultMsg{Err: errors.New("e")})
	m = nm.(Model)

	// Cluster/nodes/shards/aliases/templates
	nm, _ = m.Update(types.ClusterHealthLoadedMsg{Health: mock.Health})
	m = nm.(Model)
	nm, _ = m.Update(types.ClusterHealthLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, _ = m.Update(types.NodesLoadedMsg{Nodes: mock.Nodes})
	m = nm.(Model)
	nm, _ = m.Update(types.NodesLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, _ = m.Update(types.ShardsLoadedMsg{Shards: mock.Shards})
	m = nm.(Model)
	nm, _ = m.Update(types.ShardsLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, _ = m.Update(types.AliasesLoadedMsg{Aliases: mock.Aliases})
	m = nm.(Model)
	nm, _ = m.Update(types.AliasesLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, _ = m.Update(types.TemplatesLoadedMsg{Templates: mock.Templates})
	m = nm.(Model)
	nm, _ = m.Update(types.TemplatesLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)

	// Test connection
	nm, _ = m.Update(types.ConnectionTestMsg{Success: true, Latency: time.Millisecond, Info: mock.Info})
	m = nm.(Model)
	nm, _ = m.Update(types.ConnectionTestMsg{Err: errors.New("fail"), Latency: time.Millisecond})
	m = nm.(Model)

	// Favorites / recent
	nm, _ = m.Update(types.FavoritesLoadedMsg{Favorites: []types.Favorite{{Index: "products"}}})
	m = nm.(Model)
	nm, cmd = m.Update(types.FavoriteAddedMsg{Favorite: types.Favorite{Index: "products"}})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("fav reload")
	}
	m.Screen = types.ScreenFavorites
	nm, cmd = m.Update(types.FavoriteRemovedMsg{Index: "products"})
	m = nm.(Model)
	nm, _ = m.Update(types.RecentIndicesLoadedMsg{Indices: []types.RecentIndex{{Index: "products"}}})
	m = nm.(Model)

	// Bulk delete
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	nm, cmd = m.Update(types.BulkDeleteMsg{Index: "products", Deleted: 2})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("bulk reload")
	}
	nm, _ = m.Update(types.BulkDeleteMsg{Err: errors.New("e")})
	m = nm.(Model)

	// Live metrics
	m.LiveMetricsActive = true
	nm, cmd = m.Update(types.LiveMetricsMsg{Data: mock.Metrics})
	m = nm.(Model)
	if m.LiveMetrics == nil || cmd == nil {
		t.Fatal("metrics")
	}
	// fill history over 60
	for i := 0; i < 65; i++ {
		nm, _ = m.Update(types.LiveMetricsMsg{Data: mock.Metrics})
		m = nm.(Model)
	}
	nm, _ = m.Update(types.LiveMetricsMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, cmd = m.Update(types.LiveMetricsTickMsg{})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("tick")
	}
	m.LiveMetricsActive = false
	nm, cmd = m.Update(types.LiveMetricsTickMsg{})
	m = nm.(Model)
	if cmd != nil {
		t.Fatal("no tick when inactive")
	}

	// settings/mappings/cat
	nm, _ = m.Update(types.IndexSettingsLoadedMsg{Settings: "{}"})
	m = nm.(Model)
	nm, _ = m.Update(types.IndexSettingsLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, _ = m.Update(types.IndexMappingsLoadedMsg{Mappings: "{}"})
	m = nm.(Model)
	nm, _ = m.Update(types.IndexMappingsLoadedMsg{Err: errors.New("e")})
	m = nm.(Model)
	nm, _ = m.Update(types.CatAPIResultMsg{Body: "[]", Endpoint: "indices"})
	m = nm.(Model)
	nm, _ = m.Update(types.CatAPIResultMsg{Err: errors.New("e")})
	m = nm.(Model)

	nm, _ = m.Update(types.UpdateAvailableMsg{LatestVersion: "v1.0.0", UpgradeCmd: "brew"})
	m = nm.(Model)
	if m.UpdateAvailable != "v1.0.0" {
		t.Fatal(m.UpdateAvailable)
	}
	nm, _ = m.Update(types.DisconnectedMsg{})
	m = nm.(Model)
	if m.Screen != types.ScreenConnections {
		t.Fatal(m.Screen)
	}

	// unknown message
	nm, _ = m.Update(struct{}{})
	_ = nm
}

func TestKeyHandlersConnections(t *testing.T) {
	m, _ := testModel(t)
	m.Width, m.Height = 100, 40
	m.Connections = []types.Connection{
		{ID: 1, Name: "a", Host: "localhost", Port: 9200, Flavor: types.FlavorAuto},
		{ID: 2, Name: "b", Host: "localhost", Port: 9201, Flavor: types.FlavorOpenSearch, UseTLS: true},
	}

	m = applyKey(m, "j")
	if m.SelectedConnIdx != 1 {
		t.Fatal(m.SelectedConnIdx)
	}
	m = applyKey(m, "k")
	if m.SelectedConnIdx != 0 {
		t.Fatal(m.SelectedConnIdx)
	}
	m = applyKey(m, "G")
	if m.SelectedConnIdx != 1 {
		t.Fatal(m.SelectedConnIdx)
	}
	m = applyKey(m, "g")
	if m.SelectedConnIdx != 0 {
		t.Fatal(m.SelectedConnIdx)
	}
	m = applyKey(m, "a")
	if m.Screen != types.ScreenAddConnection {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
	m = applyKey(m, "e")
	if m.Screen != types.ScreenEditConnection {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
	m = applyKey(m, "d")
	if m.Screen != types.ScreenConfirmDelete {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "n")
	m = applyKey(m, "t")
	// test starts loading
	m.Screen = types.ScreenConnections
	m = applyKey(m, "?")
	if m.Screen != types.ScreenHelp {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
	m = applyKey(m, "L")
	if m.Screen != types.ScreenLogs {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
	m = applyKey(m, "enter")
	// connect
}

func applyKey(m Model, k string) Model {
	nm, _ := m.handleKeyPress(makeKey(k))
	return nm.(Model)
}

func TestKeyHandlersIndicesAndMore(t *testing.T) {
	m, _ := testModel(t)
	m.Width, m.Height = 100, 40
	m.Screen = types.ScreenIndices
	m.CurrentConn = &types.Connection{ID: 1, Name: "a", Host: "h", Port: 9200}
	m.Indices = []types.IndexInfo{
		{Name: "products", Health: "green"},
		{Name: "orders", Health: "yellow"},
	}
	m.Flavor = types.FlavorElasticsearch

	m = applyKey(m, "j")
	m = applyKey(m, "k")
	m = applyKey(m, "G")
	m = applyKey(m, "g")
	m = applyKey(m, "f")
	if !m.Inputs.PatternInput.Focused() {
		t.Fatal("filter focus")
	}
	m.Inputs.PatternInput.SetValue("prod*")
	m = applyKey(m, "enter")
	m = applyKey(m, "f")
	m = applyKey(m, "esc")
	m = applyKey(m, "r")
	m = applyKey(m, "a")
	if m.Screen != types.ScreenIndexCreate {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
	m = applyKey(m, "d")
	if m.Screen != types.ScreenConfirmDelete {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "n")
	m = applyKey(m, "i")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "enter")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "c")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "n")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "m")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "s")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "A")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "T")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "C")
	if m.Screen != types.ScreenCatAPI {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
	m = applyKey(m, "*")
	m = applyKey(m, "*") // unfavorite
	m = applyKey(m, "F")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "R")
	m.Screen = types.ScreenIndices
	m = applyKey(m, "?")
	m = applyKey(m, "esc")
	m = applyKey(m, "L")
	m = applyKey(m, "esc")
	m = applyKey(m, "/")
	if m.Screen != types.ScreenSearch {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
}

func TestKeyHandlersFormsAndDocs(t *testing.T) {
	m, _ := testModel(t)
	m.Width, m.Height = 100, 40

	// Add connection form
	m.Screen = types.ScreenAddConnection
	m.ConnInputs = createConnectionInputs()
	m.ConnFocusIdx = 0
	m.ConnInputs[0].Focus()
	m.ConnInputs[0].SetValue("local")
	m = applyKey(m, "tab")
	m.ConnInputs[1].SetValue("localhost")
	m = applyKey(m, "tab")
	m.ConnInputs[2].SetValue("9200")
	m = applyKey(m, "shift+tab")
	m = applyKey(m, "enter") // should save

	// Invalid host
	m.Screen = types.ScreenAddConnection
	m.ConnInputs = createConnectionInputs()
	m.ConnFocusIdx = 1
	m.ConnInputs[1].Focus()
	m.ConnInputs[1].SetValue("")
	m = applyKey(m, "enter")
	if m.Err == nil {
		// host required - connectionFromInputs
	}

	// Documents
	m.Screen = types.ScreenDocuments
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Documents = []types.Document{{ID: "1", Index: "products", Raw: `{"a":1}`}}
	m = applyKey(m, "j")
	m = applyKey(m, "k")
	m = applyKey(m, "enter")
	m.Screen = types.ScreenDocuments
	m = applyKey(m, "/")
	m = applyKey(m, "esc")
	m = applyKey(m, "e")
	if m.Screen != types.ScreenEditDocument {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
	m = applyKey(m, "d")
	m = applyKey(m, "n")
	m = applyKey(m, "D")
	if m.Screen != types.ScreenBulkDelete {
		t.Fatal(m.Screen)
	}
	m = applyKey(m, "esc")
	m = applyKey(m, "r")

	// Document detail
	m.Screen = types.ScreenDocumentDetail
	m.CurrentDocument = &types.Document{ID: "1", Index: "products", Raw: "{\n\"a\":1\n}"}
	m = applyKey(m, "j")
	m = applyKey(m, "k")
	m = applyKey(m, "e")
	m = applyKey(m, "esc")
	m.Screen = types.ScreenDocumentDetail
	m = applyKey(m, "d")
	m = applyKey(m, "y")

	// Index detail
	m.Screen = types.ScreenIndexDetail
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m = applyKey(m, "s")
	m.Screen = types.ScreenIndexDetail
	m = applyKey(m, "m")
	m.Screen = types.ScreenIndexDetail
	m = applyKey(m, "/")
	m = applyKey(m, "esc")
	m.Screen = types.ScreenIndexDetail
	m = applyKey(m, "enter")
	m.Screen = types.ScreenIndexDetail
	m = applyKey(m, "d")
	m = applyKey(m, "n")
	m.Screen = types.ScreenIndexDetail
	m = applyKey(m, "esc")

	// Confirm delete connection
	m.Screen = types.ScreenConfirmDelete
	m.ConfirmType = "connection"
	m.ConfirmData = types.Connection{ID: 1}
	m = applyKey(m, "y")

	// Confirm index
	m.Screen = types.ScreenConfirmDelete
	m.ConfirmType = "index"
	m.ConfirmData = "products"
	m = applyKey(m, "y")

	// Index create
	m.Screen = types.ScreenIndexCreate
	m.Inputs.IndexNameInput.Focus()
	m.Inputs.IndexNameInput.SetValue("newidx")
	m = applyKey(m, "tab")
	m = applyKey(m, "enter")

	// Edit document
	m.Screen = types.ScreenEditDocument
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Inputs.DocBodyInput.Focus()
	m.Inputs.DocBodyInput.SetValue(`{"x":1}`)
	m = applyKey(m, "tab")
	m = applyKey(m, "enter")

	// Bulk delete
	m.Screen = types.ScreenBulkDelete
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Inputs.BulkDeleteInput.Focus()
	m.Inputs.BulkDeleteInput.SetValue("*")
	m = applyKey(m, "enter")

	// Favorites
	m.Screen = types.ScreenFavorites
	m.Favorites = []types.Favorite{{Index: "products"}}
	m.CurrentConn = &types.Connection{ID: 1}
	m = applyKey(m, "j")
	m = applyKey(m, "k")
	m = applyKey(m, "enter")
	m.Screen = types.ScreenFavorites
	m = applyKey(m, "d")
	m = applyKey(m, "esc")

	// Recent
	m.Screen = types.ScreenRecentIndices
	m.RecentIndices = []types.RecentIndex{{Index: "products"}}
	m = applyKey(m, "j")
	m = applyKey(m, "enter")
	m.Screen = types.ScreenRecentIndices
	m = applyKey(m, "esc")

	// Cat API
	m.Screen = types.ScreenCatAPI
	m.Inputs.CatInput.Focus()
	m.Inputs.CatInput.SetValue("indices")
	m = applyKey(m, "enter")
	m = applyKey(m, "esc")

	// Search
	m.Screen = types.ScreenSearch
	m.SearchIndex = "products"
	m.Inputs.SearchInput.Focus()
	m = applyKey(m, "enter")
	m.Screen = types.ScreenSearch
	m.Documents = []types.Document{{ID: "1", Index: "products"}}
	m = applyKey(m, "j")
	m = applyKey(m, "o")
	m.Screen = types.ScreenSearch
	m = applyKey(m, "esc")

	// Live metrics
	m.Screen = types.ScreenLiveMetrics
	m.LiveMetricsActive = true
	m = applyKey(m, "esc")
	if m.LiveMetricsActive {
		t.Fatal("should stop metrics")
	}

	// Simple back + refresh
	m.Screen = types.ScreenClusterHealth
	m = applyKey(m, "r")
	m.Screen = types.ScreenNodes
	m.Nodes = []types.NodeInfo{{Name: "n1"}, {Name: "n2"}}
	m = applyKey(m, "j")
	m = applyKey(m, "k")
	m = applyKey(m, "r")
	m.Screen = types.ScreenIndexSettings
	m = applyKey(m, "j")
	m = applyKey(m, "k")
	m = applyKey(m, "esc")

	// Help
	m.Screen = types.ScreenHelp
	m = applyKey(m, "?")

	// Quit
	m.Screen = types.ScreenConnections
	nm, cmd := m.handleKeyPress(makeKey("q"))
	_ = nm
	if cmd == nil {
		// quit returns tea.Quit
	}
	nm, cmd = m.handleKeyPress(makeKey("ctrl+c"))
	_ = nm
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}

	// connectionFromInputs validation
	m.ConnInputs = createConnectionInputs()
	m.ConnInputs[1].SetValue("h")
	m.ConnInputs[2].SetValue("bad")
	if _, err := m.connectionFromInputs(); err == nil {
		t.Fatal("bad port")
	}
	m.ConnInputs[2].SetValue("9200")
	m.ConnInputs[6].SetValue("nope")
	if _, err := m.connectionFromInputs(); err == nil {
		t.Fatal("bad flavor")
	}
	m.ConnInputs[6].SetValue("elasticsearch")
	m.ConnInputs[0].SetValue("")
	conn, err := m.connectionFromInputs()
	if err != nil || conn.Name == "" {
		t.Fatal(err)
	}

	// patternOrAll
	m.Inputs.PatternInput.SetValue("x*")
	if patternOrAll(m) != "x*" {
		t.Fatal(patternOrAll(m))
	}
	m.Inputs.PatternInput.SetValue("")
	if patternOrAll(m) != "*" {
		t.Fatal(patternOrAll(m))
	}

	// goBack paths
	m.Screen = types.ScreenDocumentDetail
	gb, _ := m.goBack()
	m = gb.(Model)
	if m.Screen != types.ScreenDocuments {
		t.Fatal(m.Screen)
	}
	m.Screen = types.ScreenDocuments
	gb, _ = m.goBack()
	m = gb.(Model)
	if m.Screen != types.ScreenIndices {
		t.Fatal(m.Screen)
	}
	m.CurrentConn = nil
	m.Screen = types.ScreenHelp
	gb, _ = m.goBack()
	m = gb.(Model)
	if m.Screen != types.ScreenConnections {
		t.Fatal(m.Screen)
	}
	_ = press
}

func TestConnectionFormTyping(t *testing.T) {
	m, _ := testModel(t)
	m.Screen = types.ScreenAddConnection
	m.ConnInputs = createConnectionInputs()
	m.ConnFocusIdx = 0
	m.ConnInputs[0].Focus()
	// type a character
	nm, _ := m.handleKeyPress(tea.KeyPressMsg{Text: "x", Code: 'x'})
	m = nm.(Model)
	_ = m
}
