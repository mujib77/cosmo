package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	cyan    = lipgloss.Color("#22D3EE")
	blue    = lipgloss.Color("#60A5FA")
	green   = lipgloss.Color("#34D399")
	amber   = lipgloss.Color("#FBBF24")
	red     = lipgloss.Color("#FB7185")
	purple  = lipgloss.Color("#A78BFA")
	white   = lipgloss.Color("#F8FAFC")
	text    = lipgloss.Color("#CBD5E1")
	muted   = lipgloss.Color("#64748B")
	dim     = lipgloss.Color("#334155")
	surface = lipgloss.Color("#0F172A")
)

var (
	goodStyle  = lipgloss.NewStyle().Foreground(green).Bold(true)
	warnStyle  = lipgloss.NewStyle().Foreground(amber).Bold(true)
	critStyle  = lipgloss.NewStyle().Foreground(red).Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(muted)
	valueStyle = lipgloss.NewStyle().Foreground(white).Bold(true)
)

var panelNames = []string{"OVERVIEW", "QUERIES", "WAL / MVCC", "LOCKS"}

func RenderDashboard(m Model) string {
	if m.width < 72 || m.height < 22 {
		return renderCompact(m)
	}

	contentWidth := m.width - 4
	header := renderHeader(m, contentWidth)
	kpis := renderKPIRow(m, contentWidth)
	tabs := renderTabs(m, contentWidth)
	bodyHeight := max(8, m.height-lipgloss.Height(header)-lipgloss.Height(kpis)-lipgloss.Height(tabs)-4)
	body := renderGrid(m, contentWidth, bodyHeight)
	footer := renderFooter(m, contentWidth)

	app := lipgloss.JoinVertical(lipgloss.Left, header, kpis, tabs, body, footer)
	return lipgloss.NewStyle().Padding(0, 2).Render(app)
}

func renderHeader(m Model, width int) string {
	wordmark := lipgloss.NewStyle().Foreground(cyan).Bold(true).Render("COSMO")
	product := lipgloss.NewStyle().Foreground(white).Bold(true).Render(" / POSTGRES FLIGHT DECK")
	version := lipgloss.NewStyle().Foreground(muted).Render("  v0.3.0")

	dbName := "CONNECTING"
	if m.overview != nil {
		dbName = strings.ToUpper(m.overview.DatabaseName)
	}
	statusDot := lipgloss.NewStyle().Foreground(green).Render("●")
	status := fmt.Sprintf("%s %s  %s  %s",
		statusDot,
		lipgloss.NewStyle().Foreground(text).Bold(true).Render("LIVE"),
		lipgloss.NewStyle().Foreground(dim).Render("│"),
		lipgloss.NewStyle().Foreground(blue).Bold(true).Render(dbName),
	)
	clock := lipgloss.NewStyle().Foreground(text).Render(time.Now().Format("15:04:05"))
	right := status + "  " + lipgloss.NewStyle().Foreground(dim).Render("│") + "  " + clock
	left := wordmark + product + version

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(max(1, width-lipgloss.Width(right))).Render(left),
		right,
	)
}

func renderKPIRow(m Model, width int) string {
	if m.overview == nil || m.walStats == nil {
		return ""
	}

	gap := 1
	cardWidth := max(12, (width-gap*3)/4)
	queryTone := green
	if len(m.queries) > 5 {
		queryTone = amber
	}

	cards := []string{
		renderKPI("CONNECTIONS", fmt.Sprintf("%d / %d", m.overview.ActiveConns, m.overview.MaxConns), sparkline(m.connectionData, 10), cyan, cardWidth),
		renderKPI("CACHE HIT", fmt.Sprintf("%.2f%%", m.overview.CacheHitRatio), sparkline(m.cacheData, 10), green, cardWidth),
		renderKPI("ACTIVE SESSIONS", strconv.Itoa(len(m.queries)), sparkline(m.queryData, 10), queryTone, cardWidth),
		renderKPI("WAL THROUGHPUT", fmt.Sprintf("%.2f MB/s", m.walStats.WALRateMBPS), sparkline(m.walData, 10), purple, cardWidth),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		cards[0], strings.Repeat(" ", gap), cards[1], strings.Repeat(" ", gap),
		cards[2], strings.Repeat(" ", gap), cards[3],
	)
}

func renderKPI(label, value, graph string, tone lipgloss.Color, width int) string {
	top := lipgloss.NewStyle().Foreground(muted).Bold(true).Render(label)
	bottom := lipgloss.JoinHorizontal(lipgloss.Bottom,
		lipgloss.NewStyle().Width(max(1, width-lipgloss.Width(graph)-4)).
			Foreground(white).Bold(true).Render(value),
		lipgloss.NewStyle().Foreground(tone).Render(graph),
	)
	return lipgloss.NewStyle().
		Width(width-2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(dim).
		Padding(0, 1).
		Render(top + "\n" + bottom)
}

func renderTabs(m Model, width int) string {
	var parts []string
	for i, name := range panelNames {
		number := lipgloss.NewStyle().Foreground(muted).Render(strconv.Itoa(i + 1))
		style := lipgloss.NewStyle().Foreground(muted).Padding(0, 2)
		if i == m.activePanel {
			style = style.Foreground(surface).Background(cyan).Bold(true)
			number = lipgloss.NewStyle().Foreground(surface).Render(strconv.Itoa(i + 1))
		}
		parts = append(parts, style.Render(number+"  "+name))
	}
	line := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return lipgloss.NewStyle().Width(width).BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(dim).PaddingTop(1).Render(line)
}

func renderGrid(m Model, width, height int) string {
	gap := 1
	leftWidth := (width - gap) * 3 / 5
	rightWidth := width - gap - leftWidth
	topHeight := max(6, (height-gap)/2)
	bottomHeight := max(6, height-gap-topHeight)

	overview := renderPanel(m, 0, leftWidth, topHeight, renderOverview(m, leftWidth-4))
	queries := renderPanel(m, 1, rightWidth, topHeight, renderQueries(m, rightWidth-4, topHeight-3))
	wal := renderPanel(m, 2, leftWidth, bottomHeight, renderWAL(m, leftWidth-4))
	locks := renderPanel(m, 3, rightWidth, bottomHeight, renderLocks(m, rightWidth-4, bottomHeight-3))

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, overview, " ", queries),
		lipgloss.JoinHorizontal(lipgloss.Top, wal, " ", locks),
	)
}

func renderPanel(m Model, index, width, height int, content string) string {
	border := dim
	if m.activePanel == index {
		border = cyan
	}
	return lipgloss.NewStyle().
		Width(max(1, width-2)).
		Height(max(1, height-2)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(content)
}

func panelTitle(index int, subtitle string) string {
	number := lipgloss.NewStyle().Foreground(cyan).Bold(true).Render(fmt.Sprintf("0%d", index+1))
	title := lipgloss.NewStyle().Foreground(white).Bold(true).Render(panelNames[index])
	meta := lipgloss.NewStyle().Foreground(muted).Render(subtitle)
	return number + "  " + title + "  " + meta
}

func renderOverview(m Model, width int) string {
	o := m.overview
	if o == nil {
		return "Awaiting telemetry..."
	}
	connPct := float64(o.ActiveConns) / float64(max(1, o.MaxConns)) * 100
	cacheTone := healthTone(o.CacheHitRatio, 95, 85)
	connTone := inverseHealthTone(connPct, 70, 90)

	leftWidth := max(18, width/2)
	left := strings.Join([]string{
		field("DATABASE", o.DatabaseName),
		field("ENGINE", "PostgreSQL "+o.Version),
		field("DATA SIZE", o.TotalSize),
		field("UPTIME", o.Uptime),
	}, "\n")
	right := strings.Join([]string{
		meterLine("CONNECTION LOAD", fmt.Sprintf("%d%%", int(connPct)), connPct, connTone, max(8, width-leftWidth-12)),
		meterLine("CACHE EFFICIENCY", fmt.Sprintf("%.2f%%", o.CacheHitRatio), o.CacheHitRatio, cacheTone, max(8, width-leftWidth-12)),
		field("TRANSACTIONS", formatNumber(o.TransactionsPS)),
		field("IDLE SESSIONS", strconv.Itoa(o.IdleConns)),
	}, "\n")

	return panelTitle(0, "DATABASE VITALS") + "\n\n" +
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(leftWidth).Render(left),
			lipgloss.NewStyle().Width(max(1, width-leftWidth)).Render(right),
		)
}

func renderQueries(m Model, width, maxRows int) string {
	count := len(m.queries)
	if count == 0 {
		return panelTitle(1, "0 SESSIONS") + "\n\n" + emptyState("NO ACTIVE WORKLOAD", "query activity will appear here")
	}

	var rows []string
	limit := min(count, max(1, (maxRows-3)/2))
	for i, q := range m.queries[:limit] {
		tone := stateTone(q.State)
		state := lipgloss.NewStyle().Foreground(tone).Bold(true).Render(strings.ToUpper(q.State))
		meta := fmt.Sprintf("#%d  %s  %s", q.PID, state, q.Duration)
		rows = append(rows,
			lipgloss.NewStyle().Foreground(text).Render(meta)+"\n"+
				lipgloss.NewStyle().Foreground(muted).Render(truncate(cleanQuery(q.Query), width-2)),
		)
		if i < limit-1 {
			rows = append(rows, lipgloss.NewStyle().Foreground(dim).Render(strings.Repeat("─", max(1, width))))
		}
	}
	return panelTitle(1, fmt.Sprintf("%d SESSIONS", count)) + "\n\n" + strings.Join(rows, "\n")
}

func renderWAL(m Model, width int) string {
	w := m.walStats
	if w == nil {
		return panelTitle(2, "AWAITING TELEMETRY")
	}
	rateTone := green
	if w.WALRateMBPS > 10 {
		rateTone = amber
	}
	if w.WALRateMBPS > 50 {
		rateTone = red
	}
	ratio := float64(w.DeadTuples) / float64(max64(1, w.DeadTuples+w.LiveTuples)) * 100

	leftWidth := max(20, width/2)
	left := strings.Join([]string{
		field("CURRENT LSN", w.CurrentLSN),
		field("WAL RATE", lipgloss.NewStyle().Foreground(rateTone).Bold(true).Render(fmt.Sprintf("%.3f MB/s", w.WALRateMBPS))),
		lipgloss.NewStyle().Foreground(purple).Render(sparkline(m.walData, max(8, leftWidth-2))),
	}, "\n")
	right := strings.Join([]string{
		field("LIVE TUPLES", formatNumber(w.LiveTuples)),
		field("DEAD TUPLES", formatNumber(w.DeadTuples)),
		meterLine("TABLE BLOAT SIGNAL", fmt.Sprintf("%.1f%%", ratio), ratio, inverseHealthTone(ratio, 10, 20), max(8, width-leftWidth-13)),
		field("LAST AUTOVACUUM", w.LastVacuum),
	}, "\n")

	return panelTitle(2, "WRITE PATH + VACUUM") + "\n\n" +
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(leftWidth).Render(left),
			lipgloss.NewStyle().Width(max(1, width-leftWidth)).Render(right),
		)
}

func renderLocks(m Model, width, maxRows int) string {
	if len(m.locks) == 0 {
		return panelTitle(3, "SYSTEM CLEAR") + "\n\n" + emptyState("NO CONTENTION", "all lock queues are clear")
	}
	var rows []string
	limit := min(len(m.locks), max(1, (maxRows-3)/2))
	for _, lock := range m.locks[:limit] {
		status := goodStyle.Render("GRANTED")
		if !lock.Granted {
			status = critStyle.Render("WAITING")
		}
		rows = append(rows,
			fmt.Sprintf("#%d  %s  %s  %s", lock.PID, status, lock.LockType, lock.Table)+"\n"+
				lipgloss.NewStyle().Foreground(muted).Render(truncate(cleanQuery(lock.Query), width-2)),
		)
	}
	return panelTitle(3, fmt.Sprintf("%d EVENTS", len(m.locks))) + "\n\n" + strings.Join(rows, "\n")
}

func renderFooter(m Model, width int) string {
	age := "waiting for telemetry"
	if !m.lastUpdated.IsZero() {
		age = "synced " + relativeAge(m.lastUpdated)
	}
	if m.refreshing {
		age = lipgloss.NewStyle().Foreground(cyan).Render("refreshing telemetry...")
	}
	keys := key("TAB", "navigate") + "  " + key("1-4", "jump") + "  " + key("R", "refresh") + "  " + key("Q", "quit")
	sync := lipgloss.NewStyle().Foreground(muted).Render(age + "  •  2s cadence")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(max(1, width-lipgloss.Width(sync))).PaddingTop(1).Render(keys),
		lipgloss.NewStyle().PaddingTop(1).Render(sync),
	)
}

func renderCompact(m Model) string {
	width := max(24, m.width-4)
	header := lipgloss.NewStyle().Foreground(cyan).Bold(true).Render("COSMO") +
		lipgloss.NewStyle().Foreground(muted).Render(" / FLIGHT DECK")
	content := ""
	switch m.activePanel {
	case 0:
		content = renderOverview(m, width-4)
	case 1:
		content = renderQueries(m, width-4, max(6, m.height-8))
	case 2:
		content = renderWAL(m, width-4)
	case 3:
		content = renderLocks(m, width-4, max(6, m.height-8))
	}
	panel := renderPanel(m, m.activePanel, width, max(8, m.height-5), content)
	return lipgloss.NewStyle().Padding(0, 2).Render(header + "\n" + renderTabs(m, width) + "\n" + panel)
}

func field(label, value string) string {
	return lipgloss.NewStyle().Foreground(muted).Render(fmt.Sprintf("%-17s", label)) +
		lipgloss.NewStyle().Foreground(white).Bold(true).Render(value)
}

func meterLine(label, value string, pct float64, tone lipgloss.Color, width int) string {
	pct = math.Max(0, math.Min(100, pct))
	filled := int(math.Round(pct / 100 * float64(width)))
	bar := strings.Repeat("━", filled) + strings.Repeat("─", max(0, width-filled))
	return lipgloss.NewStyle().Foreground(muted).Render(label) + "  " +
		lipgloss.NewStyle().Foreground(tone).Render(bar) + " " +
		lipgloss.NewStyle().Foreground(tone).Bold(true).Render(value)
}

func emptyState(title, subtitle string) string {
	return goodStyle.Render("✓  "+title) + "\n" + lipgloss.NewStyle().Foreground(muted).Render("   "+subtitle)
}

func key(k, action string) string {
	return lipgloss.NewStyle().Foreground(surface).Background(dim).Bold(true).Padding(0, 1).Render(k) +
		lipgloss.NewStyle().Foreground(muted).Render(" "+action)
}

func sparkline(values []float64, width int) string {
	const bars = "▁▂▃▄▅▆▇█"
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return strings.Repeat("▁", width)
	}
	if len(values) > width {
		values = values[len(values)-width:]
	}
	minValue, maxValue := values[0], values[0]
	for _, value := range values[1:] {
		minValue = math.Min(minValue, value)
		maxValue = math.Max(maxValue, value)
	}
	var result strings.Builder
	result.WriteString(strings.Repeat("▁", width-len(values)))
	for _, value := range values {
		index := 0
		if maxValue > minValue {
			index = int(math.Round((value - minValue) / (maxValue - minValue) * 7))
		} else if value > 0 {
			index = 4
		}
		result.WriteRune([]rune(bars)[index])
	}
	return result.String()
}

func healthTone(value, good, warn float64) lipgloss.Color {
	if value >= good {
		return green
	}
	if value >= warn {
		return amber
	}
	return red
}

func inverseHealthTone(value, warn, critical float64) lipgloss.Color {
	if value >= critical {
		return red
	}
	if value >= warn {
		return amber
	}
	return green
}

func stateTone(state string) lipgloss.Color {
	switch strings.ToLower(state) {
	case "active":
		return green
	case "idle in transaction":
		return amber
	default:
		return blue
	}
}

func cleanQuery(query string) string {
	return strings.Join(strings.Fields(query), " ")
}

func truncate(s string, n int) string {
	if n < 2 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func relativeAge(t time.Time) string {
	seconds := int(time.Since(t).Seconds())
	if seconds <= 0 {
		return "just now"
	}
	return fmt.Sprintf("%ds ago", seconds)
}

func formatNumber(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	raw := strconv.FormatInt(n, 10)
	for i := len(raw) - 3; i > 0; i -= 3 {
		raw = raw[:i] + "," + raw[i:]
	}
	return sign + raw
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
