package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) viewIndices() string {
	totalWidth := m.Width
	if totalWidth < 100 {
		return m.viewIndicesListOnly()
	}

	leftWidth := (totalWidth * 58) / 100
	rightWidth := totalWidth - leftWidth - 1
	// Leave room for wrapped key help (2 lines) + status bar.
	panelHeight := max(m.Height-5, 12)

	leftContent := m.buildIndicesListPanel(leftWidth - 4)
	rightContent := m.buildIndexPreviewPanel(rightWidth - 4)

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Padding(0, 1).
		Render(leftContent)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(panelHeight).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("240")).
		Render(rightContent)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	help := renderKeyHelpWidth(m.Width-2, []struct{ key, desc string }{
		{"j/k", "nav"},
		{"enter", "docs"},
		{"/", "search"},
		{":", "palette"},
		{"f", "filter"},
		{"O/X", "open/close"},
		{"u", "refresh"},
		{"M", "merge"},
		{"I", "reindex"},
		{"c/n/m", "health/nodes/metrics"},
		{"V/W", "alloc/tasks"},
		{"#", "count"},
		{"*", "favorite"},
		{"q", "back"},
	})
	return content + "\n" + help
}

func (m Model) viewIndicesListOnly() string {
	var b strings.Builder
	b.WriteString(m.buildIndicesListPanel(max(m.Width-4, 40)))
	b.WriteString("\n")
	b.WriteString(renderKeyHelpWidth(m.Width-2, []struct{ key, desc string }{
		{"j/k", "nav"},
		{"enter", "docs"},
		{"i", "detail"},
		{"f", "filter"},
		{"/", "search"},
		{"q", "back"},
	}))
	return b.String()
}

func (m Model) buildIndicesListPanel(width int) string {
	var b strings.Builder

	connInfo := ""
	if m.CurrentConn != nil {
		connInfo = " - " + m.CurrentConn.Name
	}
	title := "Indices" + connInfo
	if len(m.Indices) > 0 {
		title += fmt.Sprintf(" [%d]", len(m.Indices))
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	b.WriteString(keyStyle.Render("Filter: "))
	if m.Inputs != nil && m.Inputs.PatternInput.Focused() {
		b.WriteString(m.Inputs.PatternInput.View())
	} else {
		pattern := "*"
		if m.Inputs != nil {
			if p := strings.TrimSpace(m.Inputs.PatternInput.Value()); p != "" {
				pattern = p
			}
		}
		b.WriteString(normalStyle.Render(pattern))
	}
	b.WriteString("\n")

	if len(m.Indices) == 0 {
		b.WriteString(dimStyle.Render("No indices found.  Press 'a' to create one."))
		return b.String()
	}

	// Fixed meta cols; index name uses the rest of the left pane (roomy, not capped tiny).
	const (
		healthW = 8
		statusW = 8
		docsW   = 9
		sizeW   = 9
	)
	// "▶ " + name + gaps + fixed cols
	fixed := 2 + 2 + healthW + 2 + statusW + 2 + docsW + 2 + sizeW
	nameW := width - fixed
	if nameW < 16 {
		nameW = 16
	}
	// Soft cap only on absurd ultra-wide terminals
	if nameW > 56 {
		nameW = 56
	}
	rowW := 2 + nameW + 2 + healthW + 2 + statusW + 2 + docsW + 2 + sizeW

	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %*s  %-*s",
		nameW, "Index", healthW, "Health", statusW, "Status", docsW, "Docs", sizeW, "Size")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(width, rowW))))
	b.WriteString("\n")

	// Header uses ~4 lines (title, filter, header, sep)
	maxVisible := max(m.Height-10, 8)
	selectedIdx := clamp(m.SelectedIndexIdx, 0, len(m.Indices)-1)
	start, end := listWindow(selectedIdx, len(m.Indices), maxVisible)

	for i := start; i < end; i++ {
		idx := m.Indices[i]
		name := truncate(idx.Name, nameW)
		docsStr := fmt.Sprintf("%d", idx.DocsCount)
		sizeStr := idx.StoreSize
		if sizeStr == "" {
			sizeStr = "-"
		}
		sizeStr = truncate(sizeStr, sizeW)

		nameCell := fmt.Sprintf("%-*s", nameW, name)
		healthCell := fmt.Sprintf("%-*s", healthW, truncate(idx.Health, healthW))
		statusCell := fmt.Sprintf("%-*s", statusW, truncate(idx.Status, statusW))
		docsCell := fmt.Sprintf("%*s", docsW, docsStr)
		sizeCell := fmt.Sprintf("%-*s", sizeW, sizeStr)

		if i == selectedIdx {
			b.WriteString(selectedStyle.Render("▶ " + nameCell))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(nameCell))
		}
		b.WriteString("  ")
		if i == selectedIdx {
			b.WriteString(healthStyleBold(idx.Health).Render(healthCell))
		} else {
			b.WriteString(healthStyle(idx.Health).Render(healthCell))
		}
		b.WriteString("  ")
		if i == selectedIdx {
			b.WriteString(indexStatusStyleBold(idx.Status).Render(statusCell))
		} else {
			b.WriteString(indexStatusStyle(idx.Status).Render(statusCell))
		}
		b.WriteString("  ")
		if i == selectedIdx {
			b.WriteString(normalStyle.Render(docsCell))
		} else {
			b.WriteString(dimStyle.Render(docsCell))
		}
		b.WriteString("  ")
		if i == selectedIdx {
			b.WriteString(normalStyle.Render(sizeCell))
		} else {
			b.WriteString(dimStyle.Render(sizeCell))
		}
		b.WriteString("\n")
	}

	if len(m.Indices) > maxVisible {
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d-%d of %d", start+1, end, len(m.Indices))))
	}

	return b.String()
}

func (m Model) buildIndexPreviewPanel(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Preview"))
	b.WriteString("\n")
	sepW := max(min(width, 40), 12)
	b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")

	if len(m.Indices) == 0 {
		b.WriteString(dimStyle.Render("No index selected"))
		return b.String()
	}
	selectedIdx := clamp(m.SelectedIndexIdx, 0, len(m.Indices)-1)
	idx := m.Indices[selectedIdx]

	writeKV := func(k, v string, vs lipgloss.Style) {
		b.WriteString(keyStyle.Render(k + ": "))
		b.WriteString(vs.Render(v))
		b.WriteString("\n")
	}

	writeKV("Index", truncate(idx.Name, max(width-8, 8)), normalStyle)
	b.WriteString(keyStyle.Render("Health: "))
	b.WriteString(healthStyleBold(idx.Health).Render(idx.Health))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Status: "))
	b.WriteString(indexStatusStyleBold(idx.Status).Render(idx.Status))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")

	writeKV("Docs", fmt.Sprintf("%d", idx.DocsCount), normalStyle)
	store := idx.StoreSize
	if store == "" {
		store = "-"
	}
	if idx.PriStoreSize != "" {
		store = fmt.Sprintf("%s  (pri %s)", store, idx.PriStoreSize)
	}
	writeKV("Store", store, normalStyle)
	writeKV("Shards", fmt.Sprintf("%d primary / %d replica", idx.PrimaryShards, idx.ReplicaShards), normalStyle)
	if idx.UUID != "" {
		writeKV("UUID", truncate(idx.UUID, max(width-8, 8)), dimStyle)
	}
	if idx.IsFavorite {
		b.WriteString(yellowStyle.Render("★ favorited"))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("enter · browse docs"))
	return b.String()
}

func (m Model) viewIndexDetail() string {
	if m.CurrentIndex == nil {
		return dimStyle.Render("No index selected")
	}
	idx := *m.CurrentIndex
	boxWidth := min(m.Width-4, 80)
	if boxWidth < 40 {
		boxWidth = 40
	}

	var b strings.Builder
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, titleStyle.Render("Index Detail")))
	b.WriteString("\n\n")

	const labelW = 8
	var meta strings.Builder
	meta.WriteString(keyStyle.Render(fmt.Sprintf("%*s", labelW, "Index:")) + " " + normalStyle.Render(idx.Name) + "\n")
	meta.WriteString(keyStyle.Render(fmt.Sprintf("%*s", labelW, "Health:")) + " " + healthStyleBold(idx.Health).Render(idx.Health))
	meta.WriteString("  ")
	meta.WriteString(keyStyle.Render("Status:") + " " + indexStatusStyleBold(idx.Status).Render(idx.Status) + "\n")
	meta.WriteString(keyStyle.Render(fmt.Sprintf("%*s", labelW, "Docs:")) + " " + normalStyle.Render(fmt.Sprintf("%d", idx.DocsCount)))
	meta.WriteString("  ")
	meta.WriteString(keyStyle.Render("Store:") + " " + normalStyle.Render(idx.StoreSize) + "\n")
	meta.WriteString(keyStyle.Render(fmt.Sprintf("%*s", labelW, "Shards:")) + " " + normalStyle.Render(fmt.Sprintf("%d pri / %d rep", idx.PrimaryShards, idx.ReplicaShards)))
	if idx.UUID != "" {
		meta.WriteString("\n")
		meta.WriteString(keyStyle.Render(fmt.Sprintf("%*s", labelW, "UUID:")) + " " + dimStyle.Render(truncate(idx.UUID, 40)))
	}
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, meta.String()))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, renderKeyHelpWidth(boxWidth, []struct{ key, desc string }{
		{"enter", "docs"},
		{"s", "settings"},
		{"m", "mappings"},
		{"/", "search"},
		{"d", "delete"},
		{"esc", "back"},
	})))
	return b.String()
}
