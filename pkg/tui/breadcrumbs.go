package tui

func (m model) BreadcrumbsView() string {
	if m.IsSubscribing() {
		return subscribeBreadcrumbsView(m)
	} else {
		return cartBreadcrumbsView(m)
	}
}

func cartBreadcrumbsView(m model) string {
	accent := m.theme.TextAccent().Render
	base := m.theme.Base().Render
	sep := m.theme.Base().Render("/")

	var labels []string
	switch m.size {
	case small:
		fallthrough
	case medium:
		labels = []string{"cart", "ship", "pay", "confirm"}
	default:
		labels = []string{"cart", "shipping", "payment", "confirmation"}
	}

	var selected int
	switch m.page {
	case cartPage:
		selected = 0
	case shippingPage:
		selected = 1
	case paymentPage:
		selected = 2
	case confirmPage:
		selected = 3
	default:
		return ""
	}

	items := []string{}
	for i, label := range labels {
		if i == selected {
			items = append(items, accent(label))
			items = append(items, sep)
		} else {
			items = append(items, base(label))
			items = append(items, sep)
		}
	}

	// remove last separator
	items = items[:len(items)-1]

	return m.theme.Base().
		MarginTop(1).
		MarginBottom(1).
		PaddingLeft(1).
		Render(items...)
}

func subscribeBreadcrumbsView(m model) string {
	accent := m.theme.TextAccent().Render
	base := m.theme.Base().Render
	sep := m.theme.Base().Render("/")

	var labels []string
	switch m.size {
	case small:
		fallthrough
	case medium:
		labels = []string{"subscribe", "ship", "pay", "confirm"}
	default:
		labels = []string{"subscribe", "shipping", "payment", "confirmation"}
	}

	var selected int
	switch m.page {
	case subscribePage:
		selected = 0
	case shippingPage:
		selected = 1
	case paymentPage:
		selected = 2
	case confirmPage:
		selected = 3
	default:
		return ""
	}

	items := []string{}
	for i, label := range labels {
		if i == selected {
			items = append(items, accent(label))
			items = append(items, sep)
		} else {
			items = append(items, base(label))
			items = append(items, sep)
		}
	}

	// remove last separator
	items = items[:len(items)-1]

	return m.theme.Base().
		MarginTop(1).
		MarginBottom(1).
		PaddingLeft(1).
		Render(items...)
}
