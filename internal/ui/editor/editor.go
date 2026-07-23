// Package editor provides a plain multiline text editor backed by bubbles textarea.
package editor

import (
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"

	"github.com/davidbudnick/es-tui/internal/types"
)

// Model is a focused multiline editor.
type Model struct {
	area     textarea.Model
	fileName string
}

// New creates an editor with content sized for the given terminal region.
func New(content string, width, height int, fileName string) *Model {
	area := textarea.New()
	area.ShowLineNumbers = true
	area.CharLimit = 0
	area.SetValue(content)
	area.SetWidth(max(width, 20))
	area.SetHeight(max(height, 5))
	area.Focus()
	return &Model{area: area, fileName: fileName}
}

// FileName returns the optional buffer label (e.g. document.json).
func (m *Model) FileName() string {
	return m.fileName
}

// Value returns the editor contents.
func (m *Model) Value() string {
	return m.area.Value()
}

// SetSize resizes the textarea.
func (m *Model) SetSize(width, height int) {
	m.area.SetWidth(max(width, 20))
	m.area.SetHeight(max(height, 5))
}

// View renders the textarea.
func (m *Model) View() string {
	return m.area.View()
}

// Save emits EditorSaveMsg with current content.
func (m *Model) Save() tea.Cmd {
	content := m.area.Value()
	return func() tea.Msg {
		return types.EditorSaveMsg{Content: content}
	}
}

// Cancel emits EditorQuitMsg.
func (m *Model) Cancel() tea.Cmd {
	return func() tea.Msg {
		return types.EditorQuitMsg{}
	}
}

// Update handles keys; ctrl+s saves, esc/ctrl+q cancels.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch key := msg.(type) {
	case tea.KeyPressMsg:
		switch key.String() {
		case "ctrl+s":
			return m, m.Save()
		case "esc", "ctrl+q":
			return m, m.Cancel()
		}
	}
	area, cmd := m.area.Update(msg)
	m.area = area
	return m, cmd
}
