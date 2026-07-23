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
	fullContent := content
	if status != "" {
		// redis-tui: blank line then status (connections often has empty status)
		fullContent = content + "\n\n" + status
	}

	// Fullscreen browse / editor = top-left. Compact modals + detail = centered.
	// Edit document matches redis-tui: full-height editor, not a tiny centered form.
	vPos := lipgloss.Position(lipgloss.Top)
	hPos := lipgloss.Left
	switch m.Screen {
	case types.ScreenConnections, types.ScreenAddConnection, types.ScreenEditConnection,
		types.ScreenConfirmDelete, types.ScreenTestConnection, types.ScreenHelp,
		types.ScreenIndexCreate, types.ScreenBulkDelete,
		types.ScreenDocumentDetail, types.ScreenIndexDetail, types.ScreenCommandPalette,
		types.ScreenReindex, types.ScreenExport:
		hPos = lipgloss.Center
		vPos = lipgloss.Center
	}

	return lipgloss.Place(m.Width, m.Height, hPos, vPos, fullContent,
		lipgloss.WithWhitespaceChars(" "))
}

func (m Model) getStatusBar() string {
	// Connection/main screens are vertically centered. Never append a status line here —
	// variable height (Disconnected, Loading, errors) re-centers the block and the UI
	// appears to jump down/up each time you return. Loading + errors render in-view.
	if m.Screen == types.ScreenConnections || m.Screen == types.ScreenAddConnection ||
		m.Screen == types.ScreenEditConnection || m.Screen == types.ScreenHelp {
		return ""
	}

	if m.Loading {
		return dimStyle.Render("Loading...")
	}
	if m.StatusMsg != "" {
		return successStyle.Render(m.StatusMsg)
	}
	if m.Err != nil {
		return errorStyle.Render(m.Err.Error())
	}
	if m.UpdateAvailable != "" {
		hint := "es-tui --update"
		if m.UpdateCmd != "" {
			hint = m.UpdateCmd
		}
		return yellowStyle.Render("update " + m.UpdateAvailable + " · " + hint)
	}

	// Connected browse screens only.
	if m.CurrentConn == nil {
		return ""
	}

	var parts []string
	parts = append(parts, successStyle.Render("Connected"))
	if m.CurrentConn.Name != "" {
		parts = append(parts, tealStyle.Render(m.CurrentConn.Name))
	}
	f := m.Flavor
	if !f.IsKnown() && m.CurrentConn.Flavor.IsKnown() {
		f = m.CurrentConn.Flavor
	}
	parts = append(parts, flavorBadge(string(f)))
	if m.ReadOnly {
		parts = append(parts, yellowStyle.Render("RO"))
	}
	if m.ClusterHealth.Status != "" {
		parts = append(parts, healthStyle(m.ClusterHealth.Status).Render(m.ClusterHealth.Status))
	}
	parts = append(parts, dimStyle.Render("? help · q quit"))
	return strings.Join(parts, "  ·  ")
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
	b.WriteString(keyStyle.Render("Name"))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(m.Inputs.IndexNameInput.View())
	}
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("Settings JSON"))
	b.WriteString(dimStyle.Render("  (optional)"))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(m.Inputs.IndexBodyInput.View())
	}
	b.WriteString("\n\n")
	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"enter", "create"},
		{"esc", "cancel"},
	}))
	return m.formModal(b.String())
}

func (m Model) viewEditDocument() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Edit Document"))
	if m.CurrentIndex != nil {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render(m.CurrentIndex.Name))
	}
	if m.Inputs != nil {
		id := strings.TrimSpace(m.Inputs.DocIDInput.Value())
		if id != "" {
			b.WriteString(dimStyle.Render(" / " + id))
		}
	}
	b.WriteString("\n")
	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"ctrl+s", "save"},
		{"esc", "cancel"},
		{"tab", "id/body"},
	}))
	b.WriteString("\n\n")

	// Document ID (single line, redis-style labeled field).
	idLabel := keyStyle
	if m.DocEditFocus == "id" {
		idLabel = accentStyle
	}
	b.WriteString(idLabel.Render("Document ID"))
	b.WriteString(dimStyle.Render("  (optional — leave empty to auto-generate)"))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(m.Inputs.DocIDInput.View())
	}
	b.WriteString("\n\n")

	bodyLabel := keyStyle
	if m.DocEditFocus != "id" {
		bodyLabel = accentStyle
	}
	b.WriteString(bodyLabel.Render("Body"))
	if m.DocEditor != nil && m.DocEditor.FileName() != "" {
		b.WriteString(dimStyle.Render("  ·  " + m.DocEditor.FileName()))
	}
	b.WriteString("\n")
	if m.DocEditor != nil {
		b.WriteString(m.DocEditor.View())
	} else if m.Inputs != nil {
		// Fallback if editor not ready.
		b.WriteString(m.Inputs.DocBodyInput.View())
	}
	return b.String()
}

func (m Model) viewBulkDelete() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Bulk Delete"))
	b.WriteString("\n\n")
	if m.CurrentIndex != nil {
		b.WriteString(keyStyle.Render("Index"))
		b.WriteString("\n")
		b.WriteString(normalStyle.Render(m.CurrentIndex.Name))
		b.WriteString("\n\n")
	}
	b.WriteString(keyStyle.Render("Query"))
	b.WriteString(dimStyle.Render("  ·  query_string or JSON"))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(m.Inputs.BulkDeleteInput.View())
	}
	b.WriteString("\n\n")
	b.WriteString(errorStyle.Render("Warning: permanently deletes matching documents."))
	b.WriteString("\n\n")
	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"enter", "run"},
		{"esc", "cancel"},
	}))
	return m.formModal(b.String())
}

func (m Model) viewConfirmDelete() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Confirm Delete"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Delete %s:\n", m.ConfirmType))
	b.WriteString(normalStyle.Render(fmt.Sprint(m.ConfirmData)))
	b.WriteString("\n\n")
	b.WriteString(errorStyle.Render("This cannot be undone."))
	b.WriteString("\n\n")
	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"y", "confirm"},
		{"n", "cancel"},
		{"esc", "cancel"},
	}))
	return m.formModal(b.String())
}

func (m Model) viewTestConnection() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Test Connection"))
	b.WriteString("\n\n")
	if m.TestConnResult != "" {
		if strings.Contains(strings.ToLower(m.TestConnResult), "fail") || strings.Contains(strings.ToLower(m.TestConnResult), "error") {
			b.WriteString(errorStyle.Render(m.TestConnResult))
		} else {
			b.WriteString(successStyle.Render(m.TestConnResult))
		}
	} else {
		b.WriteString(dimStyle.Render("Testing..."))
	}
	b.WriteString("\n\n")
	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"esc", "back"},
	}))
	return m.formModal(b.String())
}

func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Help"))
	if m.Version != "" && m.Version != "dev" {
		b.WriteString(dimStyle.Render("  ·  es-tui " + m.Version))
	} else {
		b.WriteString(dimStyle.Render("  ·  es-tui"))
	}
	b.WriteString("\n\n")

	sections := []struct {
		title string
		keys  [][2]string
	}{
		{"Global", [][2]string{
			{"q / esc", "Quit or go back"},
			{"?", "Toggle this help"},
			{"j / k", "Move selection"},
			{"g / G", "Top / bottom"},
			{"r", "Refresh"},
			{"Ctrl+C", "Force quit"},
		}},
		{"Connections", [][2]string{
			{"enter", "Connect"},
			{"a", "Add connection"},
			{"e", "Edit connection"},
			{"d", "Delete connection"},
			{"t", "Test connection"},
			{"↑ / ↓", "Navigate cards"},
		}},
		{"Indices", [][2]string{
			{"enter", "Browse documents"},
			{"/", "Search cluster"},
			{":", "Command palette"},
			{"f", "Filter pattern"},
			{"O / X", "Open / close index"},
			{"u", "Refresh indices"},
			{"M", "Force-merge"},
			{"I", "Reindex"},
			{"c", "Cluster health"},
			{"n", "Nodes"},
			{"m", "Live metrics"},
			{"s", "Shards"},
			{"A", "Aliases"},
			{"T", "Templates"},
			{"F", "Favorites"},
			{"R", "Recent indices"},
			{"V", "Allocation"},
			{"W", "Tasks"},
			{"#", "Count docs"},
			{"*", "Toggle favorite"},
		}},
		{"Documents", [][2]string{
			{"enter", "Open document"},
			{"/", "Search query"},
			{"f", "Inline filter"},
			{"y", "Copy JSON"},
			{"n / p", "Next / prev page"},
			{"e", "Edit document"},
			{"d", "Delete document"},
			{"D", "Bulk delete"},
			{"r", "Reload page"},
		}},
		{"Search", [][2]string{
			{"/", "Focus query editor"},
			{"ctrl+enter", "Run query"},
			{"1–5", "Insert template"},
			{"enter", "Open selected hit"},
			{"j / k", "Navigate hits"},
			{"n / p", "Page results"},
			{"ctrl+p/n", "Query history"},
			{"y", "Copy hit"},
			{"S", "Save query"},
			{"x", "Explain"},
			{"#", "Count"},
		}},
		{"Detail", [][2]string{
			{"j / k", "Scroll lines"},
			{"y", "Copy value"},
			{"esc", "Back"},
		}},
	}

	// Two-column binding rows; key chips stay fixed-width for clean alignment.
	const keyCol = 12
	const bindCol = 36
	colStyle := lipgloss.NewStyle().Width(bindCol)

	for _, sec := range sections {
		b.WriteString(accentStyle.Bold(true).Render(sec.title))
		b.WriteString("\n")
		half := (len(sec.keys) + 1) / 2
		var left, right strings.Builder
		for i, binding := range sec.keys {
			line := helpBindingLine(binding[0], binding[1], keyCol)
			if i < half {
				left.WriteString(line)
				left.WriteString("\n")
			} else {
				right.WriteString(line)
				right.WriteString("\n")
			}
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			colStyle.Render(strings.TrimRight(left.String(), "\n")),
			colStyle.Render(strings.TrimRight(right.String(), "\n")),
		))
		b.WriteString("\n\n")
	}

	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"?", "close"},
		{"esc", "close"},
	}))

	modalWidth := min(80, max(m.Width-6, 50))
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth).
		Render(strings.TrimRight(b.String(), "\n"))
	return modal
}

func helpBindingLine(key, desc string, keyWidth int) string {
	chip := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("255")).
		Padding(0, 1).
		Render(key)
	pad := keyWidth - lipgloss.Width(chip)
	if pad < 1 {
		pad = 1
	}
	return chip + strings.Repeat(" ", pad) + descStyle.Render(desc)
}
