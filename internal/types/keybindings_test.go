package types

import "testing"

func TestDefaultKeyBindings(t *testing.T) {
	kb := DefaultKeyBindings()
	if kb.Up != "k" || kb.Down != "j" || kb.Quit != "q" {
		t.Fatalf("unexpected defaults: %+v", kb)
	}
	if kb.Cluster != "c" || kb.Metrics != "m" {
		t.Fatalf("unexpected feature bindings: %+v", kb)
	}
}
