package es

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestAllMethodsNotConnectedAndHTTPErrors(t *testing.T) {
	c := NewClient()
	methods := []func() error{
		func() error { _, err := c.GetClusterInfo(); return err },
		func() error { _, err := c.GetClusterHealth(); return err },
		func() error { _, err := c.GetNodes(); return err },
		func() error { _, err := c.GetShards(""); return err },
		func() error { _, err := c.ListIndices("*"); return err },
		func() error { _, err := c.GetIndex("x"); return err },
		func() error { return c.CreateIndex("x", "") },
		func() error { return c.DeleteIndex("x") },
		func() error { _, err := c.GetIndexSettings("x"); return err },
		func() error { _, err := c.GetIndexMappings("x"); return err },
		func() error { return c.RefreshIndex("x") },
		func() error { return c.OpenIndex("x") },
		func() error { return c.CloseIndex("x") },
		func() error { return c.ForceMerge("x", 0) },
		func() error { _, err := c.Search("x", "*", 0, 10); return err },
		func() error { _, err := c.GetDocument("x", "1"); return err },
		func() error { return c.IndexDocument("x", "1", `{}`) },
		func() error { return c.DeleteDocument("x", "1") },
		func() error { _, err := c.DeleteByQuery("x", "*"); return err },
		func() error { _, err := c.ListAliases(); return err },
		func() error { _, err := c.ListTemplates(); return err },
		func() error { _, err := c.Cat("indices"); return err },
		func() error { _, err := c.GetLiveMetrics(); return err },
	}
	for i, fn := range methods {
		if err := fn(); err == nil {
			t.Fatalf("method %d expected not connected", i)
		}
	}

	// Root ok, everything else 500 / bad json
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		if r.URL.Path == "/_cluster/health" {
			http.Error(w, "fail", 500)
			return
		}
		http.Error(w, "fail", 500)
	}))
	defer srv.Close()
	c = connectToServer(t, srv, types.FlavorAuto)
	for i, fn := range []func() error{
		func() error { _, err := c.GetClusterHealth(); return err },
		func() error { _, err := c.GetNodes(); return err },
		func() error { _, err := c.GetShards("x"); return err },
		func() error { _, err := c.ListIndices("*"); return err },
		func() error { _, err := c.GetIndex("x"); return err },
		func() error { return c.CreateIndex("x", "{}") },
		func() error { return c.DeleteIndex("x") },
		func() error { _, err := c.GetIndexSettings("x"); return err },
		func() error { _, err := c.GetIndexMappings("x"); return err },
		func() error { return c.RefreshIndex("x") },
		func() error { return c.OpenIndex("x") },
		func() error { return c.CloseIndex("x") },
		func() error { return c.ForceMerge("x", 1) },
		func() error { _, err := c.Search("x", "*", 0, 10); return err },
		func() error { _, err := c.GetDocument("x", "1"); return err },
		func() error { return c.IndexDocument("x", "1", `{}`) },
		func() error { return c.DeleteDocument("x", "1") },
		func() error { _, err := c.DeleteByQuery("x", "status:x"); return err },
		func() error { _, err := c.ListAliases(); return err },
		func() error { _, err := c.ListTemplates(); return err },
		func() error { _, err := c.Cat("indices"); return err },
		func() error { _, err := c.GetLiveMetrics(); return err },
	} {
		if err := fn(); err == nil {
			t.Fatalf("http error method %d expected err", i)
		}
	}

	// Invalid root JSON for GetClusterInfo / Connect
	badRoot := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer badRoot.Close()
	u := strings.TrimPrefix(badRoot.URL, "http://")
	host, portStr, _ := strings.Cut(u, ":")
	port := 0
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	c2 := NewClient()
	if err := c2.Connect(types.Connection{Host: host, Port: port}); err == nil {
		t.Fatal("expected connect parse error")
	}
	// Manual base for GetClusterInfo
	c3 := NewClient()
	c3.baseURL = badRoot.URL
	c3.http = badRoot.Client()
	if _, err := c3.GetClusterInfo(); err == nil {
		t.Fatal("expected parse error")
	}

	// doUnlocked paths: auth + body variants during Connect with API key
	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "noauth", 401)
			return
		}
		rootOK(w, r)
	}))
	defer authSrv.Close()
	u = strings.TrimPrefix(authSrv.URL, "http://")
	host, portStr, _ = strings.Cut(u, ":")
	port = 0
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	c4 := NewClient()
	if err := c4.Connect(types.Connection{Host: host, Port: port, APIKey: "k"}); err != nil {
		t.Fatal(err)
	}
	// string body via doUnlocked - only used in Connect which uses nil body
	// Cover doUnlocked with client already connected by calling getClusterInfoLocked pattern
	// through Connect again
	if err := c4.Connect(types.Connection{Host: host, Port: port, Username: "u", Password: "p"}); err != nil {
		t.Fatal(err)
	}

	// GetClusterInfo with full version fields
	full := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "n", "cluster_name": "c", "cluster_uuid": "u", "tagline": "You Know, for Search",
			"version": map[string]any{
				"number": "8", "build_flavor": "f", "build_type": "t", "build_hash": "h",
				"build_date": "d", "lucene_version": "l", "distribution": "",
			},
		})
	}))
	defer full.Close()
	c5 := connectToServer(t, full, types.FlavorElasticsearch)
	if _, err := c5.GetClusterInfo(); err != nil {
		t.Fatal(err)
	}

	// GetDocument without source
	docSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"_index": "i", "_id": "1"})
	}))
	defer docSrv.Close()
	c6 := connectToServer(t, docSrv, types.FlavorAuto)
	if _, err := c6.GetDocument("i", "1"); err != nil {
		t.Fatal(err)
	}

	// ListAliases/Nodes empty arrays
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			rootOK(w, r)
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer empty.Close()
	c7 := connectToServer(t, empty, types.FlavorAuto)
	if _, err := c7.GetNodes(); err != nil {
		t.Fatal(err)
	}
	if _, err := c7.GetShards(""); err != nil {
		t.Fatal(err)
	}
	if _, err := c7.ListIndices("*"); err != nil {
		t.Fatal(err)
	}
	if _, err := c7.ListAliases(); err != nil {
		t.Fatal(err)
	}

	// Cat with pretty indent failure -> raw
	// already returns pretty or raw

	// TestConnection network error
	_, _, err := NewClient().TestConnection(types.Connection{Host: "127.0.0.1", Port: 1})
	if err == nil {
		t.Fatal("expected test connection error")
	}

	// GetLiveMetrics health error already covered; stats partial with empty nodes
	statsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			rootOK(w, r)
		case r.URL.Path == "/_cluster/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "green", "cluster_name": "c", "number_of_nodes": 1, "number_of_data_nodes": 1, "active_shards": 1, "unassigned_shards": 0})
		case r.URL.Path == "/_stats":
			_ = json.NewEncoder(w).Encode(map[string]any{"_all": map[string]any{"total": map[string]any{}}})
		case strings.HasPrefix(r.URL.Path, "/_nodes/stats"):
			_ = json.NewEncoder(w).Encode(map[string]any{"nodes": map[string]any{"n1": "bad"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer statsSrv.Close()
	c8 := connectToServer(t, statsSrv, types.FlavorAuto)
	if _, err := c8.GetLiveMetrics(); err != nil {
		t.Fatal(err)
	}
}

func TestDoUnlockedBodyVariants(t *testing.T) {
	// Connect holds write lock and uses doUnlocked - cover string/bytes/map via a custom approach:
	// set client fields and call doUnlocked directly
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := NewClient()
	c.baseURL = srv.URL
	c.http = srv.Client()
	c.apiKey = "k"
	if _, _, err := c.doUnlocked("POST", "/x", "hello"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.doUnlocked("POST", "/x", []byte("bytes")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.doUnlocked("POST", "/x", map[string]any{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.doUnlocked("POST", "/x", make(chan int)); err == nil {
		t.Fatal("marshal")
	}
	c.apiKey = ""
	c.user = "u"
	c.pass = "p"
	if _, _, err := c.doUnlocked("GET", "/", nil); err != nil {
		t.Fatal(err)
	}
	// error status
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", 400)
	}))
	defer errSrv.Close()
	c.baseURL = errSrv.URL
	c.http = errSrv.Client()
	if _, _, err := c.doUnlocked("GET", "/", nil); err == nil {
		t.Fatal("expected http error")
	}
	// invalid request URL
	c.baseURL = "://bad"
	if _, _, err := c.doUnlocked("GET", "/", nil); err == nil {
		t.Fatal("expected url error")
	}
}
