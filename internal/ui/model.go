package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	"github.com/davidbudnick/es-tui/internal/cmd"
	"github.com/davidbudnick/es-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

// Model is the Bubble Tea application model.
type Model struct {
	Cmds            *cmd.Commands
	Version         string
	Screen          types.Screen
	Connections     []types.Connection
	SelectedConnIdx int
	ConnInputs      []textinput.Model
	ConnFocusIdx    int
	EditingConn     *types.Connection
	CurrentConn     *types.Connection
	ClusterInfo     types.ClusterInfo
	Flavor          types.Flavor

	Indices          []types.IndexInfo
	SelectedIndexIdx int
	IndexPattern     string
	CurrentIndex     *types.IndexInfo
	IndexSettings    string
	IndexMappings    string

	Documents       []types.Document
	SelectedDocIdx  int
	CurrentDocument *types.Document
	DocQuery        string
	DocFrom         int
	DocTotal        int64
	DetailScroll    int
	PageSize        int

	SearchQuery    string
	SearchResult   *types.SearchResult
	SearchIndex    string
	SearchFrom     int
	SearchFocus    string // "query" | "results"
	QueryHistory   []string
	HistoryIdx     int


	ClusterHealth types.ClusterHealth
	Nodes         []types.NodeInfo
	SelectedNode  int
	Shards        []types.ShardInfo
	Aliases       []types.AliasInfo
	Templates     []types.IndexTemplate
	CatResult     string
	CatEndpoint   string

	Favorites         []types.Favorite
	RecentIndices     []types.RecentIndex
	SelectedFavIdx    int
	SelectedRecentIdx int

	LiveMetrics       *types.LiveMetrics
	LiveMetricsActive bool

	Allocation      []types.AllocationInfo
	Tasks           []types.TaskInfo
	SelectedTaskIdx int
	Plugins         []types.PluginInfo
	DataStreams     []types.DataStreamInfo
	Snapshots       []types.SnapshotInfo
	ClusterSettings string
	ExplainResult   *types.ExplainResult
	CountResult     int64
	SavedQueries    []types.SavedQuery
	SelectedSQIdx   int
	ReadOnly        bool

	// Command palette
	PaletteFilter string
	PaletteIdx    int
	PaletteItems  []PaletteItem

	// Reindex / export forms
	ReindexFocus int
	ExportPath   string

	Width           int
	Height          int
	Err             error
	StatusMsg       string
	Loading         bool
	ConfirmType     string
	ConfirmData     any
	Logs            *types.LogWriter
	SendFunc        *func(tea.Msg)
	ConnectionError string
	TestConnResult  string
	CLIConnection   *types.Connection
	UpdateAvailable string
	UpdateCmd       string
	KeyBindings     types.KeyBindings
	LogCursor       int

	Inputs            *ModelInputs
	inputsInitialized bool
}

// PaletteItem is one command-palette action.
type PaletteItem struct {
	ID   string
	Label string
	Keys string
}

// ModelInputs holds text inputs behind a pointer to keep Model small.
type ModelInputs struct {
	PatternInput     textinput.Model
	SearchInput      textinput.Model
	IndexNameInput   textinput.Model
	IndexBodyInput   textinput.Model
	DocBodyInput     textinput.Model
	DocIDInput       textinput.Model
	BulkDeleteInput  textinput.Model
	CatInput         textinput.Model
	ReindexSrcInput  textinput.Model
	ReindexDstInput  textinput.Model
	ExportInput      textinput.Model
	PaletteInput     textinput.Model
	SavedQueryName   textinput.Model
	SnapshotRepo     textinput.Model
}

// NewModel creates a default model.
func NewModel() Model {
	return Model{
		Screen:       types.ScreenConnections,
		Connections:  []types.Connection{},
		ConnInputs:   createConnectionInputs(),
		KeyBindings:  types.DefaultKeyBindings(),
		Indices:      []types.IndexInfo{},
		Documents:    []types.Document{},
		PageSize:     50,
		SearchFocus:  "query",
		QueryHistory: []string{},
		HistoryIdx:   -1,
		Inputs: &ModelInputs{
			PatternInput:    createTextInput("Filter indices (e.g. logs-*)...", 40),
			SearchInput:     createTextInput("query_string or {\"query\":{...}} JSON", 80),
			IndexNameInput:  createTextInput("Index name", 40),
			IndexBodyInput:  createTextInput("Optional settings JSON {}", 50),
			DocBodyInput:    createTextInput("Document JSON body", 60),
			DocIDInput:      createTextInput("Document ID (optional)", 40),
			BulkDeleteInput: createTextInput("Delete-by-query (query_string or JSON)", 50),
			CatInput:        createTextInput("Cat endpoint (e.g. indices, shards, nodes)", 40),
			ReindexSrcInput: createTextInput("Source index", 40),
			ReindexDstInput: createTextInput("Dest index", 40),
			ExportInput:     createTextInput("Export path (e.g. /tmp/out.ndjson)", 50),
			PaletteInput:    createTextInput("Filter commands...", 40),
			SavedQueryName:  createTextInput("Saved query name", 30),
			SnapshotRepo:    createTextInput("Snapshot repository name", 40),
		},
	}
}

func defaultPaletteItems() []PaletteItem {
	return []PaletteItem{
		{ID: "health", Label: "Cluster health", Keys: "c"},
		{ID: "nodes", Label: "Nodes", Keys: "n"},
		{ID: "metrics", Label: "Live metrics", Keys: "m"},
		{ID: "shards", Label: "Shards", Keys: "s"},
		{ID: "allocation", Label: "Disk allocation", Keys: ""},
		{ID: "aliases", Label: "Aliases", Keys: "A"},
		{ID: "templates", Label: "Index templates", Keys: "T"},
		{ID: "datastreams", Label: "Data streams", Keys: ""},
		{ID: "tasks", Label: "Tasks", Keys: ""},
		{ID: "plugins", Label: "Plugins", Keys: ""},
		{ID: "settings", Label: "Cluster settings", Keys: ""},
		{ID: "snapshots", Label: "Snapshots", Keys: ""},
		{ID: "search", Label: "Search", Keys: "/"},
		{ID: "reindex", Label: "Reindex", Keys: ""},
		{ID: "export", Label: "Export documents", Keys: ""},
		{ID: "saved", Label: "Saved queries", Keys: ""},
		{ID: "cat", Label: "Cat API", Keys: "C"},
		{ID: "favorites", Label: "Favorites", Keys: "F"},
		{ID: "recent", Label: "Recent indices", Keys: "R"},
		{ID: "logs", Label: "App logs", Keys: "L"},
		{ID: "help", Label: "Help", Keys: "?"},
	}
}

func (m *Model) pushQueryHistory(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	// de-dupe head
	if len(m.QueryHistory) > 0 && m.QueryHistory[0] == q {
		return
	}
	m.QueryHistory = append([]string{q}, m.QueryHistory...)
	if len(m.QueryHistory) > 30 {
		m.QueryHistory = m.QueryHistory[:30]
	}
	m.HistoryIdx = -1
}

func createTextInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 4096
	ti.SetWidth(width)
	return ti
}

func createConnectionInputs() []textinput.Model {
	labels := []string{
		"Name",
		"Host",
		"Port",
		"Username",
		"Password",
		"API Key",
		"Bearer Token",
		"Flavor (auto|elasticsearch|opensearch)",
		"Read-only (true|false)",
	}
	inputs := make([]textinput.Model, len(labels))
	for i, label := range labels {
		ti := textinput.New()
		ti.Placeholder = label
		ti.CharLimit = 2048
		ti.SetWidth(40)
		if i == 2 {
			ti.SetValue("9200")
		}
		if i == 7 {
			ti.SetValue("auto")
		}
		if i == 8 {
			ti.SetValue("false")
		}
		if i == 4 || i == 5 || i == 6 {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		inputs[i] = ti
	}
	return inputs
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.Cmds != nil {
		cmds = append(cmds, m.Cmds.LoadConnections())
	}
	if m.CLIConnection != nil {
		conn := *m.CLIConnection
		cmds = append(cmds, func() tea.Msg {
			return types.AutoConnectMsg{Connection: conn}
		})
	}
	return tea.Batch(cmds...)
}
