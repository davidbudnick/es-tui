// Package cmd contains Bubble Tea commands for the ES/OpenSearch TUI.
package cmd

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/davidbudnick/es-tui/internal/service"
	"github.com/davidbudnick/es-tui/internal/types"
)

// Commands wraps service dependencies and returns tea.Cmd factories.
type Commands struct {
	config service.ConfigService
	es     service.ESService
}

// NewCommands creates a Commands instance.
func NewCommands(config service.ConfigService, es service.ESService) *Commands {
	return &Commands{config: config, es: es}
}

// NewCommandsFromContainer creates Commands from a service container.
func NewCommandsFromContainer(c *service.Container) *Commands {
	return &Commands{config: c.Config, es: c.ES}
}

// Config returns the config service.
func (c *Commands) Config() service.ConfigService { return c.config }

// ES returns the ES service.
func (c *Commands) ES() service.ESService { return c.es }

// LoadConnections loads saved connections.
func (c *Commands) LoadConnections() tea.Cmd {
	return func() tea.Msg {
		conns, err := c.config.ListConnections()
		return types.ConnectionsLoadedMsg{Connections: conns, Err: err}
	}
}

// AddConnection adds a connection.
func (c *Commands) AddConnection(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		added, err := c.config.AddConnection(conn)
		return types.ConnectionAddedMsg{Connection: added, Err: err}
	}
}

// UpdateConnection updates a connection.
func (c *Commands) UpdateConnection(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		updated, err := c.config.UpdateConnection(conn)
		return types.ConnectionUpdatedMsg{Connection: updated, Err: err}
	}
}

// DeleteConnection deletes a connection.
func (c *Commands) DeleteConnection(id int64) tea.Cmd {
	return func() tea.Msg {
		err := c.config.DeleteConnection(id)
		return types.ConnectionDeletedMsg{ID: id, Err: err}
	}
}

// Connect connects to a cluster.
func (c *Commands) Connect(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		if err := c.es.Connect(conn); err != nil {
			return types.ConnectedMsg{Err: err}
		}
		info, err := c.es.GetClusterInfo()
		return types.ConnectedMsg{Info: info, Err: err}
	}
}

// Disconnect disconnects from the cluster.
func (c *Commands) Disconnect() tea.Cmd {
	return func() tea.Msg {
		_ = c.es.Disconnect()
		return types.DisconnectedMsg{}
	}
}

// TestConnection tests a connection.
func (c *Commands) TestConnection(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		latency, info, err := c.es.TestConnection(conn)
		return types.ConnectionTestMsg{
			Success: err == nil,
			Err:     err,
			Latency: latency,
			Info:    info,
		}
	}
}

// LoadIndices loads the index list.
func (c *Commands) LoadIndices(pattern string) tea.Cmd {
	return func() tea.Msg {
		indices, err := c.es.ListIndices(pattern)
		if err == nil && c.config != nil {
			// Mark favorites if we have a way — applied in UI with conn ID
		}
		return types.IndicesLoadedMsg{Indices: indices, Err: err}
	}
}

// LoadIndexDetail loads index detail, settings, and mappings.
func (c *Commands) LoadIndexDetail(name string) tea.Cmd {
	return func() tea.Msg {
		idx, err := c.es.GetIndex(name)
		if err != nil {
			return types.IndexDetailLoadedMsg{Err: err}
		}
		settings, _ := c.es.GetIndexSettings(name)
		mappings, _ := c.es.GetIndexMappings(name)
		return types.IndexDetailLoadedMsg{Index: idx, Settings: settings, Mappings: mappings}
	}
}

// CreateIndex creates an index.
func (c *Commands) CreateIndex(name, body string) tea.Cmd {
	return func() tea.Msg {
		err := c.es.CreateIndex(name, body)
		return types.IndexCreatedMsg{Name: name, Err: err}
	}
}

// DeleteIndex deletes an index.
func (c *Commands) DeleteIndex(name string) tea.Cmd {
	return func() tea.Msg {
		err := c.es.DeleteIndex(name)
		return types.IndexDeletedMsg{Name: name, Err: err}
	}
}

// LoadDocuments searches for documents in an index.
func (c *Commands) LoadDocuments(index, query string, from, size int) tea.Cmd {
	return func() tea.Msg {
		result, err := c.es.Search(index, query, from, size)
		if err != nil {
			return types.DocumentsLoadedMsg{Index: index, Err: err}
		}
		return types.DocumentsLoadedMsg{
			Index:     index,
			Documents: result.Hits,
			Total:     result.Total,
		}
	}
}

// LoadDocument loads a single document.
func (c *Commands) LoadDocument(index, id string) tea.Cmd {
	return func() tea.Msg {
		doc, err := c.es.GetDocument(index, id)
		return types.DocumentLoadedMsg{Document: doc, Err: err}
	}
}

// SaveDocument indexes a document.
func (c *Commands) SaveDocument(index, id, body string) tea.Cmd {
	return func() tea.Msg {
		if c.config != nil {
			c.config.AddValueHistory(index, id, body, "save")
		}
		err := c.es.IndexDocument(index, id, body)
		return types.DocumentSavedMsg{Index: index, ID: id, Err: err}
	}
}

// DeleteDocument deletes a document.
func (c *Commands) DeleteDocument(index, id string) tea.Cmd {
	return func() tea.Msg {
		err := c.es.DeleteDocument(index, id)
		return types.DocumentDeletedMsg{Index: index, ID: id, Err: err}
	}
}

// Search runs a search query.
func (c *Commands) Search(index, query string, from, size int) tea.Cmd {
	return func() tea.Msg {
		result, err := c.es.Search(index, query, from, size)
		return types.SearchResultMsg{Result: result, Err: err}
	}
}

// LoadClusterHealth loads cluster health.
func (c *Commands) LoadClusterHealth() tea.Cmd {
	return func() tea.Msg {
		h, err := c.es.GetClusterHealth()
		return types.ClusterHealthLoadedMsg{Health: h, Err: err}
	}
}

// LoadNodes loads cluster nodes.
func (c *Commands) LoadNodes() tea.Cmd {
	return func() tea.Msg {
		nodes, err := c.es.GetNodes()
		return types.NodesLoadedMsg{Nodes: nodes, Err: err}
	}
}

// LoadShards loads shard allocation.
func (c *Commands) LoadShards(index string) tea.Cmd {
	return func() tea.Msg {
		shards, err := c.es.GetShards(index)
		return types.ShardsLoadedMsg{Shards: shards, Err: err}
	}
}

// LoadAliases loads aliases.
func (c *Commands) LoadAliases() tea.Cmd {
	return func() tea.Msg {
		aliases, err := c.es.ListAliases()
		return types.AliasesLoadedMsg{Aliases: aliases, Err: err}
	}
}

// LoadTemplates loads index templates.
func (c *Commands) LoadTemplates() tea.Cmd {
	return func() tea.Msg {
		templates, err := c.es.ListTemplates()
		return types.TemplatesLoadedMsg{Templates: templates, Err: err}
	}
}

// LoadLiveMetrics loads live metrics.
func (c *Commands) LoadLiveMetrics() tea.Cmd {
	return func() tea.Msg {
		data, err := c.es.GetLiveMetrics()
		return types.LiveMetricsMsg{Data: data, Err: err}
	}
}

// LiveMetricsTick schedules the next metrics poll.
func (c *Commands) LiveMetricsTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return types.LiveMetricsTickMsg{}
	})
}

// BulkDelete runs delete-by-query.
func (c *Commands) BulkDelete(index, query string) tea.Cmd {
	return func() tea.Msg {
		deleted, err := c.es.DeleteByQuery(index, query)
		return types.BulkDeleteMsg{Index: index, Deleted: deleted, Err: err}
	}
}

// LoadFavorites loads favorites for a connection.
func (c *Commands) LoadFavorites(connID int64) tea.Cmd {
	return func() tea.Msg {
		favs := c.config.ListFavorites(connID)
		return types.FavoritesLoadedMsg{Favorites: favs}
	}
}

// AddFavorite favorites an index.
func (c *Commands) AddFavorite(connID int64, index, label string) tea.Cmd {
	return func() tea.Msg {
		fav, err := c.config.AddFavorite(connID, index, label)
		return types.FavoriteAddedMsg{Favorite: fav, Err: err}
	}
}

// RemoveFavorite unfavorites an index.
func (c *Commands) RemoveFavorite(connID int64, index string) tea.Cmd {
	return func() tea.Msg {
		err := c.config.RemoveFavorite(connID, index)
		return types.FavoriteRemovedMsg{Index: index, Err: err}
	}
}

// LoadRecentIndices loads recent indices.
func (c *Commands) LoadRecentIndices(connID int64) tea.Cmd {
	return func() tea.Msg {
		indices := c.config.ListRecentIndices(connID)
		return types.RecentIndicesLoadedMsg{Indices: indices}
	}
}

// CatAPI runs a cat endpoint.
func (c *Commands) CatAPI(endpoint string) tea.Cmd {
	return func() tea.Msg {
		body, err := c.es.Cat(endpoint)
		return types.CatAPIResultMsg{Endpoint: endpoint, Body: body, Err: err}
	}
}

// LoadIndexSettings loads settings for an index.
func (c *Commands) LoadIndexSettings(name string) tea.Cmd {
	return func() tea.Msg {
		s, err := c.es.GetIndexSettings(name)
		return types.IndexSettingsLoadedMsg{Settings: s, Err: err}
	}
}

// LoadIndexMappings loads mappings for an index.
func (c *Commands) LoadIndexMappings(name string) tea.Cmd {
	return func() tea.Msg {
		m, err := c.es.GetIndexMappings(name)
		return types.IndexMappingsLoadedMsg{Mappings: m, Err: err}
	}
}

// RefreshIndex refreshes an index.
func (c *Commands) RefreshIndex(name string) tea.Cmd {
	return func() tea.Msg {
		err := c.es.RefreshIndex(name)
		if err != nil {
			return types.IndicesLoadedMsg{Err: err}
		}
		indices, err := c.es.ListIndices("*")
		return types.IndicesLoadedMsg{Indices: indices, Err: err}
	}
}

// RefreshIndexOnly refreshes an index without reloading the index list.
func (c *Commands) RefreshIndexOnly(name string) tea.Cmd {
	return func() tea.Msg {
		return types.IndexOpMsg{Op: "refresh", Index: name, Err: c.es.RefreshIndex(name)}
	}
}

// OpenIndex opens a closed index.
func (c *Commands) OpenIndex(name string) tea.Cmd {
	return func() tea.Msg {
		return types.IndexOpMsg{Op: "open", Index: name, Err: c.es.OpenIndex(name)}
	}
}

// CloseIndex closes an open index.
func (c *Commands) CloseIndex(name string) tea.Cmd {
	return func() tea.Msg {
		return types.IndexOpMsg{Op: "close", Index: name, Err: c.es.CloseIndex(name)}
	}
}

// ForceMerge force-merges an index.
func (c *Commands) ForceMerge(name string) tea.Cmd {
	return func() tea.Msg {
		return types.IndexOpMsg{Op: "forcemerge", Index: name, Err: c.es.ForceMerge(name, 0)}
	}
}

// CopyToClipboard writes content to the system clipboard.
func (c *Commands) CopyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(content)
		return types.ClipboardCopiedMsg{Content: content, Err: err}
	}
}
