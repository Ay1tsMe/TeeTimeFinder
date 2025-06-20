package cmd

import "github.com/charmbracelet/lipgloss"

var (
	defaultStyle   = lipgloss.NewStyle()
	hoverStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	titleStyle     = lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("15")).Bold(true)
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	controlStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	blacklistStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
)
