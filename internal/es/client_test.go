package es

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestHelpers(t *testing.T) {
	if str(nil) != "" || str("a") != "a" || str(float64(1.5)) != "1.5" {
		t.Fatal("str")
	}
	if str(json.Number("3")) != "3" {
		t.Fatal("str number")
	}
	if str(true) != "true" {
		t.Fatal("str default")
	}
	if intVal(nil) != 0 || intVal(3.0) != 3 || intVal("4") != 4 || intVal(json.Number("5")) != 5 {
		t.Fatal("intVal")
	}
	if int64Val(int(7)) != 7 || int64Val(int64(8)) != 8 {
		t.Fatal("int64Val")
	}
	if floatVal(nil) != 0 || floatVal(1.5) != 1.5 || floatVal(2) != 2 || floatVal(int64(3)) != 3 {
		t.Fatal("floatVal")
	}
	if floatVal("1.25") != 1.25 || floatVal(json.Number("2.5")) != 2.5 {
		t.Fatal("floatVal str")
	}
	if floatVal(true) != 0 {
		t.Fatal("floatVal default")
	}
	if boolVal(nil) || !boolVal(true) || !boolVal("true") || !boolVal("yes") || !boolVal("1") || boolVal("no") {
		t.Fatal("boolVal")
	}
	if truncateErr([]byte(strings.Repeat("x", 400))) == "" {
		t.Fatal("truncate")
	}
	if truncateErr([]byte("short")) != "short" {
		t.Fatal("truncate short")
	}
}

func TestDetectFlavor(t *testing.T) {
	if detectFlavor(types.ClusterInfo{Tagline: "You Know, for Search"}) != types.FlavorElasticsearch {
		t.Fatal("es tagline")
	}
	if detectFlavor(types.ClusterInfo{Tagline: "The OpenSearch Project"}) != types.FlavorOpenSearch {
		t.Fatal("os tagline")
	}
	if detectFlavor(types.ClusterInfo{Version: types.VersionInfo{Distribution: "opensearch"}}) != types.FlavorOpenSearch {
		t.Fatal("os distribution")
	}
	if detectFlavor(types.ClusterInfo{Version: types.VersionInfo{Distribution: "OpenSearch"}}) != types.FlavorOpenSearch {
		t.Fatal("os distribution case")
	}
	if detectFlavor(types.ClusterInfo{Version: types.VersionInfo{Number: "8.0.0"}}) != types.FlavorElasticsearch {
		t.Fatal("es version")
	}
	if detectFlavor(types.ClusterInfo{Version: types.VersionInfo{BuildFlavor: "default", Number: "8.17.0"}}) != types.FlavorElasticsearch {
		t.Fatal("es build_flavor")
	}
	if detectFlavor(types.ClusterInfo{}) != types.FlavorAuto {
		t.Fatal("auto")
	}
}

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient()
	// parse host/port from URL
	u := strings.TrimPrefix(srv.URL, "http://")
	parts := strings.Split(u, ":")
	port := 80
	if len(parts) == 2 {
		var p int
		_, _ = parsePort(parts[1], &p)
		port = p
	}
	conn := types.Connection{Host: parts[0], Port: port, Flavor: types.FlavorAuto}
	// Use Connect which hits /
	// Override by manually setting for some tests
	_ = conn
	return srv, c
}

func parsePort(s string, p *int) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	*p = n
	return n, nil
}

func connectToServer(t *testing.T, srv *httptest.Server, flavor types.Flavor) *Client {
	t.Helper()
	u := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, _ := strings.Cut(u, ":")
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	c := NewClient()
	err := c.Connect(types.Connection{
		Host: host, Port: port, Flavor: flavor,
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return c
}

func rootOK(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":         "node-1",
			"cluster_name": "test-cluster",
			"cluster_uuid": "uuid",
			"tagline":      "You Know, for Search",
			"version": map[string]any{
				"number":         "8.17.0",
				"build_flavor":   "default",
				"build_type":     "docker",
				"build_hash":     "abc",
				"build_date":     "2024",
				"lucene_version": "9.0",
			},
		})
		return
	}
	http.NotFound(w, r)
}

func TestClientConnectAndBasics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			rootOK(w, r)
		case r.URL.Path == "/_cluster/health":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"cluster_name":                     "test-cluster",
				"status":                           "green",
				"number_of_nodes":                  1,
				"number_of_data_nodes":             1,
				"active_primary_shards":            1,
				"active_shards":                    1,
				"relocating_shards":                0,
				"initializing_shards":              0,
				"unassigned_shards":                0,
				"delayed_unassigned_shards":        0,
				"number_of_pending_tasks":          0,
				"number_of_in_flight_fetch":        0,
				"task_max_waiting_in_queue_millis": 0,
				"active_shards_percent_as_number":  100.0,
			})
		case strings.HasPrefix(r.URL.Path, "/_cat/nodes"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"name": "n1", "id": "id1", "ip": "127.0.0.1", "host": "localhost",
				"node.role": "dimr", "master": "*", "heap.percent": "50", "ram.percent": "60",
				"cpu": "10", "load_1m": "0.1", "version": "8.17.0",
				"disk.used_percent": "20", "disk.total": "100gb", "disk.used": "20gb", "disk.avail": "80gb",
			}})
		case strings.HasPrefix(r.URL.Path, "/_cat/shards"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"index": "products", "shard": "0", "prirep": "p", "state": "STARTED",
				"docs": "4", "store": "1kb", "ip": "127.0.0.1", "node": "n1",
			}})
		case strings.HasPrefix(r.URL.Path, "/_cat/indices"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"index": "products", "health": "green", "status": "open", "uuid": "u1",
				"pri": "1", "rep": "0", "docs.count": "4", "docs.deleted": "0",
				"store.size": "1kb", "pri.store.size": "1kb",
			}})
		case strings.HasPrefix(r.URL.Path, "/_cat/aliases"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"alias": "prod", "index": "products", "filter": "-", "routing.index": "-",
				"routing.search": "-", "is_write_index": "true",
			}})
		case r.URL.Path == "/_index_template":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"index_templates": []map[string]any{{
					"name": "logs",
					"index_template": map[string]any{
						"index_patterns": []any{"logs-*"},
						"composed_of":    []any{},
						"version":        1,
					},
				}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/newidx":
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/newidx":
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case strings.HasSuffix(r.URL.Path, "/_settings"):
			_, _ = w.Write([]byte(`{"products":{"settings":{}}}`))
		case strings.HasSuffix(r.URL.Path, "/_mapping"):
			_, _ = w.Write([]byte(`{"products":{"mappings":{}}}`))
		case strings.HasSuffix(r.URL.Path, "/_refresh") || strings.HasSuffix(r.URL.Path, "/_open") || strings.HasSuffix(r.URL.Path, "/_close") || strings.HasSuffix(r.URL.Path, "/_forcemerge"):
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case strings.Contains(r.URL.Path, "/_search") || r.URL.Path == "/_search":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"took": 5, "timed_out": false,
				"hits": map[string]any{
					"total":     map[string]any{"value": 1, "relation": "eq"},
					"max_score": 1.0,
					"hits": []map[string]any{{
						"_index": "products", "_id": "1", "_score": 1.0,
						"_source": map[string]any{"name": "Widget"},
					}},
				},
				"aggregations": map[string]any{"x": map[string]any{"value": 1}},
			})
		case strings.Contains(r.URL.Path, "/_doc/"):
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"_index": "products", "_id": "1",
					"_source": map[string]any{"name": "Widget"},
				})
			} else if r.Method == http.MethodDelete {
				_, _ = w.Write([]byte(`{"result":"deleted"}`))
			} else {
				_, _ = w.Write([]byte(`{"result":"created"}`))
			}
		case strings.HasSuffix(r.URL.Path, "/_doc"):
			_, _ = w.Write([]byte(`{"result":"created"}`))
		case strings.HasSuffix(r.URL.Path, "/_delete_by_query"):
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": 2})
		case r.URL.Path == "/_stats":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"_all": map[string]any{
					"total": map[string]any{
						"docs":     map[string]any{"count": 10},
						"store":    map[string]any{"size_in_bytes": 1024},
						"search":   map[string]any{"query_total": 100, "query_time_in_millis": 50},
						"indexing": map[string]any{"index_total": 20},
					},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/_nodes/stats"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"nodes": map[string]any{
					"n1": map[string]any{
						"jvm": map[string]any{
							"mem": map[string]any{"heap_used_in_bytes": 50, "heap_max_in_bytes": 100},
						},
						"os": map[string]any{
							"cpu": map[string]any{"percent": 25},
						},
					},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/_cat/"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"ok": true}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := connectToServer(t, srv, types.FlavorAuto)
	if !c.IsConnected() {
		t.Fatal("not connected")
	}
	if c.Flavor() != types.FlavorElasticsearch {
		t.Fatalf("flavor=%s", c.Flavor())
	}

	info, err := c.GetClusterInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.ClusterName != "test-cluster" {
		t.Fatal(info)
	}

	health, err := c.GetClusterHealth()
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "green" {
		t.Fatal(health)
	}

	nodes, err := c.GetNodes()
	if err != nil || len(nodes) != 1 {
		t.Fatalf("nodes=%v err=%v", nodes, err)
	}
	shards, err := c.GetShards("products")
	if err != nil || len(shards) != 1 {
		t.Fatalf("shards=%v err=%v", shards, err)
	}
	shards, err = c.GetShards("")
	if err != nil {
		t.Fatal(err)
	}

	indices, err := c.ListIndices("*")
	if err != nil || len(indices) != 1 {
		t.Fatalf("indices=%v err=%v", indices, err)
	}
	indices, err = c.ListIndices("products")
	if err != nil {
		t.Fatal(err)
	}
	idx, err := c.GetIndex("products")
	if err != nil || idx.Name != "products" {
		t.Fatalf("get index: %+v %v", idx, err)
	}

	if err := c.CreateIndex("newidx", `{"settings":{}}`); err != nil {
		t.Fatal(err)
	}
	if err := c.CreateIndex("newidx", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteIndex("newidx"); err != nil {
		t.Fatal(err)
	}
	s, err := c.GetIndexSettings("products")
	if err != nil || s == "" {
		t.Fatal(err)
	}
	m, err := c.GetIndexMappings("products")
	if err != nil || m == "" {
		t.Fatal(err)
	}
	if err := c.RefreshIndex("products"); err != nil {
		t.Fatal(err)
	}
	if err := c.OpenIndex("products"); err != nil {
		t.Fatal(err)
	}
	if err := c.CloseIndex("products"); err != nil {
		t.Fatal(err)
	}
	if err := c.ForceMerge("products", 0); err != nil {
		t.Fatal(err)
	}
	if err := c.ForceMerge("products", 1); err != nil {
		t.Fatal(err)
	}

	sr, err := c.Search("products", "", 0, 10)
	if err != nil || sr.Total != 1 {
		t.Fatalf("search: %+v %v", sr, err)
	}
	sr, err = c.Search("products", "name:Widget", -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	sr, err = c.Search("", `{"query":{"match_all":{}}}`, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search("products", "{bad", 0, 10); err == nil {
		t.Fatal("expected bad json error")
	}

	doc, err := c.GetDocument("products", "1")
	if err != nil || doc.ID != "1" {
		t.Fatal(err)
	}
	if err := c.IndexDocument("products", "1", `{"name":"x"}`); err != nil {
		t.Fatal(err)
	}
	if err := c.IndexDocument("products", "", `{"name":"y"}`); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteDocument("products", "1"); err != nil {
		t.Fatal(err)
	}
	n, err := c.DeleteByQuery("products", "*")
	if err != nil || n != 2 {
		t.Fatalf("dbq %d %v", n, err)
	}
	n, err = c.DeleteByQuery("products", "")
	if err != nil {
		t.Fatal(err)
	}
	n, err = c.DeleteByQuery("products", `{"query":{"match_all":{}}}`)
	if err != nil {
		t.Fatal(err)
	}
	n, err = c.DeleteByQuery("products", "status:pending")
	if err != nil {
		t.Fatal(err)
	}

	aliases, err := c.ListAliases()
	if err != nil || len(aliases) != 1 {
		t.Fatal(err)
	}
	tmpls, err := c.ListTemplates()
	if err != nil || len(tmpls) != 1 {
		t.Fatal(err)
	}

	body, err := c.Cat("indices")
	if err != nil || body == "" {
		t.Fatal(err)
	}
	body, err = c.Cat("_cat/nodes")
	if err != nil {
		t.Fatal(err)
	}
	body, err = c.Cat("allocation?v")
	if err != nil {
		t.Fatal(err)
	}

	metrics, err := c.GetLiveMetrics()
	if err != nil {
		t.Fatal(err)
	}
	if metrics.DocsCount != 10 || metrics.JVMHeapUsedPct != 50 {
		t.Fatalf("%+v", metrics)
	}

	// TestConnection
	u := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, _ := strings.Cut(u, ":")
	port := 0
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	lat, tinfo, err := c.TestConnection(types.Connection{Host: host, Port: port})
	if err != nil || tinfo.ClusterName != "test-cluster" || lat < 0 {
		t.Fatalf("test: %v %+v", err, tinfo)
	}

	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if c.IsConnected() {
		t.Fatal("still connected")
	}
	if _, _, err := c.do("GET", "/", nil); err == nil {
		t.Fatal("expected not connected")
	}
}

func TestClientAuthAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "unauthorized", 401)
			return
		}
		if r.URL.Path == "/" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "n", "cluster_name": "c", "tagline": "The OpenSearch Project: https://opensearch.org/",
				"version": map[string]any{"number": "2.18.0", "distribution": "opensearch"},
			})
			return
		}
		http.Error(w, "nope", 500)
	}))
	defer srv.Close()

	u := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, _ := strings.Cut(u, ":")
	port := 0
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}

	c := NewClient()
	err := c.Connect(types.Connection{Host: host, Port: port, Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Flavor() != types.FlavorOpenSearch {
		t.Fatalf("flavor %s", c.Flavor())
	}
	if _, err := c.ListIndices("*"); err == nil {
		t.Fatal("expected 500")
	}

	// API key
	c2 := NewClient()
	err = c2.Connect(types.Connection{Host: host, Port: port, APIKey: "abc"})
	if err != nil {
		t.Fatal(err)
	}

	// Connect failure without auth
	c3 := NewClient()
	if err := c3.Connect(types.Connection{Host: host, Port: port}); err == nil {
		t.Fatal("expected auth failure")
	}

	// Template fallback
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			rootOK(w, r)
		case "/_index_template":
			http.Error(w, "no", 404)
		case "/_template":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"old": map[string]any{
					"order":          1,
					"version":        2,
					"index_patterns": []any{"old-*"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv2.Close()
	c4 := connectToServer(t, srv2, types.FlavorElasticsearch)
	tmpls, err := c4.ListTemplates()
	if err != nil || len(tmpls) != 1 || tmpls[0].Name != "old" {
		t.Fatalf("%+v %v", tmpls, err)
	}

	// GetIndex not found
	srv3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv3.Close()
	c5 := connectToServer(t, srv3, types.FlavorAuto)
	if _, err := c5.GetIndex("missing"); err == nil {
		t.Fatal("expected not found")
	}

	// parseSearchResult total as float64
	srv4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"took": 1, "hits": map[string]any{
				"total": 2.0, "max_score": 1.0,
				"hits": []any{map[string]any{"_index": "i", "_id": "1", "_score": 1.0}},
			},
		})
	}))
	defer srv4.Close()
	c6 := connectToServer(t, srv4, types.FlavorAuto)
	sr, err := c6.Search("i", "*", 0, 10)
	if err != nil || sr.Total != 2 {
		t.Fatalf("%+v %v", sr, err)
	}

	// health fallback path - invalid for direct unmarshal but valid map
	// covered by normal path already

	// TLS build error
	c7 := NewClient()
	err = c7.Connect(types.Connection{
		Host: "localhost", Port: 1, UseTLS: true,
		TLSConfig: &types.TLSConfig{CertFile: "/nope", KeyFile: "/nope"},
	})
	if err == nil {
		t.Fatal("expected tls error")
	}
	_, _, err = c7.TestConnection(types.Connection{
		Host: "localhost", Port: 1, UseTLS: true,
		TLSConfig: &types.TLSConfig{CertFile: "/nope", KeyFile: "/nope"},
	})
	if err == nil {
		t.Fatal("expected tls error")
	}

	// do with body types
	srv5 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv5.Close()
	c8 := connectToServer(t, srv5, types.FlavorAuto)
	_, _, err = c8.do("POST", "/x", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c8.do("POST", "/x", map[string]any{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseSearchResultAndGetIndexSingle(t *testing.T) {
	// single index wildcard match
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/_cat/indices") {
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"index": "logs-1", "health": "yellow", "status": "open", "uuid": "u",
				"pri": "1", "rep": "0", "docs.count": "0", "docs.deleted": "0",
				"store.size": "0", "pri.store.size": "0",
			}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := connectToServer(t, srv, types.FlavorAuto)
	idx, err := c.GetIndex("logs-*")
	if err != nil || idx.Name != "logs-1" {
		t.Fatalf("%+v %v", idx, err)
	}

	// invalid search JSON response
	if _, err := parseSearchResult([]byte("notjson")); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestClientNewMethods(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			if auth := r.Header.Get("Authorization"); auth != "" && !strings.HasPrefix(auth, "Bearer ") && !strings.HasPrefix(auth, "ApiKey ") && !strings.HasPrefix(auth, "Basic ") {
				http.Error(w, "bad auth", 401)
				return
			}
			rootOK(w, r)
		case strings.HasSuffix(r.URL.Path, "/_count"):
			_ = json.NewEncoder(w).Encode(map[string]any{"count": 42})
		case strings.Contains(r.URL.Path, "/_explain/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"matched":     true,
				"explanation": map[string]any{"value": 1.0, "description": "match"},
			})
		case r.URL.Path == "/_reindex":
			_ = json.NewEncoder(w).Encode(map[string]any{"task": "node:123"})
		case strings.HasPrefix(r.URL.Path, "/_cat/allocation"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"shards": "5", "disk.indices": "1b", "disk.used": "2b", "disk.avail": "3b",
				"disk.total": "5b", "disk.percent": "40", "host": "h", "ip": "1.1.1.1", "node": "n1",
			}})
		case r.URL.Path == "/_tasks":
			if strings.HasSuffix(r.URL.Path, "/_cancel") {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"nodes": map[string]any{
					"nodeA": map[string]any{
						"tasks": map[string]any{
							"nodeA:1": map[string]any{
								"action": "indices:data/write/reindex", "type": "transport",
								"start_time_in_millis": 1, "running_time_in_nanos": 2,
								"cancellable": true, "node": "nodeA", "description": "reindex",
							},
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/_cancel"):
			_ = json.NewEncoder(w).Encode(map[string]any{"nodes": map[string]any{}})
		case strings.HasPrefix(r.URL.Path, "/_cat/plugins"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"name": "n1", "component": "analysis-icu", "version": "8.0",
			}})
		case r.URL.Path == "/_cluster/settings":
			_, _ = w.Write([]byte(`{"persistent":{}}`))
		case r.URL.Path == "/_data_stream":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data_streams": []map[string]any{{
					"name": "logs", "timestamp_field": map[string]any{"name": "@timestamp"},
					"indices":    []any{map[string]any{"index_name": "logs-0001"}},
					"generation": 1, "status": "GREEN", "template": "logs-tpl",
				}},
			})
		case r.URL.Path == "/_snapshot":
			_ = json.NewEncoder(w).Encode(map[string]any{"repo1": map[string]any{"type": "fs"}})
		case r.URL.Path == "/_snapshot/repo1/_all":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"snapshots": []map[string]any{{
					"snapshot": "snap1", "repository": "repo1", "state": "SUCCESS",
					"start_time": "t1", "end_time": "t2",
					"indices": []any{"idx1", "idx2"},
				}},
			})
		case strings.Contains(r.URL.Path, "/_search"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"took": 1, "hits": map[string]any{
					"total": map[string]any{"value": 1, "relation": "eq"},
					"hits": []any{map[string]any{
						"_index": "products", "_id": "1", "_score": 1.0,
						"_source": map[string]any{"name": "a"},
					}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Bearer auth connect
	u := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, _ := strings.Cut(u, ":")
	port := 0
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	c := NewClient()
	if err := c.Connect(types.Connection{Host: host, Port: port, BearerToken: "tok", ReadOnly: true}); err != nil {
		t.Fatal(err)
	}
	if !c.IsReadOnly() {
		t.Fatal("expected read-only")
	}

	n, err := c.Count("products", "")
	if err != nil || n != 42 {
		t.Fatalf("count: %v %d", err, n)
	}
	n, err = c.Count("products", "name:a")
	if err != nil || n != 42 {
		t.Fatal(err)
	}
	n, err = c.Count("products", `{"query":{"match_all":{}}}`)
	if err != nil || n != 42 {
		t.Fatal(err)
	}

	er, err := c.Explain("products", "1", "name:a")
	if err != nil || !er.Matched || er.Explanation == "" {
		t.Fatalf("explain: %+v %v", er, err)
	}
	er, err = c.Explain("products", "1", "")
	if err != nil || !er.Matched {
		t.Fatal(err)
	}
	er, err = c.Explain("products", "1", `{"match_all":{}}`)
	if err != nil {
		t.Fatal(err)
	}

	task, err := c.Reindex(`{"source":{"index":"a"},"dest":{"index":"b"}}`)
	if err != nil || task != "node:123" {
		t.Fatalf("reindex: %q %v", task, err)
	}

	alloc, err := c.ListAllocation()
	if err != nil || len(alloc) != 1 || alloc[0].Node != "n1" {
		t.Fatalf("alloc: %+v %v", alloc, err)
	}

	tasks, err := c.ListTasks()
	if err != nil || len(tasks) != 1 || tasks[0].ID != "nodeA:1" {
		t.Fatalf("tasks: %+v %v", tasks, err)
	}
	if err := c.CancelTask("nodeA:1"); err != nil {
		t.Fatal(err)
	}

	plugins, err := c.ListPlugins()
	if err != nil || len(plugins) != 1 || plugins[0].Component != "analysis-icu" {
		t.Fatalf("plugins: %+v %v", plugins, err)
	}

	settings, err := c.GetClusterSettings()
	if err != nil || settings == "" {
		t.Fatal(err)
	}

	streams, err := c.ListDataStreams()
	if err != nil || len(streams) != 1 || streams[0].Name != "logs" || streams[0].TimestampField != "@timestamp" {
		t.Fatalf("streams: %+v %v", streams, err)
	}

	snaps, err := c.ListSnapshots("")
	if err != nil || len(snaps) != 1 || snaps[0].Snapshot != "snap1" {
		t.Fatalf("snaps: %+v %v", snaps, err)
	}
	snaps, err = c.ListSnapshots("repo1")
	if err != nil || len(snaps) != 1 {
		t.Fatal(err)
	}

	docs, err := c.ExportDocs("products", "", 10)
	if err != nil || len(docs) != 1 {
		t.Fatalf("export: %+v %v", docs, err)
	}
	docs, err = c.ExportDocs("products", "", 10000)
	if err != nil || len(docs) != 1 {
		t.Fatal(err)
	}

	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if c.IsReadOnly() {
		t.Fatal("read-only after disconnect")
	}
}

func TestListTasksFlatAndDataStream404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			rootOK(w, r)
		case "/_tasks":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tasks": []any{map[string]any{
					"id": "t1", "action": "a", "type": "t", "node": "n",
				}},
			})
		case "/_data_stream":
			http.Error(w, "no", 404)
		case "/_snapshot":
			_ = json.NewEncoder(w).Encode(map[string]any{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := connectToServer(t, srv, types.FlavorAuto)
	tasks, err := c.ListTasks()
	if err != nil || len(tasks) != 1 || tasks[0].ID != "t1" {
		t.Fatalf("%+v %v", tasks, err)
	}
	streams, err := c.ListDataStreams()
	if err != nil || len(streams) != 0 {
		t.Fatalf("%+v %v", streams, err)
	}
	snaps, err := c.ListSnapshots("")
	if err != nil || len(snaps) != 0 {
		t.Fatalf("%+v %v", snaps, err)
	}
}

func TestBearerAuthPriority(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		rootOK(w, r)
	}))
	defer srv.Close()
	u := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, _ := strings.Cut(u, ":")
	port := 0
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}

	// API key wins over bearer
	c := NewClient()
	if err := c.Connect(types.Connection{Host: host, Port: port, APIKey: "k", BearerToken: "b", Username: "u", Password: "p"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotAuth, "ApiKey ") {
		t.Fatalf("auth %q", gotAuth)
	}

	// Bearer over basic
	c2 := NewClient()
	if err := c2.Connect(types.Connection{Host: host, Port: port, BearerToken: "btok", Username: "u", Password: "p"}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer btok" {
		t.Fatalf("auth %q", gotAuth)
	}

	// TestConnection with bearer
	_, _, err := NewClient().TestConnection(types.Connection{Host: host, Port: port, BearerToken: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer t" {
		t.Fatalf("test auth %q", gotAuth)
	}
}
