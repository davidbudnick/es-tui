package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

// jsonMarshalIndent is overridable in tests to simulate marshal errors.
var jsonMarshalIndent = json.MarshalIndent

// Config stores all application configuration.
type Config struct {
	Connections      []types.Connection        `json:"connections"`
	Groups           []types.ConnectionGroup   `json:"groups,omitempty"`
	Favorites        []types.Favorite          `json:"favorites,omitempty"`
	RecentIndices    []types.RecentIndex       `json:"recent_indices,omitempty"`
	KeyBindings      types.KeyBindings         `json:"key_bindings"`
	ValueHistory     []types.ValueHistoryEntry `json:"-"`
	MaxRecentIndices int                       `json:"max_recent_indices"`
	MaxValueHistory  int                       `json:"max_value_history"`
	nextID           int64
	path             string
	mu               sync.RWMutex
}

// NewConfig creates or loads configuration from the given path.
func NewConfig(configPath string) (*Config, error) {
	c := &Config{
		path:             configPath,
		Connections:      []types.Connection{},
		Groups:           []types.ConnectionGroup{},
		Favorites:        []types.Favorite{},
		RecentIndices:    []types.RecentIndex{},
		KeyBindings:      types.DefaultKeyBindings(),
		MaxRecentIndices: 20,
		MaxValueHistory:  50,
		nextID:           1,
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}

	if err := c.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	for _, conn := range c.Connections {
		if conn.ID >= c.nextID {
			c.nextID = conn.ID + 1
		}
	}

	return c, nil
}

func (c *Config) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, c)
}

func (c *Config) save() error {
	safeConnections := make([]types.Connection, len(c.Connections))
	for i, conn := range c.Connections {
		safeConnections[i] = conn
		safeConnections[i].Password = ""
		safeConnections[i].APIKey = ""
	}

	safeCfg := &Config{
		Connections:      safeConnections,
		Groups:           c.Groups,
		Favorites:        c.Favorites,
		RecentIndices:    c.RecentIndices,
		KeyBindings:      c.KeyBindings,
		MaxRecentIndices: c.MaxRecentIndices,
		MaxValueHistory:  c.MaxValueHistory,
	}

	data, err := jsonMarshalIndent(safeCfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0o600)
}

// Close implements ConfigService.
func (c *Config) Close() error {
	return nil
}

// ListConnections returns all saved connections sorted by ID.
func (c *Config) ListConnections() ([]types.Connection, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]types.Connection, len(c.Connections))
	copy(result, c.Connections)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

// AddConnection adds a new connection.
func (c *Config) AddConnection(conn types.Connection) (types.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	conn.ID = c.nextID
	conn.Created = now
	conn.Updated = now
	if conn.Flavor == "" {
		conn.Flavor = types.FlavorAuto
	}
	c.nextID++
	c.Connections = append(c.Connections, conn)

	if err := c.save(); err != nil {
		c.Connections = c.Connections[:len(c.Connections)-1]
		c.nextID--
		return types.Connection{}, err
	}
	return c.Connections[len(c.Connections)-1], nil
}

// UpdateConnection updates an existing connection, preserving Group/Color/TLS.
func (c *Config) UpdateConnection(conn types.Connection) (types.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, existing := range c.Connections {
		if existing.ID == conn.ID {
			if conn.Group == "" {
				conn.Group = existing.Group
			}
			if conn.Color == "" {
				conn.Color = existing.Color
			}
			if !conn.UseTLS {
				conn.UseTLS = existing.UseTLS
			}
			if conn.TLSConfig == nil {
				conn.TLSConfig = existing.TLSConfig
			}
			if conn.Password == "" {
				conn.Password = existing.Password
			}
			if conn.APIKey == "" {
				conn.APIKey = existing.APIKey
			}
			conn.Created = existing.Created
			conn.Updated = time.Now()
			c.Connections[i] = conn
			if err := c.save(); err != nil {
				c.Connections[i] = existing
				return types.Connection{}, err
			}
			return conn, nil
		}
	}
	return types.Connection{}, os.ErrNotExist
}

// DeleteConnection removes a connection by ID.
func (c *Config) DeleteConnection(id int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, conn := range c.Connections {
		if conn.ID == id {
			c.Connections = append(c.Connections[:i], c.Connections[i+1:]...)
			return c.save()
		}
	}
	return os.ErrNotExist
}

// AddFavorite adds an index to favorites.
func (c *Config) AddFavorite(connID int64, index, label string) (types.Favorite, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, f := range c.Favorites {
		if f.ConnectionID == connID && f.Index == index {
			return f, nil
		}
	}

	connName := ""
	for _, conn := range c.Connections {
		if conn.ID == connID {
			connName = conn.Name
			break
		}
	}

	fav := types.Favorite{
		ConnectionID: connID,
		Connection:   connName,
		Index:        index,
		Label:        label,
		AddedAt:      time.Now(),
	}
	c.Favorites = append(c.Favorites, fav)
	if err := c.save(); err != nil {
		c.Favorites = c.Favorites[:len(c.Favorites)-1]
		return types.Favorite{}, err
	}
	return fav, nil
}

// RemoveFavorite removes an index from favorites.
func (c *Config) RemoveFavorite(connID int64, index string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, f := range c.Favorites {
		if f.ConnectionID == connID && f.Index == index {
			c.Favorites = append(c.Favorites[:i], c.Favorites[i+1:]...)
			return c.save()
		}
	}
	return nil
}

// ListFavorites returns favorites for a connection.
func (c *Config) ListFavorites(connID int64) []types.Favorite {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []types.Favorite
	for _, f := range c.Favorites {
		if f.ConnectionID == connID {
			result = append(result, f)
		}
	}
	return result
}

// IsFavorite reports whether an index is favorited.
func (c *Config) IsFavorite(connID int64, index string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, f := range c.Favorites {
		if f.ConnectionID == connID && f.Index == index {
			return true
		}
	}
	return false
}

// AddRecentIndex records a recently accessed index.
func (c *Config) AddRecentIndex(connID int64, index string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	filtered := make([]types.RecentIndex, 0, len(c.RecentIndices))
	for _, r := range c.RecentIndices {
		if !(r.ConnectionID == connID && r.Index == index) {
			filtered = append(filtered, r)
		}
	}
	c.RecentIndices = append([]types.RecentIndex{{
		ConnectionID: connID,
		Index:        index,
		AccessedAt:   time.Now(),
	}}, filtered...)

	if len(c.RecentIndices) > c.MaxRecentIndices {
		c.RecentIndices = c.RecentIndices[:c.MaxRecentIndices]
	}
	_ = c.save()
}

// ListRecentIndices returns recent indices for a connection.
func (c *Config) ListRecentIndices(connID int64) []types.RecentIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []types.RecentIndex
	for _, r := range c.RecentIndices {
		if r.ConnectionID == connID {
			result = append(result, r)
		}
	}
	return result
}

// ClearRecentIndices clears recent indices for a connection.
func (c *Config) ClearRecentIndices(connID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	filtered := make([]types.RecentIndex, 0)
	for _, r := range c.RecentIndices {
		if r.ConnectionID != connID {
			filtered = append(filtered, r)
		}
	}
	c.RecentIndices = filtered
	_ = c.save()
}

// AddValueHistory records a document value change.
func (c *Config) AddValueHistory(index, docID, value, action string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ValueHistory = append([]types.ValueHistoryEntry{{
		Index:     index,
		DocID:     docID,
		Value:     value,
		Timestamp: time.Now(),
		Action:    action,
	}}, c.ValueHistory...)

	if len(c.ValueHistory) > c.MaxValueHistory {
		c.ValueHistory = c.ValueHistory[:c.MaxValueHistory]
	}
}

// GetValueHistory returns history entries for a document.
func (c *Config) GetValueHistory(index, docID string) []types.ValueHistoryEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []types.ValueHistoryEntry
	for _, e := range c.ValueHistory {
		if e.Index == index && e.DocID == docID {
			result = append(result, e)
		}
	}
	return result
}

// ClearValueHistory clears all value history.
func (c *Config) ClearValueHistory() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ValueHistory = nil
}

// ListGroups returns connection groups.
func (c *Config) ListGroups() []types.ConnectionGroup {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]types.ConnectionGroup, len(c.Groups))
	copy(result, c.Groups)
	return result
}

// AddGroup adds a connection group.
func (c *Config) AddGroup(name, color string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, g := range c.Groups {
		if g.Name == name {
			return nil
		}
	}
	c.Groups = append(c.Groups, types.ConnectionGroup{Name: name, Color: color})
	return c.save()
}

// AddConnectionToGroup adds a connection ID to a group.
func (c *Config) AddConnectionToGroup(groupName string, connID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, g := range c.Groups {
		if g.Name == groupName {
			for _, id := range g.Connections {
				if id == connID {
					return nil
				}
			}
			c.Groups[i].Connections = append(c.Groups[i].Connections, connID)
			return c.save()
		}
	}
	return os.ErrNotExist
}

// RemoveConnectionFromGroup removes a connection ID from a group.
func (c *Config) RemoveConnectionFromGroup(groupName string, connID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, g := range c.Groups {
		if g.Name == groupName {
			for j, id := range g.Connections {
				if id == connID {
					c.Groups[i].Connections = append(g.Connections[:j], g.Connections[j+1:]...)
					return c.save()
				}
			}
			return nil
		}
	}
	return os.ErrNotExist
}

// GetKeyBindings returns current key bindings.
func (c *Config) GetKeyBindings() types.KeyBindings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.KeyBindings
}

// SetKeyBindings updates key bindings.
func (c *Config) SetKeyBindings(kb types.KeyBindings) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.KeyBindings = kb
	return c.save()
}

// ResetKeyBindings restores default key bindings.
func (c *Config) ResetKeyBindings() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.KeyBindings = types.DefaultKeyBindings()
	return c.save()
}
