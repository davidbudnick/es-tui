package types

// Screen represents the current view in the application.
type Screen int

const (
	ScreenConnections Screen = iota
	ScreenAddConnection
	ScreenEditConnection
	ScreenIndices
	ScreenIndexDetail
	ScreenDocuments
	ScreenDocumentDetail
	ScreenSearch
	ScreenHelp
	ScreenConfirmDelete
	ScreenClusterHealth
	ScreenNodes
	ScreenIndexCreate
	ScreenIndexSettings
	ScreenIndexMappings
	ScreenAliases
	ScreenShards
	ScreenLiveMetrics
	ScreenTestConnection
	ScreenLogs
	ScreenFavorites
	ScreenRecentIndices
	ScreenBulkDelete
	ScreenEditDocument
	ScreenIndexTemplates
	ScreenCatAPI
	ScreenReindex
	ScreenAllocation
	ScreenTasks
	ScreenPlugins
	ScreenClusterSettings
	ScreenDataStreams
	ScreenSnapshots
	ScreenSavedQueries
	ScreenCommandPalette
	ScreenExport
	ScreenExplain
)

// String returns a human-readable name for the screen.
func (s Screen) String() string {
	names := map[Screen]string{
		ScreenConnections:     "Connections",
		ScreenAddConnection:   "Add Connection",
		ScreenEditConnection:  "Edit Connection",
		ScreenIndices:         "Indices",
		ScreenIndexDetail:     "Index Detail",
		ScreenDocuments:       "Documents",
		ScreenDocumentDetail:  "Document Detail",
		ScreenSearch:          "Search",
		ScreenHelp:            "Help",
		ScreenConfirmDelete:   "Confirm Delete",
		ScreenClusterHealth:   "Cluster Health",
		ScreenNodes:           "Nodes",
		ScreenIndexCreate:     "Create Index",
		ScreenIndexSettings:   "Index Settings",
		ScreenIndexMappings:   "Index Mappings",
		ScreenAliases:         "Aliases",
		ScreenShards:          "Shards",
		ScreenLiveMetrics:     "Live Metrics",
		ScreenTestConnection:  "Test Connection",
		ScreenLogs:            "Logs",
		ScreenFavorites:       "Favorites",
		ScreenRecentIndices:   "Recent Indices",
		ScreenBulkDelete:      "Bulk Delete",
		ScreenEditDocument:    "Edit Document",
		ScreenIndexTemplates:  "Index Templates",
		ScreenCatAPI:          "Cat API",
		ScreenReindex:         "Reindex",
		ScreenAllocation:      "Allocation",
		ScreenTasks:           "Tasks",
		ScreenPlugins:         "Plugins",
		ScreenClusterSettings: "Cluster Settings",
		ScreenDataStreams:     "Data Streams",
		ScreenSnapshots:       "Snapshots",
		ScreenSavedQueries:    "Saved Queries",
		ScreenCommandPalette:  "Command Palette",
		ScreenExport:          "Export",
		ScreenExplain:         "Explain",
	}
	if name, ok := names[s]; ok {
		return name
	}
	return "Unknown"
}
