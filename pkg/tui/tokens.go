package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/terminaldotshop/terminal-sdk-go"
)

type TokenAddedMsg struct {
	newToken terminal.TokenNewResponseData
	tokens   []terminal.Token
}

type tokensState struct {
	selected int
	deleting *int
	newToken *terminal.TokenNewResponseData
}

func (m model) nextToken() (model, tea.Cmd) {
	next := m.state.tokens.selected + 1
	max := len(m.tokens)
	if next > max {
		next = max
	}

	m.state.tokens.selected = next
	return m, nil
}

func (m model) previousToken() (model, tea.Cmd) {
	next := m.state.tokens.selected - 1
	if next < 0 {
		next = 0
	}

	m.state.tokens.selected = next
	return m, nil
}

func (m model) TokensUpdate(msg tea.Msg) (model, tea.Cmd) {
	m.state.footer.commands = []footerCommand{
		{key: "↑/↓", value: "navigate"},
		{key: "x/del", value: "revoke"},
		{key: "esc", value: "back"},
	}

	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "tab":
			if m.state.tokens.deleting == nil {
				return m.nextToken()
			}
		case "k", "up", "shift+tab":
			if m.state.tokens.deleting == nil {
				return m.previousToken()
			}
		case "delete", "d", "backspace", "x":
			if m.state.tokens.deleting == nil {
				m.state.tokens.deleting = &m.state.tokens.selected
			}
			return m, nil
		case "y":
			if m.state.tokens.deleting != nil {
				m.state.tokens.deleting = nil
				_, err := m.client.Token.Delete(m.context, m.tokens[m.state.tokens.selected].ID)
				if err != nil {
					return m, func() tea.Msg { return err }
				}
				if len(m.tokens)-1 == 0 {
					m.state.account.focused = false
				}
				return m, func() tea.Msg {
					tokens, err := m.client.Token.List(m.context)
					if err != nil {
						return err
					}
					return tokens.Data
				}
			}
			return m, nil
		case "n", "esc":
			m.state.tokens.deleting = nil
			return m, nil
		case "enter":
			if m.state.tokens.deleting == nil && m.state.tokens.selected == len(m.tokens) {
				return m, func() tea.Msg {
					response, err := m.client.Token.New(m.context)
					if err != nil {
						return err
					}
					tokens, err := m.client.Token.List(m.context)
					if err != nil {
						return err
					}
					// if m.output != nil {
					// 	m.output.Copy(m.state.tokens.newToken.Token)
					// }
					return TokenAddedMsg{
						newToken: response.Data,
						tokens:   tokens.Data,
					}
				}
			}
		}
	case TokenAddedMsg:
		m.state.tokens.newToken = &msg.newToken
		m.tokens = msg.tokens
	}

	return m, tea.Batch(cmds...)
}

func (m model) formatToken(token terminal.Token, totalWidth int) string {
	space := totalWidth - lipgloss.Width(
		token.ID,
	)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.theme.TextAccent().Render(token.ID),
		m.theme.Base().Width(space).Render(),
		// m.theme.Base().Render(price),
	)

	lines := []string{}
	lines = append(lines, content)
	lines = append(lines, "created: "+token.Created)

	if m.state.tokens.newToken != nil && token.ID == m.state.tokens.newToken.ID {
		lines = append(
			lines,
			m.theme.TextBrand().Bold(true).Render(m.state.tokens.newToken.Token),
		)
		lines = append(lines, "(will not be shown again)")
	} else {
		lines = append(lines, token.Token)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) TokensView(totalWidth int, focused bool) string {
	base := m.theme.Base().Render
	accent := m.theme.TextAccent().Render

	tokens := []string{}
	for i, token := range m.tokens {
		content := m.formatToken(token, totalWidth)
		if m.state.tokens.deleting != nil && *m.state.tokens.deleting == i {
			content = accent("are you sure you want to revoke?") + base("\n(y/n)")
		}
		box := m.CreateBoxCustom(
			content,
			focused && i == m.state.tokens.selected,
			totalWidth,
		)
		tokens = append(tokens, box)
	}

	newTokenIndex := len(m.tokens)
	newToken := m.CreateBoxCustom(
		m.formatListItemCustom("add access token", m.state.tokens.selected == newTokenIndex, totalWidth, false),
		focused && m.state.tokens.selected == newTokenIndex,
		totalWidth,
	)
	tokens = append(tokens, newToken)
	tokenList := lipgloss.JoinVertical(lipgloss.Left, tokens...)

	return m.theme.Base().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		tokenList,
	))
}
