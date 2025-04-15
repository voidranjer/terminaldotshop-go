package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m model) _createBoxInner(
	content string,
	selected bool,
	position lipgloss.Position,
	padding int,
	totalWidth int,
) string {
	padded := lipgloss.PlaceHorizontal(totalWidth, position, content)
	base := m.theme.Base().Border(lipgloss.NormalBorder()).Width(totalWidth)

	var style lipgloss.Style
	if selected {
		style = base.BorderForeground(m.theme.Accent())
	} else {
		style = base.BorderForeground(m.theme.Border())
	}
	return style.PaddingLeft(padding).Render(padded)
}

func (m model) CreateBox(content string, selected bool) string {
	return m._createBoxInner(content, selected, lipgloss.Left, 1, m.widthContent-2)
}

func (m model) CreateBoxCustom(content string, selected bool, totalWidth int) string {
	return m._createBoxInner(content, selected, lipgloss.Left, 1, totalWidth)
}

func (m model) CreateCenteredBox(content string, selected bool) string {
	return m._createBoxInner(content, selected, lipgloss.Center, 0, m.widthContent-2)
}

func (m model) CreateCenteredBoxCustom(content string, selected bool, totalWidth int) string {
	return m._createBoxInner(content, selected, lipgloss.Center, 0, totalWidth)
}
