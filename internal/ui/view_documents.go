package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func (m Model) viewDocuments() string {
	totalWidth := m.Width
	if totalWidth < 100 {
		return m.viewDocumentsListOnly()
	}

	leftWidth := (totalWidth * 58) / 100
	rightWidth := totalWidth - leftWidth - 1
	panelHeight := max(m.Height-4, 12)

	leftContent := m.buildDocumentsListPanel(leftWidth - 2)
	rightContent := m.buildDocumentPreviewPanel(rightWidth - 2)

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
	help := helpStyle.Render("j/k:nav  enter:view  e:edit  d:del  /:search  y:copy  n/p:page  D:bulk  r:reload  q:back")
	return content + "\n" + help
}

func (m Model) viewDocumentsListOnly() string {
	var b strings.Builder
	b.WriteString(m.buildDocumentsListPanel(max(m.Width-4, 40)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:view  e:edit  d:del  /:search  q:back"))
	return b.String()
}

func (m Model) buildDocumentsListPanel(width int) string {
	var b strings.Builder

	name := ""
	if m.CurrentIndex != nil {
		name = m.CurrentIndex.Name
	}
	title := "Documents - " + name
	if m.DocTotal > 0 {
		title += fmt.Sprintf(" [%d]", m.DocTotal)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Query: "))
	if m.Inputs != nil && m.Inputs.SearchInput.Focused() {
		b.WriteString(m.Inputs.SearchInput.View())
	} else {
		q := m.DocQuery
		if q == "" {
			q = "*"
		}
		b.WriteString(normalStyle.Render(truncate(q, max(width-10, 10))))
	}
	b.WriteString("\n\n")

	if len(m.Documents) == 0 {
		b.WriteString(dimStyle.Render("No documents."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press enter to run match_all."))
		return b.String()
	}

	// Discover useful columns from the current page of docs
	colA, colB := pickListColumns(m.Documents)
	idW := 10
	scoreW := 7
	// remaining width for summary + optional cols
	rest := width - idW - scoreW - 10
	if rest < 20 {
		rest = 20
	}
	sumW := rest
	colAW, colBW := 0, 0
	if colA != "" {
		colAW = min(16, rest/3)
		sumW = rest - colAW - 2
	}
	if colB != "" {
		colBW = min(12, rest/4)
		sumW = rest - colAW - colBW - 4
	}
	if sumW < 12 {
		sumW = 12
	}

	// Header
	var hdr strings.Builder
	fmt.Fprintf(&hdr, "  %-*s  %-*s", idW, "ID", sumW, "Summary")
	if colA != "" {
		fmt.Fprintf(&hdr, "  %-*s", colAW, truncate(titleCase(colA), colAW))
	}
	if colB != "" {
		fmt.Fprintf(&hdr, "  %-*s", colBW, truncate(titleCase(colB), colBW))
	}
	fmt.Fprintf(&hdr, "  %s", "Score")
	b.WriteString(headerStyle.Render(hdr.String()))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(width, idW+sumW+colAW+colBW+scoreW+16))))
	b.WriteString("\n")

	maxVisible := max(m.Height-12, 5)
	selectedIdx := clamp(m.SelectedDocIdx, 0, len(m.Documents)-1)
	start := 0
	if selectedIdx >= maxVisible {
		start = selectedIdx - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.Documents))

	for i := start; i < end; i++ {
		doc := m.Documents[i]
		id := truncate(doc.ID, idW)
		summary := truncate(docSummary(doc), sumW)
		aVal, bVal := "", ""
		if colA != "" {
			aVal = truncate(fieldString(doc.Source, colA), colAW)
		}
		if colB != "" {
			bVal = truncate(fieldString(doc.Source, colB), colBW)
		}
		score := fmt.Sprintf("%.3f", doc.Score)

		if i == selectedIdx {
			// Blue bar on ID + summary (redis-style), keep colored/meta after
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-*s", idW, id)))
			b.WriteString("  ")
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-*s", sumW, summary)))
			if colA != "" {
				b.WriteString("  ")
				b.WriteString(yellowStyle.Render(fmt.Sprintf("%-*s", colAW, aVal)))
			}
			if colB != "" {
				b.WriteString("  ")
				b.WriteString(tealStyle.Render(fmt.Sprintf("%-*s", colBW, bVal)))
			}
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(score))
		} else {
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-*s", idW, id)))
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-*s", sumW, summary)))
			if colA != "" {
				b.WriteString("  ")
				b.WriteString(yellowStyle.Render(fmt.Sprintf("%-*s", colAW, aVal)))
			}
			if colB != "" {
				b.WriteString("  ")
				b.WriteString(tealStyle.Render(fmt.Sprintf("%-*s", colBW, bVal)))
			}
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(score))
		}
		b.WriteString("\n")
	}

	from := m.DocFrom
	if from < 0 {
		from = 0
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("%d-%d of %d  (n/p page)", from+start+1, from+end, m.DocTotal)))
	return b.String()
}

// preferred summary field names, in order of usefulness
var summaryFieldPriority = []string{
	"name", "title", "email", "customer_id", "order_id", "sku", "message",
	"event_id", "host", "service", "level", "status", "company", "plan",
}

// preferred extra columns after summary
var listColumnPriority = []string{
	"email", "plan", "status", "company", "country", "city", "category", "brand",
	"level", "service", "host", "customer", "tags", "type", "active",
}

func pickListColumns(docs []types.Document) (colA, colB string) {
	// count field presence across page
	counts := map[string]int{}
	for _, d := range docs {
		if d.Source == nil {
			continue
		}
		for k, v := range d.Source {
			if isScalarish(v) {
				counts[k]++
			}
		}
	}
	// pick up to 2 from priority that exist often enough, skip ones used as summary-only
	for _, k := range listColumnPriority {
		if counts[k] == 0 {
			continue
		}
		// avoid duplicating the primary summary field
		if colA == "" {
			colA = k
			continue
		}
		if k != colA {
			colB = k
			break
		}
	}
	return colA, colB
}

func docSummary(doc types.Document) string {
	if doc.Source == nil {
		if doc.Raw != "" {
			return truncate(strings.ReplaceAll(doc.Raw, "\n", " "), 60)
		}
		return doc.ID
	}
	for _, k := range summaryFieldPriority {
		if s := fieldString(doc.Source, k); s != "" {
			return s
		}
	}
	// fallback: first string-ish field alphabetically
	keys := make([]string, 0, len(doc.Source))
	for k := range doc.Source {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if s := fieldString(doc.Source, k); s != "" {
			return s
		}
	}
	return doc.ID
}

func fieldString(src map[string]any, key string) string {
	if src == nil {
		return ""
	}
	v, ok := src[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case []any:
		parts := make([]string, 0, len(t))
		for _, x := range t {
			parts = append(parts, fmt.Sprint(x))
			if len(parts) >= 3 {
				break
			}
		}
		return strings.Join(parts, ",")
	default:
		s := fmt.Sprint(t)
		if len(s) > 40 {
			return s[:37] + "..."
		}
		return s
	}
}

func isScalarish(v any) bool {
	switch v.(type) {
	case string, float64, bool, int, int64:
		return true
	case []any:
		return true
	default:
		return false
	}
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (m Model) buildDocumentPreviewPanel(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Preview"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n\n")

	if len(m.Documents) == 0 {
		b.WriteString(dimStyle.Render("No document selected"))
		return b.String()
	}
	selectedIdx := clamp(m.SelectedDocIdx, 0, len(m.Documents)-1)
	doc := m.Documents[selectedIdx]

	b.WriteString(keyStyle.Render("ID: "))
	b.WriteString(normalStyle.Render(truncate(doc.ID, max(width-6, 8))))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Index: "))
	b.WriteString(normalStyle.Render(doc.Index))
	b.WriteString("\n\n")

	if s := docSummary(doc); s != "" && s != doc.ID {
		b.WriteString(keyStyle.Render("Summary: "))
		b.WriteString(normalStyle.Render(truncate(s, max(width-10, 8))))
		b.WriteString("\n\n")
	}

	b.WriteString(keyStyle.Render("Score: "))
	b.WriteString(normalStyle.Render(fmt.Sprintf("%.4f", doc.Score)))
	b.WriteString("\n\n")

	// Field chips: top-level scalars for quick scan
	if len(doc.Source) > 0 {
		b.WriteString(keyStyle.Render("Fields"))
		b.WriteString("\n\n")
		keys := make([]string, 0, len(doc.Source))
		for k := range doc.Source {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// prefer known fields first
		ordered := make([]string, 0, len(keys))
		seen := map[string]bool{}
		for _, k := range summaryFieldPriority {
			if _, ok := doc.Source[k]; ok {
				ordered = append(ordered, k)
				seen[k] = true
			}
		}
		for _, k := range keys {
			if !seen[k] {
				ordered = append(ordered, k)
			}
		}
		shown := 0
		for _, k := range ordered {
			if shown >= 10 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d more", len(ordered)-shown)))
				b.WriteString("\n")
				break
			}
			v := fieldString(doc.Source, k)
			if v == "" {
				continue
			}
			b.WriteString(dimStyle.Render("  " + k + ": "))
			b.WriteString(normalStyle.Render(truncate(v, max(width-len(k)-6, 8))))
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
		b.WriteString(dimStyle.Render("(empty source)"))
		return b.String()
	}

	maxLines := max(m.Height-24, 4)
	colored := colorizeJSON(raw)
	lines := strings.Split(colored, "\n")
	shown := 0
	for _, line := range lines {
		if shown >= maxLines {
			b.WriteString(dimStyle.Render(fmt.Sprintf("… %d more lines", len(lines)-shown)))
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
		shown++
	}
	return b.String()
}

func (m Model) viewDocumentDetail() string {
	if m.CurrentDocument == nil {
		return dimStyle.Render("No document")
	}
	doc := *m.CurrentDocument
	boxWidth := detailBoxWidth(m.Width)

	var b strings.Builder
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, titleStyle.Render("Document Detail")))
	b.WriteString("\n\n")

	var meta strings.Builder
	meta.WriteString(keyStyle.Render("  Index: "))
	meta.WriteString(normalStyle.Render(doc.Index))
	meta.WriteString("\n")
	meta.WriteString(keyStyle.Render("     ID: "))
	meta.WriteString(normalStyle.Render(doc.ID))
	if s := docSummary(doc); s != "" && s != doc.ID {
		meta.WriteString("\n")
		meta.WriteString(keyStyle.Render("Summary: "))
		meta.WriteString(normalStyle.Render(truncate(s, 50)))
	}
	meta.WriteString("\n")
	meta.WriteString(keyStyle.Render("  Score: "))
	meta.WriteString(normalStyle.Render(fmt.Sprintf("%.4f", doc.Score)))
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, meta.String()))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, keyStyle.Render("Value:")))
	b.WriteString("\n")

	raw := doc.Raw
	if raw == "" && doc.Source != nil {
		if bb, err := json.MarshalIndent(doc.Source, "", "  "); err == nil {
			raw = string(bb)
		}
	}
	body := colorizeJSON(raw)
	lines := strings.Split(body, "\n")
	maxVisible := max(m.Height-14, 6)
	start := clamp(m.DetailScroll, 0, max(0, len(lines)-1))
	end := min(start+maxVisible, len(lines))

	var display strings.Builder
	for i := start; i < end; i++ {
		display.WriteString(lines[i])
		display.WriteString("\n")
	}
	if len(lines) > maxVisible {
		display.WriteString(dimStyle.Render(fmt.Sprintf("[%d-%d / %d lines]", start+1, end, len(lines))))
	}

	box := valueBoxStyle.Width(boxWidth).Render(display.String())
	b.WriteString(lipgloss.PlaceHorizontal(min(m.Width, boxWidth+4), lipgloss.Center, box))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("e:edit  d:delete  j/k:scroll  y:copy  esc:back"))
	return b.String()
}

func detailBoxWidth(termWidth int) int {
	w := termWidth - 8
	if w > 100 {
		w = 100
	}
	if w < 40 {
		w = 40
	}
	return w
}

var _ = types.Document{}
