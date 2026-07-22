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

	leftWidth := (totalWidth * 60) / 100
	rightWidth := totalWidth - leftWidth - 1
	panelHeight := max(m.Height-4, 12)

	leftContent := m.buildIndicesListPanel(leftWidth - 2)
	rightContent := m.buildIndexPreviewPanel(rightWidth - 2)

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
	help := helpStyle.Render("j/k:nav  enter:docs  /:search  i:detail  O:open  X:close  u:refresh  M:merge  a:create  d:del  c:health  n:nodes  m:metrics  *:fav  r:reload  q:back")
	return content + "\n" + help
}

func (m Model) viewIndicesListOnly() string {
	var b strings.Builder
	b.WriteString(m.buildIndicesListPanel(max(m.Width-4, 40)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:docs  i:detail  f:filter  /:search  q:back"))
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
	b.WriteString("\n\n")

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
	b.WriteString("\n\n")

	if len(m.Indices) == 0 {
		b.WriteString(dimStyle.Render("No indices found."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press 'a' to create one."))
		return b.String()
	}

	// Columns: name flexible, health, status, docs, size (redis-style spacing)
	healthW, statusW, docsW, sizeW := 8, 8, 8, 8
	nameW := width - healthW - statusW - docsW - sizeW - 12
	if nameW < 18 {
		nameW = 18
	}

	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %*s  %s",
		nameW, "Index", healthW, "Health", statusW, "Status", docsW, "Docs", "Size")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(width, nameW+healthW+statusW+docsW+sizeW+16))))
	b.WriteString("\n")

	maxVisible := max(m.Height-12, 5)
	selectedIdx := clamp(m.SelectedIndexIdx, 0, len(m.Indices)-1)
	start := 0
	if selectedIdx >= maxVisible {
		start = selectedIdx - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.Indices))
	if end-start < maxVisible && end == len(m.Indices) {
		start = max(end-maxVisible, 0)
	}

	for i := start; i < end; i++ {
		idx := m.Indices[i]
		name := truncate(idx.Name, nameW)
		docsStr := fmt.Sprintf("%d", idx.DocsCount)
		sizeStr := idx.StoreSize
		if sizeStr == "" {
			sizeStr = "-"
		}

		if i == selectedIdx {
			// Blue selection band on cursor + name (redis pattern)
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-*s", nameW, name)))
			b.WriteString("  ")
			b.WriteString(healthStyleBold(idx.Health).Render(fmt.Sprintf("%-*s", healthW, idx.Health)))
			b.WriteString("  ")
			b.WriteString(indexStatusStyleBold(idx.Status).Render(fmt.Sprintf("%-*s", statusW, idx.Status)))
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%*s", docsW, docsStr)))
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(sizeStr))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-*s", nameW, name)))
			b.WriteString("  ")
			b.WriteString(healthStyle(idx.Health).Render(fmt.Sprintf("%-*s", healthW, idx.Health)))
			b.WriteString("  ")
			b.WriteString(indexStatusStyle(idx.Status).Render(fmt.Sprintf("%-*s", statusW, idx.Status)))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("%*s", docsW, docsStr)))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(sizeStr))
		}
		b.WriteString("\n")
	}

	if len(m.Indices) > maxVisible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d-%d of %d", start+1, end, len(m.Indices))))
	}

	return b.String()
}

func (m Model) buildIndexPreviewPanel(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Preview"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n\n")

	if len(m.Indices) == 0 {
		b.WriteString(dimStyle.Render("No index selected"))
		return b.String()
	}
	selectedIdx := clamp(m.SelectedIndexIdx, 0, len(m.Indices)-1)
	idx := m.Indices[selectedIdx]

	name := idx.Name
	if width > 12 && len(name) > width-8 {
		name = name[:width-11] + "..."
	}

	b.WriteString(keyStyle.Render("Index: "))
	b.WriteString(normalStyle.Render(name))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Health: "))
	b.WriteString(healthStyleBold(idx.Health).Render(idx.Health))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Status: "))
	b.WriteString(indexStatusStyleBold(idx.Status).Render(idx.Status))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Docs"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render(fmt.Sprintf("%d", idx.DocsCount)))
	if idx.DocsDeleted > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  (%d deleted)", idx.DocsDeleted)))
	}
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Store: "))
	store := idx.StoreSize
	if store == "" {
		store = "-"
	}
	b.WriteString(normalStyle.Render(store))
	if idx.PriStoreSize != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  (pri %s)", idx.PriStoreSize)))
	}
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Shards: "))
	b.WriteString(normalStyle.Render(fmt.Sprintf("%d primary / %d replica", idx.PrimaryShards, idx.ReplicaShards)))
	b.WriteString("\n\n")

	if idx.UUID != "" {
		b.WriteString(keyStyle.Render("UUID: "))
		b.WriteString(dimStyle.Render(truncate(idx.UUID, max(width-8, 8))))
		b.WriteString("\n\n")
	}

	if idx.IsFavorite {
		b.WriteString(yellowStyle.Render("★ favorited"))
		b.WriteString("\n\n")
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

	var meta strings.Builder
	meta.WriteString(keyStyle.Render("  Index: "))
	meta.WriteString(normalStyle.Render(idx.Name))
	meta.WriteString("\n")
	meta.WriteString(keyStyle.Render(" Health: "))
	meta.WriteString(healthStyleBold(idx.Health).Render(idx.Health))
	meta.WriteString("  ")
	meta.WriteString(keyStyle.Render("Status: "))
	meta.WriteString(indexStatusStyleBold(idx.Status).Render(idx.Status))
	meta.WriteString("\n")
	meta.WriteString(keyStyle.Render("   Docs: "))
	meta.WriteString(normalStyle.Render(fmt.Sprintf("%d", idx.DocsCount)))
	meta.WriteString("  ")
	meta.WriteString(keyStyle.Render("Store: "))
	meta.WriteString(normalStyle.Render(idx.StoreSize))
	meta.WriteString("\n")
	meta.WriteString(keyStyle.Render(" Shards: "))
	meta.WriteString(normalStyle.Render(fmt.Sprintf("%d pri / %d rep", idx.PrimaryShards, idx.ReplicaShards)))
	if idx.UUID != "" {
		meta.WriteString("\n")
		meta.WriteString(keyStyle.Render("   UUID: "))
		meta.WriteString(dimStyle.Render(idx.UUID))
	}
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, meta.String()))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter docs · s settings · m mappings · / search · d delete · esc back"))
	return b.String()
}
