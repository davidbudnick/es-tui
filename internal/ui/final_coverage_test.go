package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func TestFinalUICoverage(t *testing.T) {
	m, _ := testModel(t)
	m.Width, m.Height = 120, 50

	// Init with CLI only (cmds set)
	m.CLIConnection = &types.Connection{Host: "h", Port: 9200}
	if m.Init() == nil {
		t.Fatal("expected cmds")
	}

	// connection form enter without cmds
	m.Screen = types.ScreenAddConnection
	m.ConnInputs = createConnectionInputs()
	m.ConnInputs[1].SetValue("localhost")
	m.ConnFocusIdx = 0
	m.Cmds = nil
	m = applyKey(m, "enter")

	// connectionFromInputs empty flavor -> auto
	m, _ = testModel(t)
	m.Width, m.Height = 120, 50
	m.ConnInputs = createConnectionInputs()
	m.ConnInputs[1].SetValue("h")
	m.ConnInputs[6].SetValue("")
	conn, err := m.connectionFromInputs()
	if err != nil || conn.Flavor != types.FlavorAuto {
		t.Fatal(conn, err)
	}

	// handleConnectionFormKeys edit path
	m.Screen = types.ScreenEditConnection
	m.EditingConn = &types.Connection{ID: 1}
	m.ConnInputs = createConnectionInputs()
	m.ConnInputs[0].SetValue("n")
	m.ConnInputs[1].SetValue("h")
	m.ConnInputs[2].SetValue("9200")
	m.ConnFocusIdx = 0
	m = applyKey(m, "enter")

	// indices without cmds for various actions
	m.Screen = types.ScreenIndices
	m.Indices = []types.IndexInfo{{Name: "a"}}
	m.Cmds = nil
	m = applyKey(m, "r")
	m = applyKey(m, "c")
	m = applyKey(m, "n")
	m = applyKey(m, "m")
	m = applyKey(m, "s")
	m = applyKey(m, "A")
	m = applyKey(m, "T")
	m = applyKey(m, "F")
	m = applyKey(m, "R")
	m = applyKey(m, "enter")
	m = applyKey(m, "i")
	m = applyKey(m, "*")

	// restore cmds
	m, _ = testModel(t)
	m.Width, m.Height = 120, 50
	m.Screen = types.ScreenIndices
	m.CurrentConn = &types.Connection{ID: 1}
	m.Indices = []types.IndexInfo{{Name: "products"}, {Name: "orders"}}
	m.SelectedIndexIdx = 0
	// home/end aliases
	m = applyKey(m, "down")
	m = applyKey(m, "up")
	m = applyKey(m, "end")
	m = applyKey(m, "home")

	// index detail without cmds
	m.Screen = types.ScreenIndexDetail
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Cmds = nil
	m = applyKey(m, "enter")
	m = applyKey(m, "s")
	m = applyKey(m, "m")
	m = applyKey(m, "/")

	// documents without cmds
	m, _ = testModel(t)
	m.Screen = types.ScreenDocuments
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Documents = []types.Document{{ID: "1", Index: "products"}}
	m.Cmds = nil
	m = applyKey(m, "enter")
	m = applyKey(m, "r")
	m.Inputs.SearchInput.Focus()
	m = applyKey(m, "enter")

	// document detail without current
	m.Screen = types.ScreenDocumentDetail
	m.CurrentDocument = nil
	m = applyKey(m, "e")
	m = applyKey(m, "d")

	// search without cmds
	m.Screen = types.ScreenSearch
	m.Inputs.SearchInput.Focus()
	m.Cmds = nil
	m = applyKey(m, "enter")
	m.Inputs.SearchInput.Blur()
	m.Documents = []types.Document{{ID: "1", Index: "i"}}
	m = applyKey(m, "o")

	// confirm wrong types
	m, _ = testModel(t)
	m.Screen = types.ScreenConfirmDelete
	m.ConfirmType = "connection"
	m.ConfirmData = "not-a-conn"
	m = applyKey(m, "y")
	m.Screen = types.ScreenConfirmDelete
	m.ConfirmType = "index"
	m.ConfirmData = 123
	m = applyKey(m, "y")
	m.Screen = types.ScreenConfirmDelete
	m.ConfirmType = "document"
	m.ConfirmData = "nope"
	m = applyKey(m, "y")
	// goBackFromConfirm with current index for index type
	m.ConfirmType = "index"
	m.CurrentIndex = &types.IndexInfo{Name: "x"}
	gb, _ := m.goBackFromConfirm()
	m = gb.(Model)

	// favorites/recent navigation
	m.Screen = types.ScreenFavorites
	m.Favorites = []types.Favorite{{Index: "a"}, {Index: "b"}}
	m = applyKey(m, "up")
	m = applyKey(m, "down")
	m.Screen = types.ScreenRecentIndices
	m.RecentIndices = []types.RecentIndex{{Index: "a"}, {Index: "b"}}
	m = applyKey(m, "up")
	m = applyKey(m, "k")

	// cat without cmds
	m.Screen = types.ScreenCatAPI
	m.Inputs.CatInput.SetValue("indices")
	m.Cmds = nil
	m = applyKey(m, "enter")

	// goBack default
	m.Screen = types.ScreenConnections
	gb, _ = m.goBack()
	m = gb.(Model)

	// views remaining
	m.Width, m.Height = 100, 40
	m.Documents = []types.Document{{ID: "1", Score: 1}, {ID: "2", Score: 2}}
	m.SelectedDocIdx = 1
	_ = m.viewDocuments()
	m.Nodes = []types.NodeInfo{{Name: "a"}, {Name: "b"}}
	m.SelectedNode = 1
	_ = m.viewNodes()
	// logs with newline suffix
	m.Logs = types.NewLogWriter()
	_, _ = m.Logs.Write([]byte("line\n"))
	_ = m.viewLogs()
	// recent selected style
	m.RecentIndices = []types.RecentIndex{{Index: "a"}, {Index: "b"}}
	m.SelectedRecentIdx = 1
	_ = m.viewRecentIndices()
	// sparkline max edge
	_ = sparkline([]types.LiveMetricsData{{QueryTotal: 0}, {QueryTotal: 0}})
	// connections endIdx adjust
	m.Connections = make([]types.Connection, 5)
	for i := range m.Connections {
		m.Connections[i] = types.Connection{Name: "c", Host: "h", Port: 9200}
	}
	m.SelectedConnIdx = 4
	m.Height = 18
	_ = m.viewConnections()

	// Update ConnectionDeleted clamping
	m.Connections = []types.Connection{{ID: 1}, {ID: 2}}
	m.SelectedConnIdx = 1
	nm, _ := m.Update(types.ConnectionDeletedMsg{ID: 2})
	m = nm.(Model)

	// handleKeyPress help on unknown keys for simple screens
	m.Screen = types.ScreenLogs
	m = applyKey(m, "x")

	// Document detail j when at top k
	m.Screen = types.ScreenDocumentDetail
	m.CurrentDocument = &types.Document{ID: "1", Index: "i", Raw: "a\nb\nc"}
	m.DetailScroll = 0
	m = applyKey(m, "k")

	// connections delete/backspace
	m, _ = testModel(t)
	m.Connections = []types.Connection{{ID: 1, Name: "a", Host: "h", Port: 1}}
	m.Screen = types.ScreenConnections
	m = applyKey(m, "backspace")

	_ = time.Now()
	_ = tea.KeyPressMsg{}
}
