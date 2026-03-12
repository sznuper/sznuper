package initcmd

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	styleSubtle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	styleHighlight = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	styleSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("78"))

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	styleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	styleCursor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	stylePreview = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")).
			Padding(0, 1).
			MarginTop(1)
)
