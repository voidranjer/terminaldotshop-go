package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/terminaldotshop/terminal-sdk-go"
)

type footerState struct {
	commands []footerCommand
}

type footerCommand struct {
	key   string
	value string
}

// wordWrap breaks a string into multiple lines to fit within maxWidth
func wordWrap(text string, maxWidth int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	lines := []string{}
	currentLine := words[0]

	for _, word := range words[1:] {
		// Check if adding this word would exceed the width
		testLine := currentLine + " " + word
		if lipgloss.Width(testLine) <= maxWidth {
			currentLine = testLine
		} else {
			// Line would be too long, start a new line
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	// Add the last line
	lines = append(lines, currentLine)

	return strings.Join(lines, "\n")
}

// ToggleRegion switches between regions and creates a new client with the updated region header
func (m model) ToggleRegion() (model, tea.Cmd) {
	// Toggle between "na" and "eu"
	newRegion := terminal.RegionEu
	if m.region != nil && *m.region == terminal.RegionEu {
		newRegion = terminal.RegionNa
	}

	// Update the model's region
	m.region = &newRegion

	// Create new client with updated region
	m.client = m.CreateSDKClient()

	// Return command to reload data
	cmd := func() tea.Msg {
		_, err := m.client.Cart.Clear(m.context)
		if err != nil {
			return err
		}

		response, err := m.client.View.Init(m.context)
		if err != nil {
			return err
		}
		return response.Data
	}

	return m, cmd
}

func (m model) FooterView() string {
	bold := m.theme.TextAccent().Bold(true).Render
	base := m.theme.Base().Render

	table := m.theme.Base().
		Width(m.widthContainer).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(m.theme.Border()).
		PaddingBottom(1).
		Align(lipgloss.Center)

	if m.size == small && m.hasMenu {
		return table.Render(bold("m") + base(" menu"))
	}

	// Note: Region selection is now handled server-side based on client IP
	// but we keep the UI indicator to show which region's products are displayed
	naFlag := "🇺🇸" // US flag for North America
	euFlag := "🇪🇺" // EU flag

	var regionSelector string
	if m.region == nil || *m.region == terminal.RegionNa {
		regionSelector = base(" " + naFlag + " (US)")
	} else {
		regionSelector = base(" " + euFlag + " (EU)")
	}

	// Add other commands
	commands := []string{}
	for _, cmd := range m.state.footer.commands {
		commands = append(commands, bold(" "+cmd.key+" ")+base(cmd.value+"  "))
	}

	lines := []string{}
	if m.page == shopPage {
		lines = append(lines, bold("r")+regionSelector)
		lines = append(lines, base("  "))
	}
	lines = append(lines, commands...)

	var content string
	if m.error != nil {
		hint := "esc"

		// Calculate maximum width for error message to ensure it fits
		maxErrorWidth := m.widthContainer - lipgloss.Width(hint) - 6

		// Handle wrapping for long error messages
		errorMsg := m.error.message
		if lipgloss.Width(errorMsg) > maxErrorWidth {
			// Split into multiple lines
			errorMsg = wordWrap(errorMsg, maxErrorWidth)
		}

		msg := m.theme.PanelError().Padding(0, 1).Render(errorMsg)

		// Calculate remaining space after rendering the message
		space := max(m.widthContainer-lipgloss.Width(msg)-lipgloss.Width(hint)-2, 0)

		height := lipgloss.Height(msg)

		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			msg,
			m.theme.PanelError().Width(space).Height(height).Render(),
			m.theme.PanelError().Bold(true).Padding(0, 1).Height(height).Render(hint),
		)
	} else {
		content = "free shipping on US orders over $40"
	}

	// Add the region selector and the rest of the commands
	return lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		content,
		table.Render(
			lipgloss.JoinHorizontal(
				lipgloss.Center,
				lines...,
			),
		))
}
