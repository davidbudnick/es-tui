package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunSeedRich(t *testing.T) {
	indices := map[string]bool{}
	bulkDocs := 0
	aliases := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case r.Method == http.MethodPut && path == "/_aliases":
			aliases++
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case r.Method == http.MethodPut && !strings.Contains(path, "/_doc"):
			name := strings.TrimPrefix(path, "/")
			indices[name] = true
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case r.Method == http.MethodPost && path == "/_bulk":
			// count ndjson pairs
			var body []byte
			buf := make([]byte, 64*1024)
			for {
				n, err := r.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if err != nil {
					break
				}
			}
			lines := 0
			for _, line := range strings.Split(string(body), "\n") {
				if strings.TrimSpace(line) != "" {
					lines++
				}
			}
			bulkDocs += lines / 2
			_, _ = w.Write([]byte(`{"errors":false,"items":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/_refresh"):
			_, _ = w.Write([]byte(`{"_shards":{"successful":1}}`))
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	oldOut := seedStdout
	seedStdout = ioDiscard{}
	t.Cleanup(func() { seedStdout = oldOut })

	if err := run(srv.URL, true); err != nil {
		t.Fatal(err)
	}
	for _, name := range demoIndexNames() {
		if !indices[name] {
			t.Fatalf("missing index create for %s: %v", name, indices)
		}
	}
	if bulkDocs < 50 {
		t.Fatalf("expected many bulk docs, got %d", bulkDocs)
	}
	if aliases == 0 {
		t.Fatal("expected alias call")
	}
}

func TestSeedMainAndHelpers(t *testing.T) {
	if err := seedMain([]string{"-h"}); err == nil {
		// help returns error from flag set
	}
	if err := seedMain([]string{"-nope"}); err == nil {
		t.Fatal("expected flag error")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_bulk" {
			_, _ = w.Write([]byte(`{"errors":true,"items":[{"index":{"error":"x"}}]}`))
			return
		}
		if r.Method == http.MethodPut {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(404)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	oldOut := seedStdout
	seedStdout = ioDiscard{}
	t.Cleanup(func() { seedStdout = oldOut })

	// products create ok then bulk fails
	if err := run(srv.URL, false); err == nil {
		t.Fatal("expected bulk errors")
	}

	if err := put(http.DefaultClient, "://bad", map[string]any{}); err == nil {
		t.Fatal("bad url")
	}
	if err := put(http.DefaultClient, "http://127.0.0.1:1", map[string]any{"ch": make(chan int)}); err == nil {
		t.Fatal("marshal")
	}
	if truncate("hi", 10) != "hi" || !strings.HasSuffix(truncate(strings.Repeat("x", 50), 10), "...") {
		t.Fatal("truncate")
	}
	_ = flag.ErrHelp
	_ = json.Number("1")
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
