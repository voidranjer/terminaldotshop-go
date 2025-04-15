package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m model) ErrorView() string {
	return lipgloss.Place(
		m.viewportWidth,
		m.viewportHeight,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Center,
			m.CreateCenteredBox(m.error.message, true),
		),
	)
}
