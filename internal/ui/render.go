package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	goodStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff88")).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffaa00")).
			Bold(true)

	critStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff4444")).
			Bold(true)
)

func healthColor(value float64, good float64, warn float64) lipgloss.Style {
	if value >= good {
		return goodStyle
	} else if value >= warn {
		return warnStyle
	}
	return critStyle
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00d4ff"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00d4ff")).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true)
)

func RenderDashboard(m Model) string {
	halfWidth := m.width/2 - 4
	halfHeight := m.height/2 - 4

	topLeft := panelStyle.
		Width(halfWidth).
		Height(halfHeight).
		Render(renderOverview(m))

	topRight := panelStyle.
		Width(halfWidth).
		Height(halfHeight).
		Render(renderQueries(m))

	bottomLeft := panelStyle.
		Width(halfWidth).
		Height(halfHeight).
		Render(renderWAL(m))

	bottomRight := panelStyle.
		Width(halfWidth).
		Height(halfHeight).
		Render(renderLocks(m))

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topLeft, topRight)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, bottomLeft, bottomRight)

	now := time.Now().Format("15:04:05")

	logo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00d4ff")).
		Bold(true).
		Render("◆ COSMO")

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Render(" POSTGRESQL MISSION CONTROL  v0.2.0")

	connInfo := ""
	if m.overview != nil {
		connInfo = labelStyle.Render("  ● LIVE  |  ") +
			valueStyle.Render(m.overview.DatabaseName)
	}

	clock := labelStyle.Render(now)

	headerLeft := lipgloss.JoinHorizontal(lipgloss.Top, logo, title)
	headerRight := lipgloss.JoinHorizontal(lipgloss.Top, connInfo, labelStyle.Render("  |  "), clock)

	header := lipgloss.NewStyle().
		Width(m.width).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.NewStyle().Width(m.width/2).Render(headerLeft),
				lipgloss.NewStyle().Width(m.width/2).Align(lipgloss.Right).Render(headerRight),
			),
		)

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Render("  [TAB] switch panel  |  [R] refresh  |  [Q] quit  |  auto-refresh 2s |  cosmo v0.2.0")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		topRow,
		bottomRow,
		footer,
	)
}

func renderOverview(m Model) string {
	if m.overview == nil {
		return "loading..."
	}
	o := m.overview

	connPct := float64(o.ActiveConns) / float64(o.MaxConns) * 100
	connStyle := healthColor(100-connPct, 80, 50)
	cacheStyle := healthColor(o.CacheHitRatio, 95, 80)

	connBar := progressBar(o.ActiveConns, o.MaxConns, 20)

    return fmt.Sprintf("%s\n\n%s %s\n%s %s\n%s %s\n%s %s\n%s\n%s %s\n%s %s\n%s %s",
    titleStyle.Render("✦ DB OVERVIEW"),
    labelStyle.Render("database:"), valueStyle.Render(o.DatabaseName),
    labelStyle.Render("version:"), valueStyle.Render("PostgreSQL "+o.Version),
    labelStyle.Render("size:"), valueStyle.Render(o.TotalSize),
    labelStyle.Render("connections:"), connStyle.Render(
        fmt.Sprintf("%d / %d", o.ActiveConns, o.MaxConns),
    ),
    connBar,
    labelStyle.Render("cache hit:"), cacheStyle.Render(
        fmt.Sprintf("%.2f%%", o.CacheHitRatio),
    ),
    labelStyle.Render("uptime:"), valueStyle.Render(o.Uptime),
    labelStyle.Render("transactions:"), valueStyle.Render(
        formatNumber(o.TransactionsPS),
    ),
)

}

func renderQueries(m Model) string {
	if len(m.queries) == 0 {
		return titleStyle.Render("✦ ACTIVE QUERIES") + "\n\nno active queries"
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("✦ ACTIVE QUERIES") + "\n\n")

	for _, q := range m.queries {
		sb.WriteString(fmt.Sprintf("%s %s  %s %s\n%s\n\n",
			labelStyle.Render("state:"), valueStyle.Render(q.State),
			labelStyle.Render("duration:"), valueStyle.Render(q.Duration),
			labelStyle.Render(truncate(q.Query, 50)),
		))
	}
	return sb.String()
}

func renderWAL(m Model) string {
	walRate := fmt.Sprintf("%.3f MB/s", m.walStats.WALRateMBPS)
	walRateStyle := goodStyle
	if m.walStats.WALRateMBPS > 10 {
		walRateStyle = warnStyle
	}
	if m.walStats.WALRateMBPS > 50 {
		walRateStyle = critStyle
	}
	walBar := progressBar(int(m.walStats.WALRateMBPS*10), 100, 20)

	return fmt.Sprintf("%s\n\n%s %s\n%s\n%s %s\n%s\n%s %s\n%s %s\n%s %s",
		titleStyle.Render("✦ WAL & MVCC"),
		labelStyle.Render("current lsn:"), goodStyle.Render(m.walStats.CurrentLSN),
		labelStyle.Render("wal rate:"), walRateStyle.Render(walRate),
		labelStyle.Render("wal rate:   ")+walRateStyle.Render(walRate)+" "+walBar,
		labelStyle.Render("────────────────────"),
		labelStyle.Render("dead tuples:"), valueStyle.Render(formatNumber(m.walStats.DeadTuples)),
		labelStyle.Render("live tuples:"), goodStyle.Render(formatNumber(m.walStats.LiveTuples)),
		labelStyle.Render("checkpoints:"), valueStyle.Render(formatNumber(m.walStats.CheckpointsPS)),
	)
}

func renderLocks(m Model) string {
	if len(m.locks) == 0 {
		return titleStyle.Render("✦ LOCKS & WAITS") + "\n\nno locks"
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("✦ LOCKS & WAITS") + "\n\n")

	for _, l := range m.locks {
		granted := "waiting"
		if l.Granted {
			granted = "granted"
		}
		sb.WriteString(fmt.Sprintf("%s %s  %s %s\n%s\n\n",
			labelStyle.Render("type:"), valueStyle.Render(l.LockType),
			labelStyle.Render("status:"), valueStyle.Render(granted),
			labelStyle.Render(truncate(l.Query, 50)),
		))
	}
	return sb.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func progressBar(current int, max int, width int) string {
	if max == 0 {
		return ""
	}
	pct := float64(current) / float64(max)
	filled := int(pct * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	if pct > 0.9 {
		return critStyle.Render(bar)
	} else if pct > 0.7 {
		return warnStyle.Render(bar)
	}
	return goodStyle.Render(bar)
}

func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}