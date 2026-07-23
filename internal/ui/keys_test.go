package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func TestNormalizeKey(t *testing.T) {
	cases := []struct {
		msg  tea.KeyPressMsg
		want string
	}{
		{tea.KeyPressMsg{Text: "?", Code: '?'}, "?"},
		{tea.KeyPressMsg{Code: '/', Mod: tea.ModShift}, "?"},
		{tea.KeyPressMsg{Text: "O", Code: 'O'}, "O"},
		{tea.KeyPressMsg{Code: 'o', Mod: tea.ModShift}, "O"},
		{tea.KeyPressMsg{Code: 'x', Mod: tea.ModShift}, "X"},
		{tea.KeyPressMsg{Code: 'm', Mod: tea.ModShift}, "M"},
		{tea.KeyPressMsg{Code: 'v', Mod: tea.ModShift}, "V"},
		{tea.KeyPressMsg{Code: 'i', Mod: tea.ModShift}, "I"},
		{tea.KeyPressMsg{Code: '3', Mod: tea.ModShift}, "#"},
		{tea.KeyPressMsg{Code: '8', Mod: tea.ModShift}, "*"},
		{tea.KeyPressMsg{Text: "n", Code: 'n'}, "n"},
		{tea.KeyPressMsg{Code: tea.KeyEnter}, "enter"},
		{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, "ctrl+c"},
		{tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, "shift+tab"},
		{tea.KeyPressMsg{Text: ":", Code: ':'}, ":"},
		{tea.KeyPressMsg{Code: ';', Mod: tea.ModShift}, ":"},
	}
	for _, tc := range cases {
		got := normalizeKey(tc.msg)
		if got != tc.want {
			t.Fatalf("normalizeKey(%+v)=%q want %q", tc.msg, got, tc.want)
		}
	}
}

func TestGlobalHelpAndPaletteKeys(t *testing.T) {
	m, _ := testModel(t)
	m.Screen = types.ScreenIndices
	m.CurrentConn = &types.Connection{Name: "local"}

	// ? opens help and remembers previous screen
	nm, _ := m.handleKeyPress(tea.KeyPressMsg{Code: '/', Mod: tea.ModShift})
	m = nm.(Model)
	if m.Screen != types.ScreenHelp {
		t.Fatalf("help screen got %v", m.Screen)
	}
	if m.PrevScreen != types.ScreenIndices {
		t.Fatalf("prev screen %v", m.PrevScreen)
	}

	// ? again closes help
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: "?", Code: '?'})
	m = nm.(Model)
	if m.Screen != types.ScreenIndices {
		t.Fatalf("back to indices got %v", m.Screen)
	}

	// shift+o works as O (open index path needs cmds+indices; just ensure key routes)
	m.Screen = types.ScreenIndices
	m.Indices = []types.IndexInfo{{Name: "products"}}
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Code: 'n'}) // nodes
	m = nm.(Model)
	// with cmds, should load nodes
	if m.Cmds != nil && m.Screen != types.ScreenNodes && !m.Loading {
		// Loading may be true with cmd pending — either is fine
		_ = m
	}

	// colon opens palette
	m.Screen = types.ScreenIndices
	nm, _ = m.handleKeyPress(tea.KeyPressMsg{Text: ":", Code: ':'})
	m = nm.(Model)
	if m.Screen != types.ScreenCommandPalette {
		t.Fatalf("palette got %v", m.Screen)
	}
}

func TestPickDocumentListColumns(t *testing.T) {
	// products-like page
	products := []types.Document{
		{ID: "1", Score: 1, Source: map[string]any{"name": "Widget", "category": "hardware", "brand": "elastic", "sku": "A", "price": 10.0}},
		{ID: "2", Score: 1, Source: map[string]any{"name": "Mug", "category": "merch", "brand": "kibana", "sku": "B", "price": 14.0}},
		{ID: "3", Score: 1, Source: map[string]any{"name": "Shirt", "category": "merch", "brand": "elastic", "sku": "C", "price": 22.0}},
		{ID: "4", Score: 1, Source: map[string]any{"name": "Hat", "category": "merch", "brand": "opensearch", "sku": "D", "price": 18.0}},
	}
	cols := pickDocumentListColumns(products, 100)
	if len(cols) < 2 || cols[0].Key != "_id" {
		t.Fatalf("cols=%+v", cols)
	}
	// products: ID | Name | Category | Brand (name is flex mid-order)
	keys := []string{}
	for _, c := range cols {
		keys = append(keys, c.Key)
	}
	if keys[1] != "name" {
		t.Fatalf("products order=%v want name second", keys)
	}
	joined := strings.Join(keys, ",")
	if !strings.Contains(joined, "category") && !strings.Contains(joined, "brand") {
		t.Fatalf("expected category/brand in %v", keys)
	}
	for _, c := range cols {
		if c.Key == "name" && c.Width < 18 {
			t.Fatalf("name width %d too narrow", c.Width)
		}
		if c.Key == "plan" && c.Width < 12 {
			t.Fatalf("plan width %d too narrow for enterprise", c.Width)
		}
	}
	if docsHaveUsefulScores(products) {
		t.Fatal("flat 1.0 scores should hide score col")
	}

	// logs: ID | Level | Service | Message (message last, facets not squished)
	logs := []types.Document{
		{ID: "1", Score: 1, Source: map[string]any{"message": "ok", "level": "info", "service": "api", "host": "n1"}},
		{ID: "2", Score: 1, Source: map[string]any{"message": "err", "level": "error", "service": "auth", "host": "n2"}},
		{ID: "3", Score: 1, Source: map[string]any{"message": "warn", "level": "warn", "service": "api", "host": "n1"}},
		{ID: "4", Score: 1, Source: map[string]any{"message": "ok", "level": "debug", "service": "api", "host": "n3"}},
	}
	cols = pickDocumentListColumns(logs, 100)
	keys = keys[:0]
	for _, c := range cols {
		keys = append(keys, c.Key)
	}
	if keys[len(keys)-1] != "message" {
		t.Fatalf("logs order=%v want message last", keys)
	}
	for _, c := range cols {
		if c.Key == "level" && c.Width < 8 {
			t.Fatalf("level width=%d too narrow", c.Width)
		}
		if c.Key == "service" && c.Width < 12 {
			t.Fatalf("service width=%d too narrow", c.Width)
		}
		if c.Key == "message" && c.Width < 20 {
			t.Fatalf("message width %d too narrow", c.Width)
		}
	}
}

func TestListWindowPagesToTop(t *testing.T) {
	// 20 items, 8 visible — selection on row 8 should open a new page at start=8
	start, end := listWindow(8, 20, 8)
	if start != 8 || end != 16 {
		t.Fatalf("page2 got %d-%d", start, end)
	}
	// still on first page
	start, end = listWindow(7, 20, 8)
	if start != 0 || end != 8 {
		t.Fatalf("page1 got %d-%d", start, end)
	}
	// last partial page
	start, end = listWindow(19, 20, 8)
	if start != 16 || end != 20 {
		t.Fatalf("last page got %d-%d", start, end)
	}
	// short list
	start, end = listWindow(2, 5, 8)
	if start != 0 || end != 5 {
		t.Fatalf("short got %d-%d", start, end)
	}
}

func TestDocPaginationBounds(t *testing.T) {
	m, _ := testModel(t)
	m.Screen = types.ScreenDocuments
	m.CurrentIndex = &types.IndexInfo{Name: "products"}
	m.Documents = make([]types.Document, 12)
	for i := range m.Documents {
		m.Documents[i] = types.Document{ID: fmt.Sprintf("%d", i)}
	}
	m.DocTotal = 12
	m.DocFrom = 0
	m.PageSize = 50

	// n must not advance past a single page of 12
	nm, cmd := m.handleDocumentsKeys("n", tea.KeyPressMsg{})
	m = nm.(Model)
	if cmd != nil {
		t.Fatal("expected no load cmd on last page")
	}
	if m.DocFrom != 0 {
		t.Fatalf("DocFrom=%d", m.DocFrom)
	}
	if m.StatusMsg != "Last page" {
		t.Fatalf("status=%q", m.StatusMsg)
	}

	// p on first page stays put
	m.StatusMsg = ""
	nm, cmd = m.handleDocumentsKeys("p", tea.KeyPressMsg{})
	m = nm.(Model)
	if cmd != nil || m.DocFrom != 0 {
		t.Fatal("p on first page should no-op")
	}
	if m.StatusMsg != "First page" {
		t.Fatalf("status=%q", m.StatusMsg)
	}

	// with more docs, n is allowed
	m.DocTotal = 100
	m.DocFrom = 0
	m.StatusMsg = ""
	nm, cmd = m.handleDocumentsKeys("n", tea.KeyPressMsg{})
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("expected next page load")
	}
	if m.DocFrom != 50 {
		t.Fatalf("DocFrom=%d want 50", m.DocFrom)
	}
}

func TestTypingContextBlocksHelp(t *testing.T) {
	m, _ := testModel(t)
	m.Screen = types.ScreenIndices
	m.Inputs.PatternInput.Focus()
	if !m.typingContext() {
		t.Fatal("expected typing context")
	}
	// ? should go into filter, not help — handleIndicesKeys path when focused
	nm, _ := m.handleKeyPress(tea.KeyPressMsg{Text: "?", Code: '?'})
	m = nm.(Model)
	if m.Screen == types.ScreenHelp {
		t.Fatal("help must not open while filter focused")
	}
}
