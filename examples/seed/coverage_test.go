package main

import (
	"bytes"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMainAndSeedMain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut, http.MethodPost:
			_, _ = w.Write([]byte(`{}`))
		case http.MethodDelete:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`missing`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if err := seedMain([]string{"-addr", srv.URL}); err != nil {
		t.Fatal(err)
	}
	if err := seedMain([]string{"-addr", srv.URL, "-flush"}); err != nil {
		t.Fatal(err)
	}
	if err := seedMain([]string{"-h"}); err == nil {
		// help returns ErrHelp
		if err != flag.ErrHelp && err == nil {
			// ContinueOnError returns error for -h
		}
	}
	// unknown flag
	if err := seedMain([]string{"-nope"}); err == nil {
		t.Fatal("expected flag error")
	}

	// main success
	oldArgs := os.Args
	oldExit := osExit
	oldOut := seedStdout
	oldErr := seedStderr
	var out, errBuf bytes.Buffer
	seedStdout = &out
	seedStderr = &errBuf
	exited := -1
	osExit = func(code int) { exited = code }
	os.Args = []string{"seed", "-addr", srv.URL}
	t.Cleanup(func() {
		os.Args = oldArgs
		osExit = oldExit
		seedStdout = oldOut
		seedStderr = oldErr
	})
	main()
	if exited != -1 {
		t.Fatalf("exit %d: %s", exited, errBuf.String())
	}
	if out.Len() == 0 {
		t.Fatal("expected output")
	}

	// main failure
	os.Args = []string{"seed", "-addr", "http://127.0.0.1:1"}
	exited = -1
	errBuf.Reset()
	main()
	if exited != 1 {
		t.Fatalf("expected exit 1 got %d", exited)
	}

	// put paths
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("err"))
	}))
	defer bad.Close()
	if err := put(bad.Client(), bad.URL+"/x", map[string]any{"a": 1}); err == nil {
		t.Fatal("expected error")
	}
	if err := put(http.DefaultClient, "://bad", map[string]any{}); err == nil {
		t.Fatal("bad url")
	}
	if err := put(http.DefaultClient, "http://127.0.0.1:1", map[string]any{"ch": make(chan int)}); err == nil {
		t.Fatal("marshal")
	}
	if err := run("http://127.0.0.1:1", true); err == nil {
		t.Fatal("network")
	}
	_ = io.Discard
}
