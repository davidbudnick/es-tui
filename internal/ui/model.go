package ui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"github.com/davidbudnick/es-tui/internal/cmd"
	"github.com/davidbudnick/es-tui/internal/types"
	"github.com/davidbudnick/es-tui/internal/ui/editor"

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
	ConnFlavorIdx   int  // index into connFlavorOptions
	ConnFlavorOpen  bool // flavor dropdown expanded
	ConnReadOnly    bool
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
	DetailScroll    int // viewport top line in JSON/detail body
	DetailCursor    int // selected line in document detail (blue band)
	PageSize        int
	// DocEditor is the redis-style multiline body editor (ScreenEditDocument).
	DocEditor *editor.Model
	// DocEditFocus is "id" or "body" while editing a document.
	DocEditFocus string
	// PrevScreen is restored when leaving Help.
	PrevScreen types.Screen

	// Cached renders (redis-tui): pretty-print once on load, not every View().
	DetailBody       string   // bounded pretty JSON for CurrentDocument
	DetailTruncated  bool     // body cut at maxJSONPrettyBytes
	DetailLinesCache []string // wrapped lines for DetailWrapWidth
	DetailWrapWidth  int      // content width used for DetailLinesCache
	PreviewDocID     string   // index/id of cached list preview
	PreviewBody      string   // bounded pretty JSON for list preview
	PreviewTruncated bool

	SearchQuery  string
	SearchResult *types.SearchResult
	SearchIndex  string
	SearchFrom   int
	SearchFocus  string // "query" | "results"
	// SearchArea is the multiline query editor on the search screen.
	SearchArea   *textarea.Model
	QueryHistory []string
	HistoryIdx   int

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
	ID    string
	Label string
	Keys  string
	Group string
}

// ModelInputs holds text inputs behind a pointer to keep Model small.
type ModelInputs struct {
	PatternInput    textinput.Model
	SearchInput     textinput.Model
	IndexNameInput  textinput.Model
	IndexBodyInput  textinput.Model
	DocBodyInput    textinput.Model
	DocIDInput      textinput.Model
	BulkDeleteInput textinput.Model
	CatInput        textinput.Model
	ReindexSrcInput textinput.Model
	ReindexDstInput textinput.Model
	ExportInput     textinput.Model
	PaletteInput    textinput.Model
	SavedQueryName  textinput.Model
	SnapshotRepo    textinput.Model
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
		{ID: "health", Label: "Cluster health", Keys: "c", Group: "Cluster"},
		{ID: "nodes", Label: "Nodes", Keys: "n", Group: "Cluster"},
		{ID: "metrics", Label: "Live metrics", Keys: "m", Group: "Cluster"},
		{ID: "shards", Label: "Shards", Keys: "s", Group: "Cluster"},
		{ID: "allocation", Label: "Disk allocation", Keys: "", Group: "Cluster"},
		{ID: "settings", Label: "Cluster settings", Keys: "", Group: "Cluster"},
		{ID: "tasks", Label: "Tasks", Keys: "", Group: "Cluster"},
		{ID: "plugins", Label: "Plugins", Keys: "", Group: "Cluster"},
		{ID: "aliases", Label: "Aliases", Keys: "A", Group: "Indices"},
		{ID: "templates", Label: "Index templates", Keys: "T", Group: "Indices"},
		{ID: "datastreams", Label: "Data streams", Keys: "", Group: "Indices"},
		{ID: "snapshots", Label: "Snapshots", Keys: "", Group: "Indices"},
		{ID: "favorites", Label: "Favorites", Keys: "F", Group: "Indices"},
		{ID: "recent", Label: "Recent indices", Keys: "R", Group: "Indices"},
		{ID: "search", Label: "Search", Keys: "/", Group: "Tools"},
		{ID: "reindex", Label: "Reindex", Keys: "", Group: "Tools"},
		{ID: "export", Label: "Export documents", Keys: "", Group: "Tools"},
		{ID: "saved", Label: "Saved queries", Keys: "", Group: "Tools"},
		{ID: "cat", Label: "Cat API", Keys: "C", Group: "Tools"},
		{ID: "logs", Label: "App logs", Keys: "L", Group: "Tools"},
		{ID: "help", Label: "Help", Keys: "?", Group: "Tools"},
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

// Connection form field indices (text inputs 0–6, then selectors).
const (
	connFieldName = iota
	connFieldHost
	connFieldPort
	connFieldUser
	connFieldPass
	connFieldAPIKey
	connFieldBearer
	connFieldFlavor
	connFieldReadOnly
	connFieldCount
)

const connTextCount = 7 // name…bearer

var connFlavorOptions = []types.Flavor{
	types.FlavorAuto,
	types.FlavorElasticsearch,
	types.FlavorOpenSearch,
}

var connTextLabels = []string{
	"Name",
	"Host",
	"Port",
	"Username",
	"Password",
	"API Key",
	"Bearer Token",
}

func createConnectionInputs() []textinput.Model {
	placeholders := []string{
		"my-cluster",
		"localhost",
		"9200",
		"optional",
		"optional",
		"optional",
		"optional",
	}
	inputs := make([]textinput.Model, connTextCount)
	for i, ph := range placeholders {
		ti := textinput.New()
		ti.Placeholder = ph
		ti.CharLimit = 2048
		ti.SetWidth(42)
		if i == connFieldPort {
			ti.SetValue("9200")
		}
		if i == connFieldHost {
			ti.SetValue("localhost")
		}
		if i == connFieldPass || i == connFieldAPIKey || i == connFieldBearer {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		inputs[i] = ti
	}
	return inputs
}

func flavorIndex(f types.Flavor) int {
	for i, opt := range connFlavorOptions {
		if opt == f || (f == "" && opt == types.FlavorAuto) {
			return i
		}
	}
	return 0
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
	// Background release check (no-ops for dev builds).
	cmds = append(cmds, cmd.CheckForUpdate(m.Version))
	return tea.Batch(cmds...)
}
