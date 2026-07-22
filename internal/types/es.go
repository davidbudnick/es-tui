package types

import "time"

// IndexInfo holds metadata about an Elasticsearch/OpenSearch index.
type IndexInfo struct {
	Name          string
	Health        string
	Status        string
	UUID          string
	PrimaryShards int
	ReplicaShards int
	DocsCount     int64
	DocsDeleted   int64
	StoreSize     string
	PriStoreSize  string
	IsFavorite    bool
}

// Document represents a search hit / document.
type Document struct {
	Index  string
	ID     string
	Score  float64
	Source map[string]any
	Raw    string
}

// ClusterHealth holds cluster health information.
type ClusterHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

// ClusterInfo holds / cluster-level version and name info.
type ClusterInfo struct {
	Name        string
	ClusterName string
	ClusterUUID string
	Version     VersionInfo
	Tagline     string
	Flavor      Flavor
}

// VersionInfo holds version fields from the root API.
type VersionInfo struct {
	Number                           string
	BuildFlavor                      string
	BuildType                        string
	BuildHash                        string
	BuildDate                        string
	BuildSnapshot                    bool
	LuceneVersion                    string
	MinimumWireCompatibilityVersion  string
	MinimumIndexCompatibilityVersion string
	Distribution                     string // OpenSearch
}

// NodeInfo holds information about a cluster node.
type NodeInfo struct {
	Name            string
	ID              string
	IP              string
	Host            string
	Roles           []string
	Version         string
	HeapPercent     int
	RamPercent      int
	CPU             int
	Load1m          string
	Master          string
	NodeRole        string
	DiskUsedPercent string
	DiskTotal       string
	DiskUsed        string
	DiskAvail       string
}

// ShardInfo holds cat shards row data.
type ShardInfo struct {
	Index  string
	Shard  string
	Prirep string
	State  string
	Docs   string
	Store  string
	IP     string
	Node   string
}

// AliasInfo holds index alias information.
type AliasInfo struct {
	Alias         string
	Index         string
	Filter        string
	RoutingIndex  string
	RoutingSearch string
	IsWriteIndex  string
}

// IndexTemplate holds index template metadata.
type IndexTemplate struct {
	Name          string
	IndexPatterns []string
	Order         int
	Version       int
	ComposedOf    []string
}

// SearchResult holds a search response summary.
type SearchResult struct {
	Took         int
	TimedOut     bool
	Total        int64
	TotalRel     string
	MaxScore     float64
	Hits         []Document
	Aggregations map[string]any
	Raw          string
}

// LiveMetricsData holds real-time cluster metrics.
type LiveMetricsData struct {
	Timestamp        time.Time
	Status           string
	Nodes            int
	DataNodes        int
	ActiveShards     int
	UnassignedShards int
	DocsCount        int64
	StoreSizeBytes   int64
	QueryTotal       int64
	IndexingTotal    int64
	SearchLatencyMs  float64
	JVMHeapUsedPct   float64
	CPUPercent       float64
}

// LiveMetrics accumulates metric history for charts.
type LiveMetrics struct {
	History []LiveMetricsData
	Latest  LiveMetricsData
}

// ValueHistoryEntry stores a previous document value for undo.
type ValueHistoryEntry struct {
	Index     string
	DocID     string
	Value     string
	Timestamp time.Time
	Action    string
}

// AllocationInfo holds cat allocation row data.
type AllocationInfo struct {
	Shards      string
	DiskIndices string
	DiskUsed    string
	DiskAvail   string
	DiskTotal   string
	DiskPercent string
	Host        string
	IP          string
	Node        string
}

// TaskInfo holds cluster task metadata.
type TaskInfo struct {
	ID          string
	Action      string
	Type        string
	StartTime   string
	RunningTime string
	Cancellable string
	Node        string
	Description string
}

// PluginInfo holds installed plugin metadata.
type PluginInfo struct {
	Name      string
	Component string
	Version   string
}

// DataStreamInfo holds data stream metadata.
type DataStreamInfo struct {
	Name           string
	TimestampField string
	IndicesCount   string
	Generation     string
	Status         string
	Template       string
}

// SnapshotInfo holds snapshot metadata.
type SnapshotInfo struct {
	Snapshot   string
	Repository string
	State      string
	StartTime  string
	EndTime    string
	Indices    string
}

// SavedQuery stores a named search query.
type SavedQuery struct {
	Name    string    `json:"name"`
	Index   string    `json:"index"`
	Query   string    `json:"query"`
	Created time.Time `json:"created"`
}

// ExplainResult holds a query explain response.
type ExplainResult struct {
	Matched     bool
	Explanation string
	Raw         string
}
