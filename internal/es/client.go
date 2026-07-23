// Package es provides an Elasticsearch/OpenSearch HTTP client.
package es

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

// Client is an HTTP client for Elasticsearch and OpenSearch.
type Client struct {
	mu      sync.RWMutex
	http    *http.Client
	baseURL string
	user    string
	pass    string
	apiKey  string
	bearer  string
	flavor  types.Flavor
	conn    *types.Connection
	ctx     context.Context
}

// NewClient creates a disconnected client.
func NewClient() *Client {
	return &Client{
		ctx: context.Background(),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Connect establishes a connection and detects the engine flavor.
func (c *Client) Connect(conn types.Connection) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	httpClient, err := buildHTTPClient(conn)
	if err != nil {
		return err
	}

	c.http = httpClient
	c.baseURL = strings.TrimRight(conn.BaseURL(), "/")
	c.user = conn.Username
	c.pass = conn.Password
	c.apiKey = conn.APIKey
	c.bearer = conn.BearerToken
	c.conn = &conn
	c.flavor = conn.Flavor

	info, err := c.getClusterInfoLocked()
	if err != nil {
		c.http = &http.Client{Timeout: 30 * time.Second}
		c.baseURL = ""
		c.conn = nil
		c.user = ""
		c.pass = ""
		c.apiKey = ""
		c.bearer = ""
		return fmt.Errorf("connect: %w", err)
	}

	if c.flavor == "" || c.flavor == types.FlavorAuto {
		c.flavor = detectFlavor(info)
	}
	// Keep the live connection object in sync with what we actually talk to.
	if c.conn != nil {
		c.conn.Flavor = c.flavor
	}
	return nil
}

// Disconnect closes the connection.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = ""
	c.conn = nil
	c.user = ""
	c.pass = ""
	c.apiKey = ""
	c.bearer = ""
	c.flavor = ""
	c.http = &http.Client{Timeout: 30 * time.Second}
	return nil
}

// IsReadOnly reports whether the active connection is read-only.
func (c *Client) IsReadOnly() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && c.conn.ReadOnly
}

// IsConnected reports whether the client has an active connection.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL != ""
}

// Flavor returns the detected or configured engine flavor.
func (c *Client) Flavor() types.Flavor {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.flavor
}

// TestConnection tests connectivity without mutating the active client state.
func (c *Client) TestConnection(conn types.Connection) (time.Duration, types.ClusterInfo, error) {
	httpClient, err := buildHTTPClient(conn)
	if err != nil {
		return 0, types.ClusterInfo{}, err
	}

	tmp := &Client{
		http:    httpClient,
		baseURL: strings.TrimRight(conn.BaseURL(), "/"),
		user:    conn.Username,
		pass:    conn.Password,
		apiKey:  conn.APIKey,
		bearer:  conn.BearerToken,
		ctx:     context.Background(),
	}

	start := time.Now()
	info, err := tmp.getClusterInfoLocked()
	latency := time.Since(start)
	if err != nil {
		return latency, types.ClusterInfo{}, err
	}
	info.Flavor = detectFlavor(info)
	return latency, info, nil
}

func buildHTTPClient(conn types.Connection) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if conn.UseTLS {
		tlsCfg := &tls.Config{} // #nosec G402
		if conn.TLSConfig != nil {
			var err error
			tlsCfg, err = conn.TLSConfig.BuildTLSConfig()
			if err != nil {
				return nil, err
			}
		}
		transport.TLSClientConfig = tlsCfg
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}

func detectFlavor(info types.ClusterInfo) types.Flavor {
	dist := strings.ToLower(strings.TrimSpace(info.Version.Distribution))
	if dist == "opensearch" || strings.Contains(dist, "opensearch") {
		return types.FlavorOpenSearch
	}
	tag := strings.ToLower(info.Tagline)
	if strings.Contains(tag, "opensearch") {
		return types.FlavorOpenSearch
	}
	// Elasticsearch root tagline is the classic "You Know, for Search".
	if strings.Contains(tag, "you know, for search") || strings.Contains(tag, "elastic") {
		return types.FlavorElasticsearch
	}
	// build_flavor is present on ES; OpenSearch uses distribution instead.
	bf := strings.ToLower(info.Version.BuildFlavor)
	if bf == "default" || bf == "oss" || bf == "serverless" {
		return types.FlavorElasticsearch
	}
	if info.Version.Number != "" {
		// Versioned root response without OpenSearch markers → ES family.
		return types.FlavorElasticsearch
	}
	return types.FlavorAuto
}

func (c *Client) do(method, path string, body any) ([]byte, int, error) {
	c.mu.RLock()
	base := c.baseURL
	httpClient := c.http
	user := c.user
	pass := c.pass
	apiKey := c.apiKey
	bearer := c.bearer
	ctx := c.ctx
	c.mu.RUnlock()

	if base == "" {
		return nil, 0, fmt.Errorf("not connected")
	}

	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, 0, fmt.Errorf("marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(b)
		}
	}

	u := base + path
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	setAuth(req, apiKey, bearer, user, pass)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode >= 400 {
		return data, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateErr(data))
	}
	return data, resp.StatusCode, nil
}

func setAuth(req *http.Request, apiKey, bearer, user, pass string) {
	if apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+apiKey)
	} else if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	} else if user != "" {
		req.SetBasicAuth(user, pass)
	}
}

func truncateErr(data []byte) string {
	s := string(data)
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}

func (c *Client) getClusterInfoLocked() (types.ClusterInfo, error) {
	// Note: caller may or may not hold lock; do uses RLock so nested is OK with RWMutex
	// but Connect holds write lock. Use unlocked path for root request.
	data, _, err := c.doUnlocked("GET", "/", nil)
	if err != nil {
		return types.ClusterInfo{}, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return types.ClusterInfo{}, err
	}

	info := types.ClusterInfo{
		Name:        str(raw["name"]),
		ClusterName: str(raw["cluster_name"]),
		ClusterUUID: str(raw["cluster_uuid"]),
		Tagline:     str(raw["tagline"]),
	}
	if v, ok := raw["version"].(map[string]any); ok {
		info.Version = types.VersionInfo{
			Number:                           str(v["number"]),
			BuildFlavor:                      str(v["build_flavor"]),
			BuildType:                        str(v["build_type"]),
			BuildHash:                        str(v["build_hash"]),
			BuildDate:                        str(v["build_date"]),
			BuildSnapshot:                    boolVal(v["build_snapshot"]),
			LuceneVersion:                    str(v["lucene_version"]),
			MinimumWireCompatibilityVersion:  str(v["minimum_wire_compatibility_version"]),
			MinimumIndexCompatibilityVersion: str(v["minimum_index_compatibility_version"]),
			Distribution:                     str(v["distribution"]),
		}
	}
	info.Flavor = detectFlavor(info)
	return info, nil
}

// doUnlocked is used when the write lock is already held (Connect).
func (c *Client) doUnlocked(method, path string, body any) ([]byte, int, error) {
	if c.baseURL == "" {
		return nil, 0, fmt.Errorf("not connected")
	}

	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, 0, fmt.Errorf("marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequestWithContext(c.ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	setAuth(req, c.apiKey, c.bearer, c.user, c.pass)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return data, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateErr(data))
	}
	return data, resp.StatusCode, nil
}

// GetClusterInfo returns root cluster info.
func (c *Client) GetClusterInfo() (types.ClusterInfo, error) {
	data, _, err := c.do("GET", "/", nil)
	if err != nil {
		return types.ClusterInfo{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return types.ClusterInfo{}, err
	}
	info := types.ClusterInfo{
		Name:        str(raw["name"]),
		ClusterName: str(raw["cluster_name"]),
		ClusterUUID: str(raw["cluster_uuid"]),
		Tagline:     str(raw["tagline"]),
	}
	if v, ok := raw["version"].(map[string]any); ok {
		info.Version = types.VersionInfo{
			Number:        str(v["number"]),
			BuildFlavor:   str(v["build_flavor"]),
			BuildType:     str(v["build_type"]),
			BuildHash:     str(v["build_hash"]),
			BuildDate:     str(v["build_date"]),
			LuceneVersion: str(v["lucene_version"]),
			Distribution:  str(v["distribution"]),
		}
	}
	// Prefer live client flavor (forced or previously detected); else detect from body.
	detected := detectFlavor(info)
	if f := c.Flavor(); f.IsKnown() {
		info.Flavor = f
	} else {
		info.Flavor = detected
	}
	return info, nil
}

// GetClusterHealth returns cluster health.
func (c *Client) GetClusterHealth() (types.ClusterHealth, error) {
	data, _, err := c.do("GET", "/_cluster/health", nil)
	if err != nil {
		return types.ClusterHealth{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return types.ClusterHealth{}, err
	}
	return types.ClusterHealth{
		ClusterName:                 str(raw["cluster_name"]),
		Status:                      str(raw["status"]),
		TimedOut:                    boolVal(raw["timed_out"]),
		NumberOfNodes:               intVal(raw["number_of_nodes"]),
		NumberOfDataNodes:           intVal(raw["number_of_data_nodes"]),
		ActivePrimaryShards:         intVal(raw["active_primary_shards"]),
		ActiveShards:                intVal(raw["active_shards"]),
		RelocatingShards:            intVal(raw["relocating_shards"]),
		InitializingShards:          intVal(raw["initializing_shards"]),
		UnassignedShards:            intVal(raw["unassigned_shards"]),
		DelayedUnassignedShards:     intVal(raw["delayed_unassigned_shards"]),
		NumberOfPendingTasks:        intVal(raw["number_of_pending_tasks"]),
		NumberOfInFlightFetch:       intVal(raw["number_of_in_flight_fetch"]),
		TaskMaxWaitingInQueueMillis: intVal(raw["task_max_waiting_in_queue_millis"]),
		ActiveShardsPercentAsNumber: floatVal(raw["active_shards_percent_as_number"]),
	}, nil
}

// GetNodes returns node list via _cat/nodes.
func (c *Client) GetNodes() ([]types.NodeInfo, error) {
	data, _, err := c.do("GET", "/_cat/nodes?format=json&h=name,id,ip,host,node.role,master,heap.percent,ram.percent,cpu,load_1m,version,disk.used_percent,disk.total,disk.used,disk.avail", nil)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	nodes := make([]types.NodeInfo, 0, len(rows))
	for _, r := range rows {
		nodes = append(nodes, types.NodeInfo{
			Name:            str(r["name"]),
			ID:              str(r["id"]),
			IP:              str(r["ip"]),
			Host:            str(r["host"]),
			NodeRole:        str(r["node.role"]),
			Master:          str(r["master"]),
			HeapPercent:     intVal(r["heap.percent"]),
			RamPercent:      intVal(r["ram.percent"]),
			CPU:             intVal(r["cpu"]),
			Load1m:          str(r["load_1m"]),
			Version:         str(r["version"]),
			DiskUsedPercent: str(r["disk.used_percent"]),
			DiskTotal:       str(r["disk.total"]),
			DiskUsed:        str(r["disk.used"]),
			DiskAvail:       str(r["disk.avail"]),
		})
	}
	return nodes, nil
}

// GetShards returns shard info, optionally filtered by index.
func (c *Client) GetShards(index string) ([]types.ShardInfo, error) {
	path := "/_cat/shards?format=json&h=index,shard,prirep,state,docs,store,ip,node"
	if index != "" {
		path = "/_cat/shards/" + url.PathEscape(index) + "?format=json&h=index,shard,prirep,state,docs,store,ip,node"
	}
	data, _, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	shards := make([]types.ShardInfo, 0, len(rows))
	for _, r := range rows {
		shards = append(shards, types.ShardInfo{
			Index:  str(r["index"]),
			Shard:  str(r["shard"]),
			Prirep: str(r["prirep"]),
			State:  str(r["state"]),
			Docs:   str(r["docs"]),
			Store:  str(r["store"]),
			IP:     str(r["ip"]),
			Node:   str(r["node"]),
		})
	}
	return shards, nil
}

// ListIndices returns index catalog via _cat/indices.
func (c *Client) ListIndices(pattern string) ([]types.IndexInfo, error) {
	path := "/_cat/indices?format=json&h=index,health,status,uuid,pri,rep,docs.count,docs.deleted,store.size,pri.store.size&s=index"
	if pattern != "" && pattern != "*" {
		path = "/_cat/indices/" + url.PathEscape(pattern) + "?format=json&h=index,health,status,uuid,pri,rep,docs.count,docs.deleted,store.size,pri.store.size&s=index"
	}
	data, _, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	indices := make([]types.IndexInfo, 0, len(rows))
	for _, r := range rows {
		indices = append(indices, types.IndexInfo{
			Name:          str(r["index"]),
			Health:        str(r["health"]),
			Status:        str(r["status"]),
			UUID:          str(r["uuid"]),
			PrimaryShards: intVal(r["pri"]),
			ReplicaShards: intVal(r["rep"]),
			DocsCount:     int64Val(r["docs.count"]),
			DocsDeleted:   int64Val(r["docs.deleted"]),
			StoreSize:     str(r["store.size"]),
			PriStoreSize:  str(r["pri.store.size"]),
		})
	}
	return indices, nil
}

// GetIndex returns a single index's cat info.
func (c *Client) GetIndex(name string) (types.IndexInfo, error) {
	indices, err := c.ListIndices(name)
	if err != nil {
		return types.IndexInfo{}, err
	}
	for _, idx := range indices {
		if idx.Name == name {
			return idx, nil
		}
	}
	if len(indices) == 1 {
		return indices[0], nil
	}
	return types.IndexInfo{}, fmt.Errorf("index %q not found", name)
}

// CreateIndex creates an index with optional body.
func (c *Client) CreateIndex(name string, body string) error {
	var payload any
	if strings.TrimSpace(body) != "" {
		payload = body
	} else {
		payload = map[string]any{}
	}
	_, _, err := c.do("PUT", "/"+url.PathEscape(name), payload)
	return err
}

// DeleteIndex deletes an index.
func (c *Client) DeleteIndex(name string) error {
	_, _, err := c.do("DELETE", "/"+url.PathEscape(name), nil)
	return err
}

// GetIndexSettings returns index settings as pretty JSON.
func (c *Client) GetIndexSettings(name string) (string, error) {
	data, _, err := c.do("GET", "/"+url.PathEscape(name)+"/_settings?pretty", nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetIndexMappings returns index mappings as pretty JSON.
func (c *Client) GetIndexMappings(name string) (string, error) {
	data, _, err := c.do("GET", "/"+url.PathEscape(name)+"/_mapping?pretty", nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RefreshIndex refreshes an index.
func (c *Client) RefreshIndex(name string) error {
	_, _, err := c.do("POST", "/"+url.PathEscape(name)+"/_refresh", nil)
	return err
}

// OpenIndex opens a closed index.
func (c *Client) OpenIndex(name string) error {
	_, _, err := c.do("POST", "/"+url.PathEscape(name)+"/_open", nil)
	return err
}

// CloseIndex closes an open index.
func (c *Client) CloseIndex(name string) error {
	_, _, err := c.do("POST", "/"+url.PathEscape(name)+"/_close", nil)
	return err
}

// ForceMerge force-merges an index. Omits max_num_segments when maxNumSegments <= 0.
func (c *Client) ForceMerge(name string, maxNumSegments int) error {
	path := "/" + url.PathEscape(name) + "/_forcemerge"
	if maxNumSegments > 0 {
		path += "?max_num_segments=" + strconv.Itoa(maxNumSegments)
	}
	_, _, err := c.do("POST", path, nil)
	return err
}

// Search runs a search query. query may be a simple string (query_string) or full JSON body.
func (c *Client) Search(index, query string, from, size int) (types.SearchResult, error) {
	// Bound page size like redis-tui preview caps — never ask ES for huge pages.
	const maxSearchSize = 200
	if size <= 0 {
		size = 50
	}
	if size > maxSearchSize {
		size = maxSearchSize
	}
	if from < 0 {
		from = 0
	}

	path := "/_search"
	if index != "" {
		path = "/" + url.PathEscape(index) + "/_search"
	}

	var body any
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		body = map[string]any{
			"query": map[string]any{"match_all": map[string]any{}},
			"from":  from,
			"size":  size,
		}
	} else if strings.HasPrefix(trimmed, "{") {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return types.SearchResult{}, fmt.Errorf("invalid query JSON: %w", err)
		}
		if _, ok := parsed["from"]; !ok {
			parsed["from"] = from
		}
		if _, ok := parsed["size"]; !ok {
			parsed["size"] = size
		}
		body = parsed
	} else {
		body = map[string]any{
			"query": map[string]any{
				"query_string": map[string]any{"query": trimmed},
			},
			"from": from,
			"size": size,
		}
	}

	data, _, err := c.do("POST", path, body)
	if err != nil {
		return types.SearchResult{}, err
	}
	return parseSearchResult(data)
}

func parseSearchResult(data []byte) (types.SearchResult, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return types.SearchResult{}, err
	}

	result := types.SearchResult{
		Took:     intVal(raw["took"]),
		TimedOut: boolVal(raw["timed_out"]),
		Raw:      string(data),
	}
	if hits, ok := raw["hits"].(map[string]any); ok {
		result.MaxScore = floatVal(hits["max_score"])
		switch t := hits["total"].(type) {
		case float64:
			result.Total = int64(t)
			result.TotalRel = "eq"
		case map[string]any:
			result.Total = int64Val(t["value"])
			result.TotalRel = str(t["relation"])
		}
		if arr, ok := hits["hits"].([]any); ok {
			for _, item := range arr {
				hm, ok := item.(map[string]any)
				if !ok {
					continue
				}
				doc := types.Document{
					Index: str(hm["_index"]),
					ID:    str(hm["_id"]),
					Score: floatVal(hm["_score"]),
				}
				if src, ok := hm["_source"].(map[string]any); ok {
					doc.Source = src
					if b, err := json.MarshalIndent(src, "", "  "); err == nil {
						doc.Raw = string(b)
					}
				}
				result.Hits = append(result.Hits, doc)
			}
		}
	}
	if aggs, ok := raw["aggregations"].(map[string]any); ok {
		result.Aggregations = aggs
	}
	return result, nil
}

// GetDocument fetches a document by ID.
func (c *Client) GetDocument(index, id string) (types.Document, error) {
	data, _, err := c.do("GET", "/"+url.PathEscape(index)+"/_doc/"+url.PathEscape(id), nil)
	if err != nil {
		return types.Document{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return types.Document{}, err
	}
	doc := types.Document{
		Index: str(raw["_index"]),
		ID:    str(raw["_id"]),
	}
	if src, ok := raw["_source"].(map[string]any); ok {
		doc.Source = src
		if b, err := json.MarshalIndent(src, "", "  "); err == nil {
			doc.Raw = string(b)
		}
	}
	return doc, nil
}

// IndexDocument indexes or updates a document.
func (c *Client) IndexDocument(index, id, body string) error {
	path := "/" + url.PathEscape(index) + "/_doc"
	if id != "" {
		path += "/" + url.PathEscape(id)
	}
	method := "POST"
	if id != "" {
		method = "PUT"
	}
	_, _, err := c.do(method, path, body)
	return err
}

// DeleteDocument deletes a document by ID.
func (c *Client) DeleteDocument(index, id string) error {
	_, _, err := c.do("DELETE", "/"+url.PathEscape(index)+"/_doc/"+url.PathEscape(id), nil)
	return err
}

// DeleteByQuery deletes documents matching a query.
func (c *Client) DeleteByQuery(index, query string) (int64, error) {
	var body any
	trimmed := strings.TrimSpace(query)
	if trimmed == "" || trimmed == "*" {
		body = map[string]any{"query": map[string]any{"match_all": map[string]any{}}}
	} else if strings.HasPrefix(trimmed, "{") {
		body = trimmed
	} else {
		body = map[string]any{
			"query": map[string]any{
				"query_string": map[string]any{"query": trimmed},
			},
		}
	}
	data, _, err := c.do("POST", "/"+url.PathEscape(index)+"/_delete_by_query", body)
	if err != nil {
		return 0, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, err
	}
	return int64Val(raw["deleted"]), nil
}

// ListAliases returns aliases via _cat/aliases.
func (c *Client) ListAliases() ([]types.AliasInfo, error) {
	data, _, err := c.do("GET", "/_cat/aliases?format=json&h=alias,index,filter,routing.index,routing.search,is_write_index", nil)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	aliases := make([]types.AliasInfo, 0, len(rows))
	for _, r := range rows {
		aliases = append(aliases, types.AliasInfo{
			Alias:         str(r["alias"]),
			Index:         str(r["index"]),
			Filter:        str(r["filter"]),
			RoutingIndex:  str(r["routing.index"]),
			RoutingSearch: str(r["routing.search"]),
			IsWriteIndex:  str(r["is_write_index"]),
		})
	}
	return aliases, nil
}

// ListTemplates returns index templates.
func (c *Client) ListTemplates() ([]types.IndexTemplate, error) {
	data, _, err := c.do("GET", "/_index_template", nil)
	if err != nil {
		// Fallback for older ES / OpenSearch
		data, _, err = c.do("GET", "/_template", nil)
		if err != nil {
			return nil, err
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		var templates []types.IndexTemplate
		for name, v := range raw {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			t := types.IndexTemplate{Name: name, Order: intVal(m["order"]), Version: intVal(m["version"])}
			if patterns, ok := m["index_patterns"].([]any); ok {
				for _, p := range patterns {
					t.IndexPatterns = append(t.IndexPatterns, str(p))
				}
			}
			templates = append(templates, t)
		}
		return templates, nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var templates []types.IndexTemplate
	if arr, ok := raw["index_templates"].([]any); ok {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			t := types.IndexTemplate{Name: str(m["name"])}
			if it, ok := m["index_template"].(map[string]any); ok {
				if patterns, ok := it["index_patterns"].([]any); ok {
					for _, p := range patterns {
						t.IndexPatterns = append(t.IndexPatterns, str(p))
					}
				}
				if composed, ok := it["composed_of"].([]any); ok {
					for _, p := range composed {
						t.ComposedOf = append(t.ComposedOf, str(p))
					}
				}
				t.Version = intVal(it["version"])
			}
			templates = append(templates, t)
		}
	}
	return templates, nil
}

// Cat runs a _cat API endpoint and returns the body.
func (c *Client) Cat(endpoint string) (string, error) {
	ep := strings.TrimPrefix(endpoint, "/")
	if !strings.HasPrefix(ep, "_cat") {
		ep = "_cat/" + ep
	}
	if !strings.Contains(ep, "format=") {
		if strings.Contains(ep, "?") {
			ep += "&format=json"
		} else {
			ep += "?format=json"
		}
	}
	data, _, err := c.do("GET", "/"+ep, nil)
	if err != nil {
		return "", err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, data, "", "  "); err == nil {
		return pretty.String(), nil
	}
	return string(data), nil
}

// Count returns the number of documents matching a query.
func (c *Client) Count(index, query string) (int64, error) {
	path := "/_count"
	if index != "" {
		path = "/" + url.PathEscape(index) + "/_count"
	}
	var body any
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		body = map[string]any{"query": map[string]any{"match_all": map[string]any{}}}
	} else if strings.HasPrefix(trimmed, "{") {
		body = trimmed
	} else {
		body = map[string]any{
			"query": map[string]any{
				"query_string": map[string]any{"query": trimmed},
			},
		}
	}
	data, _, err := c.do("POST", path, body)
	if err != nil {
		return 0, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, err
	}
	return int64Val(raw["count"]), nil
}

// Explain explains why a document matches a query.
func (c *Client) Explain(index, id, query string) (types.ExplainResult, error) {
	path := "/" + url.PathEscape(index) + "/_explain/" + url.PathEscape(id)
	var body any
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		body = map[string]any{"query": map[string]any{"match_all": map[string]any{}}}
	} else if strings.HasPrefix(trimmed, "{") {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return types.ExplainResult{}, fmt.Errorf("invalid query JSON: %w", err)
		}
		if _, ok := parsed["query"]; ok {
			body = parsed
		} else {
			body = map[string]any{"query": parsed}
		}
	} else {
		body = map[string]any{
			"query": map[string]any{
				"query_string": map[string]any{"query": trimmed},
			},
		}
	}
	data, _, err := c.do("POST", path, body)
	if err != nil {
		return types.ExplainResult{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return types.ExplainResult{}, err
	}
	result := types.ExplainResult{
		Matched: boolVal(raw["matched"]),
		Raw:     string(data),
	}
	if expl, ok := raw["explanation"]; ok {
		if b, err := json.MarshalIndent(expl, "", "  "); err == nil {
			result.Explanation = string(b)
		} else {
			result.Explanation = fmt.Sprint(expl)
		}
	}
	return result, nil
}

// Reindex starts an async reindex and returns the task ID.
func (c *Client) Reindex(body string) (string, error) {
	data, _, err := c.do("POST", "/_reindex?wait_for_completion=false", body)
	if err != nil {
		return "", err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", err
	}
	return str(raw["task"]), nil
}

// ListAllocation returns disk allocation via _cat/allocation.
func (c *Client) ListAllocation() ([]types.AllocationInfo, error) {
	data, _, err := c.do("GET", "/_cat/allocation?format=json", nil)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	out := make([]types.AllocationInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.AllocationInfo{
			Shards:      str(r["shards"]),
			DiskIndices: str(r["disk.indices"]),
			DiskUsed:    str(r["disk.used"]),
			DiskAvail:   str(r["disk.avail"]),
			DiskTotal:   str(r["disk.total"]),
			DiskPercent: str(r["disk.percent"]),
			Host:        str(r["host"]),
			IP:          str(r["ip"]),
			Node:        str(r["node"]),
		})
	}
	return out, nil
}

// ListTasks returns cluster tasks.
func (c *Client) ListTasks() ([]types.TaskInfo, error) {
	data, _, err := c.do("GET", "/_tasks?detailed=true", nil)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var tasks []types.TaskInfo
	if nodes, ok := raw["nodes"].(map[string]any); ok {
		for nodeID, nv := range nodes {
			nm, ok := nv.(map[string]any)
			if !ok {
				continue
			}
			nodeTasks, _ := nm["tasks"].(map[string]any)
			for taskID, tv := range nodeTasks {
				tm, ok := tv.(map[string]any)
				if !ok {
					continue
				}
				tasks = append(tasks, types.TaskInfo{
					ID:          taskID,
					Action:      str(tm["action"]),
					Type:        str(tm["type"]),
					StartTime:   str(tm["start_time_in_millis"]),
					RunningTime: str(tm["running_time_in_nanos"]),
					Cancellable: str(tm["cancellable"]),
					Node:        firstNonEmpty(str(tm["node"]), nodeID),
					Description: str(tm["description"]),
				})
			}
		}
	}
	if len(tasks) == 0 {
		if arr, ok := raw["tasks"].([]any); ok {
			for _, item := range arr {
				tm, ok := item.(map[string]any)
				if !ok {
					continue
				}
				tasks = append(tasks, types.TaskInfo{
					ID:          str(tm["id"]),
					Action:      str(tm["action"]),
					Type:        str(tm["type"]),
					StartTime:   str(tm["start_time_in_millis"]),
					RunningTime: str(tm["running_time_in_nanos"]),
					Cancellable: str(tm["cancellable"]),
					Node:        str(tm["node"]),
					Description: str(tm["description"]),
				})
			}
		}
	}
	return tasks, nil
}

// CancelTask cancels a running task.
func (c *Client) CancelTask(taskID string) error {
	_, _, err := c.do("POST", "/_tasks/"+url.PathEscape(taskID)+"/_cancel", nil)
	return err
}

// ListPlugins returns installed plugins via _cat/plugins.
func (c *Client) ListPlugins() ([]types.PluginInfo, error) {
	data, _, err := c.do("GET", "/_cat/plugins?format=json", nil)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	out := make([]types.PluginInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.PluginInfo{
			Name:      str(r["name"]),
			Component: str(r["component"]),
			Version:   str(r["version"]),
		})
	}
	return out, nil
}

// GetClusterSettings returns cluster settings as pretty JSON.
func (c *Client) GetClusterSettings() (string, error) {
	data, _, err := c.do("GET", "/_cluster/settings?include_defaults=true&pretty", nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ListDataStreams returns data streams; empty slice on 404.
func (c *Client) ListDataStreams() ([]types.DataStreamInfo, error) {
	data, status, err := c.do("GET", "/_data_stream", nil)
	if err != nil {
		if status == 404 {
			return []types.DataStreamInfo{}, nil
		}
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var out []types.DataStreamInfo
	if arr, ok := raw["data_streams"].([]any); ok {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			info := types.DataStreamInfo{
				Name:       str(m["name"]),
				Generation: str(m["generation"]),
				Status:     str(m["status"]),
				Template:   str(m["template"]),
			}
			if tf, ok := m["timestamp_field"].(map[string]any); ok {
				info.TimestampField = str(tf["name"])
			} else {
				info.TimestampField = str(m["timestamp_field"])
			}
			if indices, ok := m["indices"].([]any); ok {
				info.IndicesCount = strconv.Itoa(len(indices))
			} else {
				info.IndicesCount = str(m["indices_count"])
			}
			out = append(out, info)
		}
	}
	return out, nil
}

// ListSnapshots lists snapshots for a repository.
func (c *Client) ListSnapshots(repo string) ([]types.SnapshotInfo, error) {
	if repo == "" {
		data, _, err := c.do("GET", "/_snapshot", nil)
		if err != nil {
			return nil, err
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		for name := range raw {
			repo = name
			break
		}
		if repo == "" {
			return []types.SnapshotInfo{}, nil
		}
	}
	data, _, err := c.do("GET", "/_snapshot/"+url.PathEscape(repo)+"/_all", nil)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var out []types.SnapshotInfo
	if arr, ok := raw["snapshots"].([]any); ok {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			indices := ""
			if idx, ok := m["indices"].([]any); ok {
				parts := make([]string, 0, len(idx))
				for _, p := range idx {
					parts = append(parts, str(p))
				}
				indices = strings.Join(parts, ",")
			}
			out = append(out, types.SnapshotInfo{
				Snapshot:   str(m["snapshot"]),
				Repository: firstNonEmpty(str(m["repository"]), repo),
				State:      str(m["state"]),
				StartTime:  str(m["start_time"]),
				EndTime:    str(m["end_time"]),
				Indices:    indices,
			})
		}
	}
	return out, nil
}

// ExportDocs searches and returns documents up to maxDocs (capped at 5000).
// Pages through Search (which bounds each page) so large exports stay memory-safe.
func (c *Client) ExportDocs(index, query string, maxDocs int) ([]types.Document, error) {
	if maxDocs <= 0 || maxDocs > 5000 {
		maxDocs = 5000
	}
	const page = 200
	var all []types.Document
	from := 0
	for len(all) < maxDocs {
		n := page
		if maxDocs-len(all) < n {
			n = maxDocs - len(all)
		}
		result, err := c.Search(index, query, from, n)
		if err != nil {
			return nil, err
		}
		if len(result.Hits) == 0 {
			break
		}
		all = append(all, result.Hits...)
		from += len(result.Hits)
		if len(result.Hits) < n {
			break
		}
		if int64(from) >= result.Total && result.Total > 0 {
			break
		}
	}
	return all, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// GetLiveMetrics collects cluster-level metrics.
func (c *Client) GetLiveMetrics() (types.LiveMetricsData, error) {
	health, err := c.GetClusterHealth()
	if err != nil {
		return types.LiveMetricsData{}, err
	}

	m := types.LiveMetricsData{
		Timestamp:        time.Now(),
		Status:           health.Status,
		Nodes:            health.NumberOfNodes,
		DataNodes:        health.NumberOfDataNodes,
		ActiveShards:     health.ActiveShards,
		UnassignedShards: health.UnassignedShards,
	}

	// Stats
	data, _, err := c.do("GET", "/_stats", nil)
	if err == nil {
		var raw map[string]any
		if json.Unmarshal(data, &raw) == nil {
			if all, ok := raw["_all"].(map[string]any); ok {
				if total, ok := all["total"].(map[string]any); ok {
					if docs, ok := total["docs"].(map[string]any); ok {
						m.DocsCount = int64Val(docs["count"])
					}
					if store, ok := total["store"].(map[string]any); ok {
						m.StoreSizeBytes = int64Val(store["size_in_bytes"])
					}
					if search, ok := total["search"].(map[string]any); ok {
						m.QueryTotal = int64Val(search["query_total"])
						if qt := int64Val(search["query_total"]); qt > 0 {
							m.SearchLatencyMs = floatVal(search["query_time_in_millis"]) / float64(qt)
						}
					}
					if indexing, ok := total["indexing"].(map[string]any); ok {
						m.IndexingTotal = int64Val(indexing["index_total"])
					}
				}
			}
		}
	}

	// Nodes stats for JVM/CPU
	ndata, _, err := c.do("GET", "/_nodes/stats/jvm,os,process", nil)
	if err == nil {
		var raw map[string]any
		if json.Unmarshal(ndata, &raw) == nil {
			if nodes, ok := raw["nodes"].(map[string]any); ok {
				var heapSum, heapMax float64
				var cpuSum float64
				var count float64
				for _, nv := range nodes {
					nm, ok := nv.(map[string]any)
					if !ok {
						continue
					}
					count++
					if jvm, ok := nm["jvm"].(map[string]any); ok {
						if mem, ok := jvm["mem"].(map[string]any); ok {
							heapSum += floatVal(mem["heap_used_in_bytes"])
							heapMax += floatVal(mem["heap_max_in_bytes"])
						}
					}
					if os, ok := nm["os"].(map[string]any); ok {
						if cpu, ok := os["cpu"].(map[string]any); ok {
							cpuSum += floatVal(cpu["percent"])
						}
					}
				}
				if heapMax > 0 {
					m.JVMHeapUsedPct = (heapSum / heapMax) * 100
				}
				if count > 0 {
					m.CPUPercent = cpuSum / count
				}
			}
		}
	}

	return m, nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}

func intVal(v any) int {
	return int(int64Val(v))
}

func int64Val(v any) int64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	case json.Number:
		n, _ := t.Int64()
		return n
	default:
		return 0
	}
}

func floatVal(v any) float64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		n, _ := strconv.ParseFloat(t, 64)
		return n
	case json.Number:
		n, _ := t.Float64()
		return n
	default:
		return 0
	}
}

func boolVal(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "yes" || t == "1"
	default:
		return false
	}
}
