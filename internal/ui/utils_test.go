package ui

import (
	"testing"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestClampTruncateFormat(t *testing.T) {
	if clamp(5, 0, 3) != 3 || clamp(-1, 0, 3) != 0 || clamp(2, 0, 3) != 2 {
		t.Fatal("clamp")
	}
	if clamp(1, 5, 3) != 5 {
		t.Fatal("clamp inverted")
	}
	if truncate("hello", 0) != "" || truncate("hi", 5) != "hi" || truncate("abcdef", 5) != "ab..." {
		t.Fatal("truncate")
	}
	if truncate("abcd", 3) != "abc" {
		t.Fatal("truncate short n")
	}
	if formatBytes(500) != "500 B" {
		t.Fatal(formatBytes(500))
	}
	if formatBytes(2048) == "" {
		t.Fatal("formatBytes kb")
	}
	if formatBytes(5<<20) == "" {
		t.Fatal("formatBytes mb")
	}
}

func TestColorizeJSON(t *testing.T) {
	if colorizeJSON("") != "" {
		t.Fatal("empty")
	}
	out := colorizeJSON(`{"name":"x","n":1,"ok":true,"z":null,"arr":[1,"a"]}`)
	if out == "" {
		t.Fatal("expected output")
	}
	if colorizeJSON("plain") != "plain" {
		t.Fatal("plain")
	}
	if colorizeJSON(`[1,2]`) == "" {
		t.Fatal("array")
	}
	if colorizeJSON(`{"a":"b\"c"}`) == "" {
		t.Fatal("escape")
	}
}

func TestHealthStyle(t *testing.T) {
	_ = healthStyle("green")
	_ = healthStyle("yellow")
	_ = healthStyle("red")
	_ = healthStyle("unknown")
}

func TestSparkline(t *testing.T) {
	if sparkline(nil) != "" {
		t.Fatal("empty")
	}
	s := sparkline([]types.LiveMetricsData{
		{QueryTotal: 0},
		{QueryTotal: 10},
		{QueryTotal: 5},
	})
	if s == "" {
		t.Fatal("sparkline")
	}
}
