package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/davidbudnick/es-tui/internal/types"
)

func TestNew(t *testing.T) {
	m := New("hello", 80, 24, "file.json")
	if m == nil {
		t.Fatal("expected non-nil editor")
	}
	if m.Value() != "hello" {
		t.Errorf("expected content hello, got %q", m.Value())
	}
	if m.FileName() != "file.json" {
		t.Errorf("expected fileName file.json, got %q", m.FileName())
	}
}

func TestSetSize(t *testing.T) {
	m := New("content", 40, 10, "")
	m.SetSize(120, 30)
	if m.View() == "" {
		t.Error("expected non-empty view after resize")
	}
}

func TestSaveCancel(t *testing.T) {
	m := New("payload", 80, 24, "")
	cmd := m.Save()
	msg, ok := cmd().(types.EditorSaveMsg)
	if !ok || msg.Content != "payload" {
		t.Fatalf("save: %v %v", ok, msg)
	}
	cmd = m.Cancel()
	if _, ok := cmd().(types.EditorQuitMsg); !ok {
		t.Fatal("cancel")
	}
}

func TestUpdateKeys(t *testing.T) {
	m := New("data", 80, 24, "")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected save cmd")
	}
	if _, ok := cmd().(types.EditorSaveMsg); !ok {
		t.Fatalf("got %T", cmd())
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cancel")
	}
	if _, ok := cmd().(types.EditorQuitMsg); !ok {
		t.Fatalf("got %T", cmd())
	}
}
