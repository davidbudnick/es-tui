package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestLooksLikeSemver(t *testing.T) {
	if !looksLikeSemver("1.2.3") || !looksLikeSemver("v1.2.3") || !looksLikeSemver("1.2.3-rc.1") {
		t.Fatal("expected semver")
	}
	if looksLikeSemver("dev") || looksLikeSemver("ci") || looksLikeSemver("latest") || looksLikeSemver("") {
		t.Fatal("expected non-semver")
	}
}

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.0.1", "1.0.0", true},
		{"1.0.0", "v1.0.0", false},
		{"v2.0.0", "1.9.9", true},
		{"v1.0.0", "1.0.1", false},
		{"v1.1.0", "1.0.9", true},
		{"v1.0.0-rc.1", "0.9.0", true},
		{"not-a-version", "also-not", true},
		{"same", "same", false},
	}
	for _, tc := range cases {
		if got := isNewerVersion(tc.latest, tc.current); got != tc.want {
			t.Fatalf("isNewerVersion(%q,%q)=%v want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	if p := parseSemver("v1.2.3"); p == nil || p[0] != 1 || p[1] != 2 || p[2] != 3 {
		t.Fatalf("%v", p)
	}
	if p := parseSemver("1.2.3-beta"); p == nil || p[2] != 3 {
		t.Fatalf("%v", p)
	}
	if parseSemver("1.2") != nil || parseSemver("a.b.c") != nil {
		t.Fatal("expected nil")
	}
}

func TestCheckForUpdateSkipsDev(t *testing.T) {
	for _, v := range []string{"", "dev", "ci", "latest"} {
		msg := checkForUpdate(v)
		if msg.LatestVersion != "" || msg.Err != nil {
			t.Fatalf("version %q: %+v", v, msg)
		}
	}
}

func TestCheckForUpdateHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/davidbudnick/es-tui/releases/latest" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v9.9.9"})
	}))
	defer srv.Close()

	oldBase, oldClient := updateAPIBase, updateHTTPClient
	updateAPIBase = srv.URL
	updateHTTPClient = srv.Client()
	defer func() {
		updateAPIBase = oldBase
		updateHTTPClient = oldClient
	}()

	msg := checkForUpdate("1.0.0")
	if msg.Err != nil {
		t.Fatal(msg.Err)
	}
	if msg.LatestVersion != "v9.9.9" {
		t.Fatalf("got %q", msg.LatestVersion)
	}
	if msg.UpgradeCmd == "" {
		t.Fatal("expected upgrade hint")
	}

	// already latest
	msg = checkForUpdate("v9.9.9")
	if msg.LatestVersion != "" || msg.Err != nil {
		t.Fatalf("%+v", msg)
	}

	// API error
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bad.Close()
	updateAPIBase = bad.URL
	updateHTTPClient = bad.Client()
	msg = checkForUpdate("1.0.0")
	if msg.Err == nil {
		t.Fatal("expected error")
	}

	// empty tag
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": ""})
	}))
	defer empty.Close()
	updateAPIBase = empty.URL
	updateHTTPClient = empty.Client()
	msg = checkForUpdate("1.0.0")
	if msg.Err == nil {
		t.Fatal("expected empty tag error")
	}

	// invalid JSON
	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer badJSON.Close()
	updateAPIBase = badJSON.URL
	updateHTTPClient = badJSON.Client()
	msg = checkForUpdate("1.0.0")
	if msg.Err == nil {
		t.Fatal("expected json error")
	}
}

func TestCheckForUpdateCmd(t *testing.T) {
	c := CheckForUpdate("dev")
	if c == nil {
		t.Fatal("nil cmd")
	}
	msg := c()
	u, ok := msg.(types.UpdateAvailableMsg)
	if !ok {
		t.Fatalf("type %T", msg)
	}
	if u.LatestVersion != "" {
		t.Fatalf("%+v", u)
	}
}

func TestFetchLatestReleaseTagErrors(t *testing.T) {
	oldBase, oldClient := updateAPIBase, updateHTTPClient
	defer func() {
		updateAPIBase = oldBase
		updateHTTPClient = oldClient
	}()

	// invalid URL → NewRequest fails
	updateAPIBase = "://not-a-url"
	if _, err := fetchLatestReleaseTag(); err == nil {
		t.Fatal("expected NewRequest error")
	}

	// transport failure → Do fails
	updateAPIBase = "http://127.0.0.1:1"
	updateHTTPClient = &http.Client{Timeout: 50 * time.Millisecond}
	if _, err := fetchLatestReleaseTag(); err == nil {
		t.Fatal("expected dial error")
	}
}

func TestUpgradeHint(t *testing.T) {
	old := osExecutable
	defer func() { osExecutable = old }()

	osExecutable = func() (string, error) { return "/opt/homebrew/bin/es-tui", nil }
	if got := upgradeHint(); got != "brew upgrade es-tui" {
		t.Fatal(got)
	}
	osExecutable = func() (string, error) { return "/usr/local/bin/es-tui", nil }
	if got := upgradeHint(); got != "es-tui --update" {
		t.Fatal(got)
	}
	osExecutable = func() (string, error) { return "", assertErr{} }
	if got := upgradeHint(); got != "es-tui --update" {
		t.Fatal(got)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "x" }
