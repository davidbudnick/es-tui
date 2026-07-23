package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func (m Model) viewAllocation() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Disk Allocation [%d]", len(m.Allocation))))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render("  " + m.CurrentConn.Name))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.Allocation) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No allocation data"), keyDesc{"r", "refresh"}, keyDesc{"q", "back"})
	}

	header := fmt.Sprintf("  %-18s %-14s %-12s %-12s %8s %s",
		"Node", "IP", "Used", "Avail", "Disk%", "Shards")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	maxVisible := max(m.Height-10, 8)
	end := min(maxVisible, len(m.Allocation))
	for i := 0; i < end; i++ {
		a := m.Allocation[i]
		diskStyle := healthGreen
		if pct := parsePct(a.DiskPercent); pct >= 90 {
			diskStyle = healthRed
		} else if pct >= 80 {
			diskStyle = healthYellow
		}
		b.WriteString("  ")
		b.WriteString(normalStyle.Render(fmt.Sprintf("%-18s", truncate(a.Node, 18))))
		b.WriteString(" ")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%-14s %-12s %-12s ", a.IP, a.DiskUsed, a.DiskAvail)))
		b.WriteString(diskStyle.Render(fmt.Sprintf("%8s", a.DiskPercent)))
		b.WriteString(" ")
		b.WriteString(dimStyle.Render(a.Shards))
		b.WriteString("\n")
	}
	if len(m.Allocation) > maxVisible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("1-%d of %d", end, len(m.Allocation))))
	}
	return m.fullScreenFrame(b.String(), keyDesc{"r", "refresh"}, keyDesc{"q", "back"})
}

func parsePct(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func (m Model) viewTasks() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Tasks [%d]", len(m.Tasks))))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render("  " + m.CurrentConn.Name))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.Tasks) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No running tasks"), keyDesc{"r", "refresh"}, keyDesc{"q", "back"})
	}

	header := fmt.Sprintf("  %-28s %-24s %-12s %s", "ID", "Action", "Running", "Node")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	selected := clamp(m.SelectedTaskIdx, 0, len(m.Tasks)-1)
	maxVisible := max(m.Height-10, 8)
	start, end := listWindow(selected, len(m.Tasks), maxVisible)
	for i := start; i < end; i++ {
		t := m.Tasks[i]
		if i == selected {
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-28s", truncate(t.ID, 28))))
			b.WriteString(" ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-24s %-12s %s",
				truncate(t.Action, 24), t.RunningTime, truncate(t.Node, 16))))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-28s", truncate(t.ID, 28))))
			b.WriteString(" ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-24s %-12s %s",
				truncate(t.Action, 24), t.RunningTime, truncate(t.Node, 16))))
		}
		b.WriteString("\n")
	}
	if len(m.Tasks) > maxVisible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d-%d of %d", start+1, end, len(m.Tasks))))
	}
	return m.fullScreenFrame(b.String(), keyDesc{"j/k", "nav"}, keyDesc{"x", "cancel"}, keyDesc{"r", "refresh"}, keyDesc{"q", "back"})
}

func (m Model) viewPlugins() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Plugins [%d]", len(m.Plugins))))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render("  " + m.CurrentConn.Name))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.Plugins) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No plugins reported"), keyDesc{"q", "back"})
	}

	header := fmt.Sprintf("  %-30s %-18s %s", "Name", "Component", "Version")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	maxVisible := max(m.Height-10, 8)
	end := min(maxVisible, len(m.Plugins))
	for i := 0; i < end; i++ {
		p := m.Plugins[i]
		b.WriteString("  ")
		b.WriteString(normalStyle.Render(fmt.Sprintf("%-30s", truncate(p.Name, 30))))
		b.WriteString(" ")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%-18s %s", truncate(p.Component, 18), p.Version)))
		b.WriteString("\n")
	}
	if len(m.Plugins) > maxVisible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("1-%d of %d", end, len(m.Plugins))))
	}
	return m.fullScreenFrame(b.String(), keyDesc{"q", "back"})
}

func (m Model) viewDataStreams() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Data Streams [%d]", len(m.DataStreams))))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render("  " + m.CurrentConn.Name))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.DataStreams) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No data streams (or API unsupported)"), keyDesc{"r", "refresh"}, keyDesc{"q", "back"})
	}

	header := fmt.Sprintf("  %-30s %-10s %8s  %s", "Name", "Status", "Gen", "Template")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	maxVisible := max(m.Height-10, 8)
	end := min(maxVisible, len(m.DataStreams))
	for i := 0; i < end; i++ {
		d := m.DataStreams[i]
		st := statusOpenStyle
		if strings.EqualFold(d.Status, "RED") {
			st = statusCloseStyle
		} else if strings.EqualFold(d.Status, "YELLOW") {
			st = statusOtherStyle
		}
		b.WriteString("  ")
		b.WriteString(normalStyle.Render(fmt.Sprintf("%-30s", truncate(d.Name, 30))))
		b.WriteString(" ")
		b.WriteString(st.Render(fmt.Sprintf("%-10s", d.Status)))
		b.WriteString(dimStyle.Render(fmt.Sprintf(" %8s  %s", d.Generation, truncate(d.Template, 24))))
		b.WriteString("\n")
	}
	if len(m.DataStreams) > maxVisible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("1-%d of %d", end, len(m.DataStreams))))
	}
	return m.fullScreenFrame(b.String(), keyDesc{"r", "refresh"}, keyDesc{"q", "back"})
}

func (m Model) viewSnapshots() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Snapshots [%d]", len(m.Snapshots))))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render("  " + m.CurrentConn.Name))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.Inputs != nil {
		b.WriteString(keyStyle.Render("Repo: "))
		if m.Inputs.SnapshotRepo.Focused() {
			b.WriteString(m.Inputs.SnapshotRepo.View())
		} else {
			v := m.Inputs.SnapshotRepo.Value()
			if v == "" {
				v = "(enter repository name)"
			}
			b.WriteString(normalStyle.Render(v))
		}
		b.WriteString("\n\n")
	}

	if len(m.Snapshots) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No snapshots — enter repo name and press enter"), keyDesc{"enter", "load"}, keyDesc{"q", "back"})
	}

	header := fmt.Sprintf("  %-26s %-12s %-14s %s", "Snapshot", "State", "Repo", "Start")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	maxVisible := max(m.Height-14, 6)
	end := min(maxVisible, len(m.Snapshots))
	for i := 0; i < end; i++ {
		s := m.Snapshots[i]
		st := healthGreen
		switch strings.ToUpper(s.State) {
		case "FAILED", "PARTIAL":
			st = healthRed
		case "IN_PROGRESS":
			st = healthYellow
		}
		b.WriteString("  ")
		b.WriteString(normalStyle.Render(fmt.Sprintf("%-26s", truncate(s.Snapshot, 26))))
		b.WriteString(" ")
		b.WriteString(st.Render(fmt.Sprintf("%-12s", s.State)))
		b.WriteString(dimStyle.Render(fmt.Sprintf(" %-14s %s", truncate(s.Repository, 14), s.StartTime)))
		b.WriteString("\n")
	}
	return m.fullScreenFrame(b.String(), keyDesc{"enter", "load"}, keyDesc{"q", "back"})
}

func (m Model) viewClusterSettings() string {
	return m.viewJSONPanel("Cluster Settings", m.ClusterSettings)
}

func (m Model) viewReindex() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Reindex"))
	b.WriteString("\n\n")
	if m.ReadOnly {
		b.WriteString(errorStyle.Render("Read-only connection — reindex disabled"))
		b.WriteString("\n\n")
	}
	if m.Inputs != nil {
		srcLabel, dstLabel := keyStyle, keyStyle
		if m.ReindexFocus == 0 {
			srcLabel = accentStyle
		}
		if m.ReindexFocus == 1 {
			dstLabel = accentStyle
		}
		b.WriteString(srcLabel.Render("Source index"))
		b.WriteString("\n")
		b.WriteString(m.Inputs.ReindexSrcInput.View())
		b.WriteString("\n\n")
		b.WriteString(dstLabel.Render("Destination index"))
		b.WriteString("\n")
		b.WriteString(m.Inputs.ReindexDstInput.View())
	}
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Creates async reindex task (wait_for_completion=false)"))
	b.WriteString("\n\n")
	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"tab", "field"},
		{"enter", "start"},
		{"esc", "cancel"},
	}))
	return m.formModal(b.String())
}

func (m Model) viewExport() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Export Documents"))
	b.WriteString(dimStyle.Render("  ·  NDJSON"))
	b.WriteString("\n\n")
	idx := "*"
	if m.CurrentIndex != nil {
		idx = m.CurrentIndex.Name
	}
	if m.SearchIndex != "" {
		idx = m.SearchIndex
	}
	b.WriteString(keyStyle.Render("Index"))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render(idx))
	b.WriteString("\n\n")
	q := m.SearchQuery
	if q == "" {
		q = m.DocQuery
	}
	if q == "" {
		q = "match_all"
	}
	b.WriteString(keyStyle.Render("Query"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(truncate(q, 48)))
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("Output path"))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(m.Inputs.ExportInput.View())
	}
	b.WriteString("\n\n")
	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"enter", "export"},
		{"esc", "cancel"},
	}))
	return m.formModal(b.String())
}

func (m Model) viewSavedQueries() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Saved Queries [%d]", len(m.SavedQueries))))
	b.WriteString("\n")

	if len(m.SavedQueries) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No saved queries. From search, press S to save current query."), keyDesc{"q", "back"})
	}

	header := fmt.Sprintf("  %-22s %-18s  %s", "Name", "Index", "Query")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	selected := clamp(m.SelectedSQIdx, 0, len(m.SavedQueries)-1)
	maxVisible := max(m.Height-10, 8)
	start, end := listWindow(selected, len(m.SavedQueries), maxVisible)
	for i := start; i < end; i++ {
		q := m.SavedQueries[i]
		if i == selected {
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-22s", truncate(q.Name, 22))))
			b.WriteString(" ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-18s  %s", truncate(q.Index, 18), truncate(q.Query, 40))))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-22s", truncate(q.Name, 22))))
			b.WriteString(" ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-18s  %s", truncate(q.Index, 18), truncate(q.Query, 40))))
		}
		b.WriteString("\n")
	}
	return m.fullScreenFrame(b.String(), keyDesc{"enter", "run"}, keyDesc{"d", "delete"}, keyDesc{"q", "back"})
}

func (m Model) viewExplain() string {
	var b strings.Builder
	b.WriteString(m.listHeader("Explain"))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))
	b.WriteString("\n")

	if m.ExplainResult == nil {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No explain result"), keyDesc{"q", "back"})
	}
	matched := "false"
	style := errorStyle
	if m.ExplainResult.Matched {
		matched = "true"
		style = successStyle
	}
	b.WriteString(keyStyle.Render("  Matched: "))
	b.WriteString(style.Bold(true).Render(matched))
	b.WriteString("\n\n")
	body := m.ExplainResult.Explanation
	if body == "" {
		body = m.ExplainResult.Raw
	}
	lines := strings.Split(body, "\n")
	maxLines := max(m.Height-12, 8)
	for i, line := range lines {
		if i >= maxLines {
			b.WriteString(dimStyle.Render("…"))
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return m.fullScreenFrame(b.String(), keyDesc{"q", "back"})
}

func (m Model) viewCommandPalette() string {
	const (
		labelW = 28
		rowW   = 48
	)
	var b strings.Builder
	b.WriteString(titleStyle.Render("Command Palette"))
	b.WriteString(dimStyle.Render("  ·  type to filter"))
	b.WriteString("\n\n")

	// Search box.
	searchInner := ""
	if m.Inputs != nil {
		searchInner = m.Inputs.PaletteInput.View()
	}
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(0, 1).
		Width(rowW).
		Render(searchInner)
	b.WriteString(searchBox)
	b.WriteString("\n\n")

	items := m.filteredPalette()
	if len(items) == 0 {
		b.WriteString(dimStyle.Render("No matching commands"))
		b.WriteString("\n\n")
		b.WriteString(renderKeyHelp([]struct{ key, desc string }{
			{"esc", "close"},
		}))
		return m.paletteModal(b.String())
	}

	idx := clamp(m.PaletteIdx, 0, max(len(items)-1, 0))
	// Leave room for title, search, footer, borders.
	maxVisible := max(m.Height-16, 10)
	start := 0
	if idx >= maxVisible {
		start = idx - maxVisible + 1
	}
	end := min(start+maxVisible, len(items))

	lastGroup := ""
	for i := start; i < end; i++ {
		it := items[i]
		if it.Group != "" && it.Group != lastGroup {
			if lastGroup != "" {
				b.WriteString("\n")
			}
			b.WriteString(accentStyle.Render(it.Group))
			b.WriteString("\n")
			lastGroup = it.Group
		}
		b.WriteString(paletteRow(it, i == idx, labelW, rowW))
		b.WriteString("\n")
	}

	if len(items) > maxVisible {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d–%d of %d", start+1, end, len(items))))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderKeyHelp([]struct{ key, desc string }{
		{"j/k", "nav"},
		{"enter", "run"},
		{"esc", "close"},
	}))
	return m.paletteModal(b.String())
}

func paletteRow(it PaletteItem, selected bool, labelW, rowW int) string {
	label := it.Label
	if r := []rune(label); len(r) > labelW {
		label = string(r[:labelW-1]) + "…"
	}
	key := it.Keys
	if key == "" {
		key = "·"
	}
	if selected {
		// Full-width selection band (plain text so padding is accurate).
		line := fmt.Sprintf("▶ %-*s  %s", labelW, label, key)
		return selectedStyle.Width(rowW + 2).Render(padRight(line, rowW+2))
	}
	labelPart := normalStyle.Render(fmt.Sprintf("%-*s", labelW, label))
	var keyPart string
	if it.Keys != "" {
		keyPart = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Render(it.Keys)
	} else {
		keyPart = dimStyle.Render("·")
	}
	// "  " prefix + label + gap + key chip ≈ rowW+2
	used := 2 + lipgloss.Width(labelPart) + lipgloss.Width(keyPart)
	gap := max(rowW+2-used, 2)
	return "  " + labelPart + strings.Repeat(" ", gap) + keyPart
}

func (m Model) paletteModal(body string) string {
	modalWidth := min(56, max(m.Width-8, 40))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth).
		Render(strings.TrimRight(body, "\n"))
}

func (m Model) filteredPalette() []PaletteItem {
	items := m.PaletteItems
	if len(items) == 0 {
		items = defaultPaletteItems()
	}
	filter := ""
	if m.Inputs != nil {
		filter = strings.ToLower(strings.TrimSpace(m.Inputs.PaletteInput.Value()))
	}
	if filter == "" {
		return items
	}
	var out []PaletteItem
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Label), filter) ||
			strings.Contains(strings.ToLower(it.ID), filter) ||
			strings.Contains(strings.ToLower(it.Group), filter) ||
			strings.Contains(strings.ToLower(it.Keys), filter) {
			out = append(out, it)
		}
	}
	return out
}

var _ = types.AllocationInfo{}
