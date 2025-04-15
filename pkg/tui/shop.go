package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/terminaldotshop/terminal-sdk-go"
	"github.com/terminaldotshop/terminal/go/pkg/tui/theme"
)

type shopState struct {
	selected       int
	menuViewport   viewport.Model
	detailViewport viewport.Model
	viewportsReady bool
}

func (m model) updateShopViewports() model {
	headerHeight := lipgloss.Height(m.HeaderView())
	breadcrumbsHeight := lipgloss.Height(m.BreadcrumbsView())
	footerHeight := lipgloss.Height(m.FooterView())
	verticalMarginHeight := headerHeight + footerHeight + breadcrumbsHeight

	availableHeight := m.heightContainer - verticalMarginHeight

	// Calculate menu width based on products
	menuWidth := 0
	for _, p := range m.products {
		w := lipgloss.Width(p.Name)
		if w > menuWidth {
			menuWidth = w
		}
	}

	// Add padding for section headers
	if menuWidth > 0 {
		menuWidth += 4 // padding for menu items
	}

	// For small screens, make the menu full width
	if m.size < large {
		menuWidth = m.widthContent
	}

	detailWidth := m.widthContent - menuWidth
	if m.size < large {
		detailWidth = m.widthContent
	}

	if !m.state.shop.viewportsReady {
		// Initialize viewports for the first time
		m.state.shop.menuViewport = viewport.New(menuWidth, availableHeight)
		m.state.shop.menuViewport.KeyMap = viewport.KeyMap{}

		m.state.shop.detailViewport = viewport.New(detailWidth, availableHeight)
		m.state.shop.detailViewport.KeyMap = modifiedKeyMap

		m.state.shop.viewportsReady = true
	} else {
		// Update existing viewports
		m.state.shop.menuViewport.Width = menuWidth
		m.state.shop.menuViewport.Height = availableHeight

		m.state.shop.detailViewport.Width = detailWidth
		m.state.shop.detailViewport.Height = availableHeight
	}

	return m
}

func (m model) ShopSwitch() (model, tea.Cmd) {
	m = m.SwitchPage(shopPage)
	m.state.subscribe.product = nil

	m.state.footer.commands = []footerCommand{
		{key: "+/-", value: "qty"},
		{key: "c", value: "cart"},
		{key: "q", value: "quit"},
	}

	if len(m.products) > 1 {
		m.state.footer.commands = append(
			[]footerCommand{{key: "↑/↓", value: "products"}},
			m.state.footer.commands...,
		)
	}

	m = m.UpdateSelectedTheme()
	m = m.updateShopViewports()
	return m, nil
}

func (m model) ShopUpdate(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Update viewport dimensions if window size changed
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m = m.updateShopViewports()
	}

	// Handle different message types
	switch msg := msg.(type) {
	case tea.KeyMsg:
		product := m.products[m.state.shop.selected]

		switch msg.String() {
		case "r":
			return m.ToggleRegion()
		case "tab", "down", "j":
			m, cmd = m.UpdateSelected(false)
			cmds = append(cmds, cmd)
		case "shift+tab", "up", "k":
			m, cmd = m.UpdateSelected(true)
			cmds = append(cmds, cmd)
		case "+", "=", "right", "l":
			if product.Subscription == terminal.ProductSubscriptionRequired {
				break
			}
			productVariantID := m.products[m.state.shop.selected].Variants[0].ID
			return m.UpdateCart(productVariantID, 1)
		case "-", "left", "h":
			if product.Subscription == terminal.ProductSubscriptionRequired {
				break
			}
			productVariantID := m.products[m.state.shop.selected].Variants[0].ID
			return m.UpdateCart(productVariantID, -1)
		case "enter":
			if product.Subscription == terminal.ProductSubscriptionRequired {
				subscribed := false
				subscriptionId := ""
				for _, s := range m.subscriptions {
					for _, v := range product.Variants {
						if v.ID == s.ProductVariantID {
							subscriptionId = s.ID
							subscribed = true
						}
					}
				}
				if subscribed {
					return m.SubscriptionManageSwitch(subscriptionId)
				} else {
					if m.anonymous {
						m.error = &VisibleError{
							message: "ssh public key required to subscribe, see trm.sh/faq",
						}
						return m, nil
					}
					m.state.subscribe.product = &product
					return m.SubscribeSwitch()
				}
			}
			return m.CartSwitch()
		}
	}

	// Update viewports with new content
	if m.state.shop.viewportsReady {
		// Update menu content
		menuContent := m.getShopMenuContent()
		m.state.shop.menuViewport.SetContent(menuContent)

		// Update detail content
		detailContent := m.getShopDetailContent()
		m.state.shop.detailViewport.SetContent(detailContent)
	}

	m.state.shop.detailViewport, cmd = m.state.shop.detailViewport.Update(msg)
	cmds = append(cmds, cmd)

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m model) UpdateSelected(previous bool) (model, tea.Cmd) {
	var next int
	if previous {
		next = m.state.shop.selected - 1
	} else {
		next = m.state.shop.selected + 1
	}

	if next < 0 {
		next = 0
	}
	max := len(m.products) - 1
	if next > max {
		next = max
	}

	m.state.shop.selected = next
	m = m.UpdateSelectedTheme()

	// If viewports are ready, update them with new content after selection change
	if m.state.shop.viewportsReady {
		// Calculate approximate position of selected item
		itemHeight := 1
		featuredCount := 0
		for _, p := range m.products {
			if p.Tags.Featured {
				featuredCount++
			}
		}

		var targetY int
		if featuredCount > 0 && m.state.shop.selected >= featuredCount {
			// Add header rows for sections
			targetY = (m.state.shop.selected + 4) * itemHeight
		} else {
			targetY = (m.state.shop.selected + 2) * itemHeight
		}

		// Keep selected item in view
		m.state.shop.menuViewport.SetYOffset(targetY - (m.state.shop.menuViewport.Height / 2))

		// Reset detail viewport to top when selection changes
		m.state.shop.detailViewport.GotoTop()
	}

	return m, nil
}

func (m model) reorderProducts() model {
	var featured, originals []terminal.Product

	// Split into featured and originals while maintaining relative order within each category
	for _, p := range m.products {
		if p.Tags.Featured {
			featured = append(featured, p)
		} else {
			originals = append(originals, p)
		}
	}

	// Combine featured first, then originals
	m.products = append(featured, originals...)

	// Reset selection to avoid any out-of-bounds issues
	if len(m.products) > 0 {
		m.state.shop.selected = 0
	}

	return m
}

// Helper function to generate content for the menu viewport
func (m model) getShopMenuContent() string {
	menuWidth := 0
	var featuredCount int

	// Calculate max width and count featured products
	for _, p := range m.products {
		w := lipgloss.Width(p.Name)
		if w > menuWidth {
			menuWidth = w
		}
		if p.Tags.Featured {
			featuredCount++
		}
	}

	// Only consider section header widths if we have featured products
	if featuredCount > 0 {
		featuredHeader := "~ featured ~"
		originalsHeader := "~ originals ~"
		headerWidth := lipgloss.Width(featuredHeader)
		if w := lipgloss.Width(originalsHeader); w > headerWidth {
			headerWidth = w
		}
		if headerWidth > menuWidth {
			menuWidth = headerWidth
		}
	}

	var menuItem lipgloss.Style
	var highlightedMenuItem lipgloss.Style
	var sectionHeader lipgloss.Style

	if m.size < large {
		menuWidth = m.widthContent
		menuItem = m.theme.Base().
			Width(m.widthContent - 1).
			Align(lipgloss.Center)
		highlightedMenuItem = m.theme.Base().
			Width(m.widthContent - 1).
			Align(lipgloss.Center).
			Background(m.theme.Highlight()).
			Foreground(m.theme.Accent())
		sectionHeader = m.theme.Base().
			Width(menuWidth).
			Align(lipgloss.Center).
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
		sectionHeader = m.theme.Base().
			Width(menuWidth+2).
			Padding(0, 1).
			Foreground(m.theme.Accent())
	}

	product := m.products[m.state.shop.selected]
	if product.Name == "cron" {
		highlightedMenuItem = highlightedMenuItem.Foreground(lipgloss.Color("#000000"))
	}

	var products strings.Builder

	// If we have featured products, show sections
	if featuredCount > 0 {
		products.WriteString(sectionHeader.Render("~ featured ~"))
		products.WriteString("\n")

		for i := range featuredCount {
			var content string
			if i == m.state.shop.selected {
				content = highlightedMenuItem.Render(m.products[i].Name)
			} else {
				content = menuItem.Render(m.products[i].Name)
			}
			products.WriteString(content + "\n")
		}

		if featuredCount < len(m.products) {
			products.WriteString("\n")
			products.WriteString(sectionHeader.Render("~ originals ~"))
			products.WriteString("\n")

			for i := featuredCount; i < len(m.products); i++ {
				var content string
				if i == m.state.shop.selected {
					content = highlightedMenuItem.Render(m.products[i].Name)
				} else {
					content = menuItem.Render(m.products[i].Name)
				}
				products.WriteString(content + "\n")
			}
			products.WriteString("\n")
		}
	} else {
		// No sections, just list all products
		for i, p := range m.products {
			var content string
			if i == m.state.shop.selected {
				content = highlightedMenuItem.Render(p.Name)
			} else {
				content = menuItem.Render(p.Name)
			}
			products.WriteString(content + "\n")
		}
	}

	return m.theme.Base().Padding(0, 1).Render(products.String())
}

// Helper function to generate content for the detail viewport
func (m model) getShopDetailContent() string {
	base := m.theme.Base().Render
	accent := m.theme.TextAccent().Render
	boldStyle := m.theme.TextHighlight().Bold(true)

	if product := m.products[m.state.shop.selected]; product.Tags.Color == "#000000" {
		boldStyle = boldStyle.Foreground(lipgloss.Color("#FFFFFF"))
	}

	bold := boldStyle.Render
	button := m.theme.Base().
		PaddingLeft(1).
		PaddingRight(1).
		Align(lipgloss.Center).
		Background(m.theme.Highlight()).
		Foreground(m.theme.Background()).
		Render

	product := m.products[m.state.shop.selected]
	variantID := product.Variants[0].ID
	cartItem, _ := m.GetCartItem(variantID)
	minus := base("- ")
	plus := base(" +")
	count := accent(fmt.Sprintf(" %d ", cartItem.Quantity))
	quantity := minus + count + plus

	if product.Subscription == terminal.ProductSubscriptionRequired {
		subscribed := false
		for _, s := range m.subscriptions {
			for _, v := range product.Variants {
				if v.ID == s.ProductVariantID {
					subscribed = true
				}
			}
		}

		if subscribed {
			quantity = button("manage sub") + " enter"
		} else {
			quantity = button("subscribe") + " enter"
		}
	}

	menuWidth := 0
	if m.size >= large {
		// Calculate menu width for large screens
		for _, p := range m.products {
			w := lipgloss.Width(p.Name)
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

	detailStyle := m.theme.Base().Width(detailWidth)

	name := accent(product.Name)
	variantNames := ""
	for _, variant := range product.Variants {
		if variant.Name == product.Variants[len(product.Variants)-1].Name {
			variantNames += variant.Name
		} else {
			variantNames += variant.Name + "/"
		}
	}

	detail := lipgloss.JoinVertical(
		lipgloss.Left,
		name,
		base(strings.ToLower(variantNames)),
		"",
		bold(fmt.Sprintf("$%.2v", product.Variants[0].Price/100)),
		"",
		product.Description,
		"",
		quantity,
	)

	return detailStyle.Render(detail)
}

func (m model) ShopView() string {
	if !m.state.shop.viewportsReady {
		m = m.updateShopViewports()
	}

	// Update viewport contents
	menuContent := m.getShopMenuContent()
	m.state.shop.menuViewport.SetContent(menuContent)

	detailContent := m.getShopDetailContent()
	m.state.shop.detailViewport.SetContent(detailContent)

	// Combine viewport views
	if m.size < large {
		// For small screens, stack the viewports vertically
		return lipgloss.JoinVertical(
			lipgloss.Top,
			m.state.shop.menuViewport.View(),
			m.state.shop.detailViewport.View(),
		)
	} else {
		// For large screens, place viewports side by side
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.state.shop.menuViewport.View(),
			"  ",
			m.state.shop.detailViewport.View(),
		)
	}
}

func (m model) UpdateSelectedTheme() model {
	var highlight string
	product := m.products[m.state.shop.selected]
	highlight = product.Tags.Color

	if highlight != "" {
		m.theme = theme.BasicTheme(m.renderer, &highlight)
	} else {
		m.theme = theme.BasicTheme(m.renderer, nil)
	}

	return m
}
