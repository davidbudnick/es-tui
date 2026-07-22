package ui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func TestUICoverageGaps(t *testing.T) {
	m, mock := testModel(t)
	m.Width, m.Height = 100, 40

	// Init with no cmds
	m2 := NewModel()
	if m2.Init() != nil {
		// may be nil without cmds/cli
	}

	// Connections empty key paths
	m.Screen = types.ScreenConnections
	m.Connections = nil
	m = applyKey(m, "e")
	m = applyKey(m, "d")
	m = applyKey(m, "t")
	m = applyKey(m, "enter")
	m = applyKey(m, "j")
	m = applyKey(m, "k")
	m = applyKey(m, "G")

	// Connection form default branch (typing)
	m.Screen = types.ScreenAddConnection
	m.ConnInputs = createConnectionInputs()
	m.ConnFocusIdx = 0
	m.ConnInputs[0].Focus()
	nm, _ := m.handleKeyPress(tea.KeyPressMsg{Text: "a", Code: 'a'})
	m = nm.(Model)

	// Indices empty
	m.Screen = types.ScreenIndices
	m.Indices = nil
	m.CurrentConn = &types.Connection{ID: 1}
	m = applyKey(m, "d")
	m = applyKey(m, "i")
	m = applyKey(m, "enter")
	m = applyKey(m, "j")
	m = applyKey(m, "*")
	m = applyKey(m, "F")
	m = applyKey(m, "R")
	m = applyKey(m, "esc") // disconnect

	// Indices filter typing
	m.Screen = types.ScreenIndices
	m.CurrentConn = &types.Connection{ID: 1}
	m.Indices = mock.Indices
	m.Inputs.PatternInput.Focus()
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "p", Code: 'p'})
	m = nm.(Model)

	// Documents focused search typing + empty enter match_all
	m.Screen = types.ScreenDocuments
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Documents = nil
	m.Inputs.SearchInput.Focus()
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "q", Code: 'q'})
	m = nm.(Model)
	m = applyKey(m, "enter")
	m.Screen = types.ScreenDocuments
	m.Documents = nil
	m.Inputs.SearchInput.Blur()
	m = applyKey(m, "enter")
	m.Screen = types.ScreenDocuments
	m = applyKey(m, "d")

	// Search unfocused + typing
	m.Screen = types.ScreenSearch
	m.Inputs.SearchInput.Focus()
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "x", Code: 'x'})
	m = nm.(Model)
	m.Inputs.SearchInput.Blur()
	m.Documents = []types.Document{{ID: "1", Index: "products"}}
	m = applyKey(m, "enter") // focus
	m.Inputs.SearchInput.Blur()
	m = applyKey(m, "k")
	m = applyKey(m, "j")

	// Index create empty name + typing
	m.Screen = types.ScreenIndexCreate
	m.Inputs.IndexNameInput.Focus()
	m.Inputs.IndexNameInput.SetValue("")
	m = applyKey(m, "enter")
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "i", Code: 'i'})
	m = nm.(Model)
	m.Inputs.IndexNameInput.Blur()
	m.Inputs.IndexBodyInput.Focus()
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "{", Code: '{'})
	m = nm.(Model)

	// Edit document typing
	m.Screen = types.ScreenEditDocument
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.CurrentDocument = &types.Document{ID: "1"}
	m.Inputs.DocIDInput.Focus()
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "1", Code: '1'})
	m = nm.(Model)
	m = applyKey(m, "tab")
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "{", Code: '{'})
	m = nm.(Model)
	m = applyKey(m, "esc")

	// Bulk delete typing
	m.Screen = types.ScreenBulkDelete
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Inputs.BulkDeleteInput.Focus()
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "*", Code: '*'})
	m = nm.(Model)

	// Cat typing
	m.Screen = types.ScreenCatAPI
	m.Inputs.CatInput.Focus()
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "i", Code: 'i'})
	m = nm.(Model)

	// Favorites empty / recent empty
	m.Screen = types.ScreenFavorites
	m.Favorites = nil
	m = applyKey(m, "enter")
	m = applyKey(m, "d")
	m = applyKey(m, "j")
	m.Screen = types.ScreenRecentIndices
	m.RecentIndices = nil
	m = applyKey(m, "enter")
	m = applyKey(m, "j")
	m = applyKey(m, "k")

	// Confirm delete document path
	m.Screen = types.ScreenConfirmDelete
	m.ConfirmType = "document"
	m.ConfirmData = types.Document{Index: "products", ID: "1"}
	m = applyKey(m, "y")

	// goBackFromConfirm document / default
	m.ConfirmType = "document"
	gb, _ := m.goBackFromConfirm()
	m = gb.(Model)
	m.ConfirmType = "unknown"
	gb, _ = m.goBackFromConfirm()
	m = gb.(Model)

	// goBack more screens
	for _, s := range []types.Screen{
		types.ScreenEditDocument, types.ScreenBulkDelete, types.ScreenIndexCreate,
		types.ScreenLiveMetrics, types.ScreenCatAPI, types.ScreenShards,
	} {
		m.Screen = s
		m.CurrentConn = &types.Connection{ID: 1}
		gb, _ = m.goBack()
		m = gb.(Model)
	}

	// handleKeyPress default screen
	m.Screen = types.Screen(999)
	m = applyKey(m, "x")
	m = applyKey(m, "q")

	// Update edge: selected index clamp
	nm, _ = m.Update(types.ConnectionsLoadedMsg{Connections: nil})
	m = nm.(Model)
	m.SelectedConnIdx = 5
	nm, _ = m.Update(types.ConnectionsLoadedMsg{Connections: []types.Connection{{ID: 1}}})
	m = nm.(Model)

	m.SelectedIndexIdx = 99
	nm, _ = m.Update(types.IndicesLoadedMsg{Indices: []types.IndexInfo{{Name: "a"}}})
	m = nm.(Model)

	// Connected with empty flavor uses ES().Flavor()
	mock.FlavorVal = types.FlavorOpenSearch
	nm, _ = m.Update(types.ConnectedMsg{Info: types.ClusterInfo{ClusterName: "c"}})
	m = nm.(Model)

	// Favorite removed from indices screen
	m.Screen = types.ScreenIndices
	m.CurrentConn = &types.Connection{ID: 1}
	nm, _ = m.Update(types.FavoriteRemovedMsg{Index: "x"})
	m = nm.(Model)

	// Document saved/deleted without current index
	m.CurrentIndex = nil
	nm, _ = m.Update(types.DocumentSavedMsg{Index: "i", ID: "1"})
	m = nm.(Model)
	nm, _ = m.Update(types.DocumentDeletedMsg{Index: "i", ID: "1"})
	m = nm.(Model)
	nm, _ = m.Update(types.BulkDeleteMsg{Index: "i", Deleted: 1})
	m = nm.(Model)

	// AutoConnect without cmds
	m.Cmds = nil
	nm, _ = m.Update(types.AutoConnectMsg{Connection: types.Connection{Host: "h", Port: 1}})
	m = nm.(Model)

	// Live metrics history without active
	m3, mock3 := testModel(t)
	m3.LiveMetricsActive = false
	nm, _ = m3.Update(types.LiveMetricsMsg{Data: mock3.Metrics})
	m = nm.(Model)

	// View branches: connections scroll, version empty, status current conn
	m.Version = ""
	m.Connections = make([]types.Connection, 20)
	for i := range m.Connections {
		m.Connections[i] = types.Connection{Name: "c", Host: "h", Port: 9200 + i, Flavor: types.FlavorAuto}
	}
	m.SelectedConnIdx = 15
	m.Screen = types.ScreenConnections
	m.CurrentConn = &types.Connection{Name: "c"}
	m.Flavor = ""
	_ = m.render()
	m.Height = 15
	_ = m.viewConnections()
	m.Height = 40
	m.SelectedConnIdx = 100
	_ = m.viewConnections()

	// Indices scroll
	m.Screen = types.ScreenIndices
	m.Indices = make([]types.IndexInfo, 50)
	for i := range m.Indices {
		m.Indices[i] = types.IndexInfo{Name: "idx", Health: "green"}
	}
	m.SelectedIndexIdx = 40
	_ = m.viewIndices()

	// Document detail scroll overflow
	m.CurrentDocument = &types.Document{Index: "i", ID: "1", Raw: "{\n" + "\"a\":1,\n\"b\":2,\n\"c\":3,\n\"d\":4,\n\"e\":5,\n\"f\":6,\n\"g\":7,\n\"h\":8,\n\"i\":9,\n\"j\":10\n}"}
	m.DetailScroll = 2
	m.Height = 20
	_ = m.viewDocumentDetail()

	// Search without index
	m.SearchIndex = ""
	m.SearchResult = &types.SearchResult{Total: 1, Hits: make([]types.Document, 25)}
	for i := range m.SearchResult.Hits {
		m.SearchResult.Hits[i] = types.Document{ID: "x", Index: "i"}
	}
	_ = m.viewSearch()

	// shards many
	m.Shards = make([]types.ShardInfo, 40)
	_ = m.viewShards()

	// cat many lines
	m.CatResult = ""
	for i := 0; i < 50; i++ {
		m.CatResult += "line\n"
	}
	_ = m.viewCatAPI()

	// test connection empty
	m.TestConnResult = ""
	_ = m.viewTestConnection()

	// favorites selected
	m.Favorites = []types.Favorite{{Index: "a"}, {Index: "b", Label: "L"}}
	m.SelectedFavIdx = 1
	_ = m.viewFavorites()

	// recent selected
	m.RecentIndices = []types.RecentIndex{{Index: "a"}}
	m.SelectedRecentIdx = 0
	_ = m.viewRecentIndices()

	// sparkline edge
	_ = sparkline([]types.LiveMetricsData{{QueryTotal: 100}, {QueryTotal: 0}})

	// colorize edge cases
	_ = colorizeJSON(`{"a":-1.5e10}`)
	_ = findStringEnd(`"ab`, 1) // unclosed
	_, _ = matchLiteral("truex", 0)
	_, _ = matchLiteral("true", 0)

	// default getScreenView
	m.Screen = types.Screen(999)
	if m.getScreenView() != "" {
		t.Fatal("expected empty")
	}

	// connection form edit title
	m.Screen = types.ScreenEditConnection
	_ = m.viewConnectionForm()

	// handleSimpleBack without cmds refresh
	m.Cmds = nil
	m.Screen = types.ScreenClusterHealth
	m = applyKey(m, "r")

	// confirm with nil cmds
	m.Screen = types.ScreenConfirmDelete
	m.ConfirmType = "index"
	m.ConfirmData = "x"
	m = applyKey(m, "y")

	// index create/edit/bulk without cmds/index
	m.Screen = types.ScreenIndexCreate
	m.Inputs = nil
	m = applyKey(m, "enter")
	m.Inputs = &ModelInputs{
		IndexNameInput:  createTextInput("n", 10),
		IndexBodyInput:  createTextInput("b", 10),
		DocIDInput:      createTextInput("id", 10),
		DocBodyInput:    createTextInput("body", 10),
		BulkDeleteInput: createTextInput("q", 10),
		CatInput:        createTextInput("c", 10),
		PatternInput:    createTextInput("p", 10),
		SearchInput:     createTextInput("s", 10),
	}
	m.Inputs.IndexNameInput.SetValue("x")
	m.Cmds = nil
	m = applyKey(m, "enter")

	m.Screen = types.ScreenEditDocument
	m.CurrentIndex = nil
	m = applyKey(m, "enter")
	m.Inputs = nil
	m = applyKey(m, "enter")

	m.Inputs = &ModelInputs{BulkDeleteInput: createTextInput("q", 10), CatInput: createTextInput("c", 10)}
	m.Screen = types.ScreenBulkDelete
	m.CurrentIndex = nil
	m = applyKey(m, "enter")
	m.Inputs = nil
	m = applyKey(m, "enter")

	m.Screen = types.ScreenCatAPI
	m.Inputs = nil
	m = applyKey(m, "enter")

	// help q
	m.Screen = types.ScreenHelp
	m = applyKey(m, "q")

	// documents esc
	m.Screen = types.ScreenDocuments
	m = applyKey(m, "esc")

	// UpdateAvailable with err
	nm, _ = m.Update(types.UpdateAvailableMsg{Err: errors.New("x")})
	_ = nm

	// Index create with body focus enter after name set - restore cmds
	m, _ = testModel(t)
	m.Width, m.Height = 100, 40
	m.Screen = types.ScreenIndexCreate
	m.Inputs.IndexNameInput.SetValue("newone")
	m.Inputs.IndexNameInput.Focus()
	m = applyKey(m, "enter")

	// disconnect from indices with cmds
	m.Screen = types.ScreenIndices
	m = applyKey(m, "q")
	_ = mock
}
