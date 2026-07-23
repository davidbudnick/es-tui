package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
	"github.com/davidbudnick/es-tui/internal/ui/editor"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if nm, cmd, ok := m.handleAdminMessages(msg); ok {
		return nm, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Rewrap detail body on next paint (cheap; avoids stale wrap at new width).
		m.invalidateDetailCache()
		if m.Screen == types.ScreenEditDocument && m.DocEditor != nil {
			m.DocEditor.SetSize(max(msg.Width-4, 20), max(msg.Height-12, 8))
		}
		if m.Screen == types.ScreenSearch && m.SearchArea != nil {
			m.SearchArea.SetWidth(max(msg.Width/2-6, 40))
			m.SearchArea.SetHeight(7)
		}
		return m, nil

	case types.EditorSaveMsg:
		if m.Screen != types.ScreenEditDocument || m.CurrentIndex == nil || m.Cmds == nil {
			return m, nil
		}
		id := ""
		if m.Inputs != nil {
			id = strings.TrimSpace(m.Inputs.DocIDInput.Value())
		}
		m.Loading = true
		return m, m.Cmds.SaveDocument(m.CurrentIndex.Name, id, msg.Content)

	case types.EditorQuitMsg:
		m.DocEditor = nil
		m.DocEditFocus = ""
		if m.CurrentDocument != nil {
			m.Screen = types.ScreenDocumentDetail
		} else {
			m.Screen = types.ScreenDocuments
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case types.ConnectionsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Connections = msg.Connections
		if m.SelectedConnIdx >= len(m.Connections) {
			m.SelectedConnIdx = max(0, len(m.Connections)-1)
		}
		return m, nil

	case types.ConnectionAddedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Connections = append(m.Connections, msg.Connection)
		m.Screen = types.ScreenConnections
		m.StatusMsg = "Connection added"
		m.Err = nil
		return m, nil

	case types.ConnectionUpdatedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		for i, c := range m.Connections {
			if c.ID == msg.Connection.ID {
				m.Connections[i] = msg.Connection
				break
			}
		}
		m.Screen = types.ScreenConnections
		m.StatusMsg = "Connection updated"
		return m, nil

	case types.ConnectionDeletedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		filtered := m.Connections[:0]
		for _, c := range m.Connections {
			if c.ID != msg.ID {
				filtered = append(filtered, c)
			}
		}
		m.Connections = filtered
		if m.SelectedConnIdx >= len(m.Connections) {
			m.SelectedConnIdx = max(0, len(m.Connections)-1)
		}
		m.StatusMsg = "Connection deleted"
		m.Screen = types.ScreenConnections
		return m, nil

	case types.AutoConnectMsg:
		m.Loading = true
		m.ConnectionError = ""
		m.CurrentConn = &msg.Connection
		if m.Cmds != nil {
			return m, m.Cmds.Connect(msg.Connection)
		}
		return m, nil

	case types.ConnectedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.ConnectionError = msg.Err.Error()
			m.Err = msg.Err
			m.Screen = types.ScreenConnections
			return m, nil
		}
		m.ClusterInfo = msg.Info
		m.Flavor = msg.Info.Flavor
		if !m.Flavor.IsKnown() && m.Cmds != nil {
			m.Flavor = m.Cmds.ES().Flavor()
		}
		if m.Cmds != nil {
			m.ReadOnly = m.Cmds.ES().IsReadOnly()
		}
		if m.CurrentConn != nil {
			m.ReadOnly = m.CurrentConn.ReadOnly
			// Session knows the concrete engine after auto-detect.
			if m.Flavor.IsKnown() {
				m.CurrentConn.Flavor = m.Flavor
			}
		}
		m.ConnectionError = ""
		m.Err = nil
		m.Screen = types.ScreenIndices
		engine := m.Flavor.DisplayName()
		status := fmt.Sprintf("Connected · %s · %s %s", engine, msg.Info.ClusterName, msg.Info.Version.Number)
		if m.ReadOnly {
			status += " [read-only]"
		}
		m.StatusMsg = status
		if m.Cmds != nil {
			return m, tea.Batch(m.Cmds.LoadIndices("*"), m.Cmds.LoadClusterHealth())
		}
		return m, nil

	case types.DisconnectedMsg:
		m.CurrentConn = nil
		m.Indices = nil
		m.Documents = nil
		m.ReadOnly = false
		m.LiveMetricsActive = false
		m.LiveMetrics = nil
		m.ClusterHealth = types.ClusterHealth{}
		m.StatusMsg = ""
		m.Err = nil
		m.Loading = false
		m.Screen = types.ScreenConnections
		return m, nil

	case types.IndicesLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Indices = msg.Indices
		if m.CurrentConn != nil && m.Cmds != nil {
			connID := m.CurrentConn.ID
			for i := range m.Indices {
				m.Indices[i].IsFavorite = m.Cmds.Config().IsFavorite(connID, m.Indices[i].Name)
			}
		}
		if m.SelectedIndexIdx >= len(m.Indices) {
			m.SelectedIndexIdx = max(0, len(m.Indices)-1)
		}
		m.Err = nil
		return m, nil

	case types.IndexDetailLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		idx := msg.Index
		m.CurrentIndex = &idx
		m.IndexSettings = msg.Settings
		m.IndexMappings = msg.Mappings
		m.Screen = types.ScreenIndexDetail
		return m, nil

	case types.IndexCreatedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.StatusMsg = "Created index " + msg.Name
		m.Screen = types.ScreenIndices
		if m.Cmds != nil {
			return m, m.Cmds.LoadIndices(patternOrAll(m))
		}
		return m, nil

	case types.IndexDeletedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.StatusMsg = "Deleted index " + msg.Name
		m.Screen = types.ScreenIndices
		if m.Cmds != nil {
			return m, m.Cmds.LoadIndices(patternOrAll(m))
		}
		return m, nil

	case types.DocumentsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Documents = msg.Documents
		m.DocTotal = msg.Total
		m.SelectedDocIdx = 0
		// If we overshot (empty page past end), snap back to last valid page.
		if len(msg.Documents) == 0 && m.DocTotal > 0 && m.DocFrom > 0 {
			pageSize := m.docPageSize()
			lastFrom := int((m.DocTotal - 1) / int64(pageSize) * int64(pageSize))
			if lastFrom < 0 {
				lastFrom = 0
			}
			if m.DocFrom > lastFrom {
				m.DocFrom = lastFrom
				m.Loading = true
				if m.Cmds != nil && m.CurrentIndex != nil {
					return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, m.DocFrom, pageSize)
				}
			}
		}
		m.refreshDocPreviewFromSelection()
		m.Screen = types.ScreenDocuments
		return m, nil

	case types.DocumentLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		doc := msg.Document
		m.CurrentDocument = &doc
		m.setDetailBody(doc)
		m.DetailScroll = 0
		m.DetailCursor = 0
		m.Screen = types.ScreenDocumentDetail
		return m, nil

	case types.DocumentSavedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.DocEditor = nil
		m.DocEditFocus = ""
		m.Screen = types.ScreenDocuments
		m.StatusMsg = fmt.Sprintf("Saved %s/%s", msg.Index, msg.ID)
		if m.Cmds != nil && m.CurrentIndex != nil {
			return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, 0, 50)
		}
		return m, nil

	case types.DocumentDeletedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.StatusMsg = fmt.Sprintf("Deleted %s/%s", msg.Index, msg.ID)
		if m.Cmds != nil && m.CurrentIndex != nil {
			return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, 0, 50)
		}
		return m, nil

	case types.SearchResultMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		r := msg.Result
		m.SearchResult = &r
		m.Documents = r.Hits
		m.DocTotal = r.Total
		m.SelectedDocIdx = 0
		m.SearchFocus = "results"
		if m.Screen != types.ScreenSearch && m.Screen != types.ScreenDocuments {
			m.Screen = types.ScreenSearch
		}
		return m, nil

	case types.IndexOpMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.StatusMsg = fmt.Sprintf("%s %s", msg.Op, msg.Index)
		if m.Cmds != nil {
			return m, m.Cmds.LoadIndices(patternOrAll(m))
		}
		return m, nil

	case types.ClipboardCopiedMsg:
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.StatusMsg = "Copied to clipboard"
		return m, nil

	case types.ClusterHealthLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		// Update status-bar chip only; do not steal the current screen
		// (connect batches LoadClusterHealth in the background).
		m.ClusterHealth = msg.Health
		return m, nil

	case types.NodesLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Nodes = msg.Nodes
		m.Screen = types.ScreenNodes
		return m, nil

	case types.ShardsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Shards = msg.Shards
		m.Screen = types.ScreenShards
		return m, nil

	case types.AliasesLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Aliases = msg.Aliases
		m.Screen = types.ScreenAliases
		return m, nil

	case types.TemplatesLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Templates = msg.Templates
		m.Screen = types.ScreenIndexTemplates
		return m, nil

	case types.ConnectionTestMsg:
		m.Loading = false
		if msg.Err != nil {
			m.TestConnResult = errorStyle.Render(fmt.Sprintf("Failed: %v (latency %s)", msg.Err, msg.Latency.Round(time.Millisecond)))
		} else {
			engine := msg.Info.Flavor.DisplayName()
			m.TestConnResult = successStyle.Render(fmt.Sprintf(
				"OK · %s · %s %s · latency %s",
				engine,
				msg.Info.ClusterName,
				msg.Info.Version.Number,
				msg.Latency.Round(time.Millisecond),
			))
		}
		m.Screen = types.ScreenTestConnection
		return m, nil

	case types.FavoritesLoadedMsg:
		m.Favorites = msg.Favorites
		m.Screen = types.ScreenFavorites
		return m, nil

	case types.FavoriteAddedMsg:
		if msg.Err == nil {
			m.StatusMsg = "Favorited " + msg.Favorite.Index
			if m.Cmds != nil {
				return m, m.Cmds.LoadIndices(patternOrAll(m))
			}
		}
		return m, nil

	case types.FavoriteRemovedMsg:
		if msg.Err == nil {
			m.StatusMsg = "Unfavorited " + msg.Index
			if m.Screen == types.ScreenFavorites && m.CurrentConn != nil && m.Cmds != nil {
				return m, m.Cmds.LoadFavorites(m.CurrentConn.ID)
			}
			if m.Cmds != nil {
				return m, m.Cmds.LoadIndices(patternOrAll(m))
			}
		}
		return m, nil

	case types.RecentIndicesLoadedMsg:
		m.RecentIndices = msg.Indices
		m.Screen = types.ScreenRecentIndices
		return m, nil

	case types.BulkDeleteMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.StatusMsg = fmt.Sprintf("Deleted %d documents from %s", msg.Deleted, msg.Index)
		m.Screen = types.ScreenDocuments
		if m.Cmds != nil {
			return m, m.Cmds.LoadDocuments(msg.Index, m.DocQuery, 0, 50)
		}
		return m, nil

	case types.LiveMetricsMsg:
		if !m.LiveMetricsActive {
			return m, nil
		}
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		if m.LiveMetrics == nil {
			m.LiveMetrics = &types.LiveMetrics{}
		}
		m.LiveMetrics.Latest = msg.Data
		m.LiveMetrics.History = append(m.LiveMetrics.History, msg.Data)
		if len(m.LiveMetrics.History) > 60 {
			m.LiveMetrics.History = m.LiveMetrics.History[len(m.LiveMetrics.History)-60:]
		}
		m.Screen = types.ScreenLiveMetrics
		if m.Cmds != nil {
			return m, m.Cmds.LiveMetricsTick()
		}
		return m, nil

	case types.LiveMetricsTickMsg:
		if m.LiveMetricsActive && m.Cmds != nil {
			return m, m.Cmds.LoadLiveMetrics()
		}
		return m, nil

	case types.IndexSettingsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.IndexSettings = msg.Settings
		m.DetailScroll = 0
		m.Screen = types.ScreenIndexSettings
		return m, nil

	case types.IndexMappingsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.IndexMappings = msg.Mappings
		m.DetailScroll = 0
		m.Screen = types.ScreenIndexMappings
		return m, nil

	case types.CatAPIResultMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.CatResult = msg.Body
		m.CatEndpoint = msg.Endpoint
		return m, nil

	case types.UpdateAvailableMsg:
		if msg.Err == nil && msg.LatestVersion != "" {
			m.UpdateAvailable = msg.LatestVersion
			m.UpdateCmd = msg.UpgradeCmd
		}
		return m, nil
	}

	return m, nil
}

func patternOrAll(m Model) string {
	if m.Inputs != nil {
		p := strings.TrimSpace(m.Inputs.PatternInput.Value())
		if p != "" {
			return p
		}
	}
	return "*"
}

func (m Model) docPageSize() int {
	return clampPageSize(m.PageSize)
}

func (m Model) canDocPageNext() bool {
	if m.DocTotal <= 0 {
		return false
	}
	return int64(m.DocFrom+m.docPageSize()) < m.DocTotal
}

func (m Model) canSearchPageNext() bool {
	if m.SearchResult == nil || m.SearchResult.Total <= 0 {
		return false
	}
	pageSize := m.docPageSize()
	return int64(m.SearchFrom+pageSize) < m.SearchResult.Total
}

func (m Model) searchQueryValue() string {
	if m.SearchArea != nil {
		return m.SearchArea.Value()
	}
	if m.Inputs != nil {
		if q := m.Inputs.SearchInput.Value(); q != "" {
			return q
		}
	}
	return m.SearchQuery
}

func (m Model) focusSearchQuery() Model {
	m = m.ensureSearchArea()
	m.SearchFocus = "query"
	m.SearchArea.Focus()
	return m
}

func (m Model) leaveSearch() Model {
	if m.SearchArea != nil {
		m.SearchQuery = m.SearchArea.Value()
	}
	m.SearchArea = nil
	m.SearchFocus = ""
	if m.CurrentIndex != nil {
		m.Screen = types.ScreenDocuments
	} else {
		m.Screen = types.ScreenIndices
	}
	return m
}

func (m Model) openSearchScreen(index, initialQuery string, clearResults bool) Model {
	m.Screen = types.ScreenSearch
	m.SearchIndex = index
	m.SearchFrom = 0
	m.SelectedDocIdx = 0
	if clearResults {
		m.SearchResult = nil
	}
	if initialQuery != "" {
		m.SearchQuery = initialQuery
	}
	m.SearchArea = nil
	return m.focusSearchQuery()
}

func (m Model) runSearchQuery() (tea.Model, tea.Cmd) {
	pageSize := m.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	m = m.ensureSearchArea()
	q := m.SearchArea.Value()
	m.SearchQuery = q
	m.pushQueryHistory(q)
	m.SearchFrom = 0
	m.SearchFocus = "results"
	m.SearchArea.Blur()
	m.SelectedDocIdx = 0
	if m.Cmds == nil {
		return m, nil
	}
	m.Loading = true
	return m, m.Cmds.Search(m.SearchIndex, q, 0, pageSize)
}

func (m Model) applySearchTemplate(n int) Model {
	templates := searchQueryTemplates()
	if n < 0 || n >= len(templates) {
		return m
	}
	m = m.ensureSearchArea()
	m.SearchArea.SetValue(templates[n].Query)
	m.SearchFocus = "query"
	m.SearchArea.Focus()
	m.HistoryIdx = -1
	return m
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := normalizeKey(msg)

	// Global force quit
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	// Clear status on any key
	m.StatusMsg = ""
	m.Err = nil

	// Global help — available from every browse screen (not while typing).
	if key == "?" && !m.typingContext() {
		if m.Screen == types.ScreenHelp {
			return m.goBack()
		}
		m.PrevScreen = m.Screen
		m.Screen = types.ScreenHelp
		return m, nil
	}

	// Global command palette (colon) when not typing into a field.
	if key == ":" && !m.typingContext() && m.Screen != types.ScreenCommandPalette {
		return m.openPalette()
	}

	switch m.Screen {
	case types.ScreenConnections:
		return m.handleConnectionsKeys(key)
	case types.ScreenAddConnection, types.ScreenEditConnection:
		return m.handleConnectionFormKeys(key, msg)
	case types.ScreenIndices:
		return m.handleIndicesKeys(key, msg)
	case types.ScreenIndexDetail:
		return m.handleIndexDetailKeys(key)
	case types.ScreenDocuments:
		return m.handleDocumentsKeys(key, msg)
	case types.ScreenDocumentDetail:
		return m.handleDocumentDetailKeys(key)
	case types.ScreenSearch:
		return m.handleSearchKeys(key, msg)
	case types.ScreenConfirmDelete:
		return m.handleConfirmDeleteKeys(key)
	case types.ScreenClusterHealth, types.ScreenNodes, types.ScreenShards,
		types.ScreenAliases, types.ScreenIndexTemplates, types.ScreenLogs,
		types.ScreenTestConnection, types.ScreenIndexSettings, types.ScreenIndexMappings:
		return m.handleSimpleBackKeys(key)
	case types.ScreenLiveMetrics:
		return m.handleLiveMetricsKeys(key)
	case types.ScreenHelp:
		if key == "esc" || key == "q" || key == "?" {
			return m.goBack()
		}
		return m, nil
	case types.ScreenIndexCreate:
		return m.handleIndexCreateKeys(key, msg)
	case types.ScreenEditDocument:
		return m.handleEditDocumentKeys(key, msg)
	case types.ScreenBulkDelete:
		return m.handleBulkDeleteKeys(key, msg)
	case types.ScreenFavorites:
		return m.handleFavoritesKeys(key)
	case types.ScreenRecentIndices:
		return m.handleRecentKeys(key)
	case types.ScreenCatAPI:
		return m.handleCatAPIKeys(key, msg)
	case types.ScreenCommandPalette:
		return m.handleCommandPaletteKeys(key, msg)
	case types.ScreenReindex:
		return m.handleReindexKeys(key, msg)
	case types.ScreenExport:
		return m.handleExportKeys(key, msg)
	case types.ScreenTasks:
		return m.handleTasksKeys(key)
	case types.ScreenSavedQueries:
		return m.handleSavedQueriesKeys(key)
	case types.ScreenSnapshots:
		return m.handleSnapshotsKeys(key, msg)
	case types.ScreenAllocation, types.ScreenPlugins, types.ScreenDataStreams,
		types.ScreenClusterSettings, types.ScreenExplain:
		return m.handleSimpleBackKeys(key)
	default:
		if key == "q" || key == "esc" {
			return m, tea.Quit
		}
		return m, nil
	}
}

func (m Model) handleConnectionsKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "j", "down":
		if len(m.Connections) > 0 {
			m.SelectedConnIdx = (m.SelectedConnIdx + 1) % len(m.Connections)
		}
	case "k", "up":
		if len(m.Connections) > 0 {
			m.SelectedConnIdx = (m.SelectedConnIdx - 1 + len(m.Connections)) % len(m.Connections)
		}
	case "g", "home":
		m.SelectedConnIdx = 0
	case "G", "end":
		if len(m.Connections) > 0 {
			m.SelectedConnIdx = len(m.Connections) - 1
		}
	case "a":
		m.Screen = types.ScreenAddConnection
		m.ConnInputs = createConnectionInputs()
		m.ConnFocusIdx = 0
		m.ConnFlavorIdx = 0
		m.ConnFlavorOpen = false
		m.ConnReadOnly = false
		m.ConnInputs[0].Focus()
		m.EditingConn = nil
		m.Err = nil
	case "e":
		if len(m.Connections) == 0 {
			return m, nil
		}
		conn := m.Connections[m.SelectedConnIdx]
		m.EditingConn = &conn
		m.Screen = types.ScreenEditConnection
		m.ConnInputs = createConnectionInputs()
		m.ConnInputs[connFieldName].SetValue(conn.Name)
		m.ConnInputs[connFieldHost].SetValue(conn.Host)
		m.ConnInputs[connFieldPort].SetValue(strconv.Itoa(conn.Port))
		m.ConnInputs[connFieldUser].SetValue(conn.Username)
		m.ConnInputs[connFieldPass].SetValue(conn.Password)
		m.ConnInputs[connFieldAPIKey].SetValue(conn.APIKey)
		m.ConnInputs[connFieldBearer].SetValue(conn.BearerToken)
		m.ConnFlavorIdx = flavorIndex(conn.Flavor)
		m.ConnFlavorOpen = false
		m.ConnReadOnly = conn.ReadOnly
		m.ConnFocusIdx = 0
		m.ConnInputs[0].Focus()
		m.Err = nil
	case "d", "delete", "backspace":
		if len(m.Connections) == 0 {
			return m, nil
		}
		m.ConfirmType = "connection"
		m.ConfirmData = m.Connections[m.SelectedConnIdx]
		m.Screen = types.ScreenConfirmDelete
	case "t":
		if len(m.Connections) == 0 || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		m.TestConnResult = ""
		return m, m.Cmds.TestConnection(m.Connections[m.SelectedConnIdx])
	case "enter":
		if len(m.Connections) == 0 || m.Cmds == nil {
			return m, nil
		}
		conn := m.Connections[m.SelectedConnIdx]
		m.CurrentConn = &conn
		m.ReadOnly = conn.ReadOnly
		m.Loading = true
		m.ConnectionError = ""
		return m, m.Cmds.Connect(conn)
	case "L":
		m.Screen = types.ScreenLogs
	}
	return m, nil
}

func (m Model) handleConnectionFormKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Flavor dropdown open: navigation stays inside the menu.
	if m.ConnFocusIdx == connFieldFlavor && m.ConnFlavorOpen {
		switch key {
		case "esc":
			m.ConnFlavorOpen = false
			return m, nil
		case "j", "down":
			m.ConnFlavorIdx = (m.ConnFlavorIdx + 1) % len(connFlavorOptions)
			return m, nil
		case "k", "up":
			m.ConnFlavorIdx = (m.ConnFlavorIdx - 1 + len(connFlavorOptions)) % len(connFlavorOptions)
			return m, nil
		case "enter", " ":
			m.ConnFlavorOpen = false
			return m, nil
		default:
			return m, nil
		}
	}

	switch key {
	case "esc":
		m.ConnFlavorOpen = false
		m.Err = nil
		m.Screen = types.ScreenConnections
		return m, nil
	case "tab", "down":
		return m.connFormMoveFocus(1), nil
	case "shift+tab", "up":
		return m.connFormMoveFocus(-1), nil
	case " ", "space":
		if m.ConnFocusIdx == connFieldFlavor {
			m.ConnFlavorOpen = !m.ConnFlavorOpen
			return m, nil
		}
		if m.ConnFocusIdx == connFieldReadOnly {
			m.ConnReadOnly = !m.ConnReadOnly
			return m, nil
		}
	case "enter":
		// Enter always saves (space/←/→ handle engine + read-only).
		conn, err := m.connectionFromInputs()
		if err != nil {
			m.Err = err
			return m, nil
		}
		m.Err = nil
		if m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		if m.Screen == types.ScreenEditConnection && m.EditingConn != nil {
			conn.ID = m.EditingConn.ID
			return m, m.Cmds.UpdateConnection(conn)
		}
		return m, m.Cmds.AddConnection(conn)
	case "left", "h":
		if m.ConnFocusIdx == connFieldFlavor {
			m.ConnFlavorIdx = (m.ConnFlavorIdx - 1 + len(connFlavorOptions)) % len(connFlavorOptions)
			return m, nil
		}
	case "right", "l":
		if m.ConnFocusIdx == connFieldFlavor {
			m.ConnFlavorIdx = (m.ConnFlavorIdx + 1) % len(connFlavorOptions)
			return m, nil
		}
	default:
		if m.ConnFocusIdx >= 0 && m.ConnFocusIdx < connTextCount && m.ConnFocusIdx < len(m.ConnInputs) {
			var cmd tea.Cmd
			m.ConnInputs[m.ConnFocusIdx], cmd = m.ConnInputs[m.ConnFocusIdx].Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) connFormMoveFocus(delta int) Model {
	// Blur current text field if any.
	if m.ConnFocusIdx >= 0 && m.ConnFocusIdx < connTextCount && m.ConnFocusIdx < len(m.ConnInputs) {
		m.ConnInputs[m.ConnFocusIdx].Blur()
	}
	m.ConnFlavorOpen = false
	m.ConnFocusIdx = (m.ConnFocusIdx + delta + connFieldCount) % connFieldCount
	if m.ConnFocusIdx >= 0 && m.ConnFocusIdx < connTextCount && m.ConnFocusIdx < len(m.ConnInputs) {
		m.ConnInputs[m.ConnFocusIdx].Focus()
	}
	return m
}

func (m Model) connectionFromInputs() (types.Connection, error) {
	if len(m.ConnInputs) < connTextCount {
		return types.Connection{}, fmt.Errorf("form not initialized")
	}
	name := strings.TrimSpace(m.ConnInputs[connFieldName].Value())
	host := strings.TrimSpace(m.ConnInputs[connFieldHost].Value())
	portStr := strings.TrimSpace(m.ConnInputs[connFieldPort].Value())
	if host == "" {
		return types.Connection{}, fmt.Errorf("host is required")
	}
	port := 9200
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 || p > 65535 {
			return types.Connection{}, fmt.Errorf("invalid port")
		}
		port = p
	}
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}
	flavor := types.FlavorAuto
	if m.ConnFlavorIdx >= 0 && m.ConnFlavorIdx < len(connFlavorOptions) {
		flavor = connFlavorOptions[m.ConnFlavorIdx]
	}
	return types.Connection{
		Name:        name,
		Host:        host,
		Port:        port,
		Username:    strings.TrimSpace(m.ConnInputs[connFieldUser].Value()),
		Password:    m.ConnInputs[connFieldPass].Value(),
		APIKey:      strings.TrimSpace(m.ConnInputs[connFieldAPIKey].Value()),
		BearerToken: strings.TrimSpace(m.ConnInputs[connFieldBearer].Value()),
		Flavor:      flavor,
		ReadOnly:    m.ConnReadOnly,
	}, nil
}

func (m Model) handleIndicesKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// When filter input focused
	if m.Inputs != nil && m.Inputs.PatternInput.Focused() {
		switch key {
		case "esc":
			m.Inputs.PatternInput.Blur()
			return m, nil
		case "enter":
			m.Inputs.PatternInput.Blur()
			if m.Cmds != nil {
				m.Loading = true
				return m, m.Cmds.LoadIndices(patternOrAll(m))
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.Inputs.PatternInput, cmd = m.Inputs.PatternInput.Update(msg)
			return m, cmd
		}
	}

	switch key {
	case "esc", "q":
		if m.Cmds != nil {
			// Disconnect() already returns DisconnectedMsg — don't double-fire.
			return m, m.Cmds.Disconnect()
		}
		m.CurrentConn = nil
		m.LiveMetricsActive = false
		m.StatusMsg = ""
		m.Screen = types.ScreenConnections
		return m, nil
	case "j", "down":
		if len(m.Indices) > 0 {
			m.SelectedIndexIdx = (m.SelectedIndexIdx + 1) % len(m.Indices)
		}
	case "k", "up":
		if len(m.Indices) > 0 {
			m.SelectedIndexIdx = (m.SelectedIndexIdx - 1 + len(m.Indices)) % len(m.Indices)
		}
	case "g", "home":
		m.SelectedIndexIdx = 0
	case "G", "end":
		if len(m.Indices) > 0 {
			m.SelectedIndexIdx = len(m.Indices) - 1
		}
	case "f", "/":
		if key == "f" && m.Inputs != nil {
			m.Inputs.PatternInput.Focus()
			return m, nil
		}
		idx := ""
		if len(m.Indices) > 0 {
			idx = m.Indices[m.SelectedIndexIdx].Name
		}
		return m.openSearchScreen(idx, "", true), nil
	case "r":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadIndices(patternOrAll(m))
		}
	case "O":
		// open index
		if len(m.Indices) == 0 || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.OpenIndex(m.Indices[m.SelectedIndexIdx].Name)
	case "X":
		// close index
		if len(m.Indices) == 0 || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.CloseIndex(m.Indices[m.SelectedIndexIdx].Name)
	case "u":
		// refresh selected index
		if len(m.Indices) == 0 || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.RefreshIndexOnly(m.Indices[m.SelectedIndexIdx].Name)
	case "M":
		// force-merge
		if len(m.Indices) == 0 || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.ForceMerge(m.Indices[m.SelectedIndexIdx].Name)
	case "a":
		m.Screen = types.ScreenIndexCreate
		if m.Inputs != nil {
			m.Inputs.IndexNameInput.SetValue("")
			m.Inputs.IndexBodyInput.SetValue("")
			m.Inputs.IndexNameInput.Focus()
		}
	case "d", "delete":
		if len(m.Indices) == 0 {
			return m, nil
		}
		m.ConfirmType = "index"
		m.ConfirmData = m.Indices[m.SelectedIndexIdx].Name
		m.Screen = types.ScreenConfirmDelete
	case "i":
		if len(m.Indices) == 0 || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.LoadIndexDetail(m.Indices[m.SelectedIndexIdx].Name)
	case "enter":
		if len(m.Indices) == 0 || m.Cmds == nil {
			return m, nil
		}
		idx := m.Indices[m.SelectedIndexIdx]
		m.CurrentIndex = &idx
		if m.CurrentConn != nil {
			m.Cmds.Config().AddRecentIndex(m.CurrentConn.ID, idx.Name)
		}
		m.Loading = true
		m.DocQuery = ""
		m.DocFrom = 0
		pageSize := m.PageSize
		if pageSize <= 0 {
			pageSize = 50
		}
		return m, m.Cmds.LoadDocuments(idx.Name, "", 0, pageSize)
	case "c":
		if m.Cmds != nil {
			m.Loading = true
			m.Screen = types.ScreenClusterHealth
			return m, m.Cmds.LoadClusterHealth()
		}
	case "n":
		if m.Cmds != nil {
			m.Loading = true
			m.Screen = types.ScreenNodes
			return m, m.Cmds.LoadNodes()
		}
	case "m":
		if m.Cmds != nil {
			m.LiveMetricsActive = true
			m.Loading = true
			m.Screen = types.ScreenLiveMetrics
			return m, m.Cmds.LoadLiveMetrics()
		}
	case "s":
		if m.Cmds != nil {
			m.Loading = true
			name := ""
			if len(m.Indices) > 0 {
				name = m.Indices[m.SelectedIndexIdx].Name
			}
			return m, m.Cmds.LoadShards(name)
		}
	case "A":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadAliases()
		}
	case "T":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadTemplates()
		}
	case "C":
		m.Screen = types.ScreenCatAPI
		if m.Inputs != nil {
			m.Inputs.CatInput.SetValue("indices")
			m.Inputs.CatInput.Focus()
		}
	case "*":
		if len(m.Indices) == 0 || m.CurrentConn == nil || m.Cmds == nil {
			return m, nil
		}
		idx := m.Indices[m.SelectedIndexIdx]
		if m.Cmds.Config().IsFavorite(m.CurrentConn.ID, idx.Name) {
			return m, m.Cmds.RemoveFavorite(m.CurrentConn.ID, idx.Name)
		}
		return m, m.Cmds.AddFavorite(m.CurrentConn.ID, idx.Name, "")
	case "F":
		if m.CurrentConn != nil && m.Cmds != nil {
			return m, m.Cmds.LoadFavorites(m.CurrentConn.ID)
		}
	case "R":
		if m.CurrentConn != nil && m.Cmds != nil {
			return m, m.Cmds.LoadRecentIndices(m.CurrentConn.ID)
		}
	case "L":
		m.Screen = types.ScreenLogs
	case "P":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadPlugins()
		}
	case "V":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadAllocation()
		}
	case "W":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadTasks()
		}
	case "E":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadDataStreams()
		}
	case "U":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadClusterSettings()
		}
	case "Z":
		m.Screen = types.ScreenSnapshots
		if m.Inputs != nil {
			m.Inputs.SnapshotRepo.Focus()
		}
	case "I":
		m.Screen = types.ScreenReindex
		m.ReindexFocus = 0
		if m.Inputs != nil {
			if len(m.Indices) > 0 {
				m.Inputs.ReindexSrcInput.SetValue(m.Indices[m.SelectedIndexIdx].Name)
			}
			m.Inputs.ReindexSrcInput.Focus()
		}
	case "Y":
		if m.Cmds != nil {
			return m, m.Cmds.LoadSavedQueries()
		}
	case "#":
		if len(m.Indices) > 0 && m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.Count(m.Indices[m.SelectedIndexIdx].Name, "")
		}
	case "Q":
		m.Screen = types.ScreenExport
		if m.Inputs != nil {
			m.Inputs.ExportInput.SetValue(fmt.Sprintf("/tmp/es-tui-export-%d.ndjson", time.Now().Unix()))
			m.Inputs.ExportInput.Focus()
		}
	}
	return m, nil
}

func (m Model) handleIndexDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.Screen = types.ScreenIndices
	case "enter":
		if m.CurrentIndex == nil || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, "", 0, 50)
	case "s":
		if m.CurrentIndex != nil && m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadIndexSettings(m.CurrentIndex.Name)
		}
	case "m":
		if m.CurrentIndex != nil && m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadIndexMappings(m.CurrentIndex.Name)
		}
	case "/":
		if m.CurrentIndex != nil {
			return m.openSearchScreen(m.CurrentIndex.Name, "", true), nil
		}
	case "d":
		if m.CurrentIndex != nil {
			m.ConfirmType = "index"
			m.ConfirmData = m.CurrentIndex.Name
			m.Screen = types.ScreenConfirmDelete
		}
	}
	return m, nil
}

func (m Model) handleDocumentsKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs != nil && m.Inputs.SearchInput.Focused() {
		switch key {
		case "esc":
			m.Inputs.SearchInput.Blur()
			return m, nil
		case "enter":
			m.Inputs.SearchInput.Blur()
			m.DocQuery = m.Inputs.SearchInput.Value()
			if m.Cmds != nil && m.CurrentIndex != nil {
				m.Loading = true
				return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, 0, 50)
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.Inputs.SearchInput, cmd = m.Inputs.SearchInput.Update(msg)
			return m, cmd
		}
	}

	switch key {
	case "esc", "q":
		m.Screen = types.ScreenIndices
	case "j", "down":
		if len(m.Documents) > 0 {
			m.SelectedDocIdx = (m.SelectedDocIdx + 1) % len(m.Documents)
			m.refreshDocPreviewFromSelection()
		}
	case "k", "up":
		if len(m.Documents) > 0 {
			m.SelectedDocIdx = (m.SelectedDocIdx - 1 + len(m.Documents)) % len(m.Documents)
			m.refreshDocPreviewFromSelection()
		}
	case "enter":
		if len(m.Documents) == 0 || m.Cmds == nil {
			// run match_all if empty
			if m.Cmds != nil && m.CurrentIndex != nil {
				m.Loading = true
				return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, 0, 50)
			}
			return m, nil
		}
		doc := m.Documents[m.SelectedDocIdx]
		m.Loading = true
		return m, m.Cmds.LoadDocument(doc.Index, doc.ID)
	case "f":
		if m.Inputs != nil {
			m.Inputs.SearchInput.Focus()
		}
	case "/":
		idx := ""
		if m.CurrentIndex != nil {
			idx = m.CurrentIndex.Name
		}
		return m.openSearchScreen(idx, m.DocQuery, true), nil
	case "e":
		body := "{\n  \n}"
		id := ""
		if len(m.Documents) > 0 {
			doc := m.Documents[m.SelectedDocIdx]
			id = doc.ID
			body = prettyJSONBody(doc)
		}
		return m.openDocEditor(id, body), nil
	case "d":
		if len(m.Documents) == 0 {
			return m, nil
		}
		m.ConfirmType = "document"
		m.ConfirmData = m.Documents[m.SelectedDocIdx]
		m.Screen = types.ScreenConfirmDelete
	case "D":
		m.Screen = types.ScreenBulkDelete
		if m.Inputs != nil {
			m.Inputs.BulkDeleteInput.SetValue("*")
			m.Inputs.BulkDeleteInput.Focus()
		}
	case "r":
		if m.Cmds != nil && m.CurrentIndex != nil {
			m.Loading = true
			pageSize := m.PageSize
			if pageSize <= 0 {
				pageSize = 50
			}
			return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, m.DocFrom, pageSize)
		}
	case "n":
		if m.Cmds == nil || m.CurrentIndex == nil {
			return m, nil
		}
		pageSize := m.docPageSize()
		next := m.DocFrom + pageSize
		if !m.canDocPageNext() {
			m.StatusMsg = "Last page"
			return m, nil
		}
		m.DocFrom = next
		m.Loading = true
		return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, m.DocFrom, pageSize)
	case "p":
		if m.Cmds == nil || m.CurrentIndex == nil {
			return m, nil
		}
		if m.DocFrom <= 0 {
			m.DocFrom = 0
			m.StatusMsg = "First page"
			return m, nil
		}
		pageSize := m.docPageSize()
		m.DocFrom -= pageSize
		if m.DocFrom < 0 {
			m.DocFrom = 0
		}
		m.Loading = true
		return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, m.DocFrom, pageSize)
	case "y":
		if len(m.Documents) == 0 || m.Cmds == nil {
			return m, nil
		}
		doc := m.Documents[clamp(m.SelectedDocIdx, 0, len(m.Documents)-1)]
		content := doc.Raw
		if content == "" && doc.Source != nil {
			if b, err := json.MarshalIndent(doc.Source, "", "  "); err == nil {
				content = string(b)
			}
		}
		return m, m.Cmds.CopyToClipboard(content)
	}
	return m, nil
}

func (m Model) handleDocumentDetailKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.Screen = types.ScreenDocuments
		m.DetailScroll = 0
		m.DetailCursor = 0
	case "j", "down":
		if m.DetailCursor < m.documentDetailLastLine() {
			m.DetailCursor++
			m.syncDocumentDetailScroll()
		}
	case "k", "up":
		if m.DetailCursor > 0 {
			m.DetailCursor--
			m.syncDocumentDetailScroll()
		}
	case "pgdown", "ctrl+d":
		m.DetailCursor += 10
		if last := m.documentDetailLastLine(); m.DetailCursor > last {
			m.DetailCursor = last
		}
		m.syncDocumentDetailScroll()
	case "pgup", "ctrl+u":
		m.DetailCursor -= 10
		if m.DetailCursor < 0 {
			m.DetailCursor = 0
		}
		m.syncDocumentDetailScroll()
	case "g", "home":
		m.DetailCursor = 0
		m.syncDocumentDetailScroll()
	case "G", "end":
		m.DetailCursor = m.documentDetailLastLine()
		m.syncDocumentDetailScroll()
	case "e":
		if m.CurrentDocument != nil {
			return m.openDocEditor(m.CurrentDocument.ID, prettyJSONBody(*m.CurrentDocument)), nil
		}
	case "d":
		if m.CurrentDocument != nil {
			m.ConfirmType = "document"
			m.ConfirmData = *m.CurrentDocument
			m.Screen = types.ScreenConfirmDelete
		}
	case "y":
		if m.CurrentDocument == nil || m.Cmds == nil {
			return m, nil
		}
		content := m.CurrentDocument.Raw
		if content == "" && m.CurrentDocument.Source != nil {
			if b, err := json.MarshalIndent(m.CurrentDocument.Source, "", "  "); err == nil {
				content = string(b)
			}
		}
		return m, m.Cmds.CopyToClipboard(content)
	}
	return m, nil
}

func (m Model) handleSearchKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	pageSize := m.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	// Multiline query editor focused.
	if m.SearchFocus == "query" {
		m = m.ensureSearchArea()
		switch key {
		case "esc":
			// Keep draft text, but leave search entirely when there is nothing to browse.
			// Avoid the awkward "stuck on Query: test" intermediate state.
			m.SearchQuery = m.SearchArea.Value()
			if m.SearchResult == nil || len(m.SearchResult.Hits) == 0 {
				return m.leaveSearch(), nil
			}
			m.SearchArea.Blur()
			m.SearchFocus = "results"
			return m, nil
		case "ctrl+enter", "ctrl+s", "ctrl+r":
			return m.runSearchQuery()
		case "ctrl+p": // previous history
			if len(m.QueryHistory) == 0 {
				return m, nil
			}
			if m.HistoryIdx < len(m.QueryHistory)-1 {
				m.HistoryIdx++
				m.SearchArea.SetValue(m.QueryHistory[m.HistoryIdx])
			}
			return m, nil
		case "ctrl+n": // next history
			if m.HistoryIdx > 0 {
				m.HistoryIdx--
				m.SearchArea.SetValue(m.QueryHistory[m.HistoryIdx])
			} else if m.HistoryIdx == 0 {
				m.HistoryIdx = -1
				m.SearchArea.SetValue("")
			}
			return m, nil
		case "ctrl+1", "ctrl+2", "ctrl+3", "ctrl+4", "ctrl+5":
			n := int(key[len(key)-1] - '1')
			return m.applySearchTemplate(n), nil
		default:
			// Digits 1-5 with empty editor insert template quickly.
			if len(key) == 1 && key[0] >= '1' && key[0] <= '5' && strings.TrimSpace(m.SearchArea.Value()) == "" {
				return m.applySearchTemplate(int(key[0] - '1')), nil
			}
			area, cmd := m.SearchArea.Update(msg)
			m.SearchArea = &area
			return m, cmd
		}
	}

	switch key {
	case "esc", "q":
		return m.leaveSearch(), nil
	case "/":
		return m.focusSearchQuery(), nil
	case "enter":
		if m.SearchResult != nil && len(m.SearchResult.Hits) > 0 {
			if m.Cmds != nil {
				doc := m.SearchResult.Hits[clamp(m.SelectedDocIdx, 0, len(m.SearchResult.Hits)-1)]
				m.Loading = true
				return m, m.Cmds.LoadDocument(doc.Index, doc.ID)
			}
		}
		return m.focusSearchQuery(), nil
	case "tab":
		return m.focusSearchQuery(), nil
	case "1", "2", "3", "4", "5":
		return m.applySearchTemplate(int(key[0] - '1')), nil
	case "j", "down":
		hits := searchHits(m)
		if len(hits) > 0 {
			m.SelectedDocIdx = (m.SelectedDocIdx + 1) % len(hits)
		}
	case "k", "up":
		hits := searchHits(m)
		if len(hits) > 0 {
			m.SelectedDocIdx = (m.SelectedDocIdx - 1 + len(hits)) % len(hits)
		}
	case "g", "home":
		m.SelectedDocIdx = 0
	case "G", "end":
		hits := searchHits(m)
		if len(hits) > 0 {
			m.SelectedDocIdx = len(hits) - 1
		}
	case "o":
		hits := searchHits(m)
		if len(hits) > 0 && m.Cmds != nil {
			doc := hits[clamp(m.SelectedDocIdx, 0, len(hits)-1)]
			m.Loading = true
			return m, m.Cmds.LoadDocument(doc.Index, doc.ID)
		}
	case "y":
		hits := searchHits(m)
		if len(hits) == 0 || m.Cmds == nil {
			return m, nil
		}
		doc := hits[clamp(m.SelectedDocIdx, 0, len(hits)-1)]
		content := doc.Raw
		if content == "" && doc.Source != nil {
			if b, err := json.MarshalIndent(doc.Source, "", "  "); err == nil {
				content = string(b)
			}
		}
		return m, m.Cmds.CopyToClipboard(content)
	case "n":
		if m.Cmds == nil {
			return m, nil
		}
		if !m.canSearchPageNext() {
			m.StatusMsg = "Last page"
			return m, nil
		}
		m.SearchFrom += pageSize
		m.Loading = true
		return m, m.Cmds.Search(m.SearchIndex, m.searchQueryValue(), m.SearchFrom, pageSize)
	case "p":
		if m.Cmds == nil {
			return m, nil
		}
		if m.SearchFrom <= 0 {
			m.SearchFrom = 0
			m.StatusMsg = "First page"
			return m, nil
		}
		m.SearchFrom -= pageSize
		if m.SearchFrom < 0 {
			m.SearchFrom = 0
		}
		m.Loading = true
		return m, m.Cmds.Search(m.SearchIndex, m.searchQueryValue(), m.SearchFrom, pageSize)
	case "r":
		if m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.Search(m.SearchIndex, m.searchQueryValue(), m.SearchFrom, pageSize)
	case "S":
		if m.Cmds == nil {
			return m, nil
		}
		q := m.searchQueryValue()
		if strings.TrimSpace(q) == "" {
			m.Err = fmt.Errorf("nothing to save")
			return m, nil
		}
		name := fmt.Sprintf("q-%d", time.Now().Unix()%100000)
		return m, m.Cmds.AddSavedQuery(types.SavedQuery{
			Name:  name,
			Index: m.SearchIndex,
			Query: q,
		})
	case "#":
		if m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.Count(m.SearchIndex, m.searchQueryValue())
	case "x":
		hits := searchHits(m)
		if len(hits) == 0 || m.Cmds == nil {
			return m, nil
		}
		doc := hits[clamp(m.SelectedDocIdx, 0, len(hits)-1)]
		m.Loading = true
		return m, m.Cmds.Explain(doc.Index, doc.ID, m.searchQueryValue())
	case ":":
		return m.openPalette()
	}
	return m, nil
}

func searchHits(m Model) []types.Document {
	if m.SearchResult != nil && len(m.SearchResult.Hits) > 0 {
		return m.SearchResult.Hits
	}
	return m.Documents
}

func (m Model) handleConfirmDeleteKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "n", "esc", "q":
		return m.goBackFromConfirm()
	case "y":
		if m.Cmds == nil {
			return m.goBackFromConfirm()
		}
		m.Loading = true
		switch m.ConfirmType {
		case "connection":
			if conn, ok := m.ConfirmData.(types.Connection); ok {
				return m, m.Cmds.DeleteConnection(conn.ID)
			}
		case "index":
			if name, ok := m.ConfirmData.(string); ok {
				return m, m.Cmds.DeleteIndex(name)
			}
		case "document":
			if doc, ok := m.ConfirmData.(types.Document); ok {
				return m, m.Cmds.DeleteDocument(doc.Index, doc.ID)
			}
		}
		return m.goBackFromConfirm()
	}
	return m, nil
}

func (m Model) goBackFromConfirm() (tea.Model, tea.Cmd) {
	switch m.ConfirmType {
	case "connection":
		m.Screen = types.ScreenConnections
	case "index":
		if m.CurrentIndex != nil {
			m.Screen = types.ScreenIndexDetail
		} else {
			m.Screen = types.ScreenIndices
		}
	case "document":
		m.Screen = types.ScreenDocuments
	default:
		m.Screen = types.ScreenConnections
	}
	return m, nil
}

func (m Model) handleSimpleBackKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		return m.goBack()
	case "j", "down":
		if m.Screen == types.ScreenNodes && len(m.Nodes) > 0 {
			m.SelectedNode = (m.SelectedNode + 1) % len(m.Nodes)
		} else {
			// Cap scroll so we don't walk past EOF (JSON panels / shards lists).
			m.DetailScroll++
			if m.DetailScroll > 100000 {
				m.DetailScroll = 100000
			}
		}
	case "k", "up":
		if m.Screen == types.ScreenNodes && len(m.Nodes) > 0 {
			m.SelectedNode = (m.SelectedNode - 1 + len(m.Nodes)) % len(m.Nodes)
		} else if m.DetailScroll > 0 {
			m.DetailScroll--
		}
	case "r":
		if m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		switch m.Screen {
		case types.ScreenClusterHealth:
			return m, m.Cmds.LoadClusterHealth()
		case types.ScreenNodes:
			return m, m.Cmds.LoadNodes()
		}
	}
	return m, nil
}

func (m Model) handleLiveMetricsKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.LiveMetricsActive = false
		m.Screen = types.ScreenIndices
	}
	return m, nil
}

func (m Model) handleIndexCreateKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs == nil {
		return m, nil
	}
	switch key {
	case "esc":
		m.Screen = types.ScreenIndices
		return m, nil
	case "tab":
		if m.Inputs.IndexNameInput.Focused() {
			m.Inputs.IndexNameInput.Blur()
			m.Inputs.IndexBodyInput.Focus()
		} else {
			m.Inputs.IndexBodyInput.Blur()
			m.Inputs.IndexNameInput.Focus()
		}
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.Inputs.IndexNameInput.Value())
		if name == "" {
			m.Err = fmt.Errorf("index name required")
			return m, nil
		}
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.CreateIndex(name, m.Inputs.IndexBodyInput.Value())
		}
		return m, nil
	default:
		var cmd tea.Cmd
		if m.Inputs.IndexNameInput.Focused() {
			m.Inputs.IndexNameInput, cmd = m.Inputs.IndexNameInput.Update(msg)
		} else {
			m.Inputs.IndexBodyInput, cmd = m.Inputs.IndexBodyInput.Update(msg)
		}
		return m, cmd
	}
}

func (m Model) openDocEditor(id, body string) Model {
	m.Screen = types.ScreenEditDocument
	m.DocEditFocus = "body"
	if m.Inputs != nil {
		m.Inputs.DocIDInput.SetValue(id)
		m.Inputs.DocIDInput.Blur()
		m.Inputs.DocBodyInput.SetValue(body)
		m.Inputs.DocBodyInput.Blur()
	}
	w := max(m.Width-4, 40)
	h := max(m.Height-12, 10)
	m.DocEditor = editor.New(body, w, h, "document.json")
	return m
}

func prettyJSONBody(doc types.Document) string {
	if s := strings.TrimSpace(doc.Raw); s != "" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(s), "", "  "); err == nil {
			return buf.String()
		}
		return doc.Raw
	}
	if doc.Source != nil {
		if b, err := json.MarshalIndent(doc.Source, "", "  "); err == nil {
			return string(b)
		}
	}
	return "{\n  \n}"
}

func (m Model) handleEditDocumentKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Tab toggles ID field vs multiline body editor (redis-style body is the main surface).
	if key == "tab" || key == "shift+tab" {
		if m.DocEditFocus == "id" {
			m.DocEditFocus = "body"
			if m.Inputs != nil {
				m.Inputs.DocIDInput.Blur()
			}
			return m, nil
		}
		m.DocEditFocus = "id"
		if m.Inputs != nil {
			m.Inputs.DocIDInput.Focus()
		}
		return m, nil
	}

	// Save / quit handled here (same as redis-tui) so state updates immediately.
	if key == "ctrl+s" {
		body := ""
		if m.DocEditor != nil {
			body = m.DocEditor.Value()
		} else if m.Inputs != nil {
			body = m.Inputs.DocBodyInput.Value()
		}
		if m.CurrentIndex == nil || m.Cmds == nil {
			return m, nil
		}
		id := ""
		if m.Inputs != nil {
			id = strings.TrimSpace(m.Inputs.DocIDInput.Value())
		}
		m.Loading = true
		return m, m.Cmds.SaveDocument(m.CurrentIndex.Name, id, body)
	}
	if key == "esc" || key == "ctrl+q" {
		m.DocEditor = nil
		m.DocEditFocus = ""
		if m.CurrentDocument != nil {
			m.Screen = types.ScreenDocumentDetail
		} else {
			m.Screen = types.ScreenDocuments
		}
		return m, nil
	}

	if m.DocEditFocus == "id" {
		if m.Inputs == nil {
			return m, nil
		}
		var cmd tea.Cmd
		m.Inputs.DocIDInput, cmd = m.Inputs.DocIDInput.Update(msg)
		return m, cmd
	}

	// Body editor (default) — multiline textarea like redis-tui.
	if m.DocEditor == nil {
		body := "{\n  \n}"
		if m.Inputs != nil && m.Inputs.DocBodyInput.Value() != "" {
			body = m.Inputs.DocBodyInput.Value()
		}
		m.DocEditor = editor.New(body, max(m.Width-4, 40), max(m.Height-12, 10), "document.json")
	}
	// Don't let editor swallow esc/ctrl+s again as async msgs — already handled above.
	if key == "esc" || key == "ctrl+q" || key == "ctrl+s" {
		return m, nil
	}
	updated, cmd := m.DocEditor.Update(msg)
	m.DocEditor = updated
	return m, cmd
}

func (m Model) handleBulkDeleteKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs == nil {
		return m, nil
	}
	switch key {
	case "esc":
		m.Screen = types.ScreenDocuments
		return m, nil
	case "enter":
		if m.CurrentIndex == nil || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.BulkDelete(m.CurrentIndex.Name, m.Inputs.BulkDeleteInput.Value())
	default:
		var cmd tea.Cmd
		m.Inputs.BulkDeleteInput, cmd = m.Inputs.BulkDeleteInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleFavoritesKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.Screen = types.ScreenIndices
	case "j", "down":
		if len(m.Favorites) > 0 {
			m.SelectedFavIdx = (m.SelectedFavIdx + 1) % len(m.Favorites)
		}
	case "k", "up":
		if len(m.Favorites) > 0 {
			m.SelectedFavIdx = (m.SelectedFavIdx - 1 + len(m.Favorites)) % len(m.Favorites)
		}
	case "enter":
		if len(m.Favorites) == 0 || m.Cmds == nil {
			return m, nil
		}
		name := m.Favorites[m.SelectedFavIdx].Index
		m.Loading = true
		return m, m.Cmds.LoadDocuments(name, "", 0, 50)
	case "d":
		if len(m.Favorites) == 0 || m.CurrentConn == nil || m.Cmds == nil {
			return m, nil
		}
		return m, m.Cmds.RemoveFavorite(m.CurrentConn.ID, m.Favorites[m.SelectedFavIdx].Index)
	}
	return m, nil
}

func (m Model) handleRecentKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.Screen = types.ScreenIndices
	case "j", "down":
		if len(m.RecentIndices) > 0 {
			m.SelectedRecentIdx = (m.SelectedRecentIdx + 1) % len(m.RecentIndices)
		}
	case "k", "up":
		if len(m.RecentIndices) > 0 {
			m.SelectedRecentIdx = (m.SelectedRecentIdx - 1 + len(m.RecentIndices)) % len(m.RecentIndices)
		}
	case "enter":
		if len(m.RecentIndices) == 0 || m.Cmds == nil {
			return m, nil
		}
		name := m.RecentIndices[m.SelectedRecentIdx].Index
		m.Loading = true
		return m, m.Cmds.LoadDocuments(name, "", 0, 50)
	}
	return m, nil
}

func (m Model) handleCatAPIKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs == nil {
		return m, nil
	}
	switch key {
	case "esc":
		m.Screen = types.ScreenIndices
		m.Inputs.CatInput.Blur()
		return m, nil
	case "enter":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.CatAPI(m.Inputs.CatInput.Value())
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.Inputs.CatInput, cmd = m.Inputs.CatInput.Update(msg)
		return m, cmd
	}
}

func (m Model) goBack() (tea.Model, tea.Cmd) {
	switch m.Screen {
	case types.ScreenHelp:
		if m.PrevScreen != types.ScreenHelp {
			m.Screen = m.PrevScreen
		} else if m.CurrentConn != nil {
			m.Screen = types.ScreenIndices
		} else {
			m.Screen = types.ScreenConnections
		}
		// leave PrevScreen as-is; next help open overwrites it
		return m, nil
	case types.ScreenLogs, types.ScreenTestConnection,
		types.ScreenClusterHealth, types.ScreenNodes, types.ScreenShards,
		types.ScreenAliases, types.ScreenIndexTemplates, types.ScreenFavorites,
		types.ScreenRecentIndices, types.ScreenCatAPI, types.ScreenLiveMetrics,
		types.ScreenSearch, types.ScreenIndexCreate, types.ScreenIndexSettings,
		types.ScreenIndexMappings, types.ScreenCommandPalette, types.ScreenAllocation,
		types.ScreenTasks, types.ScreenPlugins, types.ScreenDataStreams,
		types.ScreenClusterSettings, types.ScreenSnapshots, types.ScreenReindex,
		types.ScreenExport, types.ScreenSavedQueries, types.ScreenExplain:
		if m.CurrentConn != nil {
			m.Screen = types.ScreenIndices
		} else {
			m.Screen = types.ScreenConnections
		}
		m.LiveMetricsActive = false
	case types.ScreenIndexDetail, types.ScreenDocuments:
		m.Screen = types.ScreenIndices
	case types.ScreenDocumentDetail, types.ScreenEditDocument, types.ScreenBulkDelete:
		m.Screen = types.ScreenDocuments
	default:
		m.Screen = types.ScreenConnections
	}
	return m, nil
}

// Ensure textinput import used
var _ = textinput.EchoPassword
