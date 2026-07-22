package main

import (
	"bytes"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMainPaths(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPut:
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/_bulk":
			_, _ = w.Write([]byte(`{"errors":false}`))
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	oldArgs := os.Args
	oldExit := osExit
	oldOut := seedStdout
	oldErr := seedStderr
	var out, errBuf bytes.Buffer
	seedStdout = &out
	seedStderr = &errBuf
	exited := -1
	osExit = func(code int) { exited = code }
	os.Args = []string{"seed", "-addr", srv.URL, "-flush"}
	t.Cleanup(func() {
		os.Args = oldArgs
		osExit = oldExit
		seedStdout = oldOut
		seedStderr = oldErr
	})
	main()
	if exited != -1 {
		t.Fatalf("exit %d stderr=%s", exited, errBuf.String())
	}
	if !strings.Contains(out.String(), "seeding complete") && !strings.Contains(out.String(), "Seed") {
		// new message is "Done — seeding complete"
		if !strings.Contains(out.String(), "Done") {
			t.Fatalf("output=%q", out.String())
		}
	}

	os.Args = []string{"seed", "-addr", "http://127.0.0.1:1"}
	exited = -1
	errBuf.Reset()
	main()
	if exited != 1 {
		t.Fatalf("expected exit 1 got %d", exited)
	}

	if err := seedMain([]string{"-addr", srv.URL}); err != nil {
		t.Fatal(err)
	}
	_ = flag.CommandLine
}

func TestHTTPErrorBranches(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("err"))
	}))
	defer bad.Close()
	if err := put(bad.Client(), bad.URL+"/x", map[string]any{"a": 1}); err == nil {
		t.Fatal("put error")
	}
	if err := post(bad.Client(), bad.URL+"/x", map[string]any{"a": 1}); err == nil {
		t.Fatal("post error")
	}
	if err := bulkIndex(bad.Client(), bad.URL, "idx", []map[string]any{{"a": 1}}); err == nil {
		t.Fatal("bulk error")
	}

	// network
	if err := run("http://127.0.0.1:1", true); err == nil {
		t.Fatal("network")
	}
}
