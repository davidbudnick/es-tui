package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunSeed(t *testing.T) {
	indices := map[string]bool{}
	docs := map[string]int{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodDelete:
			name := strings.TrimPrefix(path, "/")
			delete(indices, name)
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case r.Method == http.MethodPut && !strings.Contains(path, "/_doc/"):
			name := strings.TrimPrefix(path, "/")
			indices[name] = true
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case r.Method == http.MethodPut && strings.Contains(path, "/_doc/"):
			docs[path]++
			_, _ = w.Write([]byte(`{"result":"created"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/_refresh"):
			_, _ = w.Write([]byte(`{"_shards":{"successful":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if err := run(srv.URL, true); err != nil {
		t.Fatal(err)
	}
	if !indices["products"] || !indices["orders"] || !indices["logs-demo"] {
		t.Fatalf("indices=%v", indices)
	}
	if len(docs) < 20 {
		t.Fatalf("docs=%d", len(docs))
	}

	// put error status
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", 400)
	}))
	defer bad.Close()
	if err := run(bad.URL, false); err == nil {
		t.Fatal("expected error")
	}
}

func TestPutMarshal(t *testing.T) {
	// invalid body that can't marshal - channels
	// use a server that returns body
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	client := srv.Client()
	if err := put(client, srv.URL+"/x", map[string]any{"a": 1}); err != nil {
		t.Fatal(err)
	}
}
