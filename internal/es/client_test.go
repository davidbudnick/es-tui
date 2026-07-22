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
	if detectFlavor(types.ClusterInfo{Version: types.VersionInfo{Number: "8.0.0"}}) != types.FlavorElasticsearch {
		t.Fatal("es version")
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
		case strings.HasSuffix(r.URL.Path, "/_refresh") || strings.HasSuffix(r.URL.Path, "/_open") || strings.HasSuffix(r.URL.Path, "/_close"):
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
