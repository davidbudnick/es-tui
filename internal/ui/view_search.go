package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

func (m Model) viewSearch() string {
	totalWidth := m.Width
	if totalWidth < 90 {
		return m.viewSearchCompact()
	}

	leftWidth := (totalWidth * 58) / 100
	rightWidth := totalWidth - leftWidth - 1
	panelHeight := max(m.Height-6, 12)

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
	help := renderKeyHelpWidth(m.Width-2, []struct{ key, desc string }{
		{"/", "edit"},
		{"ctrl+enter", "run"},
		{"1-5", "template"},
		{"j/k", "hits"},
		{"enter", "open"},
		{"esc", "back"},
	})
	return content + "\n" + help
}

func (m Model) viewSearchCompact() string {
	var b strings.Builder
	b.WriteString(m.buildSearchResultsPanel(max(m.Width-4, 40)))
	b.WriteString("\n")
	b.WriteString(renderKeyHelpWidth(m.Width-2, []struct{ key, desc string }{
		{"ctrl+enter", "run"},
		{"j/k", "nav"},
		{"enter", "open"},
		{"esc", "back"},
	}))
	return b.String()
}

func (m Model) buildSearchResultsPanel(width int) string {
	var b strings.Builder

	idx := m.SearchIndex
	if idx == "" {
		idx = "*"
	}
	title := fmt.Sprintf("Search · %s", idx)
	if m.SearchResult != nil {
		title += fmt.Sprintf("  ·  %d hits", m.SearchResult.Total)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Multiline query editor box.
	queryFocused := m.SearchFocus == "query" || (m.SearchArea != nil && m.SearchArea.Focused())
	borderColor := "240"
	if queryFocused {
		borderColor = "39"
	}
	queryBody := m.searchQueryEditorView(width - 4)
	queryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(max(width-2, 20)).
		Render(queryBody)
	b.WriteString(queryBox)
	b.WriteString("\n")

	// Template strip.
	b.WriteString(m.searchTemplateStrip())
	b.WriteString("\n")

	if m.SearchResult == nil {
		b.WriteString(dimStyle.Render("Write a query above, then press "))
		b.WriteString(accentStyle.Render("ctrl+enter"))
		b.WriteString(dimStyle.Render(" to run.  Empty query = match_all."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("esc cancels back to the previous screen."))
		b.WriteString("\n\n")
		if len(m.QueryHistory) > 0 {
			b.WriteString(keyStyle.Render("Recent queries"))
			b.WriteString(dimStyle.Render("  ·  ctrl+p/n while editing"))
			b.WriteString("\n")
			for i, h := range m.QueryHistory {
				if i >= 6 {
					break
				}
				mark := "  "
				if i == m.HistoryIdx {
					mark = accentStyle.Render("› ")
				}
				b.WriteString(mark)
				b.WriteString(dimStyle.Render(truncate(h, max(width-6, 20))))
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
	meta := fmt.Sprintf("took %dms · total %d (%s) · page %d · %d shown",
		r.Took, r.Total, r.TotalRel, page, len(r.Hits))
	b.WriteString(dimStyle.Render(meta))
	b.WriteString("\n\n")

	if len(r.Hits) == 0 {
		b.WriteString(dimStyle.Render("No hits for this query."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press / to edit, 1–5 for a template, or esc to leave."))
		return b.String()
	}

	// Fixed columns that fit the panel (avoid SCORE wrap).
	scoreW := 8
	indexW := min(20, max(width/4, 10))
	idW := width - indexW - scoreW - 8
	if idW < 12 {
		idW = 12
	}
	header := fmt.Sprintf("  %-*s  %-*s  %*s", idW, "ID", indexW, "INDEX", scoreW, "SCORE")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(width, idW+indexW+scoreW+8))))
	b.WriteString("\n")

	// Query box ~8 lines + chrome; leave rest for hits.
	maxVisible := max(m.Height-20, 6)
	selectedIdx := clamp(m.SelectedDocIdx, 0, len(r.Hits)-1)
	start, end := listWindow(selectedIdx, len(r.Hits), maxVisible)

	for i := start; i < end; i++ {
		hit := r.Hits[i]
		id := truncate(hit.ID, idW)
		index := truncate(hit.Index, indexW)
		score := fmt.Sprintf("%*.3f", scoreW, hit.Score)
		line := fmt.Sprintf("%-*s  %-*s  %s", idW, id, indexW, index, score)
		if i == selectedIdx {
			b.WriteString(selectedStyle.Width(min(width, idW+indexW+scoreW+6)).Render("▶ " + line))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-*s", idW, id)))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-*s", indexW, index)))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(score))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) searchQueryEditorView(width int) string {
	var b strings.Builder
	label := keyStyle.Render("Query")
	if m.SearchFocus == "query" {
		label = accentStyle.Render("Query")
	}
	b.WriteString(label)
	b.WriteString(dimStyle.Render("  ·  query_string or full JSON DSL"))
	b.WriteString("\n")
	if m.SearchArea != nil {
		b.WriteString(m.SearchArea.View())
	} else {
		q := m.searchQueryValue()
		if q == "" {
			b.WriteString(dimStyle.Render("(empty → match_all)  press / to edit"))
		} else {
			// Collapsed preview of query when not editing.
			for i, line := range strings.Split(q, "\n") {
				if i >= 4 {
					b.WriteString(dimStyle.Render("…"))
					break
				}
				b.WriteString(normalStyle.Render(truncate(line, max(width, 20))))
				b.WriteString("\n")
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) searchTemplateStrip() string {
	templates := searchQueryTemplates()
	var parts []string
	for i, t := range templates {
		chip := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Render(fmt.Sprintf("%d", i+1))
		parts = append(parts, chip+" "+dimStyle.Render(t.Name))
	}
	return dimStyle.Render("Templates  ") + strings.Join(parts, "  ")
}

type searchTemplate struct {
	Name  string
	Query string
}

func searchQueryTemplates() []searchTemplate {
	return []searchTemplate{
		{Name: "match_all", Query: "{\n  \"query\": {\n    \"match_all\": {}\n  }\n}"},
		{Name: "query_string", Query: "tags:merch"},
		{Name: "match", Query: "{\n  \"query\": {\n    \"match\": {\n      \"message\": \"error\"\n    }\n  }\n}"},
		{Name: "bool", Query: "{\n  \"query\": {\n    \"bool\": {\n      \"must\": [\n        { \"match\": { \"level\": \"error\" } }\n      ]\n    }\n  }\n}"},
		{Name: "range", Query: "{\n  \"query\": {\n    \"range\": {\n      \"@timestamp\": {\n        \"gte\": \"now-1d\"\n      }\n    }\n  }\n}"},
	}
}

func newSearchArea(content string, width, height int) *textarea.Model {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.Placeholder = "query_string (tags:merch)  or  full JSON {\"query\":{...}}"
	ta.SetValue(content)
	ta.SetWidth(max(width, 30))
	ta.SetHeight(max(height, 4))
	ta.Focus()
	ta.Prompt = "│ "
	return &ta
}

func (m Model) ensureSearchArea() Model {
	q := m.searchQueryValue()
	w := max(m.Width/2-6, 40)
	if m.SearchArea == nil {
		m.SearchArea = newSearchArea(q, w, 7)
	} else {
		m.SearchArea.SetWidth(w)
		m.SearchArea.SetHeight(7)
		if m.SearchArea.Value() == "" && q != "" {
			m.SearchArea.SetValue(q)
		}
	}
	return m
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
		b.WriteString("\n\n")
		b.WriteString(keyStyle.Render("Tips"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • / focus query editor"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • ctrl+enter run query"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • 1–5 insert template"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  • multi-line JSON DSL supported"))
		return b.String()
	}

	selectedIdx := clamp(m.SelectedDocIdx, 0, len(m.SearchResult.Hits)-1)
	doc := m.SearchResult.Hits[selectedIdx]

	b.WriteString(keyStyle.Render("ID: "))
	b.WriteString(normalStyle.Render(truncate(doc.ID, max(width-6, 8))))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Index: "))
	b.WriteString(normalStyle.Render(doc.Index))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Score: "))
	b.WriteString(normalStyle.Render(fmt.Sprintf("%.4f", doc.Score)))
	b.WriteString("\n\n")

	if len(doc.Source) > 0 {
		b.WriteString(keyStyle.Render("Fields"))
		b.WriteString("\n")
		keys := make([]string, 0, len(doc.Source))
		for k := range doc.Source {
			keys = append(keys, k)
		}
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[j] < keys[i] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		shown := 0
		for _, k := range keys {
			if shown >= 10 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d more", len(keys)-shown)))
				b.WriteString("\n")
				break
			}
			v := fmt.Sprint(doc.Source[k])
			if len(v) > max(width-len(k)-4, 12) {
				v = v[:max(width-len(k)-7, 8)] + "..."
			}
			b.WriteString(dimStyle.Render("  " + k + ": "))
			b.WriteString(normalStyle.Render(v))
			b.WriteString("\n")
			shown++
		}
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Source"))
	b.WriteString("\n")

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

	maxLines := max(m.Height-18, 8)
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
