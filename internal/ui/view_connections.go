package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func (m Model) viewConnections() string {
	var b strings.Builder
	b.WriteString(m.renderLogo())
	b.WriteString("\n\n")
	b.WriteString(m.buildStatsBar())
	b.WriteString("\n\n")

	if m.ConnectionError != "" {
		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorPink)).
			Foreground(lipgloss.Color(colorPink)).
			Padding(0, 2).
			Width(55).
			Render(fmt.Sprintf("Connection Failed\n%s", dimStyle.Render(m.ConnectionError)))
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	connCount := len(m.Connections)
	sectionTitle := fmt.Sprintf("╭─ Saved Connections (%d) ", connCount)
	sectionTitle += strings.Repeat("─", max(10, 50-len(sectionTitle))) + "╮"
	b.WriteString(accentStyle.Render(sectionTitle))
	b.WriteString("\n")

	if len(m.Connections) == 0 {
		b.WriteString("\n")
		emptyBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim)).
			Padding(1, 2).
			Render("  No connections saved.\n\n  Press 'a' to add your first Elasticsearch/OpenSearch connection.")
		b.WriteString(emptyBox)
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
		maxVisible := max((m.Height-20)/3, 3)
		selectedIdx := clamp(m.SelectedConnIdx, 0, len(m.Connections)-1)
		startIdx := 0
		if selectedIdx >= maxVisible {
			startIdx = selectedIdx - maxVisible + 1
		}
		endIdx := min(startIdx+maxVisible, len(m.Connections))
		if endIdx-startIdx < maxVisible && endIdx == len(m.Connections) {
			startIdx = max(endIdx-maxVisible, 0)
		}

		cardWidth := min(55, max(m.Width-10, 40))
		for i := startIdx; i < endIdx; i++ {
			conn := m.Connections[i]
			isSelected := i == selectedIdx
			b.WriteString(m.renderConnectionCard(conn, isSelected, cardWidth))
			b.WriteString("\n")
		}
		if len(m.Connections) > maxVisible {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  showing %d-%d of %d\n", startIdx+1, endIdx, len(m.Connections))))
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate  enter connect  a add  e edit  d delete  t test  ? help  q quit"))
	return b.String()
}

func (m Model) renderConnectionCard(conn types.Connection, selected bool, width int) string {
	var card strings.Builder

	icon := "○"
	nameStyle := normalStyle
	if selected {
		icon = "●"
		nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorTeal)).Bold(true)
	}

	fmt.Fprintf(&card, " %s %s", icon, nameStyle.Render(conn.Name))
	card.WriteString("\n")

	scheme := "http"
	if conn.UseTLS {
		scheme = "https"
	}
	hostLine := fmt.Sprintf("   %s://%s:%d", scheme, conn.Host, conn.Port)
	card.WriteString(dimStyle.Render(hostLine))

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
	if conn.APIKey != "" || conn.Username != "" {
		card.WriteString(" ")
		card.WriteString(badgeStyle.Render("AUTH"))
	}

	style := connCardStyle
	if selected {
		style = connCardSelectedStyle
	}
	return style.Width(width).Render(card.String())
}

func (m Model) renderLogo() string {
	lines := []string{
		" ███████╗██╗      █████╗ ███████╗████████╗██╗ ██████╗",
		" ██╔════╝██║     ██╔══██╗██╔════╝╚══██╔══╝██║██╔════╝",
		" █████╗  ██║     ███████║███████╗   ██║   ██║██║     ",
		" ██╔══╝  ██║     ██╔══██║╚════██║   ██║   ██║██║     ",
		" ███████╗███████╗██║  ██║███████║   ██║   ██║╚██████╗",
		" ╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝ ╚═════╝",
	}
	styles := []*lipgloss.Style{&logoPink, &logoYellow, &logoTeal, &logoBlue, &logoGreen, &logoPink}
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(styles[i%len(styles)].Render(line))
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Elasticsearch & OpenSearch TUI"))
	return b.String()
}

func (m Model) buildStatsBar() string {
	n := len(m.Connections)
	now := time.Now().Format("15:04:05")
	connChip := statsBoxStyle.Render(fmt.Sprintf("Connections\n%s saved", tealStyle.Render(fmt.Sprintf("%d", n))))
	timeChip := statsBoxStyle.Render(fmt.Sprintf("Time\n%s", dimStyle.Render(now)))
	return lipgloss.JoinHorizontal(lipgloss.Top, connChip, "  ", timeChip)
}

func (m Model) viewConnectionForm() string {
	title := "Add Connection"
	if m.Screen == types.ScreenEditConnection {
		title = "Edit Connection"
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	for i, ti := range m.ConnInputs {
		prefix := "  "
		if i == m.ConnFocusIdx {
			prefix = tealStyle.Render("❯ ")
		}
		b.WriteString(prefix)
		b.WriteString(ti.View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("tab next · enter save · esc cancel"))
	return b.String()
}
