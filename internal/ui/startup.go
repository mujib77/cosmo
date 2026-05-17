package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type StartupModel struct {
	steps    []StartupStep
	current  int
	Done     bool
}

type StartupStep struct {
	message string
	done    bool
}

type startupStepDone int
type startupComplete struct{}

func NewStartup() StartupModel {
	return StartupModel{
		steps: []StartupStep{
			{message: "Initializing..."},
			{message: "Connecting to PostgreSQL..."},
			{message: "Loading WAL metrics..."},
			{message: "Loading MVCC stats..."},
			{message: "Mission Control ready."},
		},
	}
}

func (s StartupModel) Init() tea.Cmd {
	return nextStartupStep(0)
}

func nextStartupStep(step int) tea.Cmd {
	return tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
		return startupStepDone(step)
	})
}

func (s StartupModel) Update(msg tea.Msg) (StartupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case startupStepDone:
		idx := int(msg)
		if idx < len(s.steps) {
			s.steps[idx].done = true
			s.current = idx + 1
			if idx+1 < len(s.steps) {
				return s, nextStartupStep(idx + 1)
			}
			return s, tea.Tick(600*time.Millisecond, func(t time.Time) tea.Msg {
				return startupComplete{}
			})
		}
	case startupComplete:
		s.Done = true
	}
	return s, nil
}

func (s StartupModel) View() string {
	cyan := "\033[36m"
	green := "\033[32m"
	reset := "\033[0m"
	bold := "\033[1m"

	out := fmt.Sprintf("\n\n  %s%s╔═══════════════════════════════╗%s\n", bold, cyan, reset)
	out += fmt.Sprintf("  %s%s║     COSMO MISSION CONTROL     ║%s\n", bold, cyan, reset)
	out += fmt.Sprintf("  %s%s╚═══════════════════════════════╝%s\n\n", bold, cyan, reset)

	for i, step := range s.steps {
		if i < s.current {
			out += fmt.Sprintf("  %s[COSMO]%s %s  %s✓ OK%s\n",
				cyan, reset, step.message, green, reset)
		} else if i == s.current {
			out += fmt.Sprintf("  %s[COSMO]%s %s\n",
				cyan, reset, step.message)
		}
	}

	return out
}