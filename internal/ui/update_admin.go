package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func (m Model) handleAdminMessages(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case types.AllocationLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.Allocation = msg.Allocation
		m.Screen = types.ScreenAllocation
		return m, nil, true
	case types.TasksLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.Tasks = msg.Tasks
		m.Screen = types.ScreenTasks
		return m, nil, true
	case types.PluginsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.Plugins = msg.Plugins
		m.Screen = types.ScreenPlugins
		return m, nil, true
	case types.DataStreamsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.DataStreams = msg.DataStreams
		m.Screen = types.ScreenDataStreams
		return m, nil, true
	case types.SnapshotsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.Snapshots = msg.Snapshots
		m.Screen = types.ScreenSnapshots
		return m, nil, true
	case types.ClusterSettingsLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.ClusterSettings = msg.Settings
		m.DetailScroll = 0
		m.Screen = types.ScreenClusterSettings
		return m, nil, true
	case types.ReindexMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.StatusMsg = "Reindex task: " + msg.Task
		m.Screen = types.ScreenIndices
		return m, nil, true
	case types.ExplainLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		r := msg.Result
		m.ExplainResult = &r
		m.Screen = types.ScreenExplain
		return m, nil, true
	case types.CountMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.CountResult = msg.Count
		m.StatusMsg = fmt.Sprintf("Count: %d", msg.Count)
		return m, nil, true
	case types.ExportCompleteMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.StatusMsg = fmt.Sprintf("Exported %d docs → %s", msg.Count, msg.Filename)
		m.Screen = types.ScreenIndices
		return m, nil, true
	case types.SavedQueriesLoadedMsg:
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.SavedQueries = msg.Queries
		m.Screen = types.ScreenSavedQueries
		return m, nil, true
	case types.SavedQueryAddedMsg:
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.StatusMsg = "Saved query: " + msg.Query.Name
		return m, nil, true
	case types.SavedQueryDeletedMsg:
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil, true
		}
		m.StatusMsg = "Deleted query: " + msg.Name
		if m.Cmds != nil {
			return m, m.Cmds.LoadSavedQueries(), true
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}

func (m Model) handleCommandPaletteKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs != nil && m.Inputs.PaletteInput.Focused() {
		switch key {
		case "esc":
			m.Inputs.PaletteInput.Blur()
			return m.goBack()
		case "enter":
			m.Inputs.PaletteInput.Blur()
			return m.runPaletteAction()
		case "down", "j":
			items := m.filteredPalette()
			if len(items) > 0 {
				m.PaletteIdx = (m.PaletteIdx + 1) % len(items)
			}
			return m, nil
		case "up", "k":
			items := m.filteredPalette()
			if len(items) > 0 {
				m.PaletteIdx = (m.PaletteIdx - 1 + len(items)) % len(items)
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.Inputs.PaletteInput, cmd = m.Inputs.PaletteInput.Update(msg)
			m.PaletteIdx = 0
			return m, cmd
		}
	}
	switch key {
	case "esc", "q":
		return m.goBack()
	case "j", "down":
		items := m.filteredPalette()
		if len(items) > 0 {
			m.PaletteIdx = (m.PaletteIdx + 1) % len(items)
		}
	case "k", "up":
		items := m.filteredPalette()
		if len(items) > 0 {
			m.PaletteIdx = (m.PaletteIdx - 1 + len(items)) % len(items)
		}
	case "enter":
		return m.runPaletteAction()
	case "/":
		if m.Inputs != nil {
			m.Inputs.PaletteInput.Focus()
		}
	}
	return m, nil
}

func (m Model) runPaletteAction() (tea.Model, tea.Cmd) {
	items := m.filteredPalette()
	if len(items) == 0 || m.Cmds == nil {
		m.Screen = types.ScreenIndices
		return m, nil
	}
	id := items[clamp(m.PaletteIdx, 0, len(items)-1)].ID
	m.Loading = true
	switch id {
	case "health":
		return m, m.Cmds.LoadClusterHealth()
	case "nodes":
		return m, m.Cmds.LoadNodes()
	case "metrics":
		m.LiveMetricsActive = true
		return m, m.Cmds.LoadLiveMetrics()
	case "shards":
		return m, m.Cmds.LoadShards("")
	case "allocation":
		return m, m.Cmds.LoadAllocation()
	case "aliases":
		return m, m.Cmds.LoadAliases()
	case "templates":
		return m, m.Cmds.LoadTemplates()
	case "datastreams":
		return m, m.Cmds.LoadDataStreams()
	case "tasks":
		return m, m.Cmds.LoadTasks()
	case "plugins":
		return m, m.Cmds.LoadPlugins()
	case "settings":
		return m, m.Cmds.LoadClusterSettings()
	case "snapshots":
		m.Loading = false
		m.Screen = types.ScreenSnapshots
		if m.Inputs != nil {
			m.Inputs.SnapshotRepo.Focus()
		}
		return m, nil
	case "search":
		m.Loading = false
		idx := ""
		if m.CurrentIndex != nil {
			idx = m.CurrentIndex.Name
		}
		return m.openSearchScreen(idx, "", true), nil
	case "reindex":
		m.Loading = false
		m.Screen = types.ScreenReindex
		m.ReindexFocus = 0
		if m.Inputs != nil {
			if m.CurrentIndex != nil {
				m.Inputs.ReindexSrcInput.SetValue(m.CurrentIndex.Name)
			}
			m.Inputs.ReindexSrcInput.Focus()
		}
		return m, nil
	case "export":
		m.Loading = false
		m.Screen = types.ScreenExport
		if m.Inputs != nil {
			m.Inputs.ExportInput.SetValue(fmt.Sprintf("/tmp/es-tui-export-%d.ndjson", time.Now().Unix()))
			m.Inputs.ExportInput.Focus()
		}
		return m, nil
	case "saved":
		m.Loading = false
		return m, m.Cmds.LoadSavedQueries()
	case "cat":
		m.Loading = false
		m.Screen = types.ScreenCatAPI
		if m.Inputs != nil {
			m.Inputs.CatInput.Focus()
		}
		return m, nil
	case "favorites":
		m.Loading = false
		if m.CurrentConn != nil {
			return m, m.Cmds.LoadFavorites(m.CurrentConn.ID)
		}
		return m, nil
	case "recent":
		m.Loading = false
		if m.CurrentConn != nil {
			return m, m.Cmds.LoadRecentIndices(m.CurrentConn.ID)
		}
		return m, nil
	case "logs":
		m.Loading = false
		m.Screen = types.ScreenLogs
		return m, nil
	case "help":
		m.Loading = false
		m.Screen = types.ScreenHelp
		return m, nil
	default:
		m.Loading = false
		m.Screen = types.ScreenIndices
		return m, nil
	}
}

func (m Model) handleReindexKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs == nil {
		return m, nil
	}
	switch key {
	case "esc":
		m.Screen = types.ScreenIndices
		return m, nil
	case "tab":
		m.ReindexFocus = 1 - m.ReindexFocus
		if m.ReindexFocus == 0 {
			m.Inputs.ReindexDstInput.Blur()
			m.Inputs.ReindexSrcInput.Focus()
		} else {
			m.Inputs.ReindexSrcInput.Blur()
			m.Inputs.ReindexDstInput.Focus()
		}
		return m, nil
	case "enter":
		if m.Cmds == nil {
			return m, nil
		}
		src := strings.TrimSpace(m.Inputs.ReindexSrcInput.Value())
		dst := strings.TrimSpace(m.Inputs.ReindexDstInput.Value())
		if src == "" || dst == "" {
			m.Err = fmt.Errorf("source and dest required")
			return m, nil
		}
		body := fmt.Sprintf(`{"source":{"index":%q},"dest":{"index":%q}}`, src, dst)
		m.Loading = true
		return m, m.Cmds.Reindex(body)
	default:
		var cmd tea.Cmd
		if m.ReindexFocus == 0 {
			m.Inputs.ReindexSrcInput, cmd = m.Inputs.ReindexSrcInput.Update(msg)
		} else {
			m.Inputs.ReindexDstInput, cmd = m.Inputs.ReindexDstInput.Update(msg)
		}
		return m, cmd
	}
}

func (m Model) handleExportKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs == nil {
		return m, nil
	}
	switch key {
	case "esc":
		return m.goBack()
	case "enter":
		if m.Cmds == nil {
			return m, nil
		}
		path := strings.TrimSpace(m.Inputs.ExportInput.Value())
		if path == "" {
			m.Err = fmt.Errorf("export path required")
			return m, nil
		}
		idx := ""
		if m.CurrentIndex != nil {
			idx = m.CurrentIndex.Name
		}
		if m.SearchIndex != "" {
			idx = m.SearchIndex
		}
		q := m.SearchQuery
		if q == "" {
			q = m.DocQuery
		}
		m.Loading = true
		// export via cmd then write file in message handler - ExportDocs returns docs
		// Use a custom cmd that writes file
		return m, m.exportDocsCmd(idx, q, path)
	default:
		var cmd tea.Cmd
		m.Inputs.ExportInput, cmd = m.Inputs.ExportInput.Update(msg)
		return m, cmd
	}
}

func (m Model) exportDocsCmd(index, query, path string) tea.Cmd {
	return func() tea.Msg {
		docs, err := m.Cmds.ES().ExportDocs(index, query, 5000)
		if err != nil {
			return types.ExportCompleteMsg{Filename: path, Err: err}
		}
		f, err := os.Create(path) // #nosec G304 -- path is user-chosen export destination
		if err != nil {
			return types.ExportCompleteMsg{Filename: path, Err: err}
		}
		defer f.Close()
		for _, d := range docs {
			var payload any = d.Source
			if payload == nil && d.Raw != "" {
				payload = json.RawMessage(d.Raw)
			}
			if payload == nil {
				continue
			}
			b, err := json.Marshal(payload)
			if err != nil {
				return types.ExportCompleteMsg{Filename: path, Err: err}
			}
			if _, err := f.Write(append(b, '\n')); err != nil {
				return types.ExportCompleteMsg{Filename: path, Err: err}
			}
		}
		return types.ExportCompleteMsg{Filename: path, Count: len(docs)}
	}
}

func (m Model) handleTasksKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		return m.goBack()
	case "j", "down":
		if len(m.Tasks) > 0 {
			m.SelectedTaskIdx = (m.SelectedTaskIdx + 1) % len(m.Tasks)
		}
	case "k", "up":
		if len(m.Tasks) > 0 {
			m.SelectedTaskIdx = (m.SelectedTaskIdx - 1 + len(m.Tasks)) % len(m.Tasks)
		}
	case "x":
		if len(m.Tasks) == 0 || m.Cmds == nil {
			return m, nil
		}
		m.Loading = true
		return m, m.Cmds.CancelTask(m.Tasks[m.SelectedTaskIdx].ID)
	case "r":
		if m.Cmds != nil {
			m.Loading = true
			return m, m.Cmds.LoadTasks()
		}
	}
	return m, nil
}

func (m Model) handleSavedQueriesKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		return m.goBack()
	case "j", "down":
		if len(m.SavedQueries) > 0 {
			m.SelectedSQIdx = (m.SelectedSQIdx + 1) % len(m.SavedQueries)
		}
	case "k", "up":
		if len(m.SavedQueries) > 0 {
			m.SelectedSQIdx = (m.SelectedSQIdx - 1 + len(m.SavedQueries)) % len(m.SavedQueries)
		}
	case "enter":
		if len(m.SavedQueries) == 0 || m.Cmds == nil {
			return m, nil
		}
		q := m.SavedQueries[m.SelectedSQIdx]
		m.SearchIndex = q.Index
		m.SearchQuery = q.Query
		m.SearchFrom = 0
		m.Screen = types.ScreenSearch
		m.SearchArea = nil
		m = m.ensureSearchArea()
		m.SearchArea.SetValue(q.Query)
		m.SearchFocus = "results"
		m.SearchArea.Blur()
		m.Loading = true
		pageSize := m.PageSize
		if pageSize <= 0 {
			pageSize = 50
		}
		return m, m.Cmds.Search(q.Index, q.Query, 0, pageSize)
	case "d":
		if len(m.SavedQueries) == 0 || m.Cmds == nil {
			return m, nil
		}
		return m, m.Cmds.DeleteSavedQuery(m.SavedQueries[m.SelectedSQIdx].Name)
	}
	return m, nil
}

func (m Model) handleSnapshotsKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Inputs != nil && m.Inputs.SnapshotRepo.Focused() {
		switch key {
		case "esc":
			m.Inputs.SnapshotRepo.Blur()
			return m.goBack()
		case "enter":
			m.Inputs.SnapshotRepo.Blur()
			if m.Cmds != nil {
				m.Loading = true
				return m, m.Cmds.LoadSnapshots(m.Inputs.SnapshotRepo.Value())
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.Inputs.SnapshotRepo, cmd = m.Inputs.SnapshotRepo.Update(msg)
			return m, cmd
		}
	}
	switch key {
	case "esc", "q":
		return m.goBack()
	case "enter", "/":
		if m.Inputs != nil {
			m.Inputs.SnapshotRepo.Focus()
		}
	case "r":
		if m.Cmds != nil {
			repo := ""
			if m.Inputs != nil {
				repo = m.Inputs.SnapshotRepo.Value()
			}
			m.Loading = true
			return m, m.Cmds.LoadSnapshots(repo)
		}
	}
	return m, nil
}

func (m Model) openPalette() (tea.Model, tea.Cmd) {
	m.Screen = types.ScreenCommandPalette
	m.PaletteItems = defaultPaletteItems()
	m.PaletteIdx = 0
	if m.Inputs != nil {
		m.Inputs.PaletteInput.SetValue("")
		m.Inputs.PaletteInput.Placeholder = "Type to filter commands…"
		m.Inputs.PaletteInput.SetWidth(44)
		m.Inputs.PaletteInput.Focus()
	}
	return m, nil
}
