package cmd

import (
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type (
	pbMsg  int    // how many courses we've scraped so far
	logMsg string // a line of text we want to show above the bar
)

type pbModel struct {
	bar           progress.Model
	total         int
	done          int
	logs          []string
	width, height int
}

func newPB(total int) pbModel {
	return pbModel{
		bar:   progress.New(progress.WithDefaultGradient()),
		total: total,
	}
}

func (m pbModel) Init() tea.Cmd { return nil }

func (m pbModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {

	case pbMsg:
		m.done = int(v)
		cmd := m.bar.SetPercent(float64(m.done) / float64(m.total))

		if m.done >= m.total {
			return m, tea.Batch(cmd, tea.Quit)
		}

		return m, cmd

	case logMsg:
		m.logs = append(m.logs, strings.TrimRight(string(v), "\n"))
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
		m.bar.Width = v.Width - 4
		return m, nil

	case progress.FrameMsg:
		b, cmd := m.bar.Update(msg)
		m.bar = b.(progress.Model)
		return m, cmd
	}
	return m, nil
}

func (m pbModel) View() string {
	maxLogs := m.height - 1
	if maxLogs < 0 {
		maxLogs = 0
	}
	start := 0
	if len(m.logs) > maxLogs {
		start = len(m.logs) - maxLogs
	}
	visibleLogs := strings.Join(m.logs[start:], "\n")
	return visibleLogs + "\n" + m.bar.View()
}
