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
	switch m.Screen {
	case types.ScreenConnections, types.ScreenAddConnection, types.ScreenEditConnection,
		types.ScreenConfirmDelete, types.ScreenTestConnection, types.ScreenHelp,
		types.ScreenIndexCreate, types.ScreenBulkDelete, types.ScreenEditDocument:
		vPos = lipgloss.Center
	}

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, vPos, fullContent,
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

	parts := []string{helpStyle.Render("? help  q quit")}
	if m.CurrentConn != nil {
		flavor := string(m.Flavor)
		if flavor == "" {
			flavor = "auto"
		}
		parts = append([]string{
			tealStyle.Render(m.CurrentConn.Name),
			dimStyle.Render(flavor),
		}, parts...)
	}
	if m.UpdateAvailable != "" {
		parts = append(parts, yellowStyle.Render("update: "+m.UpdateAvailable))
	}
	return strings.Join(parts, "  ·  ")
}

func (m Model) viewConnections() string {
	var b strings.Builder
	b.WriteString(m.renderLogo())
	b.WriteString("\n\n")
	b.WriteString(m.buildStatsBar())
	b.WriteString("\n\n")

	if m.ConnectionError != "" {
		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorPink)).
			Foreground(lipgloss.Color(colorPink)).
			Padding(0, 2).
			Width(55).
			Render(fmt.Sprintf("Connection Failed\n%s", dimStyle.Render(m.ConnectionError)))
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	connCount := len(m.Connections)
	sectionTitle := fmt.Sprintf("╭─ Saved Connections (%d) ", connCount)
	sectionTitle += strings.Repeat("─", max(10, 50-len(sectionTitle))) + "╮"
	b.WriteString(accentStyle.Render(sectionTitle))
	b.WriteString("\n")

	if len(m.Connections) == 0 {
		b.WriteString("\n")
		emptyBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim)).
			Padding(1, 2).
			Render("  No connections saved.\n\n  Press 'a' to add your first Elasticsearch/OpenSearch connection.")
		b.WriteString(emptyBox)
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
		maxVisible := max((m.Height-20)/3, 3)
		selectedIdx := clamp(m.SelectedConnIdx, 0, len(m.Connections)-1)
		startIdx := 0
		if selectedIdx >= maxVisible {
			startIdx = selectedIdx - maxVisible + 1
		}
		endIdx := min(startIdx+maxVisible, len(m.Connections))

		for i := startIdx; i < endIdx; i++ {
			conn := m.Connections[i]
			isSelected := i == selectedIdx
			flavor := string(conn.Flavor)
			if flavor == "" {
				flavor = "auto"
			}
			scheme := "http"
			if conn.UseTLS {
				scheme = "https"
			}
			line1 := fmt.Sprintf("%s  %s", conn.Name, dimStyle.Render(fmt.Sprintf("(%s)", flavor)))
			line2 := dimStyle.Render(fmt.Sprintf("%s://%s:%d", scheme, conn.Host, conn.Port))
			card := line1 + "\n" + line2
			if isSelected {
				b.WriteString(connCardSelectedStyle.Render(card))
			} else {
				b.WriteString(connCardStyle.Render(card))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter connect · a add · e edit · d delete · t test · ? help · q quit"))
	return b.String()
}

func (m Model) renderLogo() string {
	// Multicolor ELASTIC wordmark inspired by the logo palette
	lines := []string{
		" ███████╗██╗      █████╗ ███████╗████████╗██╗ ██████╗",
		" ██╔════╝██║     ██╔══██╗██╔════╝╚══██╔══╝██║██╔════╝",
		" █████╗  ██║     ███████║███████╗   ██║   ██║██║     ",
		" ██╔══╝  ██║     ██╔══██║╚════██║   ██║   ██║██║     ",
		" ███████╗███████╗██║  ██║███████║   ██║   ██║╚██████╗",
		" ╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝ ╚═════╝",
	}
	styles := []*lipgloss.Style{&logoPink, &logoYellow, &logoTeal, &logoBlue, &logoGreen, &logoPink}
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(styles[i%len(styles)].Render(line))
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Elasticsearch & OpenSearch TUI"))
	return b.String()
}

func (m Model) buildStatsBar() string {
	n := len(m.Connections)
	content := fmt.Sprintf("%s %d saved  %s ES + OpenSearch  %s v%s",
		pinkStyle.Render("●"),
		n,
		tealStyle.Render("●"),
		yellowStyle.Render("●"),
		m.Version,
	)
	if m.Version == "" {
		content = fmt.Sprintf("%s %d saved  %s ES + OpenSearch",
			pinkStyle.Render("●"),
			n,
			tealStyle.Render("●"),
		)
	}
	return statsBoxStyle.Render(content)
}

func (m Model) viewConnectionForm() string {
	title := "Add Connection"
	if m.Screen == types.ScreenEditConnection {
		title = "Edit Connection"
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	for i, ti := range m.ConnInputs {
		prefix := "  "
		if i == m.ConnFocusIdx {
			prefix = tealStyle.Render("❯ ")
		}
		b.WriteString(prefix)
		b.WriteString(ti.View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("tab next · enter save · esc cancel"))
	return b.String()
}

func (m Model) viewIndices() string {
	var b strings.Builder
	flavor := string(m.Flavor)
	if flavor == "" {
		flavor = "cluster"
	}
	title := fmt.Sprintf("Indices  %s", dimStyle.Render(flavor))
	if m.CurrentConn != nil {
		title = fmt.Sprintf("Indices · %s  %s", m.CurrentConn.Name, dimStyle.Render(flavor))
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if m.Inputs != nil {
		b.WriteString(dimStyle.Render("Filter: ") + m.Inputs.PatternInput.View())
		b.WriteString("\n\n")
	}

	if len(m.Indices) == 0 {
		b.WriteString(dimStyle.Render("No indices found."))
	} else {
		header := fmt.Sprintf("%-4s %-40s %-8s %-8s %10s %10s", "H", "INDEX", "STATUS", "PRI/REP", "DOCS", "SIZE")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")

		maxVisible := max(m.Height-12, 5)
		selectedIdx := clamp(m.SelectedIndexIdx, 0, len(m.Indices)-1)
		start := 0
		if selectedIdx >= maxVisible {
			start = selectedIdx - maxVisible + 1
		}
		end := min(start+maxVisible, len(m.Indices))

		for i := start; i < end; i++ {
			idx := m.Indices[i]
			h := healthStyle(idx.Health).Render(fmt.Sprintf("%-4s", idx.Health))
			line := fmt.Sprintf("%-40s %-8s %4d/%-4d %10d %10s",
				truncate(idx.Name, 40), idx.Status, idx.PrimaryShards, idx.ReplicaShards, idx.DocsCount, idx.StoreSize)
			if i == selectedIdx {
				b.WriteString(h + " " + selectedStyle.Render(line))
			} else {
				b.WriteString(h + " " + normalStyle.Render(line))
			}
			b.WriteString("\n")
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n%d indices", len(m.Indices))))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter open · / search · a create · d delete · i detail · c health · n nodes · m metrics · * fav · r refresh · esc back"))
	return b.String()
}

func (m Model) viewIndexDetail() string {
	var b strings.Builder
	if m.CurrentIndex == nil {
		return dimStyle.Render("No index selected")
	}
	idx := *m.CurrentIndex
	b.WriteString(titleStyle.Render("Index: " + idx.Name))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s %s\n", keyStyle.Render("Health:"), healthStyle(idx.Health).Render(idx.Health)))
	b.WriteString(fmt.Sprintf("%s %s\n", keyStyle.Render("Status:"), idx.Status))
	b.WriteString(fmt.Sprintf("%s %s\n", keyStyle.Render("UUID:"), idx.UUID))
	b.WriteString(fmt.Sprintf("%s %d primary / %d replica\n", keyStyle.Render("Shards:"), idx.PrimaryShards, idx.ReplicaShards))
	b.WriteString(fmt.Sprintf("%s %d (%d deleted)\n", keyStyle.Render("Docs:"), idx.DocsCount, idx.DocsDeleted))
	b.WriteString(fmt.Sprintf("%s %s (pri %s)\n", keyStyle.Render("Store:"), idx.StoreSize, idx.PriStoreSize))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter docs · s settings · m mappings · / search · d delete · esc back"))
	return b.String()
}

func (m Model) viewDocuments() string {
	var b strings.Builder
	name := ""
	if m.CurrentIndex != nil {
		name = m.CurrentIndex.Name
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("Documents · %s", name)))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(dimStyle.Render("Query: ") + m.Inputs.SearchInput.View())
		b.WriteString("\n\n")
	}

	if len(m.Documents) == 0 {
		b.WriteString(dimStyle.Render("No documents. Press enter to run match_all."))
	} else {
		for i, doc := range m.Documents {
			line := fmt.Sprintf("%-40s  score=%.3f", truncate(doc.ID, 40), doc.Score)
			if i == m.SelectedDocIdx {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(normalStyle.Render(line))
			}
			b.WriteString("\n")
			if i >= max(m.Height-14, 5) {
				break
			}
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("\nshowing %d of %d", len(m.Documents), m.DocTotal)))
	}
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter open · e edit · d delete · / query · esc back"))
	return b.String()
}

func (m Model) viewDocumentDetail() string {
	var b strings.Builder
	if m.CurrentDocument == nil {
		return dimStyle.Render("No document")
	}
	doc := *m.CurrentDocument
	b.WriteString(titleStyle.Render(fmt.Sprintf("%s / %s", doc.Index, doc.ID)))
	b.WriteString("\n\n")
	body := colorizeJSON(doc.Raw)
	lines := strings.Split(body, "\n")
	maxLines := max(m.Height-10, 5)
	start := clamp(m.DetailScroll, 0, max(0, len(lines)-1))
	end := min(start+maxLines, len(lines))
	for i := start; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	if len(lines) > maxLines {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n[%d-%d / %d lines]", start+1, end, len(lines))))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("e edit · d delete · j/k scroll · esc back"))
	return b.String()
}

func (m Model) viewSearch() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Search"))
	b.WriteString("\n")
	idx := m.SearchIndex
	if idx == "" {
		idx = "*"
	}
	b.WriteString(dimStyle.Render("Index: ") + idx + "\n")
	if m.Inputs != nil {
		b.WriteString(dimStyle.Render("Query: ") + m.Inputs.SearchInput.View())
		b.WriteString("\n\n")
	}
	if m.SearchResult != nil {
		r := m.SearchResult
		b.WriteString(fmt.Sprintf("took %dms · total %d (%s) · hits %d\n\n", r.Took, r.Total, r.TotalRel, len(r.Hits)))
		for i, hit := range r.Hits {
			line := fmt.Sprintf("%s  %s  %.3f", hit.Index, truncate(hit.ID, 30), hit.Score)
			if i == m.SelectedDocIdx {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
			if i >= 20 {
				break
			}
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter run · esc back"))
	return b.String()
}

func (m Model) viewClusterHealth() string {
	h := m.ClusterHealth
	var b strings.Builder
	b.WriteString(titleStyle.Render("Cluster Health"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s %s\n", keyStyle.Render("Cluster:"), h.ClusterName))
	b.WriteString(fmt.Sprintf("%s %s\n", keyStyle.Render("Status:"), healthStyle(h.Status).Render(h.Status)))
	b.WriteString(fmt.Sprintf("%s %d (%d data)\n", keyStyle.Render("Nodes:"), h.NumberOfNodes, h.NumberOfDataNodes))
	b.WriteString(fmt.Sprintf("%s %d active (%d primary)\n", keyStyle.Render("Shards:"), h.ActiveShards, h.ActivePrimaryShards))
	b.WriteString(fmt.Sprintf("%s %d relocating · %d initializing · %d unassigned\n",
		keyStyle.Render("Activity:"), h.RelocatingShards, h.InitializingShards, h.UnassignedShards))
	b.WriteString(fmt.Sprintf("%s %.1f%%\n", keyStyle.Render("Active %:"), h.ActiveShardsPercentAsNumber))
	if m.ClusterInfo.Version.Number != "" {
		b.WriteString(fmt.Sprintf("\n%s %s (%s)\n", keyStyle.Render("Version:"), m.ClusterInfo.Version.Number, m.Flavor))
		b.WriteString(fmt.Sprintf("%s %s\n", keyStyle.Render("Tagline:"), m.ClusterInfo.Tagline))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("r refresh · esc back"))
	return b.String()
}

func (m Model) viewNodes() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Nodes (%d)", len(m.Nodes))))
	b.WriteString("\n\n")
	header := fmt.Sprintf("%-20s %-14s %-6s %5s %5s %5s %s", "NAME", "IP", "ROLES", "HEAP%", "RAM%", "CPU", "MASTER")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	for i, n := range m.Nodes {
		line := fmt.Sprintf("%-20s %-14s %-6s %5d %5d %5d %s",
			truncate(n.Name, 20), n.IP, n.NodeRole, n.HeapPercent, n.RamPercent, n.CPU, n.Master)
		if i == m.SelectedNode {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("r refresh · esc back"))
	return b.String()
}

func (m Model) viewShards() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Shards (%d)", len(m.Shards))))
	b.WriteString("\n\n")
	for i, s := range m.Shards {
		line := fmt.Sprintf("%-30s %s %s %-10s %8s %8s %s",
			truncate(s.Index, 30), s.Shard, s.Prirep, s.State, s.Docs, s.Store, s.Node)
		if i == 0 {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
		if i >= max(m.Height-12, 5) {
			break
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc back"))
	return b.String()
}

func (m Model) viewAliases() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Aliases (%d)", len(m.Aliases))))
	b.WriteString("\n\n")
	for _, a := range m.Aliases {
		b.WriteString(fmt.Sprintf("%-30s → %s\n", a.Alias, a.Index))
	}
	if len(m.Aliases) == 0 {
		b.WriteString(dimStyle.Render("No aliases"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc back"))
	return b.String()
}

func (m Model) viewTemplates() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Index Templates (%d)", len(m.Templates))))
	b.WriteString("\n\n")
	for _, t := range m.Templates {
		b.WriteString(fmt.Sprintf("%-30s  %s\n", t.Name, strings.Join(t.IndexPatterns, ", ")))
	}
	if len(m.Templates) == 0 {
		b.WriteString(dimStyle.Render("No templates"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc back"))
	return b.String()
}

func (m Model) viewLiveMetrics() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Live Metrics"))
	b.WriteString("\n\n")
	if m.LiveMetrics == nil {
		b.WriteString(dimStyle.Render("Collecting metrics..."))
	} else {
		d := m.LiveMetrics.Latest
		b.WriteString(fmt.Sprintf("%s %s\n", keyStyle.Render("Status:"), healthStyle(d.Status).Render(d.Status)))
		b.WriteString(fmt.Sprintf("%s %d nodes (%d data)\n", keyStyle.Render("Nodes:"), d.Nodes, d.DataNodes))
		b.WriteString(fmt.Sprintf("%s %d active · %d unassigned\n", keyStyle.Render("Shards:"), d.ActiveShards, d.UnassignedShards))
		b.WriteString(fmt.Sprintf("%s %d\n", keyStyle.Render("Docs:"), d.DocsCount))
		b.WriteString(fmt.Sprintf("%s %s\n", keyStyle.Render("Store:"), formatBytes(d.StoreSizeBytes)))
		b.WriteString(fmt.Sprintf("%s %d queries · %.2f ms avg\n", keyStyle.Render("Search:"), d.QueryTotal, d.SearchLatencyMs))
		b.WriteString(fmt.Sprintf("%s %d\n", keyStyle.Render("Indexing:"), d.IndexingTotal))
		b.WriteString(fmt.Sprintf("%s %.1f%%\n", keyStyle.Render("JVM Heap:"), d.JVMHeapUsedPct))
		b.WriteString(fmt.Sprintf("%s %.1f%%\n", keyStyle.Render("CPU:"), d.CPUPercent))
		if len(m.LiveMetrics.History) > 1 {
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(sparkline(m.LiveMetrics.History)))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("auto-refresh 2s · esc back"))
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
	b.WriteString("\n\n")
	for i, f := range m.Favorites {
		line := f.Index
		if f.Label != "" {
			line += "  " + dimStyle.Render(f.Label)
		}
		if i == m.SelectedFavIdx {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	if len(m.Favorites) == 0 {
		b.WriteString(dimStyle.Render("No favorites"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter open · d remove · esc back"))
	return b.String()
}

func (m Model) viewRecentIndices() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Recent Indices (%d)", len(m.RecentIndices))))
	b.WriteString("\n\n")
	for i, r := range m.RecentIndices {
		line := fmt.Sprintf("%s  %s", r.Index, dimStyle.Render(r.AccessedAt.Format("15:04:05")))
		if i == m.SelectedRecentIdx {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	if len(m.RecentIndices) == 0 {
		b.WriteString(dimStyle.Render("No recent indices"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter open · esc back"))
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
			{"i", "Index detail"},
			{"a", "Create index"},
			{"d", "Delete index"},
			{"/", "Search"},
			{"c", "Cluster health"},
			{"n", "Nodes"},
			{"m", "Live metrics"},
			{"s", "Shards"},
			{"A", "Aliases"},
			{"T", "Templates"},
			{"C", "Cat API"},
			{"*", "Toggle favorite"},
			{"F", "Favorites"},
			{"R", "Recent"},
		}},
		{"Documents", [][2]string{
			{"enter", "Open document"},
			{"e", "Edit / index doc"},
			{"d", "Delete document"},
			{"D", "Bulk delete-by-query"},
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
