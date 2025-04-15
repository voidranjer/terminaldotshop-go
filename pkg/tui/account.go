package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type accountState struct {
	selected       int
	focused        bool
	menuViewport   viewport.Model
	detailViewport viewport.Model
	viewportsReady bool
}

func (m model) updateAccountViewports() model {
	headerHeight := lipgloss.Height(m.HeaderView())
	breadcrumbsHeight := lipgloss.Height(m.BreadcrumbsView())
	footerHeight := lipgloss.Height(m.FooterView())
	verticalMarginHeight := headerHeight + footerHeight + breadcrumbsHeight

	availableHeight := m.heightContainer - verticalMarginHeight

	// Calculate menu width based on account pages
	menuWidth := 0
	for _, p := range m.accountPages {
		w := lipgloss.Width(getAccountPageName(p))
		if w > menuWidth {
			menuWidth = w
		}
	}

	// Add padding for menu items
	if menuWidth > 0 {
		menuWidth += 4
	}

	// For small screens, make the menu full width
	if m.size < large {
		menuWidth = m.widthContent
	}

	detailWidth := m.widthContent - menuWidth
	if m.size < large {
		detailWidth = m.widthContent
	}

	if !m.state.account.viewportsReady {
		// Initialize viewports for the first time
		m.state.account.menuViewport = viewport.New(menuWidth, availableHeight)
		m.state.account.menuViewport.KeyMap = viewport.KeyMap{}

		m.state.account.detailViewport = viewport.New(detailWidth, availableHeight)
		m.state.account.detailViewport.KeyMap = modifiedKeyMap

		m.state.account.viewportsReady = true
	} else {
		// Update existing viewports
		m.state.account.menuViewport.Width = menuWidth
		m.state.account.menuViewport.Height = availableHeight

		m.state.account.detailViewport.Width = detailWidth
		m.state.account.detailViewport.Height = availableHeight
	}

	return m
}

func (m model) AccountSwitch() (model, tea.Cmd) {
	m = m.SwitchPage(accountPage)
	m.state.account.selected = 0
	m.state.account.focused = false
	m.state.tokens = tokensState{
		selected: 0,
	}
	m.state.apps = appsState{
		selected:   0,
		submitting: false,
		editing:    false,
		newApp:     nil,
		input:      appInput{},
		form:       m.createAppForm(),
	}

	m.state.footer.commands = []footerCommand{
		{key: "↑/↓", value: "navigate"},
		{key: "enter", value: "select"},
	}

	m = m.updateAccountViewports()
	m.state.account.menuViewport.GotoTop()
	m.state.account.detailViewport.GotoTop()
	return m, m.state.apps.form.Init()
}

func (m model) AccountUpdate(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	accountPage := m.accountPages[m.state.account.selected]

	// Update viewport dimensions if window size changed
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m = m.updateAccountViewports()
	}

	if m.state.account.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "left", "h":
				if !m.state.apps.editing && !m.state.orders.viewing {
					s := m.state.account.selected
					m, cmd = m.AccountSwitch()
					cmds = append(cmds, cmd)
					m.state.account.selected = s
				}
			}
		}

		var handled bool
		var nextModel model
		switch accountPage {
		case subscriptionsPage:
			nextModel, cmd = m.SubscriptionsUpdate(msg)
			handled = true
		case tokensPage:
			nextModel, cmd = m.TokensUpdate(msg)
			handled = true
		case appsPage:
			nextModel, cmd = m.AppsUpdate(msg)
			handled = true
		case ordersPage:
			nextModel, cmd = m.OrdersUpdate(msg)
			handled = true
		case shippingPage:
			nextModel, cmd = m.ShippingUpdate(msg)
			handled = true
		case paymentPage:
			nextModel, cmd = m.PaymentUpdate(msg)
			handled = true
		}

		if handled {
			// Update detail content after the selection change
			if m.state.account.viewportsReady {
				detailContent := nextModel.getAccountDetailContent()
				nextModel.state.account.detailViewport.SetContent(detailContent)

				// When navigating list items, scroll to make the selected item visible
				switch accountPage {
				case subscriptionsPage:
					if m.state.subscriptions.selected != nextModel.state.subscriptions.selected {
						nextModel = m.scrollToAccountDetailItem(nextModel, accountPage)
					}
				case tokensPage:
					if m.state.tokens.selected != nextModel.state.tokens.selected {
						nextModel = m.scrollToAccountDetailItem(nextModel, accountPage)
					}
				case appsPage:
					if m.state.apps.selected != nextModel.state.apps.selected {
						nextModel = m.scrollToAccountDetailItem(nextModel, accountPage)
					}
				case ordersPage:
					if m.state.orders.selected != nextModel.state.orders.selected {
						nextModel = m.scrollToAccountDetailItem(nextModel, accountPage)
					}
					// restore scroll position
					if nextModel.state.orders.yOffset > 0 && !nextModel.state.orders.viewing {
						nextModel.state.account.detailViewport.SetYOffset(nextModel.state.orders.yOffset)
						nextModel.state.orders.yOffset = 0
					}
				}
			}

			return nextModel, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down", "j":
			return m.UpdateSelectedAccountPage(false)
		case "shift+tab", "up", "k":
			return m.UpdateSelectedAccountPage(true)
		case "enter", "right", "l":
			if accountPage == subscriptionsPage ||
				accountPage == ordersPage ||
				accountPage == tokensPage ||
				accountPage == appsPage {
				m.state.account.focused = true
				switch accountPage {
				case subscriptionsPage:
					m.state.subscriptions.selected = 0
					return m.SubscriptionsUpdate(msg)
				case tokensPage:
					m.state.tokens.selected = 0
					return m.TokensUpdate(msg)
				case appsPage:
					m.state.apps.selected = 0
					return m.AppsUpdate(msg)
				case ordersPage:
					m.state.orders.selected = 0
					return m.OrdersUpdate(msg)
				}

			}
			return m, nil
		}
	}

	// Update viewports with new content
	if m.state.account.viewportsReady {
		// Update menu content
		menuContent := m.getAccountMenuContent()
		m.state.account.menuViewport.SetContent(menuContent)

		// Update detail content
		detailContent := m.getAccountDetailContent()
		m.state.account.detailViewport.SetContent(detailContent)
	}

	// Update the detailViewport with the message
	m.state.account.detailViewport, cmd = m.state.account.detailViewport.Update(msg)
	cmds = append(cmds, cmd)

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func getAccountPageName(accountPage page) string {
	switch accountPage {
	case ordersPage:
		return "order history"
	case subscriptionsPage:
		return "subscriptions"
	case tokensPage:
		return "access tokens"
	case appsPage:
		return "apps (oauth 2.0)"
	case shippingPage:
		return "addresses"
	case paymentPage:
		return "payment methods"
	case faqPage:
		return "faq"
	case aboutPage:
		return "about"
	}

	return ""
}

// Helper function to generate content for the menu viewport
func (m model) getAccountMenuContent() string {
	menuWidth := 0
	pages := strings.Builder{}

	// Calculate max width
	for _, p := range m.accountPages {
		w := lipgloss.Width(getAccountPageName(p))
		if w > menuWidth {
			menuWidth = w
		}
	}

	var menuItem lipgloss.Style
	var highlightedMenuItem lipgloss.Style

	if m.size < large {
		menuWidth = m.widthContent

		menuItem = m.theme.Base().
			Width(menuWidth).
			Align(lipgloss.Center)
		highlightedMenuItem = m.theme.Base().
			Width(menuWidth).
			Align(lipgloss.Center).
			Background(m.theme.Highlight()).
			Foreground(m.theme.Accent())
	} else {
		menuItem = m.theme.Base().
			Width(menuWidth+2).
			Padding(0, 1)
		highlightedMenuItem = m.theme.Base().
			Width(menuWidth+2).
			Padding(0, 1).
			Background(m.theme.Highlight()).
			Foreground(m.theme.Accent())
	}

	for i, p := range m.accountPages {
		name := getAccountPageName(p)

		var content string
		if i == m.state.account.selected {
			content = highlightedMenuItem.Render(name)
		} else {
			content = menuItem.Render(name)
		}

		pages.WriteString(content + "\n")
	}

	return m.theme.Base().Padding(0, 1).Render(pages.String())
}

// Helper function to generate content for the detail viewport
func (m model) getAccountDetailContent() string {
	accountPage := m.accountPages[m.state.account.selected]

	menuWidth := 0
	if m.size >= large {
		// Calculate menu width for large screens
		for _, p := range m.accountPages {
			w := lipgloss.Width(getAccountPageName(p))
			if w > menuWidth {
				menuWidth = w
			}
		}
		menuWidth += 4 // padding
	}

	detailWidth := m.widthContent - menuWidth - 2
	if m.size < large {
		detailWidth = m.widthContent
	}

	detail := m.GetAccountPageContent(accountPage, detailWidth)
	return detail
}

func (m model) GetAccountPageContent(accountPage page, totalWidth int) string {
	switch accountPage {
	case ordersPage:
		return m.OrdersView(totalWidth, m.state.account.focused)
	case subscriptionsPage:
		return m.SubscriptionsView(totalWidth, m.state.account.focused)
	case tokensPage:
		return m.TokensView(totalWidth, m.state.account.focused)
	case appsPage:
		return m.AppsView(totalWidth, m.state.account.focused)
	case shippingPage:
		return m.ShippingView(totalWidth, m.state.account.focused)
	case faqPage:
		return m.FaqView(totalWidth)
	case aboutPage:
		return m.AboutView(totalWidth)
	}

	return ""
}

func (m model) AccountView() string {
	if !m.state.account.viewportsReady {
		m = m.updateAccountViewports()
	}

	// Update viewport contents
	menuContent := m.getAccountMenuContent()
	m.state.account.menuViewport.SetContent(menuContent)

	detailContent := m.getAccountDetailContent()
	m.state.account.detailViewport.SetContent(detailContent)

	// Combine viewport views
	if m.size < large {
		// For small screens, stack the viewports vertically
		return lipgloss.JoinVertical(
			lipgloss.Top,
			m.state.account.menuViewport.View(),
			m.state.account.detailViewport.View(),
		)
	} else {
		// For large screens, place viewports side by side
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.state.account.menuViewport.View(),
			"  ",
			m.state.account.detailViewport.View(),
		)
	}
}

// Helper function to scroll the detail viewport to show the selected item in focused account pages
func (m model) scrollToAccountDetailItem(model model, accountPage page) model {
	// If orders page is in detail view, we don't need to scroll to a specific item
	if accountPage == ordersPage && model.state.orders.viewing {
		return model
	}

	var itemHeight int
	var itemCount int
	var selectedIndex int

	// Different item heights and counts based on the page
	switch accountPage {
	case subscriptionsPage:
		itemHeight = 5 // Estimated height of a subscription item with padding
		itemCount = len(model.subscriptions)
		selectedIndex = model.state.subscriptions.selected
	case tokensPage:
		itemHeight = 7                    // Estimated height of a token item with padding
		itemCount = len(model.tokens) + 1 // +1 for "add token" button
		selectedIndex = model.state.tokens.selected
	case appsPage:
		itemHeight = 8                  // Estimated height of an app item with padding
		itemCount = len(model.apps) + 1 // +1 for "create app" button
		selectedIndex = model.state.apps.selected
	case ordersPage:
		itemHeight = 4 // Reduced height for order item with just date (instead of all products)
		itemCount = len(model.orders)
		selectedIndex = model.state.orders.selected
	default:
		return model // No scrolling for other pages
	}

	if itemCount == 0 {
		return model // No items to scroll to
	}

	// Calculate approximate position of selected item
	targetY := (selectedIndex * itemHeight) + 2

	// Calculate offset to position item in the visible area
	viewportHeight := model.state.account.detailViewport.Height
	currentOffset := model.state.account.detailViewport.YOffset

	// If item is above viewport, scroll up to show it
	if targetY < currentOffset {
		model.state.account.detailViewport.SetYOffset(targetY - 2)
	}

	// If item is below viewport, scroll down to show it
	if targetY+itemHeight > currentOffset+viewportHeight {
		model.state.account.detailViewport.SetYOffset(targetY - viewportHeight + itemHeight)
	}

	return model
}

func (m model) UpdateSelectedAccountPage(previous bool) (model, tea.Cmd) {
	var next int
	if previous {
		next = m.state.account.selected - 1
	} else {
		next = m.state.account.selected + 1
	}

	if next < 0 {
		next = 0
	}
	max := len(m.accountPages) - 1
	if next > max {
		next = max
	}

	// Reset detailed view state if we're switching away from orders
	if m.accountPages[m.state.account.selected] == ordersPage {
		m.state.orders.viewing = false
	}

	m.state.account.selected = next
	m.switched = true

	// If viewports are ready, keep selected item in view
	if m.state.account.viewportsReady {
		// Calculate approximate position of selected item
		itemHeight := 1
		targetY := (m.state.account.selected + 1) * itemHeight

		// Keep selected item in view
		m.state.account.menuViewport.SetYOffset(targetY - (m.state.account.menuViewport.Height / 2))

		// Reset detail viewport to top when selection changes
		m.state.account.detailViewport.GotoTop()
	}

	return m, nil
}
