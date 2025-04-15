package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	terminal "github.com/terminaldotshop/terminal-sdk-go"
)

type ordersState struct {
	selected int
	viewing  bool // When true, we're viewing a single order in detail
	yOffset  int
}

func (m model) nextOrder() (model, tea.Cmd) {
	next := m.state.orders.selected + 1
	max := len(m.orders) - 1
	if next > max {
		next = max
	}

	m.state.orders.selected = next
	return m, nil
}

func (m model) previousOrder() (model, tea.Cmd) {
	next := max(m.state.orders.selected-1, 0)
	m.state.orders.selected = next
	return m, nil
}

var orderCommands = []footerCommand{
	{key: "↑/↓", value: "navigate"},
	{key: "enter", value: "view details"},
	{key: "esc", value: "back"},
}

func (m model) OrdersUpdate(msg tea.Msg) (model, tea.Cmd) {
	if len(m.state.footer.commands) != 3 && !m.state.orders.viewing {
		m.state.footer.commands = orderCommands
	}

	if m.state.orders.viewing {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "q", "backspace":
				m.state.footer.commands = orderCommands
				m.state.orders.viewing = false
				return m, nil
			}
			var cmd tea.Cmd
			m.state.account.detailViewport.KeyMap = viewport.DefaultKeyMap()
			m.state.account.detailViewport, cmd = m.state.account.detailViewport.Update(msg)
			return m, cmd
		}
	} else {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "j", "down", "tab":
				return m.nextOrder()
			case "k", "up", "shift+tab":
				return m.previousOrder()
			case "enter":
				m.state.orders.viewing = true
				m.state.orders.yOffset = m.state.account.detailViewport.YOffset
				m.state.account.detailViewport.GotoTop()
				m.state.footer.commands = []footerCommand{
					{key: "esc", value: "back to orders"},
				}
				return m, nil
			}
		}
	}

	return m, nil
}

func (m model) formatOrderItem(orderItem terminal.OrderItem) string {
	var product *terminal.Product
	// var variant *terminal.ProductVariant
	for _, p := range m.products {
		for _, v := range p.Variants {
			if v.ID == orderItem.ProductVariantID {
				product = &p
				// variant = &v
			}
		}
	}

	if product == nil {
		return "unknown product"
	}

	return fmt.Sprintf("%dx %s", orderItem.Quantity, product.Name)
}

func (m model) formatOrder(order terminal.Order, index int) string {
	orderNumber := fmt.Sprintf("order #%d", index)
	price := fmt.Sprintf("  $%2v", (order.Amount.Subtotal+order.Amount.Shipping)/100)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.theme.TextAccent().Render(orderNumber),
		m.theme.Base().Render(price),
	)

	// Show only order date instead of individual items
	lines := []string{}
	lines = append(lines, content)
	lines = append(lines, fmt.Sprintf("date: %s", order.Created))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) formatOrderDetail(order terminal.Order, index int) string {
	base := m.theme.Base().Render
	accent := m.theme.TextAccent().Render
	highlight := m.theme.TextBrand().Render

	lines := []string{}
	lines = append(lines, base("< ")+accent("esc ")+base("back to orders\n"))

	// Order header
	orderNumber := fmt.Sprintf("order #%d", index)
	lines = append(lines, highlight(orderNumber))
	lines = append(lines, base("date: ")+base(order.Created))
	lines = append(lines, "")

	// Shipping details
	if order.Tracking.Service != "" || order.Tracking.Number != "" {
		lines = append(lines, accent("shipping"))
		lines = append(lines, base("status: ")+base(order.Tracking.Status))
		if order.Tracking.Service != "" {
			lines = append(lines, base("service: ")+base(order.Tracking.Service))
		}
		if order.Tracking.Number != "" {
			lines = append(lines, base("tracking: ")+base(order.Tracking.Number))
		}
		lines = append(lines, "")
	}

	// Order items
	lines = append(lines, accent("items"))
	for _, item := range order.Items {
		itemLine := m.formatOrderItem(item)
		lines = append(lines, itemLine)
	}
	lines = append(lines, "")

	// Order totals
	lines = append(lines, accent("totals"))
	lines = append(lines, base("subtotal: ")+base(fmt.Sprintf("$%d", order.Amount.Subtotal/100)))
	lines = append(lines, base("shipping: ")+base(fmt.Sprintf("$%d", order.Amount.Shipping/100)))
	lines = append(lines, base("total: ")+base(fmt.Sprintf("$%d", (order.Amount.Subtotal+order.Amount.Shipping)/100)))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) OrdersView(totalWidth int, focused bool) string {
	base := m.theme.Base().Render

	// If we're viewing a single order in detail
	if m.state.orders.viewing && len(m.orders) > 0 {
		orderIndex := len(m.orders) - m.state.orders.selected - 1
		detailContent := m.formatOrderDetail(m.orders[m.state.orders.selected], orderIndex)
		return m.theme.Base().Width(totalWidth).Render(detailContent)
	}

	// Otherwise show order list
	orders := []string{}
	for i, order := range m.orders {
		content := m.formatListItemCustom(
			m.formatOrder(order, len(m.orders)-i-1),
			focused && i == m.state.orders.selected,
			totalWidth,
			false,
		)
		box := m.CreateBoxCustom(
			content,
			focused && i == m.state.orders.selected,
			totalWidth,
		)
		orders = append(orders, box)
	}

	orderList := lipgloss.JoinVertical(lipgloss.Left, orders...)
	if len(orders) == 0 {
		return lipgloss.Place(
			totalWidth,
			m.heightContent,
			lipgloss.Center,
			lipgloss.Center,
			base("no orders found"),
		)
	}

	return m.theme.Base().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		orderList,
	))
}
