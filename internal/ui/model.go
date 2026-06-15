package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/mujib77/cosmo/internal/db"
)

type Model struct {
	db             *db.DB
	overview       *db.OverviewStats
	queries        []db.ActiveQuery
	walStats       *db.WALStats
	locks          []db.LockInfo
	width          int
	height         int
	err            error
	loading        bool
	refreshing     bool
	activePanel    int
	lastUpdated    time.Time
	connectionData []float64
	cacheData      []float64
	walData        []float64
	queryData      []float64
}

type tickMsg time.Time
type dataMsg struct {
	overview *db.OverviewStats
	queries  []db.ActiveQuery
	walStats *db.WALStats
	locks    []db.LockInfo
	err      error
}

func New(database *db.DB) Model {
	return Model{db: database, loading: true}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchData(),
		tick(),
	)
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) fetchData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		overview, err := m.db.GetOverviewStats(ctx)
		if err != nil {
			return dataMsg{err: err}
		}

		queries, err := m.db.GetActiveQueries(ctx)
		if err != nil {
			return dataMsg{err: err}
		}

		walStats, err := m.db.GetWALStats(ctx)
		if err != nil {
			return dataMsg{err: err}
		}

		locks, err := m.db.GetLocks(ctx)
		if err != nil {
			return dataMsg{err: err}
		}

		return dataMsg{
			overview: overview,
			queries:  queries,
			walStats: walStats,
			locks:    locks,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
  case tea.KeyMsg:
		switch msg.String() {
  case "q", "ctrl+c":
			return m, tea.Quit
  case "tab":
			m.activePanel = (m.activePanel + 1) % 4
  case "shift+tab":
			m.activePanel = (m.activePanel + 3) % 4
		
  case "left", "h":
			m.activePanel = (m.activePanel + 3) % 4
   case "right", "l":
			m.activePanel = (m.activePanel + 1) % 4
   case "1", "2", "3", "4":
			m.activePanel = int(msg.Runes[0] - '1')
	case "r", "R":
			m.refreshing = true
			return m, m.fetchData()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tea.Batch(m.fetchData(), tick())

	case dataMsg:
		m.loading = false
		m.refreshing = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.overview = msg.overview
		m.queries = msg.queries
		m.walStats = msg.walStats
		m.locks = msg.locks
		m.lastUpdated = time.Now()
		m.connectionData = appendSample(m.connectionData, float64(msg.overview.ActiveConns), 24)
		m.cacheData = appendSample(m.cacheData, msg.overview.CacheHitRatio, 24)
		m.walData = appendSample(m.walData, msg.walStats.WALRateMBPS, 24)
		m.queryData = appendSample(m.queryData, float64(len(msg.queries)), 24)
	}
	return m, nil
}

func appendSample(samples []float64, value float64, limit int) []float64 {
	samples = append(samples, value)
	if len(samples) > limit {
		samples = samples[len(samples)-limit:]
	}
	return samples
}

func (m Model) View() string {
	if m.loading {
		return "\n connecting to postgres...\n"
	}
	if m.err != nil {
		return "\n error: " + m.err.Error() + "\n"
	}
	if m.width == 0 {
		return "\n  loading dashboard...\n"
	}
	return RenderDashboard(m)
}
