package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	terminal "github.com/terminaldotshop/terminal-sdk-go"
	"github.com/terminaldotshop/terminal/go/pkg/tui/validate"
)

type shippingView = int

const (
	shippingListView shippingView = iota
	shippingFormView
)

type shippingInput struct {
	name     string
	street1  string
	street2  string
	city     string
	province string
	country  string
	zip      string
	phone    string
}

type shippingState struct {
	view       shippingView
	selected   int
	deleting   *int
	input      shippingInput
	form       *huh.Form
	submitting bool
}

type SelectedShippingUpdatedMsg struct {
	shippingID string
}

type ShippingAddressAddedMsg struct {
	shippingID string
	addresses  []terminal.Address
}

func (m model) ShippingSwitch() (model, tea.Cmd) {
	m = m.SwitchPage(shippingPage)
	m.state.footer.commands = []footerCommand{
		{key: "esc", value: "back"},
		{key: "↑/↓", value: "addresses"},
		{key: "x/del", value: "remove"},
		{key: "enter", value: "select"},
	}
	m.state.shipping.submitting = false
	m.state.shipping.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("name").
				Key("name").
				Value(&m.user.User.Name).
				Validate(validate.NotEmpty("name")),
			huh.NewInput().
				Title("street 1").
				Key("street1").
				Value(&m.state.shipping.input.street1).
				Validate(validate.NotEmpty("street 1")),
			huh.NewInput().
				Title("street 2").
				Key("street2").
				Value(&m.state.shipping.input.street2),
			huh.NewInput().
				Title("city").
				Key("city").
				Value(&m.state.shipping.input.city).
				Validate(validate.NotEmpty("city")),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("state").
				Key("province").
				Value(&m.state.shipping.input.province),
			huh.NewInput().
				Title("country").
				Key("country").
				Value(&m.state.shipping.input.country).
				Validate(validate.NotEmpty("country")),
			huh.NewInput().
				Title("phone").
				Key("phone").
				Value(&m.state.shipping.input.phone),
			huh.NewInput().
				Title("postal code").
				Key("zip").
				Value(&m.state.shipping.input.zip).
				Validate(validate.NotEmpty("postal code")),
		),
	).
		WithTheme(m.theme.Form()).
		WithShowHelp(false)

	m.state.shipping.view = shippingListView
	if len(m.addresses) == 0 {
		m.state.shipping.view = shippingFormView
	}

	m = m.updateShippingForm()
	return m, m.state.shipping.form.Init()
}

func (m model) updateShippingForm() model {
	if m.size == small {
		m.state.shipping.form = m.state.shipping.form.
			WithLayout(huh.LayoutStack).
			WithWidth(m.widthContent)
	} else {
		m.state.shipping.form = m.state.shipping.form.
			WithLayout(huh.LayoutColumns(2)).
			WithWidth(m.widthContent)
	}

	return m
}

func (m model) nextAddress() (model, tea.Cmd) {
	next := m.state.shipping.selected + 1
	max := len(m.addresses)
	if next > max {
		next = max
	}

	m.state.shipping.selected = next
	return m, nil
}

func (m model) previousAddress() (model, tea.Cmd) {
	next := m.state.shipping.selected - 1
	if next < 0 {
		next = 0
	}

	m.state.shipping.selected = next
	return m, nil
}

func (m model) SetShipping(shippingID string) error {
	if m.IsSubscribing() {
		return nil
	}

	params := terminal.CartSetAddressParams{AddressID: terminal.F(shippingID)}
	_, err := m.client.Cart.SetAddress(m.context, params)
	if err != nil {
		return err
	}
	return nil
}

func (m model) GetSelectedAddress() *terminal.Address {
	if m.IsSubscribing() {
		for _, address := range m.addresses {
			if address.ID == m.subscription.AddressID.Value {
				return &address
			}
		}
		return nil
	}

	for _, address := range m.addresses {
		if address.ID == m.cart.AddressID {
			return &address
		}
	}
	return nil
}

func (m model) chooseAddress() (model, tea.Cmd) {
	if m.state.shipping.selected < len(m.addresses) { // existing address
		shippingID := m.addresses[m.state.shipping.selected].ID

		m.state.shipping.submitting = true
		return m, func() tea.Msg {
			err := m.SetShipping(shippingID)
			if err != nil {
				return err
			}
			return SelectedShippingUpdatedMsg{shippingID: shippingID}
		}
	} else { // new
		m.state.shipping.input = shippingInput{country: "US"}
		m.state.shipping.view = shippingFormView
	}

	return m, nil
}

func (m model) shippingListUpdate(msg tea.Msg) (model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "tab":
			if m.state.shipping.deleting == nil {
				return m.nextAddress()
			}
		case "k", "up", "shift+tab":
			if m.state.shipping.deleting == nil {
				return m.previousAddress()
			}
		case "delete", "d", "backspace", "x":
			if m.state.shipping.deleting == nil && m.state.shipping.selected < len(m.addresses) {
				m.state.shipping.deleting = &m.state.shipping.selected
			}
			return m, nil
		case "y":
			if m.state.shipping.deleting != nil {
				m.state.shipping.deleting = nil
				_, err := m.client.Address.Delete(m.context, m.addresses[m.state.shipping.selected].ID)
				if err != nil {
					return m, func() tea.Msg { return err }
				}
				if len(m.addresses)-1 == 0 && m.page == accountPage {
					m.state.account.focused = false
				}
				return m, func() tea.Msg {
					shipping, err := m.client.Address.List(m.context)
					if err != nil {
						return err
					}
					return shipping.Data
				}
			}
			return m, nil
		case "n":
			m.state.shipping.deleting = nil
			return m, nil
		case "enter":
			if m.state.shipping.deleting == nil {
				return m.chooseAddress()
			}
		case "esc":
			if m.state.shipping.deleting != nil {
				m.state.shipping.deleting = nil
			} else if m.IsSubscribing() {
				if m.SubscribeItemCount() == 1 {
					return m.ShopSwitch()
				}
				return m.SubscribeSwitch()
			} else {
				return m.CartSwitch()
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) shippingFormUpdate(msg tea.Msg) (model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state.shipping.view = shippingListView
			return m, nil
		}

	case ShippingAddressAddedMsg:
		m.addresses = msg.addresses

		return m, func() tea.Msg {
			err := m.SetShipping(msg.shippingID)
			if err != nil {
				return err
			}

			return SelectedShippingUpdatedMsg{shippingID: msg.shippingID}
		}
	}

	m = m.updateShippingForm()
	next, cmd := m.state.shipping.form.Update(msg)
	m.state.shipping.form = next.(*huh.Form)

	cmds = append(cmds, cmd)
	if !m.state.shipping.submitting && m.state.shipping.form.State == huh.StateCompleted {
		m.state.shipping.submitting = true

		form := m.state.shipping.form
		m.state.shipping.input = shippingInput{
			name:     form.GetString("name"),
			street1:  form.GetString("street1"),
			street2:  form.GetString("street2"),
			city:     form.GetString("city"),
			province: form.GetString("province"),
			country:  form.GetString("country"),
			zip:      form.GetString("zip"),
			phone:    form.GetString("phone"),
		}

		return m, func() tea.Msg {
			if m.state.shipping.input.country != "US" && m.state.shipping.input.phone == "" {
				return VisibleError{message: "phone is required for international orders"}
			}

			params := terminal.AddressNewParams{
				Name:     terminal.String(m.state.shipping.input.name),
				Street1:  terminal.String(m.state.shipping.input.street1),
				Street2:  terminal.String(m.state.shipping.input.street2),
				City:     terminal.String(m.state.shipping.input.city),
				Province: terminal.String(m.state.shipping.input.province),
				Country:  terminal.String(m.state.shipping.input.country),
				Zip:      terminal.String(m.state.shipping.input.zip),
				Phone:    terminal.String(m.state.shipping.input.phone),
			}
			response, err := m.client.Address.New(m.context, params)
			if err != nil {
				return err
			}
			addresses, err := m.client.Address.List(m.context)
			if err != nil {
				return err
			}
			return ShippingAddressAddedMsg{
				shippingID: response.Data,
				addresses:  addresses.Data,
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) ShippingUpdate(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		current := m.state.shipping.view
		m, cmd := m.ShippingSwitch()
		m.state.shipping.view = current
		return m, cmd
	case SelectedShippingUpdatedMsg:
		if m.IsSubscribing() {
			m.subscription.AddressID = terminal.String(msg.shippingID)
		} else {
			m.cart.AddressID = msg.shippingID
			cart, err := m.client.Cart.Get(m.context)
			if err != nil {
				return m, func() tea.Msg { return err }
			}
			m.cart = cart.Data
		}
		return m.PaymentSwitch()
	}

	if m.state.shipping.view == shippingListView {
		return m.shippingListUpdate(msg)
	} else {
		return m.shippingFormUpdate(msg)
	}
}

func (m model) ShippingView(totalWidth int, focused bool) string {
	if m.state.shipping.submitting {
		return m.theme.Base().Width(totalWidth).Render(" calculating shipping costs...")
	}

	if m.state.shipping.view == shippingListView {
		return m.shippingListView(totalWidth, focused)
	} else {
		return m.shippingFormView()
	}
}

func (m model) formatListItem(text string, focused bool) string {
	return m.formatListItemCustom(text, focused, m.widthContent, true)
}

func (m model) formatListItemCustom(text string, focused bool, totalWidth int, showRadio bool) string {
	accent := m.theme.TextAccent().Render

	content := "     " + text
	hint := ""
	if focused {
		content = accent(" ☉   " + text)
		hint = accent("enter")
	}

	if !showRadio {
		content = text
	}

	padding := 6
	if !showRadio {
		padding = 2
	}

	var lines = strings.Split(content, "\n")
	var firstLine = lines[0]
	hintSpace := totalWidth - lipgloss.Width(hint) - lipgloss.Width(firstLine) - padding
	lines[0] = firstLine + m.theme.Base().Width(hintSpace).Render() + hint
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lines...,
	)
}

func (m model) formatAddress(address terminal.Address, focused bool) string {
	parts := []string{}
	parts = append(parts, address.Street1+", ")
	if address.Street2 != "" {
		parts = append(parts, address.Street2+", ")
	}
	parts = append(parts, address.City+", "+address.Province+", "+address.Country+", ")
	parts = append(parts, address.Zip)

	return m.formatListItem(lipgloss.JoinHorizontal(lipgloss.Left, parts...), focused)
}

func (m model) shippingListView(totalWidth int, focused bool) string {
	base := m.theme.Base().Render
	accent := m.theme.TextAccent().Render

	addresses := []string{}
	for i, address := range m.addresses {
		content := m.formatAddress(
			address,
			i == m.state.shipping.selected && (focused || m.page != accountPage),
		)
		if m.state.shipping.deleting != nil && *m.state.shipping.deleting == i {
			content = m.formatListItem(accent("are you sure?")+base(" (y/n)"), true)
		}
		box := m.CreateBoxCustom(
			content,
			i == m.state.shipping.selected && (focused || m.page != accountPage),
			totalWidth,
		)
		addresses = append(addresses, box)
	}

	newAddressIndex := len(m.addresses)
	newAddress := m.CreateBoxCustom(
		m.formatListItem("add new address", m.state.shipping.selected == newAddressIndex),
		m.state.shipping.selected == newAddressIndex,
		totalWidth,
	)
	addresses = append(addresses, newAddress)
	addressList := lipgloss.JoinVertical(lipgloss.Left, addresses...)

	return m.theme.Base().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		" select shipping address",
		addressList,
	))
}

func (m model) shippingFormView() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.state.shipping.form.View(),
	)
}
