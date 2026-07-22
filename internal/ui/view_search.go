package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) viewSearch() string {
	totalWidth := m.Width
	if totalWidth < 100 {
		return m.viewSearchCompact()
	}

	leftWidth := (totalWidth * 55) / 100
	rightWidth := totalWidth - leftWidth - 1
	panelHeight := max(m.Height-4, 12)

	left := m.buildSearchResultsPanel(leftWidth - 2)
	right := m.buildSearchPreviewPanel(rightWidth - 2)

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Padding(0, 1).
		Render(left)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(panelHeight).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("240")).
		Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	help := helpStyle.Render("/:query  enter:run  j/k:hits  o:open  y:copy  n/p:page  ↑hist  tab:focus  esc:back")
	return content + "\n" + help
}

func (m Model) viewSearchCompact() string {
	var b strings.Builder
	b.WriteString(m.buildSearchResultsPanel(max(m.Width-4, 40)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter:run  j/k:nav  o:open  esc:back"))
	return b.String()
}

func (m Model) buildSearchResultsPanel(width int) string {
	var b strings.Builder

	idx := m.SearchIndex
	if idx == "" {
		idx = "*"
	}
	title := fmt.Sprintf("Search · %s", idx)
	if m.SearchResult != nil && m.SearchResult.Total > 0 {
		title += fmt.Sprintf(" [%d hits]", m.SearchResult.Total)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Query box
	focusMark := " "
	if m.SearchFocus == "query" || (m.Inputs != nil && m.Inputs.SearchInput.Focused()) {
		focusMark = "❯"
	}
	b.WriteString(keyStyle.Render(focusMark + " Query: "))
	if m.Inputs != nil {
		if m.Inputs.SearchInput.Focused() {
			b.WriteString(m.Inputs.SearchInput.View())
		} else {
			q := m.Inputs.SearchInput.Value()
			if q == "" {
				q = m.SearchQuery
			}
			if q == "" {
				q = "(empty → match_all)"
			}
			b.WriteString(normalStyle.Render(truncate(q, max(width-12, 10))))
		}
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  query_string  or  full JSON body  {\"query\":{...}}"))
	b.WriteString("\n\n")

	if m.SearchResult == nil {
		b.WriteString(dimStyle.Render("No results yet. Type a query and press enter."))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Examples:"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  tags:merch"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  status:pending"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  level:error AND service:auth"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  {\"query\":{\"match\":{\"name\":\"widget\"}}}"))
		if len(m.QueryHistory) > 0 {
			b.WriteString("\n\n")
			b.WriteString(keyStyle.Render("Recent"))
			b.WriteString("\n")
			for i, h := range m.QueryHistory {
				if i >= 5 {
					break
				}
				b.WriteString(dimStyle.Render("  " + truncate(h, max(width-4, 20))))
				b.WriteString("\n")
			}
		}
		return b.String()
	}

	r := m.SearchResult
	page := 1
	if m.PageSize > 0 {
		page = m.SearchFrom/m.PageSize + 1
	}
	meta := fmt.Sprintf("took %dms · total %d (%s) · page %d · showing %d",
		r.Took, r.Total, r.TotalRel, page, len(r.Hits))
	b.WriteString(dimStyle.Render(meta))
	b.WriteString("\n\n")

	idW := width - 28
	if idW < 16 {
		idW = 16
	}
	header := fmt.Sprintf("  %-*s  %-18s  %s", idW, "ID", "INDEX", "SCORE")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(width, idW+32))))
	b.WriteString("\n")

	if len(r.Hits) == 0 {
		b.WriteString(dimStyle.Render("  (no hits)"))
		return b.String()
	}

	maxVisible := max(m.Height-16, 5)
	selectedIdx := clamp(m.SelectedDocIdx, 0, len(r.Hits)-1)
	start := 0
	if selectedIdx >= maxVisible {
		start = selectedIdx - maxVisible + 1
	}
	end := min(start+maxVisible, len(r.Hits))

	for i := start; i < end; i++ {
		hit := r.Hits[i]
		id := truncate(hit.ID, idW)
		index := truncate(hit.Index, 18)
		score := fmt.Sprintf("%.3f", hit.Score)
		if i == selectedIdx {
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-*s", idW, id)))
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-18s", index)))
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(score))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-*s", idW, id)))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-18s", index)))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(score))
		}
		b.WriteString("\n")
	}

	if m.SearchResult.Total > int64(m.SearchFrom+len(r.Hits)) {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("n:next page  p:prev page"))
	}
	return b.String()
}

func (m Model) buildSearchPreviewPanel(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Preview"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n\n")

	if m.SearchResult == nil || len(m.SearchResult.Hits) == 0 {
		b.WriteString(dimStyle.Render("No hit selected"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Run a query to see results."))
		return b.String()
	}

	selectedIdx := clamp(m.SelectedDocIdx, 0, len(m.SearchResult.Hits)-1)
	doc := m.SearchResult.Hits[selectedIdx]

	b.WriteString(keyStyle.Render("ID: "))
	b.WriteString(normalStyle.Render(truncate(doc.ID, max(width-6, 8))))
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("Index: "))
	b.WriteString(normalStyle.Render(doc.Index))
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("Score: "))
	b.WriteString(normalStyle.Render(fmt.Sprintf("%.4f", doc.Score)))
	b.WriteString("\n\n")

	// Snippet of top-level source fields
	if len(doc.Source) > 0 {
		b.WriteString(keyStyle.Render("Fields"))
		b.WriteString("\n\n")
		keys := make([]string, 0, len(doc.Source))
		for k := range doc.Source {
			keys = append(keys, k)
		}
		// stable-ish order by name
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[j] < keys[i] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		shown := 0
		for _, k := range keys {
			if shown >= 8 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d more", len(keys)-shown)))
				b.WriteString("\n")
				break
			}
			v := fmt.Sprint(doc.Source[k])
			if len(v) > 40 {
				v = v[:37] + "..."
			}
			b.WriteString(dimStyle.Render("  " + k + ": "))
			b.WriteString(normalStyle.Render(v))
			b.WriteString("\n")
			shown++
		}
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("Source"))
	b.WriteString("\n\n")

	raw := doc.Raw
	if raw == "" && doc.Source != nil {
		if bb, err := json.MarshalIndent(doc.Source, "", "  "); err == nil {
			raw = string(bb)
		}
	}
	if raw == "" {
		b.WriteString(dimStyle.Render("(empty)"))
		return b.String()
	}

	maxLines := max(m.Height-22, 5)
	colored := colorizeJSON(raw)
	lines := strings.Split(colored, "\n")
	for i, line := range lines {
		if i >= maxLines {
			b.WriteString(dimStyle.Render(fmt.Sprintf("… %d more lines", len(lines)-i)))
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
