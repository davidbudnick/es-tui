package ui

import (
	"strings"
	"testing"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestBoundJSONBody(t *testing.T) {
	body, trunc := boundJSONBody(`{"a":1}`)
	if trunc || !strings.Contains(body, `"a"`) {
		t.Fatalf("small json: trunc=%v body=%q", trunc, body)
	}

	// Force truncate with a large pretty payload.
	big := `{"data":"` + strings.Repeat("x", maxJSONPrettyBytes) + `"}`
	body, trunc = boundJSONBody(big)
	if !trunc {
		t.Fatal("expected truncation")
	}
	if len(body) > maxJSONPrettyBytes+80 {
		t.Fatalf("truncated body still too large: %d", len(body))
	}
	if !strings.Contains(body, "truncated") {
		t.Fatalf("missing marker: %q", body[len(body)-40:])
	}
}

func TestClampPageSize(t *testing.T) {
	if clampPageSize(0) != 50 || clampPageSize(-1) != 50 {
		t.Fatal("default")
	}
	if clampPageSize(10) != 10 {
		t.Fatal("pass-through")
	}
	if clampPageSize(9999) != maxSearchPageSize {
		t.Fatal("cap")
	}
}

func TestDocumentSourceCacheOnLoad(t *testing.T) {
	m, _ := testModel(t)
	doc := types.Document{
		Index: "products",
		ID:    "1",
		Raw:   `{"name":"Widget","category":"hardware"}`,
	}
	m.setDetailBody(doc)
	if m.DetailBody == "" || m.DetailTruncated {
		t.Fatalf("detail body=%q trunc=%v", m.DetailBody, m.DetailTruncated)
	}
	m.Documents = []types.Document{doc}
	m.SelectedDocIdx = 0
	m.refreshDocPreviewFromSelection()
	if m.PreviewDocID != "products/1" || m.PreviewBody == "" {
		t.Fatalf("preview id=%q body empty=%v", m.PreviewDocID, m.PreviewBody == "")
	}
}

func TestTruncateLines(t *testing.T) {
	s, n := truncateLines("a\nb\nc\nd", 2)
	if n != 2 || s != "a\nb" {
		t.Fatalf("got %q dropped=%d", s, n)
	}
	s, n = truncateLines("only", 10)
	if n != 0 || s != "only" {
		t.Fatal("no drop")
	}
}

func TestJSONPanelCache(t *testing.T) {
	m := NewModel()
	m.Width = 120
	raw := `{"index":{"number_of_shards":"1","number_of_replicas":"0"}}`
	m.setJSONPanel(raw)
	if m.JSONPanelRaw != raw || m.JSONPanelPlain == "" {
		t.Fatalf("plain cache empty raw=%q plain=%q", m.JSONPanelRaw, m.JSONPanelPlain)
	}
	if m.JSONPanelLines == nil || m.JSONPanelWidth != jsonPanelWrapWidth(m.Width) {
		t.Fatalf("wrap not filled width=%d lines=%v", m.JSONPanelWidth, m.JSONPanelLines != nil)
	}
	w := jsonPanelWrapWidth(m.Width)
	lines1, trunc1 := m.jsonPanelLines(raw, w)
	if trunc1 || len(lines1) == 0 {
		t.Fatalf("hit miss: lines=%d trunc=%v", len(lines1), trunc1)
	}
	if &lines1[0] != &m.JSONPanelLines[0] {
		t.Fatal("expected cache hit to return stored lines")
	}

	m.invalidateJSONPanelCache()
	if m.JSONPanelLines != nil || m.JSONPanelWidth != 0 {
		t.Fatal("invalidate should drop wrap lines")
	}
	lines3, _ := m.jsonPanelLines(raw, w)
	if len(lines3) != len(lines1) {
		t.Fatalf("rewrap len %d want %d", len(lines3), len(lines1))
	}

	m.syncJSONPanelWrap(w)
	m.Width = 80
	m.invalidateJSONPanelCache()
	m.syncJSONPanelWrap(jsonPanelWrapWidth(m.Width))
	if m.JSONPanelWidth != jsonPanelWrapWidth(80) || m.JSONPanelLines == nil {
		t.Fatalf("resync failed width=%d", m.JSONPanelWidth)
	}

	other := `{"mappings":{"properties":{"name":{"type":"text"}}}}`
	linesOther, _ := m.jsonPanelLines(other, jsonPanelWrapWidth(m.Width))
	if len(linesOther) == 0 {
		t.Fatal("fallback empty")
	}
	if m.JSONPanelRaw == other {
		t.Fatal("view fallback must not mutate cache identity")
	}
}

func TestDocListColumnsCache(t *testing.T) {
	products := []types.Document{
		{ID: "1", Score: 1, Source: map[string]any{"name": "Widget", "category": "hardware", "brand": "elastic", "sku": "A", "price": 10.0}},
		{ID: "2", Score: 1, Source: map[string]any{"name": "Mug", "category": "merch", "brand": "kibana", "sku": "B", "price": 14.0}},
		{ID: "3", Score: 1, Source: map[string]any{"name": "Shirt", "category": "merch", "brand": "elastic", "sku": "C", "price": 22.0}},
		{ID: "4", Score: 1, Source: map[string]any{"name": "Hat", "category": "merch", "brand": "opensearch", "sku": "D", "price": 18.0}},
	}
	m := NewModel()
	m.Width = 120
	m.Documents = products
	m.DocFrom = 0
	width := 100
	m.refreshDocListColumns(width)
	if m.DocListCols == nil || m.DocListColsWidth != width {
		t.Fatalf("cache not filled width=%d cols=%v", m.DocListColsWidth, m.DocListCols)
	}
	if m.DocListWithScore {
		t.Fatal("flat scores should hide score")
	}
	want := pickDocumentListColumns(products, width)
	got, withScore := m.documentListColumns(width)
	if withScore {
		t.Fatal("cached score flag wrong")
	}
	if len(got) != len(want) {
		t.Fatalf("cols len %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Key != want[i].Key || got[i].Width != want[i].Width || got[i].Header != want[i].Header {
			t.Fatalf("col[%d]=%+v want %+v", i, got[i], want[i])
		}
	}
	if &got[0] != &m.DocListCols[0] {
		t.Fatal("expected memoized columns slice")
	}

	other, _ := m.documentListColumns(60)
	wantOther := pickDocumentListColumns(products, 60)
	if len(other) != len(wantOther) {
		t.Fatalf("width miss len %d want %d", len(other), len(wantOther))
	}
	if m.DocListColsWidth != width {
		t.Fatal("view miss must not write cache")
	}

	m.Documents = append([]types.Document{}, products[1:]...)
	m.DocFrom = 50
	miss, _ := m.documentListColumns(width)
	wantMiss := pickDocumentListColumns(m.Documents, width)
	if len(miss) != len(wantMiss) {
		t.Fatalf("sig miss len %d want %d", len(miss), len(wantMiss))
	}
	m.refreshDocListColumns(width)
	if m.DocListColsSig != docListSignature(m.Documents, m.DocFrom) {
		t.Fatalf("sig=%q", m.DocListColsSig)
	}

	scored := []types.Document{
		{ID: "a", Score: 0.2, Source: map[string]any{"name": "A"}},
		{ID: "b", Score: 3.5, Source: map[string]any{"name": "B"}},
	}
	m.Documents = scored
	m.DocFrom = 0
	m.refreshDocListColumns(width)
	if !m.DocListWithScore || !docsHaveUsefulScores(scored) {
		t.Fatal("expected useful scores cached")
	}
	_, withScore = m.documentListColumns(width)
	if !withScore {
		t.Fatal("score cache miss")
	}

	m.invalidateDocListColumns()
	if m.DocListCols != nil || m.DocListColsSig != "" || m.DocListWithScore {
		t.Fatal("invalidate incomplete")
	}
}
