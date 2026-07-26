package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/davidbudnick/es-tui/internal/types"
)

// Bounds mirror redis-tui: keep View() cheap and ES payloads finite.
const (
	// maxJSONPrettyBytes caps pretty-printed JSON kept in memory / colored.
	maxJSONPrettyBytes = 64 * 1024
	// maxPreviewSourceLines caps source lines shown in the documents preview pane.
	maxPreviewSourceLines = 40
	// maxSearchPageSize hard-caps page size sent to ES/OS.
	maxSearchPageSize = 200
	// maxCatDisplayLines caps cat API output rows in the TUI.
	maxCatDisplayLines = 400
	// maxJSONPanelLines caps settings/mappings/cluster settings scroll body.
	maxJSONPanelLines = 2000
)

// boundJSONBody pretty-prints JSON and truncates huge payloads (redis-style 64KB bound).
func boundJSONBody(raw string) (body string, truncated bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	body = prettyJSON(raw)
	if body == "" {
		body = raw
	}
	if len(body) <= maxJSONPrettyBytes {
		return body, false
	}
	// Cut on a line boundary when possible so the viewer stays readable.
	cut := maxJSONPrettyBytes
	if i := strings.LastIndex(body[:cut], "\n"); i > maxJSONPrettyBytes/2 {
		cut = i
	}
	return body[:cut] + "\n… (truncated — document exceeds 64KB preview)", true
}

// documentSourceJSON returns bounded pretty JSON for a document.
func documentSourceJSON(doc types.Document) (string, bool) {
	raw := doc.Raw
	if raw == "" && doc.Source != nil {
		if bb, err := json.MarshalIndent(doc.Source, "", "  "); err == nil {
			raw = string(bb)
		} else {
			raw = fmt.Sprint(doc.Source)
		}
	}
	return boundJSONBody(raw)
}

// clampPageSize enforces a sane ES size parameter (and redis-style upper bound).
func clampPageSize(size int) int {
	if size <= 0 {
		return 50
	}
	if size > maxSearchPageSize {
		return maxSearchPageSize
	}
	return size
}

// truncateLines keeps the first n lines and notes how many were dropped.
func truncateLines(s string, maxLines int) (string, int) {
	if maxLines <= 0 || s == "" {
		return s, 0
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s, 0
	}
	return strings.Join(lines[:maxLines], "\n"), len(lines) - maxLines
}

// (m *Model) setDetailBody caches pretty JSON when a document is opened.
func (m *Model) setDetailBody(doc types.Document) {
	body, trunc := documentSourceJSON(doc)
	m.DetailBody = body
	m.DetailTruncated = trunc
	m.DetailLinesCache = nil
	m.DetailWrapWidth = 0
}

// invalidateDetailCache clears detail render caches (resize / leave detail).
func (m *Model) invalidateDetailCache() {
	m.DetailLinesCache = nil
	m.DetailWrapWidth = 0
}

// setPreviewBody caches pretty JSON for the documents list preview pane.
func (m *Model) setPreviewBody(doc types.Document) {
	body, trunc := documentSourceJSON(doc)
	m.PreviewDocID = doc.Index + "/" + doc.ID
	m.PreviewBody = body
	m.PreviewTruncated = trunc
}

// refreshDocPreviewFromSelection rebuilds preview cache for the selected list row.
func (m *Model) refreshDocPreviewFromSelection() {
	if len(m.Documents) == 0 {
		m.PreviewDocID = ""
		m.PreviewBody = ""
		m.PreviewTruncated = false
		return
	}
	idx := clamp(m.SelectedDocIdx, 0, len(m.Documents)-1)
	m.setPreviewBody(m.Documents[idx])
}

// setJSONPanel bounds settings/mappings/cluster JSON once for the panel viewer.
func (m *Model) setJSONPanel(raw string) {
	m.JSONPanelRaw = raw
	if raw == "" {
		m.JSONPanelPlain = ""
		m.JSONPanelLines = nil
		m.JSONPanelWidth = 0
		m.JSONPanelTrunc = false
		return
	}
	plain, trunc := boundJSONBody(raw)
	if plain == "" {
		plain = raw
	}
	if lines, dropped := truncateLines(plain, maxJSONPanelLines); dropped > 0 {
		plain = lines
		trunc = true
	}
	m.JSONPanelPlain = plain
	m.JSONPanelTrunc = trunc
	m.JSONPanelLines = nil
	m.JSONPanelWidth = 0
	if m.Width > 0 {
		m.syncJSONPanelWrap(jsonPanelWrapWidth(m.Width))
	}
}

// syncJSONPanelWrap fills wrapped lines for the given content width.
func (m *Model) syncJSONPanelWrap(width int) {
	if width <= 0 {
		width = 40
	}
	if m.JSONPanelPlain == "" {
		m.JSONPanelLines = nil
		m.JSONPanelWidth = width
		return
	}
	if m.JSONPanelWidth == width && m.JSONPanelLines != nil {
		return
	}
	m.JSONPanelLines = wrapPlainLines(strings.Split(m.JSONPanelPlain, "\n"), width)
	m.JSONPanelWidth = width
}

// invalidateJSONPanelCache drops wrap lines so the next sync rewraps.
func (m *Model) invalidateJSONPanelCache() {
	m.JSONPanelLines = nil
	m.JSONPanelWidth = 0
}

func jsonPanelWrapWidth(termWidth int) int {
	return max(termWidth-8, 40)
}

// jsonPanelLines returns wrapped plain lines for body, preferring the Update cache.
func (m Model) jsonPanelLines(body string, width int) (lines []string, trunc bool) {
	if body == "" {
		return nil, false
	}
	if m.JSONPanelRaw == body && m.JSONPanelPlain != "" {
		trunc = m.JSONPanelTrunc
		if m.JSONPanelWidth == width && m.JSONPanelLines != nil {
			return m.JSONPanelLines, trunc
		}
		return wrapPlainLines(strings.Split(m.JSONPanelPlain, "\n"), width), trunc
	}
	plain, trunc := boundJSONBody(body)
	if plain == "" {
		plain = body
	}
	if limited, dropped := truncateLines(plain, maxJSONPanelLines); dropped > 0 {
		plain = limited
		trunc = true
	}
	return wrapPlainLines(strings.Split(plain, "\n"), width), trunc
}

func docListSignature(docs []types.Document, from int) string {
	n := len(docs)
	if n == 0 {
		return fmt.Sprintf("0@%d", from)
	}
	return fmt.Sprintf("%d@%d:%s:%s", n, from, docs[0].ID, docs[n-1].ID)
}

func (m Model) docListPanelWidth() int {
	if m.Width < 100 {
		return max(m.Width-4, 40)
	}
	return (m.Width*58)/100 - 2
}

// refreshDocListColumns memoizes page columns and score usefulness for the list panel.
func (m *Model) refreshDocListColumns(width int) {
	if width <= 0 {
		width = m.docListPanelWidth()
	}
	if width <= 0 {
		width = 80
	}
	if len(m.Documents) == 0 {
		m.DocListCols = nil
		m.DocListColsWidth = width
		m.DocListColsSig = docListSignature(nil, m.DocFrom)
		m.DocListWithScore = false
		return
	}
	m.DocListCols = pickDocumentListColumns(m.Documents, width)
	m.DocListColsWidth = width
	m.DocListColsSig = docListSignature(m.Documents, m.DocFrom)
	m.DocListWithScore = docsHaveUsefulScores(m.Documents)
}

// documentListColumns returns memoized columns when the page/width still match.
func (m Model) documentListColumns(width int) (cols []docListCol, withScore bool) {
	sig := docListSignature(m.Documents, m.DocFrom)
	if m.DocListCols != nil && m.DocListColsWidth == width && m.DocListColsSig == sig {
		return m.DocListCols, m.DocListWithScore
	}
	return pickDocumentListColumns(m.Documents, width), docsHaveUsefulScores(m.Documents)
}

// invalidateDocListColumns clears memoized document list columns.
func (m *Model) invalidateDocListColumns() {
	m.DocListCols = nil
	m.DocListColsWidth = 0
	m.DocListColsSig = ""
	m.DocListWithScore = false
}
