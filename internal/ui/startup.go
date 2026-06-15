package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StartupModel struct {
	steps   []StartupStep
	current int
	Done    bool
	width   int
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
			{message: "Negotiating PostgreSQL connection"},
			{message: "Mapping activity telemetry"},
			{message: "Calibrating WAL throughput"},
			{message: "Loading MVCC and lock sensors"},
			{message: "Flight deck online"},
		},
	}
}

func (s StartupModel) Init() tea.Cmd {
	return nextStartupStep(0)
}

func nextStartupStep(step int) tea.Cmd {
	return tea.Tick(260*time.Millisecond, func(time.Time) tea.Msg {
		return startupStepDone(step)
	})
}

func (s StartupModel) Update(msg tea.Msg) (StartupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
	case startupStepDone:
		idx := int(msg)
		if idx < len(s.steps) {
			s.steps[idx].done = true
			s.current = idx + 1
			if idx+1 < len(s.steps) {
				return s, nextStartupStep(idx + 1)
			}
			return s, tea.Tick(400*time.Millisecond, func(time.Time) tea.Msg {
				return startupComplete{}
			})
		}
	case startupComplete:
		s.Done = true
	}
	return s, nil
}

func (s StartupModel) View() string {
	logo := lipgloss.NewStyle().Foreground(cyan).Bold(true).Render(strings.Join([]string{
		"   ██████╗ ██████╗ ███████╗███╗   ███╗ ██████╗",
		"  ██╔════╝██╔═══██╗██╔════╝████╗ ████║██╔═══██╗",
		"  ██║     ██║   ██║███████╗██╔████╔██║██║   ██║",
		"  ██║     ██║   ██║╚════██║██║╚██╔╝██║██║   ██║",
		"  ╚██████╗╚██████╔╝███████║██║ ╚═╝ ██║╚██████╔╝",
		"   ╚═════╝ ╚═════╝ ╚══════╝╚═╝     ╚═╝ ╚═════╝",
	}, "\n"))
	tagline := lipgloss.NewStyle().Foreground(muted).Render("  POSTGRESQL FLIGHT DECK  /  TELEMETRY BOOT")

	var lines []string
	for i, step := range s.steps {
		marker := lipgloss.NewStyle().Foreground(dim).Render("○")
		message := lipgloss.NewStyle().Foreground(muted).Render(step.message)
		status := ""
		if i < s.current {
			marker = goodStyle.Render("●")
			message = lipgloss.NewStyle().Foreground(text).Render(step.message)
			status = goodStyle.Render("READY")
		} else if i == s.current {
			marker = lipgloss.NewStyle().Foreground(cyan).Render("◉")
			message = lipgloss.NewStyle().Foreground(white).Bold(true).Render(step.message)
			status = lipgloss.NewStyle().Foreground(cyan).Render("SYNC")
		}
		lines = append(lines, fmt.Sprintf("  %s  %-38s %s", marker, message, status))
	}

	progress := float64(s.current) / float64(len(s.steps)) * 100
	barWidth := 52
	filled := int(progress / 100 * float64(barWidth))
	bar := lipgloss.NewStyle().Foreground(cyan).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(dim).Render(strings.Repeat("─", barWidth-filled))

	body := logo + "\n\n" + tagline + "\n\n" + strings.Join(lines, "\n") +
		"\n\n  " + bar + fmt.Sprintf("  %3.0f%%", progress)
	return lipgloss.NewStyle().Padding(2, 3).Render(body)
}
