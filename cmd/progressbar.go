package cmd

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type pbMsg int // how many courses we've scraped so far

type pbModel struct {
	bar    progress.Model
	total  int
	done   int
	quitAt int // when done == quitAt we exit
}

func newPB(total int) pbModel {
	return pbModel{
		bar:    progress.New(progress.WithDefaultGradient()),
		total:  total,
		quitAt: total,
	}
}

func (m pbModel) Init() tea.Cmd { return nil }

func (m pbModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {

	case pbMsg:
		m.done = int(v)
		cmd := m.bar.SetPercent(float64(m.done) / float64(m.total))

		if m.done >= m.quitAt {
			// finish after the final frame has rendered
			return m, tea.Batch(cmd, tea.Quit)
		}
		return m, cmd

	case tea.WindowSizeMsg:
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
	const clearLine = "\r\033[K"
	return clearLine + m.bar.View()
}
