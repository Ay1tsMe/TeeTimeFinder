// Copyright (c) 2024 Adam Wyatt
//
// This software is licensed under the MIT License.
// See the LICENSE file in the root of the repository for details.

package cmd

import (
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
)

const slotsPerPage = 18 // rows per page

type pagerModel struct {
	lines     []string // fully-rendered “07:03 am – 4 spots” strings
	paginator paginator.Model
}

func newPagerModel(lines []string) pagerModel {
	p := paginator.New()
	p.Type = paginator.Dots // slick “•••” footer
	p.PerPage = slotsPerPage
	p.InactiveDot = "·" // grey “dot”
	p.ActiveDot = "●"
	p.SetTotalPages(len(lines))

	return pagerModel{lines: lines, paginator: p}
}

func (m pagerModel) Init() tea.Cmd { return nil }

func (m pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c", "enter":
			return m, tea.Quit
		}
	}

	// pass input to paginator (--, space, left/right, etc.)
	var cmd tea.Cmd
	m.paginator, cmd = m.paginator.Update(msg)
	return m, cmd
}

func (m pagerModel) View() string {
	if len(m.lines) == 0 {
		return "No available timeslots\n"
	}

	var b strings.Builder
	b.WriteString("\n  Available Tee-Times\n\n")

	// slice for the current page
	start, end := m.paginator.GetSliceBounds(len(m.lines))
	for _, l := range m.lines[start:end] {

		if strings.HasSuffix(l, ":") {
			b.WriteString("  • " + strings.TrimSuffix(l, "\n") + "\n\n")
		} else {
			b.WriteString("    • " + strings.TrimSuffix(l, "\n") + "\n\n")
		}
	}

	b.WriteString("  " + m.paginator.View())

	help := controlStyle.Render("\n\n  h/l ←/→ page • q/enter: continue\n")
	b.WriteString(help)

	return b.String()
}
