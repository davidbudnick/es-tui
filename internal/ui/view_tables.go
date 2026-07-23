package ui

import (
	"fmt"
	"strings"

	"github.com/davidbudnick/es-tui/internal/types"

	"charm.land/lipgloss/v2"
)

func (m Model) viewNodes() string {
	if m.Width < 100 {
		return m.fullScreenFrame(m.buildNodesList(m.Width-2), keyDesc{"j/k", "nav"}, keyDesc{"r", "refresh"}, keyDesc{"q", "back"})
	}
	left := m.buildNodesList((m.Width*58)/100 - 2)
	right := m.buildNodePreview((m.Width*42)/100 - 2)
	return m.splitBrowse(58, left, right,
		keyDesc{"j/k", "nav"}, keyDesc{"r", "refresh"}, keyDesc{"enter", "detail"}, keyDesc{"q", "back"})
}

func (m Model) buildNodesList(width int) string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Nodes [%d]", len(m.Nodes))))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s · %s", m.CurrentConn.Name, m.Flavor.DisplayName())))
		b.WriteString("\n")
	}

	if len(m.Nodes) == 0 {
		b.WriteString(dimStyle.Render("No nodes"))
		return b.String()
	}

	// Fixed columns; node name fills remaining left-pane width.
	const (
		ipW    = 15
		rolesW = 14 // cdfhilmrstw fits
		numW   = 5
	)
	// "▶ " + name + gaps + ip/roles/heap/ram/cpu/master
	nameW := width - (2 + 2 + ipW + 2 + rolesW + 2 + numW + 2 + numW + 2 + numW + 2 + 1)
	nameW = max(nameW, 12)
	if nameW > 48 {
		nameW = 48
	}

	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %*s  %*s  %*s  %s",
		nameW, "Name", ipW, "IP", rolesW, "Roles", numW, "Heap%", numW, "RAM%", numW, "CPU", "M")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(width))

	maxVisible := max(m.Height-10, 5)
	selectedIdx := clamp(m.SelectedNode, 0, len(m.Nodes)-1)
	start, end := listWindow(selectedIdx, len(m.Nodes), maxVisible)

	for i := start; i < end; i++ {
		n := m.Nodes[i]
		name := truncate(n.Name, nameW)
		roles := n.NodeRole
		if roles == "" {
			roles = strings.Join(n.Roles, "")
		}
		roles = truncate(roles, rolesW)
		ip := truncate(n.IP, ipW)
		master := n.Master
		if master == "" {
			master = "-"
		}

		// Build fixed-width plain cells, then style — keeps columns aligned.
		nameCell := fmt.Sprintf("%-*s", nameW, name)
		ipCell := fmt.Sprintf("%-*s", ipW, ip)
		rolesCell := fmt.Sprintf("%-*s", rolesW, roles)
		heapCell := fmt.Sprintf("%*d", numW, n.HeapPercent)
		ramCell := fmt.Sprintf("%*d", numW, n.RamPercent)
		cpuCell := fmt.Sprintf("%*d", numW, n.CPU)

		if i == selectedIdx {
			b.WriteString(selectedStyle.Render("▶ " + nameCell))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(nameCell))
		}
		b.WriteString("  ")
		if i == selectedIdx {
			b.WriteString(normalStyle.Render(ipCell))
		} else {
			b.WriteString(dimStyle.Render(ipCell))
		}
		b.WriteString("  ")
		b.WriteString(tealStyle.Render(rolesCell))
		b.WriteString("  ")
		b.WriteString(heapColor(n.HeapPercent).Render(heapCell))
		b.WriteString("  ")
		if i == selectedIdx {
			b.WriteString(normalStyle.Render(ramCell))
		} else {
			b.WriteString(dimStyle.Render(ramCell))
		}
		b.WriteString("  ")
		b.WriteString(cpuColor(n.CPU).Render(cpuCell))
		b.WriteString("  ")
		b.WriteString(masterBadge(master))
		b.WriteString("\n")
	}
	if len(m.Nodes) > maxVisible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d-%d of %d", start+1, end, len(m.Nodes))))
	}
	return b.String()
}

func (m Model) buildNodePreview(width int) string {
	var b strings.Builder
	sepW := max(min(width, 40), 12)
	b.WriteString(titleStyle.Render("Preview"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")

	if len(m.Nodes) == 0 {
		b.WriteString(dimStyle.Render("No node selected"))
		return b.String()
	}
	n := m.Nodes[clamp(m.SelectedNode, 0, len(m.Nodes)-1)]

	writeKV := func(k, v string, valStyle lipgloss.Style) {
		b.WriteString(keyStyle.Render(k + ": "))
		b.WriteString(valStyle.Render(v))
		b.WriteString("\n")
	}

	writeKV("Name", n.Name, normalStyle)
	writeKV("IP", n.IP, dimStyle)
	if n.Host != "" {
		writeKV("Host", n.Host, dimStyle)
	}
	roles := n.NodeRole
	if roles == "" {
		roles = strings.Join(n.Roles, ",")
	}
	writeKV("Roles", roles, tealStyle)
	writeKV("Version", n.Version, normalStyle)
	b.WriteString(keyStyle.Render("Master: "))
	b.WriteString(masterBadge(n.Master))
	b.WriteString("\n")

	b.WriteString(dimStyle.Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Resources"))
	b.WriteString("\n")

	b.WriteString(keyStyle.Render("Heap: "))
	b.WriteString(heapColor(n.HeapPercent).Bold(true).Render(fmt.Sprintf("%d%%", n.HeapPercent)))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("RAM:  "))
	b.WriteString(normalStyle.Render(fmt.Sprintf("%d%%", n.RamPercent)))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("CPU:  "))
	b.WriteString(cpuColor(n.CPU).Bold(true).Render(fmt.Sprintf("%d%%", n.CPU)))
	b.WriteString("\n")
	if n.Load1m != "" {
		writeKV("Load 1m", n.Load1m, normalStyle)
	}
	if n.DiskUsedPercent != "" {
		writeKV("Disk", fmt.Sprintf("%s%%  %s / %s", n.DiskUsedPercent, n.DiskUsed, n.DiskTotal), normalStyle)
	}
	if n.DiskAvail != "" {
		writeKV("Avail", n.DiskAvail, dimStyle)
	}
	return b.String()
}

func (m Model) viewShards() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Shards [%d]", len(m.Shards))))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render("  " + m.CurrentConn.Name))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.Shards) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No shards"), keyDesc{"q", "back"})
	}

	header := fmt.Sprintf("  %-28s %5s %3s %-10s %8s %8s %s", "Index", "Shard", "P/R", "State", "Docs", "Store", "Node")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	maxVisible := max(m.Height-10, 8)
	selectedIdx := clamp(m.DetailScroll, 0, len(m.Shards)-1)
	start, end := listWindow(selectedIdx, len(m.Shards), maxVisible)
	for i := start; i < end; i++ {
		s := m.Shards[i]
		stateStyle := normalStyle
		switch strings.ToUpper(s.State) {
		case "STARTED":
			stateStyle = healthGreen
		case "RELOCATING", "INITIALIZING":
			stateStyle = healthYellow
		case "UNASSIGNED":
			stateStyle = healthRed
		}
		if i == selectedIdx {
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-28s", truncate(s.Index, 28))))
			b.WriteString(fmt.Sprintf(" %5s %3s ", s.Shard, s.Prirep))
			b.WriteString(stateStyle.Bold(true).Render(fmt.Sprintf("%-10s", s.State)))
			b.WriteString(fmt.Sprintf(" %8s %8s %s", s.Docs, s.Store, s.Node))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-28s", truncate(s.Index, 28))))
			b.WriteString(dimStyle.Render(fmt.Sprintf(" %5s %3s ", s.Shard, s.Prirep)))
			b.WriteString(stateStyle.Render(fmt.Sprintf("%-10s", s.State)))
			b.WriteString(dimStyle.Render(fmt.Sprintf(" %8s %8s %s", s.Docs, s.Store, s.Node)))
		}
		b.WriteString("\n")
	}
	if len(m.Shards) > maxVisible {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d-%d of %d", start+1, end, len(m.Shards))))
	}
	return m.fullScreenFrame(b.String(), keyDesc{"j/k", "nav"}, keyDesc{"q", "back"})
}

func (m Model) viewClusterHealth() string {
	h := m.ClusterHealth
	var b strings.Builder
	b.WriteString(m.listHeader("Cluster Health"))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s · %s", m.CurrentConn.Name, m.Flavor.DisplayName())))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.tableSep(min(m.Width-4, 70)))
	b.WriteString("\n")

	row := func(label, value string, vs lipgloss.Style) {
		b.WriteString(keyStyle.Render(fmt.Sprintf("  %-16s", label)))
		b.WriteString(vs.Render(value))
		b.WriteString("\n\n")
	}

	row("Cluster", h.ClusterName, normalStyle)
	b.WriteString(keyStyle.Render(fmt.Sprintf("  %-16s", "Status")))
	b.WriteString(healthStyle(h.Status).Bold(true).Render(strings.ToUpper(h.Status)))
	b.WriteString("\n\n")
	row("Nodes", fmt.Sprintf("%d total · %d data", h.NumberOfNodes, h.NumberOfDataNodes), normalStyle)
	row("Active shards", fmt.Sprintf("%d (%d primary)", h.ActiveShards, h.ActivePrimaryShards), normalStyle)
	row("Relocating", fmt.Sprintf("%d", h.RelocatingShards), dimStyle)
	row("Initializing", fmt.Sprintf("%d", h.InitializingShards), dimStyle)
	row("Unassigned", fmt.Sprintf("%d", h.UnassignedShards), healthStyle(map[bool]string{true: "red", false: "green"}[h.UnassignedShards > 0]))
	row("Active %", fmt.Sprintf("%.1f%%", h.ActiveShardsPercentAsNumber), normalStyle)
	if m.ClusterInfo.Version.Number != "" {
		row("Version", m.ClusterInfo.Version.Number, tealStyle)
		if m.ClusterInfo.Tagline != "" {
			row("Tagline", m.ClusterInfo.Tagline, dimStyle)
		}
	}
	return m.fullScreenFrame(b.String(), keyDesc{"r", "refresh"}, keyDesc{"q", "back"})
}

func (m Model) viewAliases() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Aliases [%d]", len(m.Aliases))))
	b.WriteString("\n")
	if len(m.Aliases) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No aliases"), keyDesc{"q", "back"})
	}
	header := fmt.Sprintf("  %-32s  %s", "Alias", "Index")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	maxVisible := max(m.Height-10, 8)
	selectedIdx := clamp(m.DetailScroll, 0, len(m.Aliases)-1)
	start, end := listWindow(selectedIdx, len(m.Aliases), maxVisible)
	for i := start; i < end; i++ {
		a := m.Aliases[i]
		if i == selectedIdx {
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-32s", truncate(a.Alias, 32))))
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(a.Index))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-32s", truncate(a.Alias, 32))))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(a.Index))
		}
		b.WriteString("\n")
	}
	return m.fullScreenFrame(b.String(), keyDesc{"j/k", "nav"}, keyDesc{"q", "back"})
}

func (m Model) viewTemplates() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Index Templates [%d]", len(m.Templates))))
	b.WriteString("\n")
	if len(m.Templates) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No templates"), keyDesc{"q", "back"})
	}
	header := fmt.Sprintf("  %-28s  %s", "Name", "Patterns")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))

	maxVisible := max(m.Height-10, 8)
	selectedIdx := clamp(m.DetailScroll, 0, len(m.Templates)-1)
	start, end := listWindow(selectedIdx, len(m.Templates), maxVisible)
	for i := start; i < end; i++ {
		t := m.Templates[i]
		patterns := strings.Join(t.IndexPatterns, ", ")
		if i == selectedIdx {
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-28s", truncate(t.Name, 28))))
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(patterns))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-28s", truncate(t.Name, 28))))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(patterns))
		}
		b.WriteString("\n")
	}
	return m.fullScreenFrame(b.String(), keyDesc{"j/k", "nav"}, keyDesc{"q", "back"})
}

func (m Model) viewLiveMetrics() string {
	var b strings.Builder
	b.WriteString(m.listHeader("Live Metrics"))
	if m.CurrentConn != nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %s · auto-refresh 2s", m.CurrentConn.Name)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.LiveMetrics == nil {
		b.WriteString(dimStyle.Render("  Collecting metrics..."))
		return m.fullScreenFrame(b.String(), keyDesc{"esc", "back"})
	}
	d := m.LiveMetrics.Latest
	hist := m.LiveMetrics.History
	heapPct := int(d.JVMHeapUsedPct)
	cpuPct := int(d.CPUPercent)
	cardW := max((m.Width-10)/5, 18)

	// Top strip: five equal cards spanning the width.
	cards := []string{
		metricsCard(cardW, "Status", healthStyle(d.Status).Bold(true).Render(strings.ToUpper(d.Status))),
		metricsCard(cardW, "Nodes", normalStyle.Bold(true).Render(fmt.Sprintf("%d", d.Nodes))+dimStyle.Render(fmt.Sprintf("  ·  %d data", d.DataNodes))),
		metricsCard(cardW, "Shards", normalStyle.Bold(true).Render(fmt.Sprintf("%d", d.ActiveShards))+dimStyle.Render(fmt.Sprintf(" active  ·  %d unassigned", d.UnassignedShards))),
		metricsCard(cardW, "Docs", normalStyle.Bold(true).Render(fmt.Sprintf("%d", d.DocsCount))),
		metricsCard(cardW, "Store", normalStyle.Bold(true).Render(formatBytes(d.StoreSizeBytes))),
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cards...))
	b.WriteString("\n\n")

	// Four large metric tiles.
	tileW := max((m.Width-8)/2, 40)
	sparkW := max(tileW-6, 24)
	tiles := []string{
		metricsTile(tileW, "Search", tealStyle.Bold(true).Render(fmt.Sprintf("%d", d.QueryTotal)),
			dimStyle.Render(fmt.Sprintf("%.2f ms avg latency", d.SearchLatencyMs)),
			accentStyle.Render(sparkValues(hist, func(h types.LiveMetricsData) float64 { return float64(h.QueryTotal) }, sparkW))),
		metricsTile(tileW, "Indexing", greenStyle.Bold(true).Render(fmt.Sprintf("%d", d.IndexingTotal)),
			dimStyle.Render("total index operations"),
			greenStyle.Render(sparkValues(hist, func(h types.LiveMetricsData) float64 { return float64(h.IndexingTotal) }, sparkW))),
		metricsTile(tileW, "JVM Heap", heapColor(heapPct).Bold(true).Render(fmt.Sprintf("%.1f%%", d.JVMHeapUsedPct)),
			heapColor(heapPct).Render(barGauge(heapPct, max(tileW-10, 20))),
			heapColor(heapPct).Render(sparkValues(hist, func(h types.LiveMetricsData) float64 { return h.JVMHeapUsedPct }, sparkW))),
		metricsTile(tileW, "CPU", cpuColor(cpuPct).Bold(true).Render(fmt.Sprintf("%.1f%%", d.CPUPercent)),
			cpuColor(cpuPct).Render(barGauge(cpuPct, max(tileW-10, 20))),
			cpuColor(cpuPct).Render(sparkValues(hist, func(h types.LiveMetricsData) float64 { return h.CPUPercent }, sparkW))),
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tiles[0], tiles[1]))
	b.WriteString("\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tiles[2], tiles[3]))
	b.WriteString("\n\n")

	// Dense cluster summary table.
	b.WriteString(titleStyle.Render("Cluster"))
	b.WriteString("\n")
	b.WriteString(m.tableSep(min(m.Width-4, 80)))
	b.WriteString(metricsKV("Status", strings.ToUpper(d.Status), healthStyle(d.Status).Bold(true)))
	b.WriteString(metricsKV("Nodes", fmt.Sprintf("%d total · %d data", d.Nodes, d.DataNodes), normalStyle))
	b.WriteString(metricsKV("Shards", fmt.Sprintf("%d active · %d unassigned", d.ActiveShards, d.UnassignedShards), normalStyle))
	b.WriteString(metricsKV("Documents", fmt.Sprintf("%d", d.DocsCount), normalStyle))
	b.WriteString(metricsKV("Store size", formatBytes(d.StoreSizeBytes), normalStyle))
	b.WriteString(metricsKV("Search", fmt.Sprintf("%d queries · %.2f ms avg", d.QueryTotal, d.SearchLatencyMs), tealStyle))
	b.WriteString(metricsKV("Indexing", fmt.Sprintf("%d", d.IndexingTotal), normalStyle))
	b.WriteString(metricsKV("JVM Heap", fmt.Sprintf("%.1f%%  %s", d.JVMHeapUsedPct, barGauge(heapPct, 24)), heapColor(heapPct)))
	b.WriteString(metricsKV("CPU", fmt.Sprintf("%.1f%%  %s", d.CPUPercent, barGauge(cpuPct, 24)), cpuColor(cpuPct)))
	if n := len(hist); n > 0 {
		b.WriteString(metricsKV("Samples", fmt.Sprintf("%d (2s interval)", n), dimStyle))
	}

	return m.fullScreenFrame(b.String(), keyDesc{"auto", "refresh 2s"}, keyDesc{"esc", "back"})
}

func metricsCard(width int, label, value string) string {
	inner := dimStyle.Render(label) + "\n" + value
	return statsBoxStyle.Width(width).MarginRight(1).Render(inner)
}

func metricsTile(width int, title, primary, secondary, spark string) string {
	var b strings.Builder
	b.WriteString(dimStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(primary)
	b.WriteString("\n")
	if secondary != "" {
		b.WriteString(secondary)
		b.WriteString("\n")
	}
	if spark != "" {
		b.WriteString(spark)
	}
	return statsBoxStyle.Width(width).Height(7).MarginRight(1).MarginBottom(1).Render(b.String())
}

func metricsKV(label, value string, vs lipgloss.Style) string {
	return keyStyle.Render(fmt.Sprintf("  %-14s", label)) + vs.Render(value) + "\n"
}

func barGauge(pct, width int) string {
	if width < 4 {
		width = 4
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := (pct * width) / 100
	if filled == 0 && pct > 0 {
		filled = 1
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func sparkValues(history []types.LiveMetricsData, get func(types.LiveMetricsData) float64, width int) string {
	if len(history) == 0 || width < 4 {
		return ""
	}
	bars := []rune("▁▂▃▄▅▆▇█")
	// Stretch short history across the full spark width so idle clusters still look full.
	samples := make([]float64, 0, len(history))
	for _, h := range history {
		samples = append(samples, get(h))
	}
	stretched := make([]float64, width)
	if len(samples) == 1 {
		for i := range stretched {
			stretched[i] = samples[0]
		}
	} else {
		for i := 0; i < width; i++ {
			pos := float64(i) * float64(len(samples)-1) / float64(width-1)
			lo := int(pos)
			hi := min(lo+1, len(samples)-1)
			frac := pos - float64(lo)
			stretched[i] = samples[lo]*(1-frac) + samples[hi]*frac
		}
	}
	var maxV, minV float64
	minV = stretched[0]
	for _, v := range stretched {
		if v > maxV {
			maxV = v
		}
		if v < minV {
			minV = v
		}
	}
	// Expand tiny ranges so flat lines still show mid-height bars.
	span := maxV - minV
	if span < 1e-9 {
		span = 1
		minV = maxV - 1
	}
	var b strings.Builder
	for _, v := range stretched {
		idx := int((v - minV) / span * float64(len(bars)-1))
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		if idx < 0 {
			idx = 0
		}
		b.WriteRune(bars[idx])
	}
	return b.String()
}

func (m Model) viewFavorites() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Favorites [%d]", len(m.Favorites))))
	b.WriteString("\n")
	if len(m.Favorites) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No favorites — press * on an index"), keyDesc{"q", "back"})
	}
	header := fmt.Sprintf("  %-40s  %s", "Index", "Label")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))
	selected := clamp(m.SelectedFavIdx, 0, len(m.Favorites)-1)
	maxVisible := max(m.Height-10, 8)
	start, end := listWindow(selected, len(m.Favorites), maxVisible)
	for i := start; i < end; i++ {
		f := m.Favorites[i]
		label := f.Label
		if label == "" {
			label = "-"
		}
		if i == selected {
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-40s", truncate(f.Index, 40))))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(label))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-40s", truncate(f.Index, 40))))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(label))
		}
		b.WriteString("\n")
	}
	return m.fullScreenFrame(b.String(), keyDesc{"enter", "open"}, keyDesc{"d", "remove"}, keyDesc{"q", "back"})
}

func (m Model) viewRecentIndices() string {
	var b strings.Builder
	b.WriteString(m.listHeader(fmt.Sprintf("Recent Indices [%d]", len(m.RecentIndices))))
	b.WriteString("\n")
	if len(m.RecentIndices) == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No recent indices"), keyDesc{"q", "back"})
	}
	header := fmt.Sprintf("  %-40s  %s", "Index", "Accessed")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))
	selected := clamp(m.SelectedRecentIdx, 0, len(m.RecentIndices)-1)
	maxVisible := max(m.Height-10, 8)
	start, end := listWindow(selected, len(m.RecentIndices), maxVisible)
	for i := start; i < end; i++ {
		r := m.RecentIndices[i]
		when := r.AccessedAt.Format("15:04:05")
		if i == selected {
			b.WriteString(selectedStyle.Render("▶ "))
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%-40s", truncate(r.Index, 40))))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(when))
		} else {
			b.WriteString("  ")
			b.WriteString(normalStyle.Render(fmt.Sprintf("%-40s", truncate(r.Index, 40))))
			b.WriteString("  ")
			b.WriteString(dimStyle.Render(when))
		}
		b.WriteString("\n")
	}
	return m.fullScreenFrame(b.String(), keyDesc{"enter", "open"}, keyDesc{"q", "back"})
}

func (m Model) viewLogs() string {
	var b strings.Builder
	b.WriteString(m.listHeader("Application Logs"))
	b.WriteString("\n")
	if m.Logs == nil || m.Logs.Len() == 0 {
		return m.fullScreenFrame(b.String()+dimStyle.Render("No log entries yet"), keyDesc{"q", "back"})
	}
	logs := m.Logs.GetLogs()
	maxLines := max(m.Height-8, 10)
	start := 0
	if len(logs) > maxLines {
		start = len(logs) - maxLines
	}
	for i := start; i < len(logs); i++ {
		line := strings.TrimRight(logs[i], "\n")
		b.WriteString(dimStyle.Render(truncate(line, m.Width-4)))
		b.WriteString("\n")
	}
	return m.fullScreenFrame(b.String(), keyDesc{"q", "back"})
}

func (m Model) viewCatAPI() string {
	var b strings.Builder
	b.WriteString(m.listHeader("Cat API"))
	b.WriteString("\n")
	if m.Inputs != nil {
		b.WriteString(keyStyle.Render("Endpoint: "))
		if m.Inputs.CatInput.Focused() {
			b.WriteString(m.Inputs.CatInput.View())
		} else {
			v := m.Inputs.CatInput.Value()
			if v == "" {
				v = m.CatEndpoint
			}
			if v == "" {
				v = "indices"
			}
			b.WriteString(normalStyle.Render(v))
		}
		b.WriteString("\n\n")
	}
	b.WriteString(m.tableSep(m.Width - 4))
	b.WriteString("\n")
	if m.CatResult == "" {
		b.WriteString(dimStyle.Render("Enter an endpoint (indices, shards, nodes, aliases, …) and press enter"))
	} else {
		lines := strings.Split(m.CatResult, "\n")
		maxLines := max(m.Height-12, 8)
		if maxLines > maxCatDisplayLines {
			maxLines = maxCatDisplayLines
		}
		for i, line := range lines {
			if i >= maxLines {
				b.WriteString(dimStyle.Render(fmt.Sprintf("… %d more lines (capped)", len(lines)-i)))
				break
			}
			// Avoid painting multi‑KB rows that freeze the TTY.
			b.WriteString(truncate(line, max(m.Width-4, 40)))
			b.WriteString("\n")
		}
	}
	return m.fullScreenFrame(b.String(), keyDesc{"enter", "run"}, keyDesc{"esc", "back"})
}

func (m Model) viewJSONPanel(title, body string) string {
	var b strings.Builder
	b.WriteString(m.listHeader(title))
	b.WriteString("\n")
	b.WriteString(m.tableSep(m.Width - 4))
	b.WriteString("\n")
	if body == "" {
		b.WriteString(dimStyle.Render("(empty)"))
	} else {
		// Bound + plain first so huge mappings don't freeze paint.
		plain, trunc := boundJSONBody(body)
		if plain == "" {
			plain = body
		}
		if lines, dropped := truncateLines(plain, maxJSONPanelLines); dropped > 0 {
			plain = lines
			trunc = true
		}
		all := wrapPlainLines(strings.Split(plain, "\n"), max(m.Width-8, 40))
		maxLines := max(m.Height-10, 8)
		visible, topHint, bottomHint, _ := scrollValueLines(all, m.DetailScroll, maxLines)
		if topHint != "" {
			b.WriteString(topHint)
			b.WriteString("\n")
		}
		for i, line := range visible {
			b.WriteString(colorizeJSONLine(line))
			if i < len(visible)-1 {
				b.WriteString("\n")
			}
		}
		if bottomHint != "" {
			b.WriteString("\n")
			b.WriteString(bottomHint)
		}
		if trunc {
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("(truncated for display)"))
		}
	}
	return m.fullScreenFrame(b.String(), keyDesc{"j/k", "scroll"}, keyDesc{"q", "back"})
}
