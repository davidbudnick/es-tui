package testutil

import (
	"fmt"
	"sync"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

// MockES is a configurable mock of service.ESService.
type MockES struct {
	mu sync.Mutex

	Connected   bool
	FlavorVal   types.Flavor
	ConnectErr  error
	TestLatency time.Duration
	TestInfo    types.ClusterInfo
	TestErr     error

	Info          types.ClusterInfo
	InfoErr       error
	Health        types.ClusterHealth
	HealthErr     error
	Nodes         []types.NodeInfo
	NodesErr      error
	Shards        []types.ShardInfo
	ShardsErr     error
	Metrics       types.LiveMetricsData
	MetricsErr    error
	CatBody       string
	CatErr        error
	Indices       []types.IndexInfo
	IndicesErr    error
	IndexDetail   types.IndexInfo
	IndexErr      error
	CreateErr     error
	DeleteErr     error
	Settings      string
	SettingsErr   error
	Mappings      string
	MappingsErr   error
	RefreshErr    error
	OpenErr       error
	CloseErr      error
	ForceMergeErr error
	SearchResult  types.SearchResult
	SearchErr     error
	Document      types.Document
	DocumentErr   error
	IndexDocErr   error
	DeleteDocErr  error
	DeleteByQ     int64
	DeleteByQErr  error
	Aliases       []types.AliasInfo
	AliasesErr    error
	Templates     []types.IndexTemplate
	TemplatesErr  error
	LastConnect   types.Connection
	LastSearchIdx string
	LastSearchQ   string
}

func (m *MockES) Connect(conn types.Connection) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastConnect = conn
	if m.ConnectErr != nil {
		return m.ConnectErr
	}
	m.Connected = true
	return nil
}

func (m *MockES) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Connected = false
	return nil
}

func (m *MockES) TestConnection(conn types.Connection) (time.Duration, types.ClusterInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.TestLatency, m.TestInfo, m.TestErr
}

func (m *MockES) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Connected
}

func (m *MockES) Flavor() types.Flavor {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FlavorVal != "" {
		return m.FlavorVal
	}
	return types.FlavorElasticsearch
}

func (m *MockES) GetClusterInfo() (types.ClusterInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Info, m.InfoErr
}

func (m *MockES) GetClusterHealth() (types.ClusterHealth, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Health, m.HealthErr
}

func (m *MockES) GetNodes() ([]types.NodeInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Nodes, m.NodesErr
}

func (m *MockES) GetShards(index string) ([]types.ShardInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Shards, m.ShardsErr
}

func (m *MockES) GetLiveMetrics() (types.LiveMetricsData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Metrics, m.MetricsErr
}

func (m *MockES) Cat(endpoint string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CatBody, m.CatErr
}

func (m *MockES) ListIndices(pattern string) ([]types.IndexInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Indices, m.IndicesErr
}

func (m *MockES) GetIndex(name string) (types.IndexInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.IndexErr != nil {
		return types.IndexInfo{}, m.IndexErr
	}
	if m.IndexDetail.Name != "" {
		return m.IndexDetail, nil
	}
	for _, idx := range m.Indices {
		if idx.Name == name {
			return idx, nil
		}
	}
	return types.IndexInfo{}, fmt.Errorf("index not found")
}

func (m *MockES) CreateIndex(name string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.Indices = append(m.Indices, types.IndexInfo{Name: name, Health: "green", Status: "open"})
	return nil
}

func (m *MockES) DeleteIndex(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	filtered := m.Indices[:0]
	for _, idx := range m.Indices {
		if idx.Name != name {
			filtered = append(filtered, idx)
		}
	}
	m.Indices = filtered
	return nil
}

func (m *MockES) GetIndexSettings(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Settings, m.SettingsErr
}

func (m *MockES) GetIndexMappings(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Mappings, m.MappingsErr
}

func (m *MockES) RefreshIndex(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.RefreshErr
}

func (m *MockES) OpenIndex(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.OpenErr
}

func (m *MockES) CloseIndex(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CloseErr
}

func (m *MockES) ForceMerge(name string, maxNumSegments int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ForceMergeErr
}

func (m *MockES) Search(index, query string, from, size int) (types.SearchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastSearchIdx = index
	m.LastSearchQ = query
	return m.SearchResult, m.SearchErr
}

func (m *MockES) GetDocument(index, id string) (types.Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Document, m.DocumentErr
}

func (m *MockES) IndexDocument(index, id, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.IndexDocErr
}

func (m *MockES) DeleteDocument(index, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.DeleteDocErr
}

func (m *MockES) DeleteByQuery(index, query string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.DeleteByQ, m.DeleteByQErr
}

func (m *MockES) ListAliases() ([]types.AliasInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Aliases, m.AliasesErr
}

func (m *MockES) ListTemplates() ([]types.IndexTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Templates, m.TemplatesErr
}
