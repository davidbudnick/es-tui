package ui

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

// normalizeKey maps bubbletea v2 key events to stable binding strings.
// Kitty-protocol terminals often emit "shift+o" instead of "O" and
// "shift+/" instead of "?" when Text is empty — without this, most
// uppercase hotkeys and ? help appear dead.
func normalizeKey(msg tea.KeyPressMsg) string {
	// Printable text wins (shifted symbols: ?, #, *, O, …).
	if t := msg.Text; t != "" && t != " " {
		return t
	}

	s := msg.String()
	if s == "" {
		return s
	}

	// shift+letter → Letter  (shift+o → O)
	if strings.HasPrefix(s, "shift+") {
		rest := strings.TrimPrefix(s, "shift+")
		if len(rest) == 1 {
			r := rune(rest[0])
			if r >= 'a' && r <= 'z' {
				return string(unicode.ToUpper(r))
			}
			// Common US-layout shifted symbols used as bindings.
			switch rest {
			case "/":
				return "?"
			case "3":
				return "#"
			case "8":
				return "*"
			case ";":
				return ":"
			case "'":
				return "\""
			case "1":
				return "!"
			case ",":
				return "<"
			case ".":
				return ">"
			case "-":
				return "_"
			case "=":
				return "+"
			case "`":
				return "~"
			case "\\":
				return "|"
			case "[":
				return "{"
			case "]":
				return "}"
			}
		}
	}
	return s
}

// typingContext reports whether a text input is focused and should receive
// keystrokes (including ? : #) instead of global shortcuts.
func (m Model) typingContext() bool {
	if m.Inputs == nil {
		return false
	}
	switch m.Screen {
	case types.ScreenAddConnection, types.ScreenEditConnection,
		types.ScreenIndexCreate, types.ScreenEditDocument, types.ScreenBulkDelete,
		types.ScreenCommandPalette, types.ScreenReindex, types.ScreenExport,
		types.ScreenCatAPI, types.ScreenSnapshots:
		return true
	case types.ScreenIndices:
		return m.Inputs.PatternInput.Focused()
	case types.ScreenDocuments:
		return m.Inputs.SearchInput.Focused()
	case types.ScreenSearch:
		return m.SearchFocus == "query" || (m.SearchArea != nil && m.SearchArea.Focused())
	default:
		return false
	}
}
