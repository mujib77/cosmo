package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/mujib77/cosmo/internal/db"
)

type Model struct {
	db *db.DB
	overview *db.OverviewStats
	queries []db.ActiveQuery
	walStats *db.WALStats
	locks []db.LockInfo
	width int
	height int
	err error
	loading bool
	activePanel int
}

type tickMsg time.Time
type dataMsg struct {
	overview *db.OverviewStats
	queries []db.ActiveQuery
	walStats *db.WALStats
	locks []db.LockInfo
	err error
}

// New creates and initializes a new Model instance.
func New(database *db.DB) Model {
	return Model{db: database, loading: true,}
}

// Init initializes the model, fetching initial data and starting the ticker.
func (m Model) Init() tea.Cmd{
	return tea.Batch(
		m.fetchData(),
		tick(),
	)
}

// tick returns a tea.Cmd that sends a tickMsg after an interval.
func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchData retrieves all the required stats from the database asynchronously.
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
			queries: queries,
			walStats: walStats,
			locks: locks,
		}
	}
}

// Update handles messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "tab":
				m.activePanel = (m.activePanel + 1) % 4
			case "r", "R":
				return m, m.fetchData()
			}
		
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height

	    case tickMsg:
			return m, tea.Batch(m.fetchData(), tick())

		case dataMsg:
			m.loading = false
			if msg.err != nil {
				m.err = msg.err
				return m, nil
			}
			m.overview = msg.overview
			m.queries = msg.queries
			m.walStats = msg.walStats
			m.locks = msg.locks
		}
	return m, nil
	}

// View renders the application UI based on the current model state.
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