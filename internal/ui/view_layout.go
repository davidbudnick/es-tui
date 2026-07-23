package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// keyDesc is a footer keybinding chip.
type keyDesc struct {
	Key, Desc string
}

// fullScreenFrame is the standard fullscreen browse layout: body + wrapped key chips.
// Pads so the status bar (appended in render) sits at the bottom without clipping help.
func (m Model) fullScreenFrame(body string, keys ...keyDesc) string {
	content := body
	if len(keys) > 0 {
		content = body + "\n" + renderKeyHelpWidth(max(m.Width-2, 20), toKeyPairs(keys))
	}
	// render() may append "\n\n"+status (2 blanks + 1 line). Reserve 3 rows.
	target := max(m.Height-3, 1)
	lines := strings.Count(content, "\n") + 1
	if lines < target {
		content += strings.Repeat("\n", target-lines)
	}
	return content
}

// splitBrowse builds a redis-style left list + right preview split filling the terminal.
func (m Model) splitBrowse(leftRatio int, left, right string, keys ...keyDesc) string {
	if leftRatio <= 0 || leftRatio >= 100 {
		leftRatio = 60
	}
	totalWidth := m.Width
	if totalWidth < 100 {
		return m.fullScreenFrame(left, keys...)
	}
	leftWidth := (totalWidth * leftRatio) / 100
	rightWidth := totalWidth - leftWidth - 1
	// Room for multi-line help + blank + status.
	panelHeight := max(m.Height-6, 12)

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Padding(0, 1).
		Render(left)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(panelHeight).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("240")).
		Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	if len(keys) > 0 {
		content += "\n" + renderKeyHelpWidth(max(m.Width-2, 20), toKeyPairs(keys))
	}
	return content
}

// formModal wraps short dialogs in a redis-style bordered box.
func (m Model) formModal(body string) string {
	w := min(56, max(m.Width-8, 40))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(w).
		Render(strings.TrimRight(body, "\n"))
}

func (m Model) listHeader(title string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	return b.String()
}

func (m Model) tableSep(width int) string {
	return dimStyle.Render(strings.Repeat("─", max(min(width, m.Width-4), 20))) + "\n"
}

func heapColor(pct int) lipgloss.Style {
	switch {
	case pct >= 85:
		return healthRed
	case pct >= 70:
		return healthYellow
	default:
		return healthGreen
	}
}

func cpuColor(pct int) lipgloss.Style {
	switch {
	case pct >= 80:
		return healthRed
	case pct >= 50:
		return healthYellow
	default:
		return healthGreen
	}
}

func masterBadge(m string) string {
	if m == "*" {
		return yellowStyle.Bold(true).Render("*")
	}
	return dimStyle.Render(m)
}

// padRight pads plain text for selection bands.
func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func fmtInt(n int) string {
	return fmt.Sprintf("%d", n)
}

func toKeyPairs(keys []keyDesc) []struct{ key, desc string } {
	out := make([]struct{ key, desc string }, len(keys))
	for i, k := range keys {
		out[i] = struct{ key, desc string }{k.Key, k.Desc}
	}
	return out
}

// colWidth picks a table column width that fits the terminal.
func colWidth(total, fixed, minW, maxW int) int {
	w := total - fixed
	if w < minW {
		return minW
	}
	if maxW > 0 && w > maxW {
		return maxW
	}
	return w
}
