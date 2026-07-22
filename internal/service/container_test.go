package service

import (
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

type stubConfig struct{ closeErr error }

func (s *stubConfig) ListConnections() ([]types.Connection, error) { return nil, nil }
func (s *stubConfig) AddConnection(types.Connection) (types.Connection, error) {
	return types.Connection{}, nil
}
func (s *stubConfig) UpdateConnection(types.Connection) (types.Connection, error) {
	return types.Connection{}, nil
}
func (s *stubConfig) DeleteConnection(int64) error { return nil }
func (s *stubConfig) AddFavorite(int64, string, string) (types.Favorite, error) {
	return types.Favorite{}, nil
}
func (s *stubConfig) RemoveFavorite(int64, string) error                       { return nil }
func (s *stubConfig) ListFavorites(int64) []types.Favorite                     { return nil }
func (s *stubConfig) IsFavorite(int64, string) bool                            { return false }
func (s *stubConfig) AddRecentIndex(int64, string)                             {}
func (s *stubConfig) ListRecentIndices(int64) []types.RecentIndex              { return nil }
func (s *stubConfig) ClearRecentIndices(int64)                                 {}
func (s *stubConfig) AddValueHistory(string, string, string, string)           {}
func (s *stubConfig) GetValueHistory(string, string) []types.ValueHistoryEntry { return nil }
func (s *stubConfig) ClearValueHistory()                                       {}
func (s *stubConfig) ListGroups() []types.ConnectionGroup                      { return nil }
func (s *stubConfig) AddGroup(string, string) error                            { return nil }
func (s *stubConfig) AddConnectionToGroup(string, int64) error                 { return nil }
func (s *stubConfig) RemoveConnectionFromGroup(string, int64) error            { return nil }
func (s *stubConfig) GetKeyBindings() types.KeyBindings                        { return types.DefaultKeyBindings() }
func (s *stubConfig) SetKeyBindings(types.KeyBindings) error                   { return nil }
func (s *stubConfig) ResetKeyBindings() error                                  { return nil }
func (s *stubConfig) Close() error                                             { return s.closeErr }

type stubES struct{ discErr error }

func (s *stubES) Connect(types.Connection) error { return nil }
func (s *stubES) Disconnect() error              { return s.discErr }
func (s *stubES) TestConnection(types.Connection) (time.Duration, types.ClusterInfo, error) {
	return 0, types.ClusterInfo{}, nil
}
func (s *stubES) IsConnected() bool                              { return false }
func (s *stubES) Flavor() types.Flavor                           { return types.FlavorAuto }
func (s *stubES) GetClusterInfo() (types.ClusterInfo, error)     { return types.ClusterInfo{}, nil }
func (s *stubES) GetClusterHealth() (types.ClusterHealth, error) { return types.ClusterHealth{}, nil }
func (s *stubES) GetNodes() ([]types.NodeInfo, error)            { return nil, nil }
func (s *stubES) GetShards(string) ([]types.ShardInfo, error)    { return nil, nil }
func (s *stubES) GetLiveMetrics() (types.LiveMetricsData, error) { return types.LiveMetricsData{}, nil }
func (s *stubES) Cat(string) (string, error)                     { return "", nil }
func (s *stubES) ListIndices(string) ([]types.IndexInfo, error)  { return nil, nil }
func (s *stubES) GetIndex(string) (types.IndexInfo, error)       { return types.IndexInfo{}, nil }
func (s *stubES) CreateIndex(string, string) error               { return nil }
func (s *stubES) DeleteIndex(string) error                       { return nil }
func (s *stubES) GetIndexSettings(string) (string, error)        { return "", nil }
func (s *stubES) GetIndexMappings(string) (string, error)        { return "", nil }
func (s *stubES) RefreshIndex(string) error                      { return nil }
func (s *stubES) OpenIndex(string) error                         { return nil }
func (s *stubES) CloseIndex(string) error                        { return nil }
func (s *stubES) Search(string, string, int, int) (types.SearchResult, error) {
	return types.SearchResult{}, nil
}
func (s *stubES) GetDocument(string, string) (types.Document, error) { return types.Document{}, nil }
func (s *stubES) IndexDocument(string, string, string) error         { return nil }
func (s *stubES) DeleteDocument(string, string) error                { return nil }
func (s *stubES) DeleteByQuery(string, string) (int64, error)        { return 0, nil }
func (s *stubES) ListAliases() ([]types.AliasInfo, error)            { return nil, nil }
func (s *stubES) ListTemplates() ([]types.IndexTemplate, error)      { return nil, nil }

func TestNewContainerAndClose(t *testing.T) {
	c := NewContainer(&stubConfig{}, &stubES{})
	if c.Config == nil || c.ES == nil {
		t.Fatal("nil services")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}

	c2 := NewContainer(&stubConfig{closeErr: errors.New("c")}, &stubES{discErr: errors.New("e")})
	if err := c2.Close(); err == nil {
		t.Fatal("expected error")
	}

	c3 := NewContainer(nil, nil)
	if err := c3.Close(); err != nil {
		t.Fatal(err)
	}
}
