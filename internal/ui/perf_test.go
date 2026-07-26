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

func TestSetDetailBodyFillsAndClearsCaches(t *testing.T) {
	m, _ := testModel(t)
	m.Width = 120
	m.Height = 40
	doc := types.Document{
		Index: "products",
		ID:    "42",
		Raw:   `{"name":"Widget","price":9.99,"tags":["a","b"]}`,
	}
	m.setDetailBody(doc)
	if m.DetailBody == "" {
		t.Fatal("expected DetailBody")
	}
	if m.DetailLinesCache != nil || m.DetailColoredCache != nil || m.DetailWrapWidth != 0 {
		t.Fatal("setDetailBody should clear wrap/color caches")
	}

	m.CurrentDocument = &doc
	m.syncDocumentDetailScroll()
	if len(m.DetailLinesCache) == 0 {
		t.Fatal("expected plain wrap cache")
	}
	if len(m.DetailColoredCache) != len(m.DetailLinesCache) {
		t.Fatalf("colored len=%d plain len=%d", len(m.DetailColoredCache), len(m.DetailLinesCache))
	}
	if m.DetailWrapWidth <= 0 {
		t.Fatal("expected wrap width")
	}

	foundColor := false
	for i, plain := range m.DetailLinesCache {
		if strings.TrimSpace(plain) == "" {
			continue
		}
		if m.DetailColoredCache[i] != plain {
			foundColor = true
			break
		}
	}
	if !foundColor {
		t.Fatal("expected at least one colorized line to differ from plain")
	}
}

func TestInvalidateDetailCacheClearsColored(t *testing.T) {
	m, _ := testModel(t)
	m.Width = 100
	m.Height = 30
	doc := types.Document{Index: "idx", ID: "1", Raw: `{"a":1}`}
	m.setDetailBody(doc)
	m.CurrentDocument = &doc
	m.syncDocumentDetailScroll()
	if m.DetailLinesCache == nil || m.DetailColoredCache == nil {
		t.Fatal("expected warm caches")
	}
	m.invalidateDetailCache()
	if m.DetailLinesCache != nil || m.DetailColoredCache != nil || m.DetailWrapWidth != 0 {
		t.Fatal("invalidate should clear all detail wrap caches")
	}
	if m.DetailBody == "" {
		t.Fatal("body should remain")
	}
}

func TestPreviewLinesCacheOnSelection(t *testing.T) {
	m, _ := testModel(t)
	m.Width = 140
	m.Height = 40
	docs := []types.Document{
		{Index: "products", ID: "1", Raw: `{"name":"A","n":1}`},
		{Index: "products", ID: "2", Raw: `{"name":"B","n":2}`},
	}
	m.Documents = docs
	m.SelectedDocIdx = 0
	m.refreshDocPreviewFromSelection()
	if m.PreviewDocID != "products/1" {
		t.Fatalf("id=%q", m.PreviewDocID)
	}
	if len(m.PreviewLinesCache) == 0 || m.PreviewWrapWidth <= 0 {
		t.Fatalf("lines=%d width=%d", len(m.PreviewLinesCache), m.PreviewWrapWidth)
	}
	firstWidth := m.PreviewWrapWidth

	m.fillPreviewLinesCache(firstWidth)
	if m.PreviewWrapWidth != firstWidth {
		t.Fatal("refill same width should keep width")
	}

	m.SelectedDocIdx = 1
	m.refreshDocPreviewFromSelection()
	if m.PreviewDocID != "products/2" {
		t.Fatalf("id=%q after nav", m.PreviewDocID)
	}
	if m.PreviewBody == "" || len(m.PreviewLinesCache) == 0 {
		t.Fatal("expected rebuilt preview for new selection")
	}

	m.invalidatePreviewLinesCache()
	if m.PreviewLinesCache != nil || m.PreviewWrapWidth != 0 {
		t.Fatal("invalidate preview lines failed")
	}
	if m.PreviewBody == "" || m.PreviewDocID != "products/2" {
		t.Fatal("body/id should survive line-cache invalidate")
	}
}

func TestSearchPreviewCacheKeyChanges(t *testing.T) {
	m, _ := testModel(t)
	m.Width = 140
	m.Height = 40
	hits := []types.Document{
		{Index: "logs", ID: "a", Raw: `{"level":"info","msg":"one"}`},
		{Index: "logs", ID: "b", Raw: `{"level":"error","msg":"two"}`},
	}
	m.SearchResult = &types.SearchResult{Hits: hits, Total: 2}
	m.SelectedDocIdx = 0
	m.refreshSearchPreviewFromSelection()
	if m.SearchPreviewHitID != "logs/a" {
		t.Fatalf("hit id=%q", m.SearchPreviewHitID)
	}
	if m.SearchPreviewBody == "" || len(m.SearchPreviewLinesCache) == 0 {
		t.Fatal("expected search preview body+lines")
	}
	bodyA := m.SearchPreviewBody

	m.SelectedDocIdx = 1
	m.refreshSearchPreviewFromSelection()
	if m.SearchPreviewHitID != "logs/b" {
		t.Fatalf("hit id=%q after nav", m.SearchPreviewHitID)
	}
	if m.SearchPreviewBody == bodyA {
		t.Fatal("body should change with selection")
	}
	if len(m.SearchPreviewLinesCache) == 0 {
		t.Fatal("expected lines after nav")
	}

	m.clearSearchPreviewCache()
	if m.SearchPreviewHitID != "" || m.SearchPreviewBody != "" || m.SearchPreviewLinesCache != nil {
		t.Fatal("clear failed")
	}
}

func TestSearchPreviewUsesBoundJSONBody(t *testing.T) {
	m, _ := testModel(t)
	m.Width = 120
	m.Height = 30
	big := `{"data":"` + strings.Repeat("x", maxJSONPrettyBytes) + `"}`
	doc := types.Document{Index: "big", ID: "1", Raw: big}
	m.setSearchPreviewBody(doc)
	if !m.SearchPreviewTrunc {
		t.Fatal("expected truncation flag")
	}
	if len(m.SearchPreviewBody) > maxJSONPrettyBytes+80 {
		t.Fatalf("body still huge: %d", len(m.SearchPreviewBody))
	}
	if !strings.Contains(m.SearchPreviewBody, "truncated") {
		t.Fatal("missing truncation marker")
	}
}

func TestInvalidateJSONPaintCaches(t *testing.T) {
	m, _ := testModel(t)
	m.Width = 120
	m.Height = 40
	doc := types.Document{Index: "p", ID: "1", Raw: `{"x":1}`}
	m.setDetailBody(doc)
	m.CurrentDocument = &doc
	m.syncDocumentDetailScroll()
	m.Documents = []types.Document{doc}
	m.SelectedDocIdx = 0
	m.refreshDocPreviewFromSelection()
	m.SearchResult = &types.SearchResult{Hits: []types.Document{doc}, Total: 1}
	m.refreshSearchPreviewFromSelection()

	if m.DetailColoredCache == nil || m.PreviewLinesCache == nil || m.SearchPreviewLinesCache == nil {
		t.Fatal("setup failed")
	}
	m.invalidateJSONPaintCaches()
	if m.DetailLinesCache != nil || m.DetailColoredCache != nil {
		t.Fatal("detail not cleared")
	}
	if m.PreviewLinesCache != nil || m.PreviewWrapWidth != 0 {
		t.Fatal("preview lines not cleared")
	}
	if m.SearchPreviewLinesCache != nil || m.SearchPreviewWrapWidth != 0 {
		t.Fatal("search lines not cleared")
	}
	if m.DetailBody == "" || m.PreviewBody == "" || m.SearchPreviewBody == "" {
		t.Fatal("bodies should remain after wrap invalidate")
	}
}

func TestColorizeJSONLinesEmpty(t *testing.T) {
	out := colorizeJSONLines([]string{"", "  ", `  "k": 1`})
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0] != "" || out[1] != "  " {
		t.Fatal("empty/blank lines stay plain")
	}
	if out[2] == `  "k": 1` {
		t.Fatal("expected color on JSON line")
	}
}
