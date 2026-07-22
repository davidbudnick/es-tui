package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/davidbudnick/es-tui/internal/types"
)

func (m Model) viewDocuments() string {
	totalWidth := m.Width
	if totalWidth < 100 {
		return m.viewDocumentsListOnly()
	}

	leftWidth := (totalWidth * 55) / 100
	rightWidth := totalWidth - leftWidth - 1
	panelHeight := max(m.Height-4, 12)

	leftContent := m.buildDocumentsListPanel(leftWidth - 2)
	rightContent := m.buildDocumentPreviewPanel(rightWidth - 2)

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Padding(0, 1).
		Render(leftContent)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(panelHeight).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color(colorBorder)).
		Render(rightContent)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	help := helpStyle.Render("j/k:nav  enter:view  e:edit  d:del  /:query  D:bulk  r:refresh  q:back")
	return content + "\n" + help
}

func (m Model) viewDocumentsListOnly() string {
	var b strings.Builder
	b.WriteString(m.buildDocumentsListPanel(max(m.Width-4, 40)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:view  e:edit  d:del  /:query  q:back"))
	return b.String()
}

func (m Model) buildDocumentsListPanel(width int) string {
	var b strings.Builder

	name := ""
	if m.CurrentIndex != nil {
		name = m.CurrentIndex.Name
	}
	title := fmt.Sprintf("Documents · %s", name)
	if m.DocTotal > 0 {
		title += fmt.Sprintf("  [%d]", m.DocTotal)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(keyStyle.Render("Query: "))
	if m.Inputs != nil && m.Inputs.SearchInput.Focused() {
		b.WriteString(m.Inputs.SearchInput.View())
	} else {
		q := m.DocQuery
		if q == "" {
			q = "match_all"
		}
		b.WriteString(normalStyle.Render(truncate(q, max(width-10, 10))))
	}
	b.WriteString("\n\n")

	if len(m.Documents) == 0 {
		b.WriteString(dimStyle.Render("No documents."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press enter to run match_all."))
		return b.String()
	}

	idW := width - 14
	if idW < 16 {
		idW = 16
	}
	header := fmt.Sprintf("  %-*s  %s", idW, "ID", "SCORE")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(width, 70))))
	b.WriteString("\n")

	maxVisible := max(m.Height-12, 5)
	selectedIdx := clamp(m.SelectedDocIdx, 0, len(m.Documents)-1)
	start := 0
	if selectedIdx >= maxVisible {
		start = selectedIdx - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.Documents))

	for i := start; i < end; i++ {
		doc := m.Documents[i]
		id := truncate(doc.ID, idW)
		score := fmt.Sprintf("%.3f", doc.Score)
		if i == selectedIdx {
			b.WriteString(selectedRowStyle.Render(fmt.Sprintf("▶ %-*s  %s", idW, id, score)))
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %-*s  %s", idW, id, score)))
		}
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render(fmt.Sprintf("\nshowing %d of %d", len(m.Documents), m.DocTotal)))
	return b.String()
}

func (m Model) buildDocumentPreviewPanel(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Preview"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n\n")

	if len(m.Documents) == 0 {
		b.WriteString(dimStyle.Render("No document selected"))
		return b.String()
	}
	selectedIdx := clamp(m.SelectedDocIdx, 0, len(m.Documents)-1)
	doc := m.Documents[selectedIdx]

	b.WriteString(keyStyle.Render("ID: "))
	b.WriteString(normalStyle.Render(truncate(doc.ID, max(width-6, 8))))
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("Index: "))
	b.WriteString(normalStyle.Render(doc.Index))
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("Score: "))
	b.WriteString(normalStyle.Render(fmt.Sprintf("%.4f", doc.Score)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(width, 8))))
	b.WriteString("\n\n")
	b.WriteString(keyStyle.Render("Value"))
	b.WriteString("\n\n")

	raw := doc.Raw
	if raw == "" && doc.Source != nil {
		if bb, err := json.MarshalIndent(doc.Source, "", "  "); err == nil {
			raw = string(bb)
		}
	}
	if raw == "" {
		b.WriteString(dimStyle.Render("(empty source)"))
		return b.String()
	}

	maxLines := max(m.Height-18, 5)
	colored := colorizeJSON(raw)
	lines := strings.Split(colored, "\n")
	shown := 0
	for _, line := range lines {
		if shown >= maxLines {
			b.WriteString(dimStyle.Render(fmt.Sprintf("… %d more lines", len(lines)-shown)))
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
		shown++
	}
	return b.String()
}

func (m Model) viewDocumentDetail() string {
	if m.CurrentDocument == nil {
		return dimStyle.Render("No document")
	}
	doc := *m.CurrentDocument
	boxWidth := detailBoxWidth(m.Width)
	contentWidth := max(boxWidth-6, 20)

	var b strings.Builder
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, titleStyle.Render("Document Detail")))
	b.WriteString("\n\n")

	var meta strings.Builder
	meta.WriteString(keyStyle.Render("  Index: "))
	meta.WriteString(normalStyle.Render(doc.Index))
	meta.WriteString("\n")
	meta.WriteString(keyStyle.Render("     ID: "))
	meta.WriteString(normalStyle.Render(doc.ID))
	meta.WriteString("\n")
	meta.WriteString(keyStyle.Render("  Score: "))
	meta.WriteString(normalStyle.Render(fmt.Sprintf("%.4f", doc.Score)))
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, meta.String()))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, keyStyle.Render("Value:")))
	b.WriteString("\n")

	raw := doc.Raw
	if raw == "" && doc.Source != nil {
		if bb, err := json.MarshalIndent(doc.Source, "", "  "); err == nil {
			raw = string(bb)
		}
	}
	body := colorizeJSON(raw)
	lines := strings.Split(body, "\n")
	maxVisible := max(m.Height-14, 6)
	start := clamp(m.DetailScroll, 0, max(0, len(lines)-1))
	end := min(start+maxVisible, len(lines))

	var display strings.Builder
	for i := start; i < end; i++ {
		display.WriteString(lines[i])
		display.WriteString("\n")
	}
	if len(lines) > maxVisible {
		display.WriteString(dimStyle.Render(fmt.Sprintf("[%d-%d / %d lines]", start+1, end, len(lines))))
	}

	box := valueBoxStyle.Width(boxWidth).Render(display.String())
	b.WriteString(lipgloss.PlaceHorizontal(min(m.Width, boxWidth+4), lipgloss.Center, box))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("e:edit  d:delete  j/k:scroll  esc:back"))
	_ = contentWidth
	return b.String()
}

func detailBoxWidth(termWidth int) int {
	w := termWidth - 8
	if w > 100 {
		w = 100
	}
	if w < 40 {
		w = 40
	}
	return w
}

var _ = types.Document{}
