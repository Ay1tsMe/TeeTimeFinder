package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type spinModel struct {
	sp  spinner.Model
	msg string
}

func newSpinnerModel(msg string) spinModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return spinModel{sp: s, msg: msg}
}

func (m spinModel) Init() tea.Cmd { return m.sp.Tick }

func (m spinModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.sp, cmd = m.sp.Update(msg)
	return m, cmd
}

func (m spinModel) View() string {
	// spinner + message on one line
	return fmt.Sprintf("\n  %s %s\n", m.sp.View(), m.msg)
}
