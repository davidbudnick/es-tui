// Package service provides interfaces for dependency injection and testability.
package service

import (
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

// ConfigService defines the interface for configuration management.
type ConfigService interface {
	ListConnections() ([]types.Connection, error)
	AddConnection(conn types.Connection) (types.Connection, error)
	UpdateConnection(conn types.Connection) (types.Connection, error)
	DeleteConnection(id int64) error

	AddFavorite(connID int64, index, label string) (types.Favorite, error)
	RemoveFavorite(connID int64, index string) error
	ListFavorites(connID int64) []types.Favorite
	IsFavorite(connID int64, index string) bool

	AddRecentIndex(connID int64, index string)
	ListRecentIndices(connID int64) []types.RecentIndex
	ClearRecentIndices(connID int64)

	AddValueHistory(index, docID, value, action string)
	GetValueHistory(index, docID string) []types.ValueHistoryEntry
	ClearValueHistory()

	ListGroups() []types.ConnectionGroup
	AddGroup(name, color string) error
	AddConnectionToGroup(groupName string, connID int64) error
	RemoveConnectionFromGroup(groupName string, connID int64) error

	GetKeyBindings() types.KeyBindings
	SetKeyBindings(kb types.KeyBindings) error
	ResetKeyBindings() error

	Close() error
}

// ESService defines the interface for Elasticsearch/OpenSearch operations.
type ESService interface {
	Connect(conn types.Connection) error
	Disconnect() error
	TestConnection(conn types.Connection) (time.Duration, types.ClusterInfo, error)
	IsConnected() bool
	Flavor() types.Flavor

	// Cluster
	GetClusterInfo() (types.ClusterInfo, error)
	GetClusterHealth() (types.ClusterHealth, error)
	GetNodes() ([]types.NodeInfo, error)
	GetShards(index string) ([]types.ShardInfo, error)
	GetLiveMetrics() (types.LiveMetricsData, error)
	Cat(endpoint string) (string, error)

	// Indices
	ListIndices(pattern string) ([]types.IndexInfo, error)
	GetIndex(name string) (types.IndexInfo, error)
	CreateIndex(name string, body string) error
	DeleteIndex(name string) error
	GetIndexSettings(name string) (string, error)
	GetIndexMappings(name string) (string, error)
	RefreshIndex(name string) error
	OpenIndex(name string) error
	CloseIndex(name string) error

	// Documents
	Search(index, query string, from, size int) (types.SearchResult, error)
	GetDocument(index, id string) (types.Document, error)
	IndexDocument(index, id, body string) error
	DeleteDocument(index, id string) error
	DeleteByQuery(index, query string) (int64, error)

	// Aliases & templates
	ListAliases() ([]types.AliasInfo, error)
	ListTemplates() ([]types.IndexTemplate, error)
}
