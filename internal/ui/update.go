package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
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
		if m.Flavor == "" && m.Cmds != nil {
			m.Flavor = m.Cmds.ES().Flavor()
		}
		if m.Cmds != nil {
			m.ReadOnly = m.Cmds.ES().IsReadOnly()
		}
		if m.CurrentConn != nil {
			m.ReadOnly = m.CurrentConn.ReadOnly
		}
		m.ConnectionError = ""
		m.Err = nil
		m.Screen = types.ScreenIndices
		status := fmt.Sprintf("Connected to %s (%s)", msg.Info.ClusterName, msg.Info.Version.Number)
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
		m.Screen = types.ScreenConnections
		m.StatusMsg = "Disconnected"
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
		m.DetailScroll = 0
		m.Screen = types.ScreenDocumentDetail
		return m, nil

	case types.DocumentSavedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
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
		m.ClusterHealth = msg.Health
		m.Screen = types.ScreenClusterHealth
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
			m.TestConnResult = successStyle.Render(fmt.Sprintf(
				"OK · %s %s · latency %s · %s",
				msg.Info.ClusterName,
				msg.Info.Version.Number,
				msg.Latency.Round(time.Millisecond),
				msg.Info.Flavor,
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
		if m.LiveMetricsActive && m.Cmds != nil {
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

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global force quit
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	// Clear status on any key
	m.StatusMsg = ""
	m.Err = nil

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
		m.ConnInputs[0].Focus()
		m.EditingConn = nil
	case "e":
		if len(m.Connections) == 0 {
			return m, nil
		}
		conn := m.Connections[m.SelectedConnIdx]
		m.EditingConn = &conn
		m.Screen = types.ScreenEditConnection
		m.ConnInputs = createConnectionInputs()
		m.ConnInputs[0].SetValue(conn.Name)
		m.ConnInputs[1].SetValue(conn.Host)
		m.ConnInputs[2].SetValue(strconv.Itoa(conn.Port))
		m.ConnInputs[3].SetValue(conn.Username)
		m.ConnInputs[4].SetValue(conn.Password)
		m.ConnInputs[5].SetValue(conn.APIKey)
		m.ConnInputs[6].SetValue(conn.BearerToken)
		flavor := string(conn.Flavor)
		if flavor == "" {
			flavor = "auto"
		}
		m.ConnInputs[7].SetValue(flavor)
		if conn.ReadOnly {
			m.ConnInputs[8].SetValue("true")
		} else {
			m.ConnInputs[8].SetValue("false")
		}
		m.ConnFocusIdx = 0
		m.ConnInputs[0].Focus()
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
	case "?":
		m.Screen = types.ScreenHelp
	case "L":
		m.Screen = types.ScreenLogs
	case ":":
		m.Screen = types.ScreenCommandPalette
		m.PaletteItems = defaultPaletteItems()
		m.PaletteIdx = 0
		if m.Inputs != nil {
			m.Inputs.PaletteInput.SetValue("")
			m.Inputs.PaletteInput.Focus()
		}
	}
	return m, nil
}

func (m Model) handleConnectionFormKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.Screen = types.ScreenConnections
		return m, nil
	case "tab", "down":
		m.ConnInputs[m.ConnFocusIdx].Blur()
		m.ConnFocusIdx = (m.ConnFocusIdx + 1) % len(m.ConnInputs)
		m.ConnInputs[m.ConnFocusIdx].Focus()
		return m, nil
	case "shift+tab", "up":
		m.ConnInputs[m.ConnFocusIdx].Blur()
		m.ConnFocusIdx = (m.ConnFocusIdx - 1 + len(m.ConnInputs)) % len(m.ConnInputs)
		m.ConnInputs[m.ConnFocusIdx].Focus()
		return m, nil
	case "enter":
		conn, err := m.connectionFromInputs()
		if err != nil {
			m.Err = err
			return m, nil
		}
		if m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		if m.Screen == types.ScreenEditConnection && m.EditingConn != nil {
			conn.ID = m.EditingConn.ID
			return m, m.Cmds.UpdateConnection(conn)
		}
		return m, m.Cmds.AddConnection(conn)
	default:
		var cmd tea.Cmd
		m.ConnInputs[m.ConnFocusIdx], cmd = m.ConnInputs[m.ConnFocusIdx].Update(msg)
		return m, cmd
	}
}

func (m Model) connectionFromInputs() (types.Connection, error) {
	name := strings.TrimSpace(m.ConnInputs[0].Value())
	host := strings.TrimSpace(m.ConnInputs[1].Value())
	portStr := strings.TrimSpace(m.ConnInputs[2].Value())
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
	flavor := types.Flavor(strings.ToLower(strings.TrimSpace(m.ConnInputs[7].Value())))
	switch flavor {
	case types.FlavorElasticsearch, types.FlavorOpenSearch, types.FlavorAuto, "":
	default:
		return types.Connection{}, fmt.Errorf("flavor must be auto, elasticsearch, or opensearch")
	}
	if flavor == "" {
		flavor = types.FlavorAuto
	}
	ro := strings.EqualFold(strings.TrimSpace(m.ConnInputs[8].Value()), "true")
	return types.Connection{
		Name:        name,
		Host:        host,
		Port:        port,
		Username:    strings.TrimSpace(m.ConnInputs[3].Value()),
		Password:    m.ConnInputs[4].Value(),
		APIKey:      strings.TrimSpace(m.ConnInputs[5].Value()),
		BearerToken: strings.TrimSpace(m.ConnInputs[6].Value()),
		Flavor:      flavor,
		ReadOnly:    ro,
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
			return m, tea.Batch(m.Cmds.Disconnect(), func() tea.Msg { return types.DisconnectedMsg{} })
		}
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
		m.SearchIndex = ""
		if len(m.Indices) > 0 {
			m.SearchIndex = m.Indices[m.SelectedIndexIdx].Name
		}
		m.Screen = types.ScreenSearch
		m.SearchFocus = "query"
		m.SearchFrom = 0
		if m.Inputs != nil {
			m.Inputs.SearchInput.SetValue("")
			m.Inputs.SearchInput.Focus()
		}
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
			return m, m.Cmds.LoadClusterHealth()
		}
	case "n":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadNodes()
		}
	case "m":
		if m.Cmds != nil {
			m.LiveMetricsActive = true
			m.Loading = true
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
	case "?":
		m.Screen = types.ScreenHelp
	case "L":
		m.Screen = types.ScreenLogs
	case ":":
		return m.openPalette()
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
			m.SearchIndex = m.CurrentIndex.Name
			m.Screen = types.ScreenSearch
			if m.Inputs != nil {
				m.Inputs.SearchInput.Focus()
			}
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
		}
	case "k", "up":
		if len(m.Documents) > 0 {
			m.SelectedDocIdx = (m.SelectedDocIdx - 1 + len(m.Documents)) % len(m.Documents)
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
		if m.CurrentIndex != nil {
			m.SearchIndex = m.CurrentIndex.Name
		}
		m.Screen = types.ScreenSearch
		m.SearchFocus = "query"
		if m.Inputs != nil {
			m.Inputs.SearchInput.SetValue(m.DocQuery)
			m.Inputs.SearchInput.Focus()
		}
	case "e":
		m.Screen = types.ScreenEditDocument
		if m.Inputs != nil {
			m.Inputs.DocIDInput.SetValue("")
			m.Inputs.DocBodyInput.SetValue(`{"message":"hello"}`)
			if len(m.Documents) > 0 {
				doc := m.Documents[m.SelectedDocIdx]
				m.Inputs.DocIDInput.SetValue(doc.ID)
				m.Inputs.DocBodyInput.SetValue(doc.Raw)
			}
			m.Inputs.DocBodyInput.Focus()
		}
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
		pageSize := m.PageSize
		if pageSize <= 0 {
			pageSize = 50
		}
		m.DocFrom += pageSize
		m.Loading = true
		return m, m.Cmds.LoadDocuments(m.CurrentIndex.Name, m.DocQuery, m.DocFrom, pageSize)
	case "p":
		if m.Cmds == nil || m.CurrentIndex == nil {
			return m, nil
		}
		pageSize := m.PageSize
		if pageSize <= 0 {
			pageSize = 50
		}
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
	case "j", "down":
		m.DetailScroll++
	case "k", "up":
		if m.DetailScroll > 0 {
			m.DetailScroll--
		}
	case "e":
		if m.CurrentDocument != nil {
			m.Screen = types.ScreenEditDocument
			if m.Inputs != nil {
				m.Inputs.DocIDInput.SetValue(m.CurrentDocument.ID)
				m.Inputs.DocBodyInput.SetValue(m.CurrentDocument.Raw)
				m.Inputs.DocBodyInput.Focus()
			}
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

	if m.Inputs != nil && m.Inputs.SearchInput.Focused() {
		switch key {
		case "esc":
			m.Inputs.SearchInput.Blur()
			m.SearchFocus = "results"
			return m, nil
		case "enter":
			m.Inputs.SearchInput.Blur()
			m.SearchFocus = "results"
			q := m.Inputs.SearchInput.Value()
			m.SearchQuery = q
			m.pushQueryHistory(q)
			m.SearchFrom = 0
			if m.Cmds != nil {
				m.Loading = true
				return m, m.Cmds.Search(m.SearchIndex, q, 0, pageSize)
			}
			return m, nil
		case "up":
			// history
			if len(m.QueryHistory) == 0 {
				return m, nil
			}
			if m.HistoryIdx < len(m.QueryHistory)-1 {
				m.HistoryIdx++
				m.Inputs.SearchInput.SetValue(m.QueryHistory[m.HistoryIdx])
			}
			return m, nil
		case "down":
			if m.HistoryIdx > 0 {
				m.HistoryIdx--
				m.Inputs.SearchInput.SetValue(m.QueryHistory[m.HistoryIdx])
			} else if m.HistoryIdx == 0 {
				m.HistoryIdx = -1
				m.Inputs.SearchInput.SetValue("")
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
		if m.CurrentIndex != nil {
			m.Screen = types.ScreenDocuments
		} else {
			m.Screen = types.ScreenIndices
		}
	case "/", "enter":
		if key == "enter" && m.SearchResult != nil && len(m.SearchResult.Hits) > 0 && m.SearchFocus == "results" {
			// open selected hit
			if m.Cmds != nil {
				doc := m.SearchResult.Hits[clamp(m.SelectedDocIdx, 0, len(m.SearchResult.Hits)-1)]
				m.Loading = true
				return m, m.Cmds.LoadDocument(doc.Index, doc.ID)
			}
		}
		if m.Inputs != nil {
			m.SearchFocus = "query"
			m.Inputs.SearchInput.Focus()
		}
	case "tab":
		if m.SearchFocus == "query" {
			m.SearchFocus = "results"
			if m.Inputs != nil {
				m.Inputs.SearchInput.Blur()
			}
		} else {
			m.SearchFocus = "query"
			if m.Inputs != nil {
				m.Inputs.SearchInput.Focus()
			}
		}
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
		m.SearchFrom += pageSize
		m.Loading = true
		q := m.SearchQuery
		if m.Inputs != nil && m.Inputs.SearchInput.Value() != "" {
			q = m.Inputs.SearchInput.Value()
		}
		return m, m.Cmds.Search(m.SearchIndex, q, m.SearchFrom, pageSize)
	case "p":
		if m.Cmds == nil {
			return m, nil
		}
		m.SearchFrom -= pageSize
		if m.SearchFrom < 0 {
			m.SearchFrom = 0
		}
		m.Loading = true
		q := m.SearchQuery
		if m.Inputs != nil && m.Inputs.SearchInput.Value() != "" {
			q = m.Inputs.SearchInput.Value()
		}
		return m, m.Cmds.Search(m.SearchIndex, q, m.SearchFrom, pageSize)
	case "r":
		if m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		q := m.SearchQuery
		if m.Inputs != nil && m.Inputs.SearchInput.Value() != "" {
			q = m.Inputs.SearchInput.Value()
		}
		return m, m.Cmds.Search(m.SearchIndex, q, m.SearchFrom, pageSize)
	case "S":
		// save current query
		if m.Cmds == nil {
			return m, nil
		}
		q := m.SearchQuery
		if m.Inputs != nil && m.Inputs.SearchInput.Value() != "" {
			q = m.Inputs.SearchInput.Value()
		}
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
		q := m.SearchQuery
		if m.Inputs != nil && m.Inputs.SearchInput.Value() != "" {
			q = m.Inputs.SearchInput.Value()
		}
		m.Loading = true
		return m, m.Cmds.Count(m.SearchIndex, q)
	case "x":
		// explain selected hit
		hits := searchHits(m)
		if len(hits) == 0 || m.Cmds == nil {
			return m, nil
		}
		doc := hits[clamp(m.SelectedDocIdx, 0, len(hits)-1)]
		q := m.SearchQuery
		if m.Inputs != nil && m.Inputs.SearchInput.Value() != "" {
			q = m.Inputs.SearchInput.Value()
		}
		m.Loading = true
		return m, m.Cmds.Explain(doc.Index, doc.ID, q)
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
		m.DetailScroll++
		if m.Screen == types.ScreenNodes && len(m.Nodes) > 0 {
			m.SelectedNode = (m.SelectedNode + 1) % len(m.Nodes)
		}
	case "k", "up":
		if m.DetailScroll > 0 {
			m.DetailScroll--
		}
		if m.Screen == types.ScreenNodes && len(m.Nodes) > 0 {
			m.SelectedNode = (m.SelectedNode - 1 + len(m.Nodes)) % len(m.Nodes)
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

func (m Model) handleEditDocumentKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs == nil {
		return m, nil
	}
	switch key {
	case "esc":
		if m.CurrentDocument != nil {
			m.Screen = types.ScreenDocumentDetail
		} else {
			m.Screen = types.ScreenDocuments
		}
		return m, nil
	case "tab":
		if m.Inputs.DocIDInput.Focused() {
			m.Inputs.DocIDInput.Blur()
			m.Inputs.DocBodyInput.Focus()
		} else {
			m.Inputs.DocBodyInput.Blur()
			m.Inputs.DocIDInput.Focus()
		}
		return m, nil
	case "enter":
		if m.CurrentIndex == nil || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.SaveDocument(
			m.CurrentIndex.Name,
			strings.TrimSpace(m.Inputs.DocIDInput.Value()),
			m.Inputs.DocBodyInput.Value(),
		)
	default:
		var cmd tea.Cmd
		if m.Inputs.DocIDInput.Focused() {
			m.Inputs.DocIDInput, cmd = m.Inputs.DocIDInput.Update(msg)
		} else {
			m.Inputs.DocBodyInput, cmd = m.Inputs.DocBodyInput.Update(msg)
		}
		return m, cmd
	}
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
	case types.ScreenHelp, types.ScreenLogs, types.ScreenTestConnection,
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
