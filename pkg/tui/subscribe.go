package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	terminal "github.com/terminaldotshop/terminal-sdk-go"
)

type subscribeState struct {
	product      *terminal.Product
	selected     int
	lastUpdateID int64
}

func (m model) VisibleSubscribeItems() []terminal.ProductVariant {
	items := []terminal.ProductVariant{}
	if m.state.subscribe.product == nil {
		return items
	}

	for _, i := range m.state.subscribe.product.Variants {
		items = append(items, i)
	}
	return items
}

func (m model) SubscribeItemCount() int {
	return len(m.VisibleSubscribeItems())
}

func (m model) IsSubscribing() bool {
	return m.state.subscribe.product != nil
}

func (m model) UpdateSelectedSubscribeItem(previous bool) (model, tea.Cmd) {
	if !m.IsSubscribing() {
		return m, nil
	}

	var next int
	if previous {
		next = m.state.subscribe.selected - 1
	} else {
		next = m.state.subscribe.selected + 1
	}

	if next < 0 {
		next = 0
	}

	max := m.SubscribeItemCount() - 1
	if next > max {
		next = max
	}

	m.state.subscribe.selected = next
	return m, nil
}

func (m model) SubscribeSwitch() (model, tea.Cmd) {
	m = m.SwitchPage(subscribePage)

	m.state.footer.commands = []footerCommand{
		{key: "esc", value: "back"},
		{key: "↑/↓", value: "roast"},
		{key: "enter", value: "select"},
	}

	if m.SubscribeItemCount() == 1 {
		m.subscription.ProductVariantID = terminal.String(
			m.VisibleSubscribeItems()[0].ID,
		)
		return m.ShippingSwitch()
	}

	return m, nil
}

type SubscribeUpdatedMsg struct {
	updateID int64
	updated  terminal.Cart
}

func (m model) SubscribeUpdate(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "tab":
			return m.UpdateSelectedSubscribeItem(false)
		case "k", "up", "shift+tab":
			return m.UpdateSelectedSubscribeItem(true)
		case "enter", "c":
			if !m.IsSubscribing() {
				return m, nil
			}
			m.subscription.ProductVariantID = terminal.String(m.VisibleSubscribeItems()[m.state.subscribe.selected].ID)
			return m.ShippingSwitch()
		case "esc":
			m.state.subscribe.selected = 0
			m.state.subscribe.product = nil
			m.subscription = terminal.SubscriptionParam{}
			return m.ShopSwitch()
		}
	}

	return m, nil
}

func (m model) SubscribeView() string {
	base := m.theme.Base().Align(lipgloss.Left).Render
	accent := m.theme.TextAccent().Render

	if !m.IsSubscribing() {
		return lipgloss.Place(
			m.widthContent,
			m.heightContent,
			lipgloss.Center,
			lipgloss.Center,
			base("You haven't selected a product to subscribe to."),
		)
	}

	var lines []string
	for i, item := range m.VisibleSubscribeItems() {
		name := accent(item.Name)
		subtotal := m.theme.Base().Render(fmt.Sprintf("$%v", item.Price/100))
		space := m.widthContent - lipgloss.Width(
			name,
		) - lipgloss.Width(
			subtotal,
		) - 4

		content := lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				name,
				m.theme.Base().Width(space).Render(),
				subtotal,
			),
		)

		line := m.CreateBox(content, i == m.state.subscribe.selected)
		lines = append(lines, line)
	}

	return m.theme.Base().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		lines...,
	))
}
