package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/es-tui/internal/types"
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
		BorderForeground(lipgloss.Color(colorBorder)).
		Render(rightContent)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	help := helpStyle.Render("j/k:nav  enter:docs  i:detail  f:filter  /:search  a:create  d:del  c:health  n:nodes  m:metrics  *:fav  r:refresh  q:back")
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
		connInfo = fmt.Sprintf(" · %s", m.CurrentConn.Name)
	}
	flavor := string(m.Flavor)
	if flavor == "" {
		flavor = "cluster"
	}
	title := fmt.Sprintf("Indices%s  %s", connInfo, dimStyle.Render(flavor))
	if len(m.Indices) > 0 {
		title += fmt.Sprintf("  [%d]", len(m.Indices))
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

	healthW, statusW, docsW, sizeW := 7, 8, 10, 8
	nameW := width - healthW - statusW - docsW - sizeW - 10
	if nameW < 16 {
		nameW = 16
	}

	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %*s  %*s",
		nameW, "INDEX", healthW, "HEALTH", statusW, "STATUS", docsW, "DOCS", sizeW, "SIZE")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(width, 80))))
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
		plain := fmt.Sprintf("%-*s  %-*s  %-*s  %*d  %*s",
			nameW, name, healthW, idx.Health, statusW, idx.Status, docsW, idx.DocsCount, sizeW, idx.StoreSize)
		if i == selectedIdx {
			b.WriteString(selectedRowStyle.Render("▶ " + plain))
		} else {
			prefix := normalStyle.Render("  ")
			namePart := normalStyle.Render(fmt.Sprintf("%-*s  ", nameW, name))
			hPart := healthStyle(idx.Health).Render(fmt.Sprintf("%-*s", healthW, idx.Health))
			rest := normalStyle.Render(fmt.Sprintf("  %-*s  %*d  %*s", statusW, idx.Status, docsW, idx.DocsCount, sizeW, idx.StoreSize))
			b.WriteString(prefix + namePart + hPart + rest)
		}
		b.WriteString("\n")
	}

	if len(m.Indices) > maxVisible {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\nShowing %d-%d of %d", start+1, end, len(m.Indices))))
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n%d indices", len(m.Indices))))
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

	b.WriteString(keyStyle.Render("Index: "))
	name := idx.Name
	if width > 10 && len(name) > width-8 {
		name = name[:width-11] + "..."
	}
	b.WriteString(normalStyle.Render(name))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Health: "))
	b.WriteString(healthStyle(idx.Health).Render(idx.Health))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Status: "))
	b.WriteString(normalStyle.Render(idx.Status))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Docs: "))
	b.WriteString(normalStyle.Render(fmt.Sprintf("%d", idx.DocsCount)))
	if idx.DocsDeleted > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  (%d deleted)", idx.DocsDeleted)))
	}
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Shards: "))
	b.WriteString(normalStyle.Render(fmt.Sprintf("%d primary / %d replica", idx.PrimaryShards, idx.ReplicaShards)))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Store: "))
	b.WriteString(normalStyle.Render(idx.StoreSize))
	if idx.PriStoreSize != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  (pri %s)", idx.PriStoreSize)))
	}
	b.WriteString("\n\n")

	if idx.UUID != "" {
		b.WriteString(keyStyle.Render("UUID: "))
		b.WriteString(dimStyle.Render(truncate(idx.UUID, max(width-8, 8))))
		b.WriteString("\n\n")
	}

	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("enter browse docs"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("i full detail · s settings"))
	if idx.IsFavorite {
		b.WriteString("\n\n")
		b.WriteString(yellowStyle.Render("★ favorited"))
	}
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
	meta.WriteString(healthStyle(idx.Health).Render(idx.Health))
	meta.WriteString("  ")
	meta.WriteString(keyStyle.Render("Status: "))
	meta.WriteString(normalStyle.Render(idx.Status))
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

// silence unused import when types only used in signatures via indices
var _ = types.IndexInfo{}
