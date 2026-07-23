package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

const connectionsCardWidth = 55

func (m Model) viewConnections() string {
	var b strings.Builder
	w := connectionsCardWidth

	// Logo + centered subtitle
	b.WriteString(m.renderLogo())
	b.WriteString("\n\n")

	// Stats chips centered under logo/cards
	b.WriteString(lipgloss.PlaceHorizontal(w, lipgloss.Center, m.buildStatsBar()))
	b.WriteString("\n\n")

	// In-view loading (status bar is suppressed on this screen to keep vertical center stable).
	if m.Loading {
		b.WriteString(lipgloss.PlaceHorizontal(w, lipgloss.Center, dimStyle.Render("Connecting...")))
		b.WriteString("\n\n")
	}

	if m.ConnectionError != "" {
		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorPink)).
			Foreground(lipgloss.Color(colorPink)).
			Padding(0, 2).
			Width(w).
			Render(fmt.Sprintf("Connection Failed\n%s", dimStyle.Render(m.ConnectionError)))
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	// Section frame (redis-style open header + cards + close footer)
	connCount := len(m.Connections)
	sectionInner := w - 1
	sectionTitle := fmt.Sprintf("╭─ Saved Connections (%d) ", connCount)
	if pad := sectionInner - len([]rune(sectionTitle)); pad > 0 {
		sectionTitle += strings.Repeat("─", pad) + "╮"
	} else {
		sectionTitle += "╮"
	}
	b.WriteString(accentStyle.Render(sectionTitle))
	b.WriteString("\n")

	if len(m.Connections) == 0 {
		emptyMsg := "No connections saved.\n\nPress a to add your first connection."
		emptyBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim)).
			Align(lipgloss.Center).
			Width(w).
			Padding(1, 0).
			Render(emptyMsg)
		b.WriteString(emptyBox)
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
		maxVisible := max((m.Height-22)/3, 3)
		selectedIdx := clamp(m.SelectedConnIdx, 0, len(m.Connections)-1)
		startIdx, endIdx := listWindow(selectedIdx, len(m.Connections), maxVisible)

		for i := startIdx; i < endIdx; i++ {
			b.WriteString(m.renderConnectionCard(m.Connections[i], i == selectedIdx, w))
			b.WriteString("\n")
		}
		if len(m.Connections) > maxVisible {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↕ %d-%d of %d connections", startIdx+1, endIdx, len(m.Connections))))
			b.WriteString("\n")
		}
	}

	sectionBottom := "╰" + strings.Repeat("─", sectionInner) + "╯"
	b.WriteString(accentStyle.Render(sectionBottom))
	b.WriteString("\n\n")

	// Keybinding chips centered under the column
	b.WriteString(lipgloss.PlaceHorizontal(w, lipgloss.Center, renderKeyHelp([]struct{ key, desc string }{
		{"↑/↓", "navigate"},
		{"enter", "connect"},
		{"a", "add"},
		{"e", "edit"},
		{"d", "delete"},
		{"t", "test"},
		{"?", "help"},
		{"q", "quit"},
	})))

	return b.String()
}

func renderKeyHelp(keys []struct{ key, desc string }) string {
	return renderKeyHelpWidth(0, keys)
}

// renderKeyHelpWidth lays out key chips; when width > 0 it wraps to multiple lines
// so footers don't get squished on typical terminals.
func renderKeyHelpWidth(width int, keys []struct{ key, desc string }) string {
	chip := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("255")).
		Padding(0, 1)

	if width <= 0 {
		var b strings.Builder
		for i, kb := range keys {
			b.WriteString(chip.Render(kb.key))
			b.WriteString(" ")
			b.WriteString(dimStyle.Render(kb.desc))
			if i < len(keys)-1 {
				b.WriteString("  ")
			}
		}
		return b.String()
	}

	var lines []string
	var cur strings.Builder
	curW := 0
	for _, kb := range keys {
		piece := chip.Render(kb.key) + " " + dimStyle.Render(kb.desc)
		pw := lipgloss.Width(piece)
		need := pw
		if curW > 0 {
			need += 2
		}
		if curW > 0 && curW+need > width {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
			need = pw
		}
		if curW > 0 {
			cur.WriteString("  ")
			curW += 2
		}
		cur.WriteString(piece)
		curW += pw
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderConnectionCard(conn types.Connection, selected bool, width int) string {
	var card strings.Builder

	icon := "○"
	nameStyle := normalStyle
	if selected {
		icon = "●"
		nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	}

	fmt.Fprintf(&card, " %s %s", icon, nameStyle.Render(conn.Name))
	card.WriteString("\n")

	// Compact host:port (redis style) — scheme only if TLS
	hostPort := fmt.Sprintf("%s:%d", conn.Host, conn.Port)
	if conn.UseTLS {
		hostPort = "https://" + hostPort
	}
	card.WriteString(dimStyle.Render("   " + hostPort))

	flavor := string(conn.Flavor)
	if flavor == "" {
		flavor = "auto"
	}
	card.WriteString("  ")
	card.WriteString(flavorBadge(flavor))

	if conn.UseTLS {
		card.WriteString(" ")
		card.WriteString(badgeTLSStyle.Render("TLS"))
	}
	if conn.APIKey != "" || conn.Username != "" || conn.BearerToken != "" {
		card.WriteString(" ")
		card.WriteString(badgeStyle.Render("AUTH"))
	}
	if conn.ReadOnly {
		card.WriteString(" ")
		card.WriteString(badgeStyle.Render("RO"))
	}

	style := connCardStyle
	if selected {
		style = connCardSelectedStyle
	}
	return style.Width(width).Render(card.String())
}

func (m Model) renderLogo() string {
	lines := []string{
		"███████╗██╗      █████╗ ███████╗████████╗██╗ ██████╗",
		"██╔════╝██║     ██╔══██╗██╔════╝╚══██╔══╝██║██╔════╝",
		"█████╗  ██║     ███████║███████╗   ██║   ██║██║     ",
		"██╔══╝  ██║     ██╔══██║╚════██║   ██║   ██║██║     ",
		"███████╗███████╗██║  ██║███████║   ██║   ██║╚██████╗",
		"╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝ ╚═════╝",
	}
	styles := []*lipgloss.Style{&logoPink, &logoYellow, &logoTeal, &logoBlue, &logoGreen, &logoPink}
	var logo strings.Builder
	for i, line := range lines {
		logo.WriteString(styles[i%len(styles)].Render(line))
		if i < len(lines)-1 {
			logo.WriteString("\n")
		}
	}
	// Center logo + tagline to the same column width as connection cards.
	w := connectionsCardWidth
	logoBlock := lipgloss.PlaceHorizontal(w, lipgloss.Center, logo.String())
	subtitle := lipgloss.PlaceHorizontal(w, lipgloss.Center,
		dimStyle.Render("Elasticsearch & OpenSearch TUI"))
	return logoBlock + "\n" + subtitle
}

func (m Model) buildStatsBar() string {
	boxes := []struct {
		label string
		value string
		color string
	}{
		{"Connections", fmt.Sprintf("%d saved", len(m.Connections)), "39"},
		{"Time", time.Now().Format("15:04:05"), "245"},
	}

	var statsBoxes []string
	for _, box := range boxes {
		content := fmt.Sprintf("%s\n%s",
			dimStyle.Render(box.label),
			lipgloss.NewStyle().Foreground(lipgloss.Color(box.color)).Bold(true).Render(box.value),
		)
		statsBoxes = append(statsBoxes, statsBoxStyle.Width(18).Render(content))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, statsBoxes...)
}

func (m Model) viewConnectionForm() string {
	title := "Add Connection"
	if m.Screen == types.ScreenEditConnection {
		title = "Edit Connection"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Text fields with labels (redis-style).
	for i, label := range connTextLabels {
		if i >= len(m.ConnInputs) {
			break
		}
		labelStyle := keyStyle
		if m.ConnFocusIdx == i {
			labelStyle = accentStyle
		}
		b.WriteString(labelStyle.Render(label))
		b.WriteString("\n")
		b.WriteString(m.ConnInputs[i].View())
		b.WriteString("\n\n")
	}

	// Engine flavor dropdown.
	flavorFocused := m.ConnFocusIdx == connFieldFlavor
	flavorLabel := keyStyle
	if flavorFocused {
		flavorLabel = accentStyle
	}
	b.WriteString(flavorLabel.Render("Engine"))
	b.WriteString("\n")
	b.WriteString(m.renderFlavorDropdown(flavorFocused))
	b.WriteString("\n\n")

	// Read-only toggle.
	roFocused := m.ConnFocusIdx == connFieldReadOnly
	roLabel := keyStyle
	if roFocused {
		roLabel = accentStyle
	}
	b.WriteString(roLabel.Render("Read-only"))
	b.WriteString("\n")
	check := "[ ] Browse only — block writes"
	if m.ConnReadOnly {
		check = "[x] Browse only — block writes"
	}
	checkStyle := normalStyle
	if roFocused {
		checkStyle = accentStyle
	}
	b.WriteString(checkStyle.Render(check))
	b.WriteString("\n\n")

	if m.Err != nil && (m.Screen == types.ScreenAddConnection || m.Screen == types.ScreenEditConnection) {
		b.WriteString(errorStyle.Render(m.Err.Error()))
		b.WriteString("\n\n")
	}

	help := []struct{ key, desc string }{
		{"tab", "next"},
		{"space", "toggle"},
		{"←/→", "engine"},
		{"enter", "save"},
		{"esc", "cancel"},
	}
	if flavorFocused && m.ConnFlavorOpen {
		help = []struct{ key, desc string }{
			{"j/k", "choose"},
			{"enter/space", "pick"},
			{"esc", "close"},
		}
	}
	b.WriteString(renderKeyHelp(help))

	modalWidth := min(56, max(m.Width-8, 40))
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth).
		Render(b.String())
	// Outer Place in render() already centers this screen.
	return modal
}

func (m Model) renderFlavorDropdown(focused bool) string {
	cur := "Auto"
	if m.ConnFlavorIdx >= 0 && m.ConnFlavorIdx < len(connFlavorOptions) {
		cur = connFlavorOptions[m.ConnFlavorIdx].DisplayName()
	}
	arrow := "▾"
	if m.ConnFlavorOpen {
		arrow = "▴"
	}
	// Closed control.
	closed := fmt.Sprintf("  %-18s %s", cur, arrow)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(28)
	if focused {
		boxStyle = boxStyle.BorderForeground(lipgloss.Color("39"))
	}
	if !m.ConnFlavorOpen {
		return boxStyle.Render(closed)
	}

	// Expanded list under the control.
	var list strings.Builder
	list.WriteString(boxStyle.Render(closed))
	list.WriteString("\n")
	for i, opt := range connFlavorOptions {
		mark := "  "
		lineStyle := dimStyle
		if i == m.ConnFlavorIdx {
			mark = "● "
			lineStyle = selectedRowStyle
		}
		list.WriteString(lineStyle.Render(mark + opt.DisplayName()))
		list.WriteString("\n")
	}
	// Trim trailing newline for cleaner box join.
	return strings.TrimRight(list.String(), "\n")
}
