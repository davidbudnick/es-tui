package ui

import (
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

	SearchQuery  string
	SearchResult *types.SearchResult
	SearchIndex  string

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
}

// NewModel creates a default model.
func NewModel() Model {
	return Model{
		Screen:      types.ScreenConnections,
		Connections: []types.Connection{},
		ConnInputs:  createConnectionInputs(),
		KeyBindings: types.DefaultKeyBindings(),
		Indices:     []types.IndexInfo{},
		Documents:   []types.Document{},
		Inputs: &ModelInputs{
			PatternInput:    createTextInput("Filter indices (e.g. logs-*)...", 40),
			SearchInput:     createTextInput("Query string or JSON body...", 60),
			IndexNameInput:  createTextInput("Index name", 40),
			IndexBodyInput:  createTextInput("Optional settings JSON {}", 50),
			DocBodyInput:    createTextInput("Document JSON body", 60),
			DocIDInput:      createTextInput("Document ID (optional)", 40),
			BulkDeleteInput: createTextInput("Delete-by-query (query_string or JSON)", 50),
			CatInput:        createTextInput("Cat endpoint (e.g. indices, shards, nodes)", 40),
		},
	}
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
		"Flavor (auto|elasticsearch|opensearch)",
	}
	inputs := make([]textinput.Model, len(labels))
	for i, label := range labels {
		ti := textinput.New()
		ti.Placeholder = label
		ti.CharLimit = 512
		ti.SetWidth(40)
		if i == 2 {
			ti.SetValue("9200")
		}
		if i == 6 {
			ti.SetValue("auto")
		}
		if i == 4 || i == 5 {
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
