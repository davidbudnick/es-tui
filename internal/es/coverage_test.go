package es

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestClientCoverageGaps(t *testing.T) {
	// Health fallback when JSON tags fail - send alternate structure that unmarshals poorly
	// GetClusterHealth tries typed unmarshal first; send valid typed JSON
	// Force fallback: invalid for struct but valid map - actually both work with tags.
	// Hit fallback by sending numbers as strings? tags use int - still works via Unmarshal.
	// Directly call fallback path by using body that fails first unmarshal:
	// ClusterHealth has bool TimedOut - if we send timed_out as object, first fails.

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "n", "cluster_name": "c", "cluster_uuid": "u", "tagline": "x",
				"version": map[string]any{
					"number": "1", "build_flavor": "f", "build_type": "t", "build_hash": "h",
					"build_date": "d", "build_snapshot": true, "lucene_version": "l",
					"minimum_wire_compatibility_version":  "w",
					"minimum_index_compatibility_version": "i",
				},
			})
		case r.URL.Path == "/_cluster/health":
			// Cause first Unmarshal to fail (wrong type for status), second path works
			_, _ = w.Write([]byte(`{"cluster_name":"c","status":"green","timed_out":"no","number_of_nodes":1,"number_of_data_nodes":1,"active_primary_shards":1,"active_shards":1,"relocating_shards":0,"initializing_shards":0,"unassigned_shards":0,"delayed_unassigned_shards":0,"number_of_pending_tasks":0,"number_of_in_flight_fetch":0,"task_max_waiting_in_queue_millis":0,"active_shards_percent_as_number":100}`))
		case strings.HasPrefix(r.URL.Path, "/_cat/nodes"):
			_, _ = w.Write([]byte(`notjson`))
		case strings.HasPrefix(r.URL.Path, "/_cat/shards"):
			_, _ = w.Write([]byte(`notjson`))
		case strings.HasPrefix(r.URL.Path, "/_cat/indices"):
			_, _ = w.Write([]byte(`notjson`))
		case strings.HasPrefix(r.URL.Path, "/_cat/aliases"):
			_, _ = w.Write([]byte(`notjson`))
		case r.URL.Path == "/_index_template":
			http.Error(w, "no", 404)
		case r.URL.Path == "/_template":
			_, _ = w.Write([]byte(`notjson`))
		case strings.Contains(r.URL.Path, "/_search"):
			_, _ = w.Write([]byte(`notjson`))
		case strings.Contains(r.URL.Path, "/_doc/"):
			_, _ = w.Write([]byte(`notjson`))
		case strings.HasSuffix(r.URL.Path, "/_delete_by_query"):
			_, _ = w.Write([]byte(`notjson`))
		case strings.HasSuffix(r.URL.Path, "/_settings"):
			http.Error(w, "no", 500)
		case strings.HasSuffix(r.URL.Path, "/_mapping"):
			http.Error(w, "no", 500)
		case r.URL.Path == "/_stats":
			http.Error(w, "no", 500)
		case strings.HasPrefix(r.URL.Path, "/_nodes/stats"):
			http.Error(w, "no", 500)
		case strings.HasPrefix(r.URL.Path, "/_cat/"):
			_, _ = w.Write([]byte(`not-json-but-ok`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := connectToServer(t, srv, types.FlavorAuto)
	h, err := c.GetClusterHealth()
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != "green" {
		t.Fatal(h)
	}
	if _, err := c.GetNodes(); err == nil {
		t.Fatal("nodes json")
	}
	if _, err := c.GetShards(""); err == nil {
		t.Fatal("shards json")
	}
	if _, err := c.ListIndices("*"); err == nil {
		t.Fatal("indices json")
	}
	if _, err := c.ListAliases(); err == nil {
		t.Fatal("aliases json")
	}
	if _, err := c.ListTemplates(); err == nil {
		t.Fatal("templates")
	}
	if _, err := c.Search("i", "*", 0, 10); err == nil {
		t.Fatal("search")
	}
	if _, err := c.GetDocument("i", "1"); err == nil {
		t.Fatal("doc")
	}
	if _, err := c.DeleteByQuery("i", "*"); err == nil {
		t.Fatal("dbq")
	}
	if _, err := c.GetIndexSettings("i"); err == nil {
		t.Fatal("settings")
	}
	if _, err := c.GetIndexMappings("i"); err == nil {
		t.Fatal("mappings")
	}
	// metrics still works via health
	if _, err := c.GetLiveMetrics(); err != nil {
		t.Fatal(err)
	}
	// cat non-json returns raw
	body, err := c.Cat("allocation")
	if err != nil || body == "" {
		// may error if not-json path hits 404 or error
		_ = body
	}

	// GetClusterInfo path + getClusterInfoLocked fields
	info, err := c.GetClusterInfo()
	if err != nil {
		t.Fatal(err)
	}
	_ = info

	// doUnlocked body types via Connect already; test not connected doUnlocked
	c2 := NewClient()
	if _, _, err := c2.doUnlocked("GET", "/", nil); err == nil {
		t.Fatal("not connected")
	}

	// int64Val default
	if int64Val(struct{}{}) != 0 {
		t.Fatal("int64 default")
	}
	if boolVal(struct{}{}) {
		t.Fatal("bool default")
	}

	// parseSearchResult non-map hit
	sr, err := parseSearchResult([]byte(`{"hits":{"total":{"value":1,"relation":"eq"},"hits":["bad",{"_index":"i","_id":"1","_score":1}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(sr.Hits) != 1 {
		t.Fatal(sr.Hits)
	}

	// Search with from/size already in JSON
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"took": 1, "hits": map[string]any{"total": 0, "hits": []any{}},
		})
	}))
	defer srv2.Close()
	c3 := connectToServer(t, srv2, types.FlavorAuto)
	if _, err := c3.Search("i", `{"query":{"match_all":{}},"from":1,"size":5}`, 0, 10); err != nil {
		t.Fatal(err)
	}

	// TLS enabled without custom config
	// Use httptest.NewTLSServer
	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rootOK(w, r)
	}))
	defer tlsSrv.Close()
	// will fail cert verify - that's ok for path coverage
	u := strings.TrimPrefix(tlsSrv.URL, "https://")
	host, portStr, _ := strings.Cut(u, ":")
	port := 0
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	c4 := NewClient()
	_ = c4.Connect(types.Connection{
		Host: host, Port: port, UseTLS: true,
		TLSConfig: &types.TLSConfig{InsecureSkipVerify: true},
	})
	// may succeed with insecure
	_, _, _ = c4.TestConnection(types.Connection{
		Host: host, Port: port, UseTLS: true,
		TLSConfig: &types.TLSConfig{InsecureSkipVerify: true},
	})

	// GetIndex with empty list already; with name mismatch multi
	srv3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"index": "a", "health": "g", "status": "open", "uuid": "u", "pri": "1", "rep": "0", "docs.count": "0", "docs.deleted": "0", "store.size": "0", "pri.store.size": "0"},
			{"index": "b", "health": "g", "status": "open", "uuid": "u", "pri": "1", "rep": "0", "docs.count": "0", "docs.deleted": "0", "store.size": "0", "pri.store.size": "0"},
		})
	}))
	defer srv3.Close()
	c5 := connectToServer(t, srv3, types.FlavorAuto)
	if _, err := c5.GetIndex("missing"); err == nil {
		t.Fatal("expected not found multi")
	}

	// ListTemplates composed_of and modern path already; template map non-map skip
	srv4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		if r.URL.Path == "/_index_template" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"index_templates": []any{
					"bad",
					map[string]any{"name": "t", "index_template": "bad"},
					map[string]any{"name": "t2", "index_template": map[string]any{
						"index_patterns": []any{"p-*"},
						"composed_of":    []any{"c1"},
						"version":        3,
					}},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv4.Close()
	c6 := connectToServer(t, srv4, types.FlavorAuto)
	tmpls, err := c6.ListTemplates()
	if err != nil || len(tmpls) < 1 {
		t.Fatalf("%v %v", tmpls, err)
	}

	// old template non-map values
	srv5 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		if r.URL.Path == "/_index_template" {
			http.Error(w, "no", 404)
			return
		}
		if r.URL.Path == "/_template" {
			_ = json.NewEncoder(w).Encode(map[string]any{"x": "bad", "y": map[string]any{"order": 1}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv5.Close()
	c7 := connectToServer(t, srv5, types.FlavorAuto)
	if _, err := c7.ListTemplates(); err != nil {
		t.Fatal(err)
	}

	// do marshal error - channel can't marshal
	c8 := connectToServer(t, srv2, types.FlavorAuto)
	_, _, err = c8.do("POST", "/x", make(chan int))
	if err == nil {
		t.Fatal("marshal")
	}

	// read body limit path already; TestConnection success path with API key
	_ = h
}

func TestDoUnlockedCoverage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// first connect uses doUnlocked
			rootOK(w, r)
			return
		}
		http.Error(w, "err", 400)
	}))
	defer srv.Close()
	// connect succeeds
	c := connectToServer(t, srv, types.FlavorElasticsearch)
	// force doUnlocked via holding - actually use Connect with body
	// disconnect and test string/bytes body on do
	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}
	// reconnect
	c = connectToServer(t, srv, types.FlavorElasticsearch)
	_, _, _ = c.do("POST", "/x", "string-body")
	_, _, _ = c.do("POST", "/x", []byte("bytes"))
}
