package ui

import (
	"fmt"
	"strings"

	"github.com/davidbudnick/es-tui/internal/types"
)

func (m Model) viewAllocation() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Disk Allocation (%d)", len(m.Allocation))))
	b.WriteString("\n\n")
	header := fmt.Sprintf("  %-16s %-14s %-10s %-10s %-10s %s", "NODE", "IP", "DISK.USED", "DISK.AVAIL", "DISK.%", "SHARDS")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 80)))
	b.WriteString("\n")
	for i, a := range m.Allocation {
		line := fmt.Sprintf("%-16s %-14s %-10s %-10s %-10s %s",
			truncate(a.Node, 16), a.IP, a.DiskUsed, a.DiskAvail, a.DiskPercent, a.Shards)
		if i == 0 {
			b.WriteString(selectedStyle.Render("▶ " + line))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
		if i >= max(m.Height-12, 5) {
			break
		}
	}
	if len(m.Allocation) == 0 {
		b.WriteString(dimStyle.Render("No allocation data"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("r:refresh  esc:back"))
	return b.String()
}

func (m Model) viewTasks() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Tasks (%d)", len(m.Tasks))))
	b.WriteString("\n\n")
	header := fmt.Sprintf("  %-28s %-22s %-12s %s", "ID", "ACTION", "RUNNING", "NODE")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 80)))
	b.WriteString("\n")
	if len(m.Tasks) == 0 {
		b.WriteString(dimStyle.Render("No running tasks"))
	}
	selected := clamp(m.SelectedTaskIdx, 0, max(len(m.Tasks)-1, 0))
	maxVisible := max(m.Height-12, 5)
	start := 0
	if selected >= maxVisible {
		start = selected - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.Tasks))
	for i := start; i < end; i++ {
		t := m.Tasks[i]
		line := fmt.Sprintf("%-28s %-22s %-12s %s",
			truncate(t.ID, 28), truncate(t.Action, 22), t.RunningTime, truncate(t.Node, 16))
		if i == selected {
			b.WriteString(selectedStyle.Render("▶ " + line))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  x:cancel  r:refresh  esc:back"))
	return b.String()
}

func (m Model) viewPlugins() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Plugins (%d)", len(m.Plugins))))
	b.WriteString("\n\n")
	header := fmt.Sprintf("  %-28s %-16s %s", "NAME", "COMPONENT", "VERSION")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 70)))
	b.WriteString("\n")
	for i, p := range m.Plugins {
		line := fmt.Sprintf("%-28s %-16s %s", truncate(p.Name, 28), truncate(p.Component, 16), p.Version)
		b.WriteString(normalStyle.Render("  " + line))
		b.WriteString("\n")
		if i >= max(m.Height-12, 5) {
			break
		}
	}
	if len(m.Plugins) == 0 {
		b.WriteString(dimStyle.Render("No plugins reported"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc:back"))
	return b.String()
}

func (m Model) viewDataStreams() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Data Streams (%d)", len(m.DataStreams))))
	b.WriteString("\n\n")
	header := fmt.Sprintf("  %-28s %-10s %-8s %s", "NAME", "STATUS", "GEN", "TEMPLATE")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 70)))
	b.WriteString("\n")
	for i, d := range m.DataStreams {
		line := fmt.Sprintf("%-28s %-10s %-8s %s",
			truncate(d.Name, 28), d.Status, d.Generation, truncate(d.Template, 24))
		b.WriteString(normalStyle.Render("  " + line))
		b.WriteString("\n")
		if i >= max(m.Height-12, 5) {
			break
		}
	}
	if len(m.DataStreams) == 0 {
		b.WriteString(dimStyle.Render("No data streams (or API unsupported)"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("r:refresh  esc:back"))
	return b.String()
}

func (m Model) viewSnapshots() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Snapshots (%d)", len(m.Snapshots))))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(dimStyle.Render("Repo: ") + m.Inputs.SnapshotRepo.View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	header := fmt.Sprintf("  %-24s %-12s %-12s %s", "SNAPSHOT", "STATE", "REPO", "START")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 70)))
	b.WriteString("\n")
	for i, s := range m.Snapshots {
		line := fmt.Sprintf("%-24s %-12s %-12s %s",
			truncate(s.Snapshot, 24), s.State, truncate(s.Repository, 12), s.StartTime)
		b.WriteString(normalStyle.Render("  " + line))
		b.WriteString("\n")
		if i >= max(m.Height-14, 5) {
			break
		}
	}
	if len(m.Snapshots) == 0 {
		b.WriteString(dimStyle.Render("No snapshots — enter repo name and press enter"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter:load  esc:back"))
	return b.String()
}

func (m Model) viewClusterSettings() string {
	return m.viewJSONPanel("Cluster Settings", m.ClusterSettings)
}

func (m Model) viewReindex() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Reindex"))
	b.WriteString("\n\n")
	if m.ReadOnly {
		b.WriteString(errorStyle.Render("Read-only connection — reindex disabled"))
		b.WriteString("\n\n")
	}
	if m.Inputs != nil {
		srcMark, dstMark := "  ", "  "
		if m.ReindexFocus == 0 {
			srcMark = tealStyle.Render("❯ ")
		}
		if m.ReindexFocus == 1 {
			dstMark = tealStyle.Render("❯ ")
		}
		b.WriteString(srcMark + "Source: " + m.Inputs.ReindexSrcInput.View() + "\n")
		b.WriteString(dstMark + "Dest:   " + m.Inputs.ReindexDstInput.View() + "\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Creates async reindex task (wait_for_completion=false)"))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("tab:field  enter:start  esc:cancel"))
	return b.String()
}

func (m Model) viewExport() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Export Documents (NDJSON)"))
	b.WriteString("\n\n")
	idx := "*"
	if m.CurrentIndex != nil {
		idx = m.CurrentIndex.Name
	}
	if m.SearchIndex != "" {
		idx = m.SearchIndex
	}
	b.WriteString(keyStyle.Render("Index: ") + idx + "\n")
	q := m.SearchQuery
	if q == "" {
		q = m.DocQuery
	}
	if q == "" {
		q = "match_all"
	}
	b.WriteString(keyStyle.Render("Query: ") + truncate(q, 60) + "\n\n")
	if m.Inputs != nil {
		b.WriteString(m.Inputs.ExportInput.View())
	}
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter:export  esc:cancel"))
	return b.String()
}

func (m Model) viewSavedQueries() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Saved Queries (%d)", len(m.SavedQueries))))
	b.WriteString("\n\n")
	if len(m.SavedQueries) == 0 {
		b.WriteString(dimStyle.Render("No saved queries. From search, press S to save current query."))
	}
	selected := clamp(m.SelectedSQIdx, 0, max(len(m.SavedQueries)-1, 0))
	for i, q := range m.SavedQueries {
		line := fmt.Sprintf("%-20s  %-16s  %s", truncate(q.Name, 20), truncate(q.Index, 16), truncate(q.Query, 40))
		if i == selected {
			b.WriteString(selectedStyle.Render("▶ " + line))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter:run  d:delete  esc:back"))
	return b.String()
}

func (m Model) viewExplain() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Explain"))
	b.WriteString("\n\n")
	if m.ExplainResult == nil {
		b.WriteString(dimStyle.Render("No explain result"))
	} else {
		matched := "false"
		style := errorStyle
		if m.ExplainResult.Matched {
			matched = "true"
			style = successStyle
		}
		b.WriteString(keyStyle.Render("Matched: ") + style.Render(matched) + "\n\n")
		body := m.ExplainResult.Explanation
		if body == "" {
			body = m.ExplainResult.Raw
		}
		lines := strings.Split(body, "\n")
		maxLines := max(m.Height-12, 5)
		for i, line := range lines {
			if i >= maxLines {
				b.WriteString(dimStyle.Render("…"))
				break
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc:back"))
	return b.String()
}

func (m Model) viewCommandPalette() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Command Palette"))
	b.WriteString("\n\n")
	if m.Inputs != nil {
		b.WriteString(m.Inputs.PaletteInput.View())
		b.WriteString("\n\n")
	}
	items := m.filteredPalette()
	if len(items) == 0 {
		b.WriteString(dimStyle.Render("No matching commands"))
	}
	idx := clamp(m.PaletteIdx, 0, max(len(items)-1, 0))
	maxVisible := max(m.Height-12, 8)
	start := 0
	if idx >= maxVisible {
		start = idx - maxVisible + 1
	}
	end := min(start+maxVisible, len(items))
	for i := start; i < end; i++ {
		it := items[i]
		line := fmt.Sprintf("%-28s  %s", it.Label, dimStyle.Render(it.Keys))
		if i == idx {
			b.WriteString(selectedStyle.Render("▶ " + fmt.Sprintf("%-28s  %s", it.Label, it.Keys)))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k:nav  enter:run  esc:close"))
	return b.String()
}

func (m Model) filteredPalette() []PaletteItem {
	items := m.PaletteItems
	if len(items) == 0 {
		items = defaultPaletteItems()
	}
	filter := ""
	if m.Inputs != nil {
		filter = strings.ToLower(strings.TrimSpace(m.Inputs.PaletteInput.Value()))
	}
	if filter == "" {
		return items
	}
	var out []PaletteItem
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Label), filter) || strings.Contains(strings.ToLower(it.ID), filter) {
			out = append(out, it)
		}
	}
	return out
}

// keep types import used
var _ = types.AllocationInfo{}
