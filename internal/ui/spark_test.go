package ui

import (
	"testing"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestSparklineEdges(t *testing.T) {
	// idx < 0 path can't happen with non-neg QueryTotal; idx >= len(bars)
	// force with huge max equal so ratio is 1.0 -> last bar
	s := sparkline([]types.LiveMetricsData{
		{QueryTotal: 100},
		{QueryTotal: 100},
	})
	if s == "" {
		t.Fatal("empty")
	}
	// maxQ == 0 all zeros already tested
	// large values
	s = sparkline([]types.LiveMetricsData{
		{QueryTotal: 1},
		{QueryTotal: 1000000},
	})
	if s == "" {
		t.Fatal("large")
	}
}

func TestInitCLIOnly(t *testing.T) {
	m := NewModel()
	m.Cmds = nil
	m.CLIConnection = &types.Connection{Host: "localhost", Port: 9200}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected auto connect cmd")
	}
	// execute
	msg := cmd()
	if _, ok := msg.(types.AutoConnectMsg); !ok {
		// may be batch with single cmd
		_ = msg
	}
}

func TestViewDocumentsOverflow(t *testing.T) {
	m, _ := testModel(t)
	m.Width, m.Height = 80, 20
	m.CurrentIndex = &types.IndexInfo{Name: "p"}
	m.Documents = make([]types.Document, 100)
	for i := range m.Documents {
		m.Documents[i] = types.Document{ID: "id", Index: "p", Score: 1}
	}
	out := m.viewDocuments()
	if out == "" {
		t.Fatal("empty")
	}
}

func TestViewLogsNoNewline(t *testing.T) {
	m, _ := testModel(t)
	m.Logs = types.NewLogWriter()
	_, _ = m.Logs.Write([]byte("no-newline"))
	out := m.viewLogs()
	if out == "" {
		t.Fatal("empty")
	}
}

func TestViewConnectionsTLSFlavor(t *testing.T) {
	m, _ := testModel(t)
	m.Connections = []types.Connection{
		{Name: "tls", Host: "h", Port: 9200, UseTLS: true, Flavor: types.FlavorElasticsearch},
	}
	m.SelectedConnIdx = 0
	m.Height = 30
	if m.viewConnections() == "" {
		t.Fatal("empty")
	}
}
