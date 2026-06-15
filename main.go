package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mujib77/cosmo/config"
	"github.com/mujib77/cosmo/internal/db"
	"github.com/mujib77/cosmo/internal/ui"
)

type appState int

const (
	stateStartup appState = iota
	stateDashboard
)

type rootModel struct {
	state     appState
	startup   ui.StartupModel
	dashboard ui.Model
	db        *db.DB
}

func (r rootModel) Init() tea.Cmd {
	return r.startup.Init()
}

func (r rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return r, tea.Quit
		}
	case tea.WindowSizeMsg:
		// always pass window size to dashboard
		newDashboard, _ := r.dashboard.Update(msg)
		r.dashboard = newDashboard.(ui.Model)
	}

	if r.state == stateStartup {
		newStartup, cmd := r.startup.Update(msg)
		r.startup = newStartup
		if r.startup.Done {
			r.state = stateDashboard
			return r, r.dashboard.Init()
		}
		return r, cmd
	}

	newDashboard, cmd := r.dashboard.Update(msg)
	r.dashboard = newDashboard.(ui.Model)
	return r, cmd
}

func (r rootModel) View() string {
	if r.state == stateStartup {
		return r.startup.View()
	}
	return r.dashboard.View()
}

func main() {
	cfg := config.Load()

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		fmt.Println("error connecting to postgres:", err)
		os.Exit(1)
	}
	defer database.Close()

	root := rootModel{
		state:     stateStartup,
		startup:   ui.NewStartup(),
		dashboard: ui.New(database),
		db:        database,
	}

	p := tea.NewProgram(root, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
