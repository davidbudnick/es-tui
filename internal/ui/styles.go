package ui

import "charm.land/lipgloss/v2"

// Elastic logo multicolor palette:
// pink #F04E98, yellow #FEC514, teal #00BFB3, green #00A69C, blue #3B8EEA, navy #343741
const (
	colorPink   = "#F04E98"
	colorYellow = "#FEC514"
	colorTeal   = "#00BFB3"
	colorGreen  = "#00A69C"
	colorBlue   = "#3B8EEA"
	colorNavy   = "#343741"
	colorWhite  = "#FFFFFF"
	colorDim    = "#6B7280"
	colorMuted  = "#9CA3AF"
	colorRed    = "#EF4444"
	colorBorder = "240"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorTeal)).MarginBottom(1)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorWhite))
	normalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorWhite))
	// selectedStyle: full-width band for detail lines (redis-style)
	selectedStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color(colorNavy)).
			Background(lipgloss.Color(colorTeal))
	// selectedRowStyle: list rows — accent text + cursor, no full flood
	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorTeal))
	keyStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorBlue))
	descStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color(colorWhite))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed))
	successStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	metaDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	accentStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(colorTeal))
	pinkStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color(colorPink))
	yellowStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(colorYellow))
	tealStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color(colorTeal))
	greenStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	blueStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBlue))

	logoPink   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorPink)).Bold(true)
	logoYellow = lipgloss.NewStyle().Foreground(lipgloss.Color(colorYellow)).Bold(true)
	logoTeal   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorTeal)).Bold(true)
	logoGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen)).Bold(true)
	logoBlue   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBlue)).Bold(true)

	healthGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true)
	healthYellow = lipgloss.NewStyle().Foreground(lipgloss.Color(colorYellow)).Bold(true)
	healthRed    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed)).Bold(true)

	connCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 1).
			MarginBottom(0)

	connCardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(colorTeal)).
				Padding(0, 1).
				MarginBottom(0)

	statsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 1)

	badgeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	badgeESStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("23")).
			Foreground(lipgloss.Color(colorTeal)).
			Padding(0, 1)

	badgeOSStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("53")).
			Foreground(lipgloss.Color("213")).
			Padding(0, 1)

	badgeTLSStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("22")).
			Foreground(lipgloss.Color("46")).
			Padding(0, 1)

	panelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color(colorBorder))

	valueBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(1, 2)

	jsonKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBlue))
	jsonStringStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	jsonNumberStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorYellow))
	jsonBoolStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorPink))
	jsonNullStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	jsonBracketStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorWhite))
)

func healthStyle(status string) lipgloss.Style {
	switch status {
	case "green":
		return healthGreen
	case "yellow":
		return healthYellow
	case "red":
		return healthRed
	default:
		return dimStyle
	}
}

func flavorBadge(flavor string) string {
	switch flavor {
	case "opensearch":
		return badgeOSStyle.Render("OS")
	case "elasticsearch":
		return badgeESStyle.Render("ES")
	default:
		return badgeStyle.Render("AUTO")
	}
}
