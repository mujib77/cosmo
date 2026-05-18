package ui

import (
	"fmt"
	"time"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
    header := lipgloss.JoinHorizontal(
    lipgloss.Top,
    titleStyle.Render("  COSMO — PostgreSQL Mission Control  "),
    labelStyle.Render("  "+now+"  "),
)
	footer := labelStyle.Render("  tab: switch panel  q: quit  ")

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
	return fmt.Sprintf("%s\n\n%s %s\n%s %s\n%s %d / %d\n%s %.2f%%\n%s %s",
		titleStyle.Render("DB OVERVIEW"),
		labelStyle.Render("database:"), valueStyle.Render(o.DatabaseName),
		labelStyle.Render("size:"), valueStyle.Render(o.TotalSize),
		labelStyle.Render("connections:"), o.ActiveConns, o.MaxConns,
		labelStyle.Render("cache hit:"), o.CacheHitRatio,
		labelStyle.Render("uptime:"), valueStyle.Render(o.Uptime),
	)
}

func renderQueries(m Model) string {
	if len(m.queries) == 0 {
		return titleStyle.Render("ACTIVE QUERIES") + "\n\nno active queries"
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("ACTIVE QUERIES") + "\n\n")

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
	if m.walStats == nil {
		return "loading..."
	}
	w := m.walStats
	return fmt.Sprintf("%s\n\n%s %s\n%s %d\n%s %d\n%s %d",
		titleStyle.Render("WAL & MVCC"),
		labelStyle.Render("current lsn:"), valueStyle.Render(w.CurrentLSN),
		labelStyle.Render("dead tuples:"), w.DeadTuples,
		labelStyle.Render("live tuples:"), w.LiveTuples,
		labelStyle.Render("checkpoints:"), w.CheckpointsPS,
	)
}

func renderLocks(m Model) string {
	if len(m.locks) == 0 {
		return titleStyle.Render("LOCKS & WAITS") + "\n\nno locks"
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("LOCKS & WAITS") + "\n\n")

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