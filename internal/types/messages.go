package types

import "time"

// Bubble Tea message types

type ConnectionsLoadedMsg struct {
	Connections []Connection
	Err         error
}

type ConnectionAddedMsg struct {
	Connection Connection
	Err        error
}

type ConnectionUpdatedMsg struct {
	Connection Connection
	Err        error
}

type ConnectionDeletedMsg struct {
	ID  int64
	Err error
}

type ConnectedMsg struct {
	Info ClusterInfo
	Err  error
}

// AutoConnectMsg triggers an automatic connection from CLI flags.
type AutoConnectMsg struct {
	Connection Connection
}

type DisconnectedMsg struct{}

type IndicesLoadedMsg struct {
	Indices []IndexInfo
	Err     error
}

type IndexDetailLoadedMsg struct {
	Index    IndexInfo
	Settings string
	Mappings string
	Err      error
}

type DocumentsLoadedMsg struct {
	Index     string
	Documents []Document
	Total     int64
	Err       error
}

type DocumentLoadedMsg struct {
	Document Document
	Err      error
}

type DocumentDeletedMsg struct {
	Index string
	ID    string
	Err   error
}

type DocumentSavedMsg struct {
	Index string
	ID    string
	Err   error
}

type IndexCreatedMsg struct {
	Name string
	Err  error
}

type IndexDeletedMsg struct {
	Name string
	Err  error
}

type SearchResultMsg struct {
	Result SearchResult
	Err    error
}

type ClusterHealthLoadedMsg struct {
	Health ClusterHealth
	Err    error
}

type NodesLoadedMsg struct {
	Nodes []NodeInfo
	Err   error
}

type ShardsLoadedMsg struct {
	Shards []ShardInfo
	Err    error
}

type AliasesLoadedMsg struct {
	Aliases []AliasInfo
	Err     error
}

type TemplatesLoadedMsg struct {
	Templates []IndexTemplate
	Err       error
}

type ConnectionTestMsg struct {
	Success bool
	Err     error
	Latency time.Duration
	Info    ClusterInfo
}

type FavoritesLoadedMsg struct {
	Favorites []Favorite
	Err       error
}

type FavoriteAddedMsg struct {
	Favorite Favorite
	Err      error
}

type FavoriteRemovedMsg struct {
	Index string
	Err   error
}

type RecentIndicesLoadedMsg struct {
	Indices []RecentIndex
	Err     error
}

type BulkDeleteMsg struct {
	Index   string
	Deleted int64
	Err     error
}

type TickMsg struct{}

type LiveMetricsMsg struct {
	Data LiveMetricsData
	Err  error
}

type LiveMetricsTickMsg struct{}

type ClipboardCopiedMsg struct {
	Content string
	Err     error
}

type UpdateAvailableMsg struct {
	LatestVersion string
	UpgradeCmd    string
	Err           error
}

type IndexSettingsLoadedMsg struct {
	Settings string
	Err      error
}

type IndexMappingsLoadedMsg struct {
	Mappings string
	Err      error
}

type CatAPIResultMsg struct {
	Endpoint string
	Body     string
	Err      error
}
