package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/terminaldotshop/terminal-sdk-go"
)

type subscriptionsState struct {
	selected int
	deleting *int
}

func (m model) SubscriptionManageSwitch(id string) (model, tea.Cmd) {
	m = m.SwitchPage(accountPage)
	m.state.footer.commands = []footerCommand{
		{key: "↑/↓", value: "navigate"},
		{key: "x/del", value: "cancel"},
		{key: "esc", value: "back"},
	}
	for i, page := range m.accountPages {
		if page == subscriptionsPage {
			m.state.account.selected = i
			break
		}
	}
	m.state.account.focused = true

	for i, sub := range m.subscriptions {
		if sub.ID == id {
			m.state.subscriptions.selected = i
			break
		}
	}

	m.state.subscriptions.deleting = nil
	return m, nil
}

func (m model) nextSubscription() (model, tea.Cmd) {
	next := m.state.subscriptions.selected + 1
	max := len(m.subscriptions) - 1
	if next > max {
		next = max
	}

	m.state.subscriptions.selected = next
	return m, nil
}

func (m model) previousSubscription() (model, tea.Cmd) {
	next := m.state.subscriptions.selected - 1
	if next < 0 {
		next = 0
	}

	m.state.subscriptions.selected = next
	return m, nil
}

func (m model) SubscriptionsUpdate(msg tea.Msg) (model, tea.Cmd) {
	m.state.footer.commands = []footerCommand{
		{key: "↑/↓", value: "navigate"},
		{key: "x/del", value: "cancel"},
		{key: "esc", value: "back"},
	}

	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "tab":
			if m.state.subscriptions.deleting == nil {
				return m.nextSubscription()
			}
		case "k", "up", "shift+tab":
			if m.state.subscriptions.deleting == nil {
				return m.previousSubscription()
			}
		case "delete", "d", "backspace", "x":
			if m.state.subscriptions.deleting == nil {
				m.state.subscriptions.deleting = &m.state.subscriptions.selected
			}
			return m, nil
		case "y":
			if m.state.subscriptions.deleting != nil {
				m.state.subscriptions.deleting = nil
				_, err := m.client.Subscription.Delete(m.context, m.subscriptions[m.state.subscriptions.selected].ID)
				if err != nil {
					return m, func() tea.Msg { return err }
				}
				if len(m.subscriptions)-1 == 0 {
					m.state.account.focused = false
				}
				return m, func() tea.Msg {
					subscriptions, err := m.client.Subscription.List(m.context)
					if err != nil {
						return err
					}
					return subscriptions.Data
				}
			}
			return m, nil
		case "n", "esc":
			m.state.subscriptions.deleting = nil
			return m, nil
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) formatSubscription(subscription terminal.Subscription, totalWidth int) string {
	base := m.theme.Base().Render
	accent := m.theme.TextAccent().Render

	var product *terminal.Product
	var variant *terminal.ProductVariant
	for _, p := range m.products {
		for _, v := range p.Variants {
			if v.ID == subscription.ProductVariantID {
				product = &p
				variant = &v
			}
		}
	}

	if product == nil {
		return base("unknown product")
	}

	title := accent(fmt.Sprintf("%dx %s", subscription.Quantity, product.Name))
	if product.Name == "cron" {
		title = accent(product.Name)
	} else {
		scheduleType := ""
		if subscription.Schedule.Type == "weekly" {
			scheduleType = "weeks"
		}
		title = accent(title) + base(fmt.Sprintf(" (every %d %s)", subscription.Schedule.Interval, scheduleType))
	}

	price := fmt.Sprintf(" $%2v", subscription.Quantity*variant.Price/100)
	space := totalWidth - lipgloss.Width(
		title,
	) - lipgloss.Width(price) - 2
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		title,
		m.theme.Base().Width(space).Render(),
		m.theme.Base().Render(price),
	)

	lines := []string{}
	lines = append(lines, content)
	lines = append(lines, fmt.Sprintf("next shipment: %s", subscription.Next))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) SubscriptionsView(totalWidth int, focused bool) string {
	base := m.theme.Base().Render
	accent := m.theme.TextAccent().Render

	subscriptions := []string{}
	for i, subscription := range m.subscriptions {
		content := m.formatSubscription(subscription, totalWidth)
		if m.state.subscriptions.deleting != nil && *m.state.subscriptions.deleting == i {
			content = accent("are you sure?") + base("\n(y/n)")
		}
		box := m.CreateBoxCustom(
			content,
			focused && i == m.state.subscriptions.selected,
			totalWidth,
		)
		subscriptions = append(subscriptions, box)
	}

	subscriptionList := lipgloss.JoinVertical(lipgloss.Left, subscriptions...)
	if len(subscriptions) == 0 {
		return lipgloss.Place(
			totalWidth,
			m.heightContent,
			lipgloss.Center,
			lipgloss.Center,
			base("no active subscriptions"),
		)
	}

	return m.theme.Base().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		subscriptionList,
	))
}
