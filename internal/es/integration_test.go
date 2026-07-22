package es

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

func liveCluster(t *testing.T, url string) bool {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}

func TestIntegrationElasticsearch(t *testing.T) {
	if !liveCluster(t, "http://localhost:9200") {
		t.Skip("elasticsearch not running on :9200")
	}
	c := NewClient()
	err := c.Connect(types.Connection{Host: "localhost", Port: 9200, Flavor: types.FlavorAuto})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Disconnect() }()

	if c.Flavor() != types.FlavorElasticsearch && c.Flavor() != types.FlavorAuto {
		// 8.x tagline detection
		t.Logf("flavor=%s", c.Flavor())
	}
	info, err := c.GetClusterInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Version.Number == "" {
		t.Fatal("no version")
	}
	health, err := c.GetClusterHealth()
	if err != nil {
		t.Fatal(err)
	}
	if health.Status == "" {
		t.Fatal("no status")
	}
	indices, err := c.ListIndices("*")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ES indices: %d health=%s version=%s", len(indices), health.Status, info.Version.Number)

	sr, err := c.Search("products", "tags:demo", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("search hits=%d total=%d", len(sr.Hits), sr.Total)

	nodes, err := c.GetNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) == 0 {
		t.Fatal("no nodes")
	}
	metrics, err := c.GetLiveMetrics()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("metrics docs=%d status=%s", metrics.DocsCount, metrics.Status)
}

func TestIntegrationOpenSearch(t *testing.T) {
	if !liveCluster(t, "http://localhost:9201") {
		t.Skip("opensearch not running on :9201")
	}
	c := NewClient()
	err := c.Connect(types.Connection{Host: "localhost", Port: 9201, Flavor: types.FlavorAuto})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Disconnect() }()

	if c.Flavor() != types.FlavorOpenSearch {
		t.Fatalf("expected opensearch flavor, got %s", c.Flavor())
	}
	indices, err := c.ListIndices("products")
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) == 0 {
		t.Fatal("expected seeded products index")
	}
	doc, err := c.GetDocument("products", "1")
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID != "1" {
		t.Fatal(doc)
	}
	lat, info, err := c.TestConnection(types.Connection{Host: "localhost", Port: 9201})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("OS latency=%s cluster=%s version=%s", lat, info.ClusterName, info.Version.Number)
	_ = os.Getenv
}
