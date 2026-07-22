package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSeedCreateFailures(t *testing.T) {
	oldOut := seedStdout
	seedStdout = ioDiscard{}
	t.Cleanup(func() { seedStdout = oldOut })

	// fail first index create
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", 500)
	}))
	defer srv.Close()
	if err := run(srv.URL, false); err == nil {
		t.Fatal("expected create fail")
	}

	// products ok, customers create fails
	n := 0
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/_bulk" {
			_, _ = w.Write([]byte(`{"errors":false}`))
			return
		}
		if r.Method == http.MethodPut {
			n++
			if n == 1 {
				_, _ = w.Write([]byte(`{}`))
				return
			}
			http.Error(w, "no", 500)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv2.Close()
	if err := run(srv2.URL, false); err == nil {
		t.Fatal("expected customers fail")
	}
}
