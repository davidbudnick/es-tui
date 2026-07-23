package ui

import (
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
	panelHeight := max(m.Height-5, 12)

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
	help := renderKeyHelpWidth(m.Width-2, []struct{ key, desc string }{
		{"j/k", "nav"},
		{"enter", "view"},
		{"e", "edit"},
		{"d", "delete"},
		{"/", "search"},
		{"y", "copy"},
		{"n/p", "page"},
		{"D", "bulk"},
		{"r", "reload"},
		{"q", "back"},
	})
	return content + "\n" + help
}

func (m Model) viewDocumentsListOnly() string {
	var b strings.Builder
	b.WriteString(m.buildDocumentsListPanel(max(m.Width-4, 40)))
	b.WriteString("\n")
	b.WriteString(renderKeyHelpWidth(m.Width-2, []struct{ key, desc string }{
		{"j/k", "nav"},
		{"enter", "view"},
		{"e", "edit"},
		{"d", "delete"},
		{"/", "search"},
		{"q", "back"},
	}))
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
	b.WriteString("\n")

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
	b.WriteString("\n")

	if len(m.Documents) == 0 {
		if m.DocTotal > 0 && m.DocFrom > 0 {
			b.WriteString(dimStyle.Render("No documents on this page."))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("Press p for previous page."))
		} else {
			b.WriteString(dimStyle.Render("No documents."))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("Press enter to run match_all."))
		}
		return b.String()
	}

	// Columns: ID + best label field + up to 2 context fields (coverage-aware).
	cols := pickDocumentListColumns(m.Documents, width)
	withScore := docsHaveUsefulScores(m.Documents)

	// Header
	var hdr strings.Builder
	hdr.WriteString("  ")
	for i, c := range cols {
		if i > 0 {
			hdr.WriteString("  ")
		}
		hdr.WriteString(fmt.Sprintf("%-*s", c.Width, truncate(c.Header, c.Width)))
	}
	if withScore {
		hdr.WriteString(fmt.Sprintf("  %5s", "Score"))
	}
	b.WriteString(headerStyle.Render(hdr.String()))
	b.WriteString("\n")
	sepLen := 2
	for i, c := range cols {
		if i > 0 {
			sepLen += 2
		}
		sepLen += c.Width
	}
	if withScore {
		sepLen += 7
	}
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(width, sepLen))))
	b.WriteString("\n")

	maxVisible := max(m.Height-10, 8)
	selectedIdx := clamp(m.SelectedDocIdx, 0, len(m.Documents)-1)
	start, end := listWindow(selectedIdx, len(m.Documents), maxVisible)

	for i := start; i < end; i++ {
		doc := m.Documents[i]
		cells := make([]string, len(cols))
		for ci, c := range cols {
			cells[ci] = truncate(docColumnValue(doc, c), c.Width)
		}
		score := fmt.Sprintf("%5.2f", doc.Score)

		if i == selectedIdx {
			b.WriteString(selectedStyle.Render("▶ "))
			// Blue band on ID + primary label; context stays colored.
			for ci, c := range cols {
				if ci > 0 {
					b.WriteString("  ")
				}
				cell := fmt.Sprintf("%-*s", c.Width, cells[ci])
				if ci <= 1 {
					b.WriteString(selectedStyle.Render(cell))
				} else if ci == 2 {
					b.WriteString(yellowStyle.Render(cell))
				} else {
					b.WriteString(tealStyle.Render(cell))
				}
			}
			if withScore {
				b.WriteString("  ")
				b.WriteString(normalStyle.Render(score))
			}
		} else {
			b.WriteString("  ")
			for ci, c := range cols {
				if ci > 0 {
					b.WriteString("  ")
				}
				cell := fmt.Sprintf("%-*s", c.Width, cells[ci])
				switch {
				case ci == 0:
					b.WriteString(dimStyle.Render(cell))
				case ci == 1:
					b.WriteString(normalStyle.Render(cell))
				case ci == 2:
					b.WriteString(yellowStyle.Render(cell))
				default:
					b.WriteString(tealStyle.Render(cell))
				}
			}
			if withScore {
				b.WriteString("  ")
				b.WriteString(dimStyle.Render(score))
			}
		}
		b.WriteString("\n")
	}

	from := m.DocFrom
	if from < 0 {
		from = 0
	}
	b.WriteString("\n")
	pager := fmt.Sprintf("%d-%d of %d", from+start+1, from+end, m.DocTotal)
	if m.canDocPageNext() || from > 0 {
		pager += "  (n/p page)"
	}
	if !m.canDocPageNext() && from == 0 && m.DocTotal > 0 {
		pager += "  · single page"
	}
	b.WriteString(dimStyle.Render(pager))
	return b.String()
}

// labelFieldPriority: human-facing identity for a row (products.name, logs.message, …).
var labelFieldPriority = []string{
	"name", "title", "message", "order_id", "customer_id", "sku", "event_id",
	"email", "host", "service",
}

// contextFieldPriority: useful facets next to the label (not identity).
var contextFieldPriority = []string{
	"category", "brand", "plan", "status", "level", "service", "company",
	"country", "city", "email", "customer", "env", "type", "active",
	"tags", "price", "rating", "currency", "quantity", "in_stock",
}

// noisyFields: never promote to list columns (ids/timestamps already covered elsewhere).
var noisyFields = map[string]bool{
	"@timestamp": true, "timestamp": true, "created_at": true, "updated_at": true,
	"signup_at": true, "description": true, "trace_id": true, "uuid": true,
	"id": true, "_id": true, "ltv": true,
}

// docListCol is one auto-picked column for the documents table.
type docListCol struct {
	Key    string // source field, or "_id" for document id
	Header string
	Width  int
}

// pickDocumentListColumns chooses ID + label + up to 2 context columns from page coverage.
func pickDocumentListColumns(docs []types.Document, width int) []docListCol {
	withSource := 0
	counts := map[string]int{}
	for _, d := range docs {
		if d.Source == nil {
			continue
		}
		withSource++
		for k, v := range d.Source {
			if noisyFields[strings.ToLower(k)] {
				continue
			}
			if isScalarish(v) && fieldString(d.Source, k) != "" {
				counts[k]++
			}
		}
	}

	// Require field on a majority of sourced docs (or any if few docs).
	minCount := 1
	if withSource >= 4 {
		minCount = (withSource + 1) / 2 // ≥50%
	}

	present := func(key string) bool {
		return counts[key] >= minCount
	}
	pickFrom := func(priority []string, used map[string]bool) string {
		for _, k := range priority {
			if used[k] || !present(k) {
				continue
			}
			return k
		}
		return ""
	}

	used := map[string]bool{}
	label := pickFrom(labelFieldPriority, used)
	if label != "" {
		used[label] = true
	}
	// If no priority label, take the most common short string-ish field.
	if label == "" {
		label = mostCommonField(counts, minCount, used, true)
		if label != "" {
			used[label] = true
		}
	}

	extra := make([]string, 0, 2)
	for len(extra) < 2 {
		k := pickFrom(contextFieldPriority, used)
		if k == "" {
			k = mostCommonField(counts, minCount, used, false)
		}
		if k == "" {
			break
		}
		used[k] = true
		extra = append(extra, k)
	}

	// Column order: short facets first for logs-like data, long text last.
	// products → ID | Name | Category | Brand
	// logs     → ID | Level | Service | Message
	longLabel := label == "message" || label == "description" || label == "title"

	type colSpec struct{ key, header string }
	var specs []colSpec
	specs = append(specs, colSpec{"_id", "ID"})
	if longLabel {
		for _, k := range extra {
			specs = append(specs, colSpec{k, humanizeField(k)})
		}
		if label != "" {
			specs = append(specs, colSpec{label, humanizeField(label)})
		} else {
			specs = append(specs, colSpec{"_summary", "Summary"})
		}
	} else {
		if label != "" {
			specs = append(specs, colSpec{label, humanizeField(label)})
		} else {
			specs = append(specs, colSpec{"_summary", "Summary"})
		}
		for _, k := range extra {
			specs = append(specs, colSpec{k, humanizeField(k)})
		}
	}

	// Base widths, then pour remaining panel width into the flex label + grow facets.
	gaps := 2 * max(len(specs)-1, 0) // "  " between cols
	prefix := 2                      // "▶ " / "  "
	avail := width - prefix - gaps
	if avail < 30 {
		avail = 30
	}

	fixed := make([]int, len(specs))
	flexIdx := -1
	usedWidth := 0
	for i, s := range specs {
		if isFlexLabelKey(s.key) {
			flexIdx = i
			fixed[i] = 0
			continue
		}
		w := fieldColWidth(s.key)
		fixed[i] = w
		usedWidth += w
	}

	// Leftover → name/message first (readable), then grow secondary text columns.
	leftover := avail - usedWidth
	if flexIdx >= 0 {
		flexW := leftover
		if flexW < 20 {
			flexW = 20
		}
		// Prefer a comfortable label; still leave room on very wide panes.
		if flexW > 56 {
			// Spill extra into company/plan/service-style columns.
			spill := flexW - 56
			flexW = 56
			for i := range fixed {
				if i == 0 || i == flexIdx {
					continue
				}
				grow := min(spill, 8)
				fixed[i] += grow
				spill -= grow
				if spill <= 0 {
					break
				}
			}
		}
		fixed[flexIdx] = flexW
	} else if leftover > 0 && len(fixed) > 1 {
		// No flex label — spread leftover across non-ID columns.
		share := leftover / max(len(fixed)-1, 1)
		for i := 1; i < len(fixed); i++ {
			fixed[i] += share
		}
	}

	// Final pass: if still under-using the pane, grow the widest text col.
	total := 0
	for _, w := range fixed {
		total += w
	}
	if pad := avail - total; pad > 0 {
		target := flexIdx
		if target < 0 {
			target = len(fixed) - 1
		}
		if target >= 0 {
			fixed[target] += pad
		}
	}

	cols := make([]docListCol, len(specs))
	for i, s := range specs {
		cols[i] = docListCol{Key: s.key, Header: s.header, Width: max(fixed[i], 4)}
	}
	return cols
}

// fieldColWidth is the minimum comfortable width for facet columns.
func fieldColWidth(key string) int {
	switch strings.ToLower(key) {
	case "_id":
		return 6
	case "level", "status", "env", "type", "active":
		return 8
	case "plan":
		return 12 // "enterprise"
	case "service", "brand", "category", "host", "city", "country":
		return 14
	case "company":
		return 16
	case "email", "customer", "customer_id", "sku", "order_id":
		return 20
	case "tags", "currency":
		return 12
	case "price", "rating", "quantity", "in_stock":
		return 9
	default:
		return 14
	}
}

func isFlexLabelKey(key string) bool {
	switch strings.ToLower(key) {
	case "message", "name", "title", "description", "_summary":
		return true
	default:
		return false
	}
}

func mostCommonField(counts map[string]int, minCount int, used map[string]bool, preferStringy bool) string {
	best, bestN := "", 0
	for k, n := range counts {
		if used[k] || n < minCount {
			continue
		}
		if preferStringy {
			// Prefer fields that look like labels (not pure numbers in name).
			kl := strings.ToLower(k)
			if strings.Contains(kl, "price") || strings.Contains(kl, "qty") ||
				strings.Contains(kl, "count") || strings.Contains(kl, "percent") {
				continue
			}
		}
		if n > bestN || (n == bestN && (best == "" || k < best)) {
			best, bestN = k, n
		}
	}
	return best
}

func humanizeField(k string) string {
	k = strings.ReplaceAll(k, "_", " ")
	k = strings.ReplaceAll(k, ".", " ")
	parts := strings.Fields(k)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func docColumnValue(doc types.Document, c docListCol) string {
	switch c.Key {
	case "_id":
		return doc.ID
	case "_summary":
		return docSummary(doc)
	default:
		return fieldString(doc.Source, c.Key)
	}
}

// docsHaveUsefulScores is true when scores vary (real search), not match_all 1.0s.
func docsHaveUsefulScores(docs []types.Document) bool {
	if len(docs) == 0 {
		return false
	}
	// Flat 1.0 (or 0) from match_all / filter-only — hide Score column.
	allFlat := true
	var minS, maxS float64
	minS, maxS = docs[0].Score, docs[0].Score
	for _, d := range docs {
		if d.Score < minS {
			minS = d.Score
		}
		if d.Score > maxS {
			maxS = d.Score
		}
		// Not the boring constant-1.0 page.
		if d.Score < 0.999 || d.Score > 1.001 {
			allFlat = false
		}
	}
	if allFlat {
		return false
	}
	if maxS-minS > 0.05 {
		return true
	}
	if maxS > 1.05 {
		return true
	}
	return false
}

func docSummary(doc types.Document) string {
	if doc.Source == nil {
		if doc.Raw != "" {
			return truncate(strings.ReplaceAll(doc.Raw, "\n", " "), 60)
		}
		return doc.ID
	}
	for _, k := range labelFieldPriority {
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
		if noisyFields[strings.ToLower(k)] {
			continue
		}
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
	sepW := max(min(width, 40), 12)

	b.WriteString(titleStyle.Render("Preview"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")

	if len(m.Documents) == 0 {
		b.WriteString(dimStyle.Render("No document selected"))
		return b.String()
	}
	selectedIdx := clamp(m.SelectedDocIdx, 0, len(m.Documents)-1)
	doc := m.Documents[selectedIdx]

	writeKV := func(k, v string) {
		b.WriteString(keyStyle.Render(k + ": "))
		b.WriteString(normalStyle.Render(v))
		b.WriteString("\n")
	}
	writeKV("ID", truncate(doc.ID, max(width-6, 8)))
	writeKV("Index", doc.Index)
	if s := docSummary(doc); s != "" && s != doc.ID {
		writeKV("Summary", truncate(s, max(width-10, 8)))
	}
	writeKV("Score", fmt.Sprintf("%.4f", doc.Score))

	// Field chips: top-level scalars for quick scan
	if len(doc.Source) > 0 {
		b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
		b.WriteString("\n")
		b.WriteString(keyStyle.Render("Fields"))
		b.WriteString("\n")
		keys := make([]string, 0, len(doc.Source))
		for k := range doc.Source {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := make([]string, 0, len(keys))
		seen := map[string]bool{}
		for _, k := range labelFieldPriority {
			if _, ok := doc.Source[k]; ok {
				ordered = append(ordered, k)
				seen[k] = true
			}
		}
		for _, k := range contextFieldPriority {
			if _, ok := doc.Source[k]; ok && !seen[k] {
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
	}

	b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Source"))
	b.WriteString("\n")

	// Use cached pretty body when selection matches (refreshed on j/k in Update).
	plain := m.PreviewBody
	trunc := m.PreviewTruncated
	wantID := doc.Index + "/" + doc.ID
	if m.PreviewDocID != wantID || plain == "" {
		plain, trunc = documentSourceJSON(doc)
	}
	if plain == "" {
		b.WriteString(dimStyle.Render("(empty source)"))
		return b.String()
	}

	maxLines := max(min(m.Height-18, maxPreviewSourceLines), 6)
	lines := wrapPlainLines(strings.Split(plain, "\n"), max(width-2, 20))
	shown := 0
	for _, line := range lines {
		if shown >= maxLines {
			b.WriteString(dimStyle.Render(fmt.Sprintf("… %d more lines", len(lines)-shown)))
			b.WriteString("\n")
			break
		}
		b.WriteString(colorizeJSONLine(line))
		b.WriteString("\n")
		shown++
	}
	if trunc {
		b.WriteString(dimStyle.Render("(preview truncated at 64KB)"))
	}
	return b.String()
}

func (m Model) viewDocumentDetail() string {
	if m.CurrentDocument == nil {
		return dimStyle.Render("No document")
	}
	doc := *m.CurrentDocument
	boxWidth := detailBoxWidth(m.Width)
	contentWidth := detailContentWidth(boxWidth)

	var b strings.Builder
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, titleStyle.Render("Document Detail")))
	b.WriteString("\n\n")

	// Fixed-width labels so the whole meta block lines up (not each line re-centered).
	const labelW = 9 // "Summary: "
	metaLine := func(label, value string) string {
		return keyStyle.Render(fmt.Sprintf("%*s", labelW, label+":")) + " " + normalStyle.Render(value)
	}
	var meta strings.Builder
	meta.WriteString(metaLine("Index", doc.Index))
	meta.WriteString("\n")
	meta.WriteString(metaLine("ID", doc.ID))
	if s := docSummary(doc); s != "" && s != doc.ID {
		meta.WriteString("\n")
		meta.WriteString(metaLine("Summary", truncate(s, min(boxWidth-labelW-2, 60))))
	}
	if doc.Score > 0.001 && (doc.Score < 0.999 || doc.Score > 1.001) {
		meta.WriteString("\n")
		meta.WriteString(metaLine("Score", fmt.Sprintf("%.4f", doc.Score)))
	}
	// Center the block as one unit — keeps colons stacked.
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, meta.String()))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, keyStyle.Render("Value")))
	b.WriteString("\n")

	// Plain text first (like redis-tui) so wrap/scroll line counts match paint.
	valueLines := m.documentDetailLines()
	maxVisible := detailMaxVisible(m.Height)
	_, displayScroll := ensureDetailCursorVisible(m.DetailCursor, m.DetailScroll, len(valueLines), maxVisible)
	visible, topHint, bottomHint, displayScroll := scrollValueLines(valueLines, displayScroll, maxVisible)

	var display strings.Builder
	if topHint != "" {
		display.WriteString(topHint)
		display.WriteByte('\n')
	}
	for i, line := range visible {
		abs := displayScroll + i
		if abs == m.DetailCursor {
			plain := padRight(truncateRunes(line, contentWidth), contentWidth)
			display.WriteString(selectedStyle.Render(plain))
		} else {
			display.WriteString(colorizeJSONLine(line))
		}
		if i < len(visible)-1 {
			display.WriteByte('\n')
		}
	}
	if bottomHint != "" {
		display.WriteByte('\n')
		display.WriteString(bottomHint)
	}

	box := valueBoxStyle.Width(boxWidth).Render(display.String())
	b.WriteString(lipgloss.PlaceHorizontal(min(m.Width, boxWidth+4), lipgloss.Center, box))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(min(m.Width, boxWidth+4), lipgloss.Center, renderKeyHelpWidth(boxWidth, []struct{ key, desc string }{
		{"j/k", "move"},
		{"pgup/pgdn", "page"},
		{"e", "edit"},
		{"d", "delete"},
		{"y", "copy"},
		{"esc", "back"},
	})))
	return b.String()
}

// documentDetailLines returns width-wrapped plain JSON lines (uses load-time cache).
func (m Model) documentDetailLines() []string {
	w := detailContentWidth(detailBoxWidth(m.Width))
	if m.DetailWrapWidth == w && m.DetailLinesCache != nil {
		return m.DetailLinesCache
	}
	body := m.DetailBody
	if body == "" && m.CurrentDocument != nil {
		body, _ = documentSourceJSON(*m.CurrentDocument)
	}
	if body == "" {
		return []string{"(empty)"}
	}
	// View is by-value so we can't write cache here; wrap is still cheap vs indent.
	return wrapPlainLines(strings.Split(body, "\n"), w)
}

func (m Model) documentDetailLineCount() int {
	return len(m.documentDetailLines())
}

func (m Model) documentDetailLastLine() int {
	return max(m.documentDetailLineCount()-1, 0)
}

func (m *Model) syncDocumentDetailScroll() {
	// Populate wrap cache once per width in Update paths.
	w := detailContentWidth(detailBoxWidth(m.Width))
	if m.DetailWrapWidth != w || m.DetailLinesCache == nil {
		body := m.DetailBody
		if body == "" && m.CurrentDocument != nil {
			body, _ = documentSourceJSON(*m.CurrentDocument)
			m.DetailBody = body
		}
		if body == "" {
			m.DetailLinesCache = []string{"(empty)"}
		} else {
			m.DetailLinesCache = wrapPlainLines(strings.Split(body, "\n"), w)
		}
		m.DetailWrapWidth = w
	}
	m.DetailCursor, m.DetailScroll = ensureDetailCursorVisible(
		m.DetailCursor,
		m.DetailScroll,
		len(m.DetailLinesCache),
		detailMaxVisible(m.Height),
	)
}

func detailBoxWidth(termWidth int) int {
	w := termWidth * 3 / 5
	if w < 50 {
		w = 50
	}
	if w > termWidth-6 {
		w = max(termWidth-6, 40)
	}
	if w > 100 {
		w = 100
	}
	return w
}

// colorizeJSONLine colors a single pretty-printed JSON line (no re-indent).
func colorizeJSONLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}
	// Avoid Indent on fragments — colorizeJSON trims + may reformat full docs only.
	return colorizeJSONFragment(line)
}

var _ = types.Document{}
