package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func (m Model) getScreenView() string {
	switch m.Screen {
	case types.ScreenConnections:
		return m.viewConnections()
	case types.ScreenAddConnection, types.ScreenEditConnection:
		return m.viewConnectionForm()
	case types.ScreenIndices:
		return m.viewIndices()
	case types.ScreenIndexDetail:
		return m.viewIndexDetail()
	case types.ScreenDocuments:
		return m.viewDocuments()
	case types.ScreenDocumentDetail:
		return m.viewDocumentDetail()
	case types.ScreenSearch:
		return m.viewSearch()
	case types.ScreenHelp:
		return m.viewHelp()
	case types.ScreenConfirmDelete:
		return m.viewConfirmDelete()
	case types.ScreenClusterHealth:
		return m.viewClusterHealth()
	case types.ScreenNodes:
		return m.viewNodes()
	case types.ScreenIndexCreate:
		return m.viewIndexCreate()
	case types.ScreenIndexSettings:
		return m.viewJSONPanel("Index Settings", m.IndexSettings)
	case types.ScreenIndexMappings:
		return m.viewJSONPanel("Index Mappings", m.IndexMappings)
	case types.ScreenAliases:
		return m.viewAliases()
	case types.ScreenShards:
		return m.viewShards()
	case types.ScreenLiveMetrics:
		return m.viewLiveMetrics()
	case types.ScreenTestConnection:
		return m.viewTestConnection()
	case types.ScreenLogs:
		return m.viewLogs()
	case types.ScreenFavorites:
		return m.viewFavorites()
	case types.ScreenRecentIndices:
		return m.viewRecentIndices()
	case types.ScreenBulkDelete:
		return m.viewBulkDelete()
	case types.ScreenEditDocument:
		return m.viewEditDocument()
	case types.ScreenIndexTemplates:
		return m.viewTemplates()
	case types.ScreenCatAPI:
		return m.viewCatAPI()
	case types.ScreenAllocation:
		return m.viewAllocation()
	case types.ScreenTasks:
		return m.viewTasks()
	case types.ScreenPlugins:
		return m.viewPlugins()
	case types.ScreenDataStreams:
		return m.viewDataStreams()
	case types.ScreenSnapshots:
		return m.viewSnapshots()
	case types.ScreenClusterSettings:
		return m.viewClusterSettings()
	case types.ScreenReindex:
		return m.viewReindex()
	case types.ScreenExport:
		return m.viewExport()
	case types.ScreenSavedQueries:
		return m.viewSavedQueries()
	case types.ScreenExplain:
		return m.viewExplain()
	case types.ScreenCommandPalette:
		return m.viewCommandPalette()
	default:
		return ""
	}
}

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	if m.Width < 50 || m.Height < 15 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center,
			"Terminal too small.\nResize to at least 50x15.")
	}

	content := m.getScreenView()
	status := m.getStatusBar()
	fullContent := content + "\n\n" + status

	vPos := lipgloss.Position(lipgloss.Top)
	hPos := lipgloss.Center
	switch m.Screen {
	case types.ScreenConnections, types.ScreenAddConnection, types.ScreenEditConnection,
		types.ScreenConfirmDelete, types.ScreenTestConnection, types.ScreenHelp,
		types.ScreenIndexCreate, types.ScreenBulkDelete, types.ScreenEditDocument,
		types.ScreenDocumentDetail, types.ScreenIndexDetail:
		vPos = lipgloss.Center
	case types.ScreenIndices, types.ScreenDocuments, types.ScreenSearch:
		hPos = lipgloss.Left
		vPos = lipgloss.Top
	}

	return lipgloss.Place(m.Width, m.Height, hPos, vPos, fullContent,
		lipgloss.WithWhitespaceChars(" "))
}

func (m Model) getStatusBar() string {
	if m.Loading {
		return dimStyle.Render("Loading...")
	}
	if m.StatusMsg != "" {
		return successStyle.Render(m.StatusMsg)
	}
	if m.Err != nil {
		return errorStyle.Render(m.Err.Error())
	}

	var parts []string
	if m.CurrentConn != nil {
		parts = append(parts, successStyle.Render("Connected"))
		if m.CurrentConn.Name != "" {
			parts = append(parts, tealStyle.Render(m.CurrentConn.Name))
		}
		flavor := string(m.Flavor)
		if flavor == "" {
			flavor = "auto"
		}
		parts = append(parts, dimStyle.Render(flavor))
		if m.ReadOnly {
			parts = append(parts, yellowStyle.Render("RO"))
		}
		if m.ClusterHealth.Status != "" {
			parts = append(parts, healthStyle(m.ClusterHealth.Status).Render(m.ClusterHealth.Status))
		}
	}
	if m.UpdateAvailable != "" {
		parts = append(parts, yellowStyle.Render("update: "+m.UpdateAvailable))
	}
	parts = append(parts, helpStyle.Render("? help  q quit"))
	return strings.Join(parts, "  ·  ")
}


func (m Model) viewClusterHealth() string {
	h := m.ClusterHealth
	var b strings.Builder
	b.WriteString(titleStyle.Render("Cluster Health"))
	sepW := min(max(m.Width-4, 40), 60)
	b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")

	const labelW = 14
	writeMeta := func(label, value string) {
		b.WriteString(keyStyle.Render(fmt.Sprintf("%-*s", labelW, label)))
		b.WriteString(normalStyle.Render(value))
		b.WriteString("\n")
	}

	writeMeta("Cluster", h.ClusterName)
	b.WriteString(keyStyle.Render(fmt.Sprintf("%-*s", labelW, "Status")))
	b.WriteString(healthStyle(h.Status).Render(h.Status))
	b.WriteString("\n")
	writeMeta("Nodes", fmt.Sprintf("%d (%d data)", h.NumberOfNodes, h.NumberOfDataNodes))
	writeMeta("Shards", fmt.Sprintf("%d active (%d primary)", h.ActiveShards, h.ActivePrimaryShards))
	writeMeta("Relocating", fmt.Sprintf("%d", h.RelocatingShards))
	writeMeta("Initializing", fmt.Sprintf("%d", h.InitializingShards))
	writeMeta("Unassigned", fmt.Sprintf("%d", h.UnassignedShards))
	writeMeta("Active %", fmt.Sprintf("%.1f%%", h.ActiveShardsPercentAsNumber))
	if m.ClusterInfo.Version.Number != "" {
		writeMeta("Version", fmt.Sprintf("%s (%s)", m.ClusterInfo.Version.Number, m.Flavor))
		b.WriteString(keyStyle.Render(fmt.Sprintf("%-*s", labelW, "Tagline")))
		b.WriteString(dimStyle.Render(m.ClusterInfo.Tagline))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("r:refresh  q:back"))
	return b.String()
}

func (m Model) viewNodes() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Nodes (%d)", len(m.Nodes))))

	if len(m.Nodes) == 0 {
		b.WriteString(dimStyle.Render("No nodes"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("j/k:nav  r:refresh  q:back"))
		return b.String()
	}

	header := fmt.Sprintf("  %-20s %-14s %-6s %5s %5s %5s %s", "NAME", "IP", "ROLES", "HEAP%", "RAM%", "CPU", "MASTER")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(max(m.Width-4, 40), 80))))
	b.WriteString("\n")

	maxVisible := max(m.Height-8, 5)
	selectedIdx := clamp(m.SelectedNode, 0, len(m.Nodes)-1)
	start := 0
	if selectedIdx >= maxVisible {
		start = selectedIdx - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.Nodes))
	for i := start; i < end; i++ {
		n := m.Nodes[i]
		line := fmt.Sprintf("%-20s %-14s %-6s %5d %5d %5d %s",
			truncate(n.Name, 20), n.IP, n.NodeRole, n.HeapPercent, n.RamPercent, n.CPU, n.Master)
		if i == selectedIdx {
			b.WriteString(selectedRowStyle.Render("▶ " + line))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  r:refresh  q:back"))
	return b.String()
}

func (m Model) viewShards() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Shards (%d)", len(m.Shards))))

	if len(m.Shards) == 0 {
		b.WriteString(dimStyle.Render("No shards"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("j/k:nav  q:back"))
		return b.String()
	}

	header := fmt.Sprintf("  %-28s %5s %3s %-10s %8s %8s %s", "INDEX", "SHARD", "P/R", "STATE", "DOCS", "STORE", "NODE")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(max(m.Width-4, 40), 90))))
	b.WriteString("\n")

	maxVisible := max(m.Height-8, 5)
	selectedIdx := clamp(m.DetailScroll, 0, len(m.Shards)-1)
	start := 0
	if selectedIdx >= maxVisible {
		start = selectedIdx - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.Shards))
	for i := start; i < end; i++ {
		s := m.Shards[i]
		line := fmt.Sprintf("%-28s %5s %3s %-10s %8s %8s %s",
			truncate(s.Index, 28), s.Shard, s.Prirep, s.State, s.Docs, s.Store, s.Node)
		if i == selectedIdx {
			b.WriteString(selectedRowStyle.Render("▶ " + line))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  q:back"))
	return b.String()
}

func (m Model) viewAliases() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Aliases (%d)", len(m.Aliases))))

	if len(m.Aliases) == 0 {
		b.WriteString(dimStyle.Render("No aliases"))
	} else {
		header := fmt.Sprintf("  %-30s %s", "ALIAS", "INDEX")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", min(max(m.Width-4, 40), 60))))
		b.WriteString("\n")

		maxVisible := max(m.Height-8, 5)
		selectedIdx := clamp(m.DetailScroll, 0, len(m.Aliases)-1)
		start := 0
		if selectedIdx >= maxVisible {
			start = selectedIdx - maxVisible + 1
		}
		end := min(start+maxVisible, len(m.Aliases))
		for i := start; i < end; i++ {
			a := m.Aliases[i]
			line := fmt.Sprintf("%-30s → %s", truncate(a.Alias, 30), a.Index)
			if i == selectedIdx {
				b.WriteString(selectedRowStyle.Render("▶ " + line))
			} else {
				b.WriteString(normalStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  q:back"))
	return b.String()
}

func (m Model) viewTemplates() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Index Templates (%d)", len(m.Templates))))

	if len(m.Templates) == 0 {
		b.WriteString(dimStyle.Render("No templates"))
	} else {
		header := fmt.Sprintf("  %-28s %s", "NAME", "PATTERNS")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", min(max(m.Width-4, 40), 60))))
		b.WriteString("\n")

		maxVisible := max(m.Height-8, 5)
		selectedIdx := clamp(m.DetailScroll, 0, len(m.Templates)-1)
		start := 0
		if selectedIdx >= maxVisible {
			start = selectedIdx - maxVisible + 1
		}
		end := min(start+maxVisible, len(m.Templates))
		for i := start; i < end; i++ {
			t := m.Templates[i]
			line := fmt.Sprintf("%-28s  %s", truncate(t.Name, 28), strings.Join(t.IndexPatterns, ", "))
			if i == selectedIdx {
				b.WriteString(selectedRowStyle.Render("▶ " + line))
			} else {
				b.WriteString(normalStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  q:back"))
	return b.String()
}

func (m Model) viewLiveMetrics() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Live Metrics"))
	sepW := min(max(m.Width-4, 40), 60)
	b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")

	if m.LiveMetrics == nil {
		b.WriteString(dimStyle.Render("Collecting metrics..."))
	} else {
		d := m.LiveMetrics.Latest
		const labelW = 14
		writeMeta := func(label, value string, style lipgloss.Style) {
			b.WriteString(keyStyle.Render(fmt.Sprintf("%-*s", labelW, label)))
			b.WriteString(style.Render(value))
			b.WriteString("\n")
		}
		writeMeta("Status", d.Status, healthStyle(d.Status))
		writeMeta("Nodes", fmt.Sprintf("%d (%d data)", d.Nodes, d.DataNodes), normalStyle)
		writeMeta("Shards", fmt.Sprintf("%d active · %d unassigned", d.ActiveShards, d.UnassignedShards), normalStyle)
		writeMeta("Docs", fmt.Sprintf("%d", d.DocsCount), normalStyle)
		writeMeta("Store", formatBytes(d.StoreSizeBytes), normalStyle)
		writeMeta("Search", fmt.Sprintf("%d queries · %.2f ms avg", d.QueryTotal, d.SearchLatencyMs), normalStyle)
		writeMeta("Indexing", fmt.Sprintf("%d", d.IndexingTotal), normalStyle)
		writeMeta("JVM Heap", fmt.Sprintf("%.1f%%", d.JVMHeapUsedPct), normalStyle)
		writeMeta("CPU", fmt.Sprintf("%.1f%%", d.CPUPercent), normalStyle)
		if len(m.LiveMetrics.History) > 1 {
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(sparkline(m.LiveMetrics.History)))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("auto-refresh 2s  q:back"))
	return b.String()
}

func sparkline(history []types.LiveMetricsData) string {
	if len(history) == 0 {
		return ""
	}
	bars := []rune("▁▂▃▄▅▆▇█")
	var maxQ int64
	for _, h := range history {
		if h.QueryTotal > maxQ {
			maxQ = h.QueryTotal
		}
	}
	var b strings.Builder
	b.WriteString("queries: ")
	for _, h := range history {
		idx := 0
		if maxQ > 0 {
			idx = int(float64(h.QueryTotal) / float64(maxQ) * float64(len(bars)-1))
			if idx >= len(bars) {
				idx = len(bars) - 1
			}
		}
		b.WriteRune(bars[idx])
	}
	return b.String()
}

func (m Model) viewIndexCreate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Create Index"))
	b.WriteString("\n\n")
	if m.Inputs != nil {
		b.WriteString("Name: " + m.Inputs.IndexNameInput.View() + "\n")
		b.WriteString("Body: " + m.Inputs.IndexBodyInput.View() + "\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter create · esc cancel"))
	return b.String()
}

func (m Model) viewEditDocument() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Edit / Index Document"))
	b.WriteString("\n\n")
	if m.Inputs != nil {
		b.WriteString("ID:   " + m.Inputs.DocIDInput.View() + "\n")
		b.WriteString("Body: " + m.Inputs.DocBodyInput.View() + "\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter save · esc cancel"))
	return b.String()
}

func (m Model) viewBulkDelete() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Bulk Delete (delete-by-query)"))
	b.WriteString("\n\n")
	if m.CurrentIndex != nil {
		b.WriteString(fmt.Sprintf("Index: %s\n", m.CurrentIndex.Name))
	}
	if m.Inputs != nil {
		b.WriteString(m.Inputs.BulkDeleteInput.View())
	}
	b.WriteString("\n\n")
	b.WriteString(errorStyle.Render("Warning: this permanently deletes matching documents."))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter run · esc cancel"))
	return b.String()
}

func (m Model) viewConfirmDelete() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Confirm Delete"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Delete %s: %v\n\n", m.ConfirmType, m.ConfirmData))
	b.WriteString(errorStyle.Render("This cannot be undone."))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("y confirm · n/esc cancel"))
	return b.String()
}

func (m Model) viewTestConnection() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Test Connection"))
	b.WriteString("\n\n")
	if m.TestConnResult != "" {
		b.WriteString(m.TestConnResult)
	} else {
		b.WriteString(dimStyle.Render("Testing..."))
	}
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("esc back"))
	return b.String()
}

func (m Model) viewLogs() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Logs"))
	b.WriteString("\n\n")
	if m.Logs == nil {
		b.WriteString(dimStyle.Render("No logs"))
	} else {
		logs := m.Logs.GetLogs()
		start := 0
		maxLines := max(m.Height-8, 5)
		if len(logs) > maxLines {
			start = len(logs) - maxLines
		}
		for i := start; i < len(logs); i++ {
			b.WriteString(dimStyle.Render(logs[i]))
			if !strings.HasSuffix(logs[i], "\n") {
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc back"))
	return b.String()
}

func (m Model) viewFavorites() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Favorites (%d)", len(m.Favorites))))

	if len(m.Favorites) == 0 {
		b.WriteString(dimStyle.Render("No favorites"))
	} else {
		maxVisible := max(m.Height-8, 5)
		selectedIdx := clamp(m.SelectedFavIdx, 0, len(m.Favorites)-1)
		start := 0
		if selectedIdx >= maxVisible {
			start = selectedIdx - maxVisible + 1
		}
		end := min(start+maxVisible, len(m.Favorites))
		for i := start; i < end; i++ {
			f := m.Favorites[i]
			name := truncate(f.Index, 40)
			if i == selectedIdx {
				b.WriteString(selectedRowStyle.Render(fmt.Sprintf("▶ %-40s", name)))
			} else {
				b.WriteString(normalStyle.Render(fmt.Sprintf("  %-40s", name)))
			}
			if f.Label != "" {
				b.WriteString(dimStyle.Render("  " + f.Label))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:open  d:remove  q:back"))
	return b.String()
}

func (m Model) viewRecentIndices() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Recent Indices (%d)", len(m.RecentIndices))))

	if len(m.RecentIndices) == 0 {
		b.WriteString(dimStyle.Render("No recent indices"))
	} else {
		maxVisible := max(m.Height-8, 5)
		selectedIdx := clamp(m.SelectedRecentIdx, 0, len(m.RecentIndices)-1)
		start := 0
		if selectedIdx >= maxVisible {
			start = selectedIdx - maxVisible + 1
		}
		end := min(start+maxVisible, len(m.RecentIndices))
		for i := start; i < end; i++ {
			r := m.RecentIndices[i]
			name := truncate(r.Index, 40)
			if i == selectedIdx {
				b.WriteString(selectedRowStyle.Render(fmt.Sprintf("▶ %-40s", name)))
			} else {
				b.WriteString(normalStyle.Render(fmt.Sprintf("  %-40s", name)))
			}
			b.WriteString(dimStyle.Render("  " + r.AccessedAt.Format("15:04:05")))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:open  q:back"))
	return b.String()
}

func (m Model) viewCatAPI() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Cat API"))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(m.Inputs.CatInput.View())
		b.WriteString("\n\n")
	}
	if m.CatResult != "" {
		lines := strings.Split(m.CatResult, "\n")
		maxLines := max(m.Height-12, 5)
		for i, line := range lines {
			if i >= maxLines {
				b.WriteString(dimStyle.Render("..."))
				break
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter run · esc back"))
	return b.String()
}

func (m Model) viewJSONPanel(title, body string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	if body == "" {
		b.WriteString(dimStyle.Render("(empty)"))
	} else {
		colored := colorizeJSON(body)
		lines := strings.Split(colored, "\n")
		maxLines := max(m.Height-8, 5)
		start := clamp(m.DetailScroll, 0, max(0, len(lines)-1))
		end := min(start+maxLines, len(lines))
		for i := start; i < end; i++ {
			b.WriteString(lines[i])
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k scroll · esc back"))
	return b.String()
}

func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Help · es-tui"))
	b.WriteString("\n\n")
	sections := []struct {
		title string
		keys  [][2]string
	}{
		{"Global", [][2]string{
			{"q / esc", "Quit or go back"},
			{"?", "This help"},
			{"r", "Refresh"},
			{"Ctrl+C", "Force quit"},
		}},
		{"Connections", [][2]string{
			{"enter", "Connect"},
			{"a", "Add connection"},
			{"e", "Edit connection"},
			{"d", "Delete connection"},
			{"t", "Test connection"},
		}},
		{"Indices", [][2]string{
			{"enter", "Browse documents"},
			{"/", "Search"},
			{":", "Command palette"},
			{"O/X", "Open / close index"},
			{"u/M", "Refresh / force-merge"},
			{"I", "Reindex"},
			{"V/W", "Allocation / tasks"},
			{"E/U/Z", "Data streams / cluster settings / snapshots"},
			{"Y/Q/#", "Saved queries / export / count"},
			{"c/n/m", "Health / nodes / metrics"},
		}},
		{"Search", [][2]string{
			{"enter", "Run query / open hit"},
			{"j/k n/p", "Navigate / page"},
			{"y/S/x/#", "Copy / save query / explain / count"},
			{":", "Command palette"},
		}},
		{"Documents", [][2]string{
			{"enter", "Open"},
			{"/ f", "Search / inline filter"},
			{"y n/p", "Copy / page"},
			{"e d D", "Edit / delete / bulk"},
		}},
	}
	for _, sec := range sections {
		b.WriteString(tealStyle.Render(sec.title))
		b.WriteString("\n")
		for _, k := range sec.keys {
			b.WriteString(fmt.Sprintf("  %s  %s\n", yellowStyle.Render(fmt.Sprintf("%-10s", k[0])), k[1]))
		}
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("esc back"))
	return b.String()
}
