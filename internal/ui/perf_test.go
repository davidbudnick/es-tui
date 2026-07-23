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
