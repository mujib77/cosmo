package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mujib77/cosmo/config"
	"github.com/mujib77/cosmo/internal/db"
	"github.com/mujib77/cosmo/internal/ui"
)

func main() {
	cfg := config.Load()

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		fmt.Println("error connecting to postgres:", err)
		os.Exit(1)
	}
	defer database.Close()

	m := ui.New(database)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}