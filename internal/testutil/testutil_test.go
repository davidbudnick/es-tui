package testutil

import (
	"errors"
	"testing"
	"time"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestAssertHelpers(t *testing.T) {
	AssertEqual(t, 1, 1)
	AssertNoError(t, nil)
	AssertTrue(t, true, "ok")
}

func TestMockESFull(t *testing.T) {
	m := &MockES{
		Indices: []types.IndexInfo{{Name: "products", Health: "green"}},
		Info:    types.ClusterInfo{ClusterName: "test"},
		Health:  types.ClusterHealth{Status: "green"},
		Nodes:   []types.NodeInfo{{Name: "n1"}},
		Shards:  []types.ShardInfo{{Index: "products"}},
		Metrics: types.LiveMetricsData{Status: "green"},
		CatBody: "[]",
		SearchResult: types.SearchResult{
			Total: 1,
			Hits:  []types.Document{{ID: "1", Index: "products"}},
		},
		Document:    types.Document{ID: "1", Index: "products"},
		Aliases:     []types.AliasInfo{{Alias: "a"}},
		Templates:   []types.IndexTemplate{{Name: "t"}},
		Settings:    "{}",
		Mappings:    "{}",
		DeleteByQ:   3,
		FlavorVal:   types.FlavorOpenSearch,
		TestLatency: 5 * time.Millisecond,
		TestInfo:    types.ClusterInfo{Name: "x"},
	}

	AssertNoError(t, m.Connect(types.Connection{Host: "localhost", Port: 9200}))
	AssertTrue(t, m.IsConnected(), "connected")
	AssertEqual(t, m.Flavor(), types.FlavorOpenSearch)
	AssertNoError(t, m.Disconnect())
	AssertTrue(t, !m.IsConnected(), "disconnected")

	lat, info, err := m.TestConnection(types.Connection{})
	AssertNoError(t, err)
	AssertEqual(t, lat, 5*time.Millisecond)
	AssertEqual(t, info.Name, "x")

	ci, err := m.GetClusterInfo()
	AssertNoError(t, err)
	AssertEqual(t, ci.ClusterName, "test")

	h, err := m.GetClusterHealth()
	AssertNoError(t, err)
	AssertEqual(t, h.Status, "green")

	nodes, err := m.GetNodes()
	AssertNoError(t, err)
	AssertEqual(t, len(nodes), 1)

	shards, err := m.GetShards("products")
	AssertNoError(t, err)
	AssertEqual(t, len(shards), 1)

	metrics, err := m.GetLiveMetrics()
	AssertNoError(t, err)
	AssertEqual(t, metrics.Status, "green")

	cat, err := m.Cat("indices")
	AssertNoError(t, err)
	AssertEqual(t, cat, "[]")

	indices, err := m.ListIndices("*")
	AssertNoError(t, err)
	AssertEqual(t, len(indices), 1)

	idx, err := m.GetIndex("products")
	AssertNoError(t, err)
	AssertEqual(t, idx.Name, "products")

	AssertNoError(t, m.CreateIndex("new", "{}"))
	AssertNoError(t, m.DeleteIndex("new"))
	s, err := m.GetIndexSettings("products")
	AssertNoError(t, err)
	AssertEqual(t, s, "{}")
	mp, err := m.GetIndexMappings("products")
	AssertNoError(t, err)
	AssertEqual(t, mp, "{}")
	AssertNoError(t, m.RefreshIndex("products"))
	AssertNoError(t, m.OpenIndex("products"))
	AssertNoError(t, m.CloseIndex("products"))

	sr, err := m.Search("products", "*", 0, 10)
	AssertNoError(t, err)
	AssertEqual(t, sr.Total, int64(1))

	doc, err := m.GetDocument("products", "1")
	AssertNoError(t, err)
	AssertEqual(t, doc.ID, "1")
	AssertNoError(t, m.IndexDocument("products", "1", `{}`))
	AssertNoError(t, m.DeleteDocument("products", "1"))
	n, err := m.DeleteByQuery("products", "*")
	AssertNoError(t, err)
	AssertEqual(t, n, int64(3))

	aliases, err := m.ListAliases()
	AssertNoError(t, err)
	AssertEqual(t, len(aliases), 1)
	tmpls, err := m.ListTemplates()
	AssertNoError(t, err)
	AssertEqual(t, len(tmpls), 1)

	// error paths
	m.ConnectErr = errors.New("nope")
	AssertError(t, m.Connect(types.Connection{}))
	m.IndexErr = errors.New("missing")
	_, err = m.GetIndex("nope")
	AssertError(t, err)
	m.IndexErr = nil
	m.IndexDetail = types.IndexInfo{}
	m.Indices = nil
	_, err = m.GetIndex("nope")
	AssertError(t, err)

	m.CreateErr = errors.New("c")
	AssertError(t, m.CreateIndex("x", ""))
	m.DeleteErr = errors.New("d")
	AssertError(t, m.DeleteIndex("x"))

	// Flavor default
	m2 := &MockES{}
	AssertEqual(t, m2.Flavor(), types.FlavorElasticsearch)
}
