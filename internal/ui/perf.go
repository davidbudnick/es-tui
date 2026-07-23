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
