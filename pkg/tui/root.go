package tui

import (
	"context"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/terminaldotshop/terminal-sdk-go"
	"github.com/terminaldotshop/terminal/go/pkg/api"
	"github.com/terminaldotshop/terminal/go/pkg/tui/theme"
)

type page = int
type size = int

const (
	menuPage page = iota
	splashPage
	shopPage
	accountPage
	paymentPage
	cartPage
	subscribePage
	shippingPage
	confirmPage
	finalSubPage
	finalPage
	subscriptionsPage
	tokensPage
	appsPage
	ordersPage
	aboutPage
	faqPage
)

const (
	undersized size = iota
	small
	medium
	large
)

type model struct {
	ready         bool
	command       []string
	switched      bool
	page          page
	hasMenu       bool
	checkout      bool
	state         state
	region        *terminal.Region
	context       context.Context
	client        *terminal.Client
	user          terminal.Profile
	accountPages  []page
	products      []terminal.Product
	addresses     []terminal.Address
	cards         []terminal.Card
	subscriptions []terminal.Subscription
	tokens        []terminal.Token
	apps          []terminal.App
	orders        []terminal.Order
	order         *terminal.Order
	cart          terminal.Cart
	subscription  terminal.SubscriptionParam
	renderer      *lipgloss.Renderer
	// output          *termenv.Output
	theme           theme.Theme
	fingerprint     string
	anonymous       bool
	viewportWidth   int
	viewportHeight  int
	widthContainer  int
	heightContainer int
	widthContent    int
	heightContent   int
	size            size
	accessToken     string
	faqs            []FAQ
	error           *VisibleError
}

type VisibleError struct {
	message string
}

type state struct {
	splash        SplashState
	cursor        cursorState
	shipping      shippingState
	subscriptions subscriptionsState
	tokens        tokensState
	apps          appsState
	orders        ordersState
	shop          shopState
	account       accountState
	footer        footerState
	cart          cartState
	subscribe     subscribeState
	payment       paymentState
	confirm       confirmState
	menu          menuState
	finalSub      finalSubState
}

type children struct {
}

func NewModel(
	renderer *lipgloss.Renderer,
	fingerprint string,
	anonymous bool,
	clientIP *string,
	command []string,
) (tea.Model, error) {
	api.Init()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "client_ip", clientIP)

	result := model{
		command:  command,
		context:  ctx,
		region:   nil,
		page:     splashPage,
		renderer: renderer,
		// output:      renderer.Output(),
		fingerprint: fingerprint,
		anonymous:   anonymous,
		theme:       theme.BasicTheme(renderer, nil),
		faqs:        LoadFaqs(),
		accountPages: []page{
			ordersPage,
			subscriptionsPage,
			tokensPage,
			appsPage,
			// shippingPage,
			// paymentPage,
			faqPage,
			aboutPage,
		},
		subscription: terminal.SubscriptionParam{},
		state: state{
			splash: SplashState{},
			shop: shopState{
				selected: 0,
			},
			cart: cartState{
				selected: 0,
			},
			subscribe: subscribeState{
				selected: 0,
			},
			account: accountState{
				selected: 0,
			},
			subscriptions: subscriptionsState{
				selected: 0,
			},
			tokens: tokensState{
				selected: 0,
			},
			orders: ordersState{
				selected: 0,
			},
			payment: paymentState{
				input: paymentInput{},
			},
			shipping: shippingState{
				input: shippingInput{
					country: "US",
				},
			},
			footer: footerState{
				commands: []footerCommand{},
			},
		},
	}
	return result, nil
}

func (m model) Init() tea.Cmd {
	return m.SplashInit()
}

func (m model) SwitchPage(page page) model {
	m.page = page
	m.switched = true
	return m
}

func (m model) InitialDataLoaded() (model, tea.Cmd) {
	if len(m.command) == 0 {
		return m.ShopSwitch()
	}

	// TODO: support multiple commands?
	command := strings.ToLower(m.command[0])

	for index, product := range m.products {
		if strings.ToLower(product.Name) == command {
			m.state.shop.selected = index
			return m.ShopSwitch()
		}
	}

	if command == "cart" {
		return m.CartSwitch()
	}

	accountPageNames := []string{
		"orders",
		"subscriptions",
		"tokens",
		"apps",
		"faq",
		"about",
	}
	for _, name := range accountPageNames {
		if strings.HasPrefix(name, command) {
			m, cmd := m.AccountSwitch()

			selected := 0
			for index, page := range m.accountPages {
				if name == "orders" && page == ordersPage {
					selected = index
					break
				} else if name == "subscriptions" && page == subscriptionsPage {
					selected = index
					break
				} else if name == "tokens" && page == tokensPage {
					selected = index
					break
				} else if name == "apps" && page == appsPage {
					selected = index
					break
				} else if name == "faq" && page == faqPage {
					selected = index
					break
				} else if name == "about" && page == aboutPage {
					selected = index
					break
				}
			}

			m.state.account.selected = selected
			if m.state.account.selected > 0 {
				m.state.account.focused = true
			}
			return m, cmd
		}
	}

	return m.ShopSwitch()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case VisibleError:
		m.error = &msg
	case error:
		m.error = &VisibleError{
			message: api.GetErrorMessage(msg),
		}
		if m.page == shopPage || m.page == cartPage {
			cmds = append(cmds, func() tea.Msg {
				response, err := m.client.Cart.Get(m.context)
				if err != nil {
					return VisibleError{message: "something went wrong, restart the ssh session"}
				}
				return response.Data
			})
		}
	case tea.WindowSizeMsg:
		m.viewportWidth = msg.Width
		m.viewportHeight = msg.Height

		switch {
		case m.viewportWidth < 20 || m.viewportHeight < 10:
			m.size = undersized
			m.widthContainer = m.viewportWidth
			m.heightContainer = m.viewportHeight
		case m.viewportWidth < 50:
			m.size = small
			m.widthContainer = m.viewportWidth
			m.heightContainer = m.viewportHeight
		case m.viewportWidth < 75:
			m.size = medium
			m.widthContainer = 50
			m.heightContainer = int(math.Min(float64(msg.Height), 30))
		default:
			m.size = large
			m.widthContainer = 75
			m.heightContainer = int(math.Min(float64(msg.Height), 30))
		}

		m.widthContent = m.widthContainer - 4
		m.heightContent = m.heightContainer - lipgloss.Height(m.HeaderView()) - lipgloss.Height(m.FooterView()) - lipgloss.Height(m.BreadcrumbsView())
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.error != nil {
				if m.page == splashPage {
					return m, tea.Quit
				}
				m.error = nil
				return m, nil
			}
		case "ctrl+c":
			return m, tea.Quit
		}
	case CursorTickMsg:
		m, cmd := m.CursorUpdate(msg)
		return m, cmd
	case CartUpdatedMsg:
		if m.state.cart.lastUpdateID == msg.updateID {
			m.cart = msg.updated
		}
	case terminal.ViewInitResponseData:
		m.user = msg.Profile
		m.products = msg.Products
		m.cart = msg.Cart
		m.cards = msg.Cards
		m.addresses = msg.Addresses
		m.subscriptions = msg.Subscriptions
		m.tokens = msg.Tokens
		m.apps = msg.Apps
		m.orders = msg.Orders
		m.region = &msg.Region
		m = m.reorderProducts()
	case terminal.Profile:
		m.user = msg
	case []terminal.Product:
		m.products = msg
		m = m.reorderProducts()
	case terminal.Cart:
		m.cart = msg
	case []terminal.Card:
		m.cards = msg
	case []terminal.Address:
		m.addresses = msg
	case []terminal.Subscription:
		m.subscriptions = msg
	case []terminal.Token:
		m.tokens = msg
	case []terminal.App:
		m.apps = msg
	case []terminal.Order:
		m.orders = msg
	}

	var cmd tea.Cmd
	switch m.page {
	case menuPage:
		m, cmd = m.MenuUpdate(msg)
	case splashPage:
		m, cmd = m.SplashUpdate(msg)
	case accountPage:
		m, cmd = m.AccountUpdate(msg)
	case aboutPage:
		m, cmd = m.AboutUpdate(msg)
	case shopPage:
		m, cmd = m.ShopUpdate(msg)
	case cartPage:
		m, cmd = m.CartUpdate(msg)
	case subscribePage:
		m, cmd = m.SubscribeUpdate(msg)
	case paymentPage:
		m, cmd = m.PaymentUpdate(msg)
	case shippingPage:
		m, cmd = m.ShippingUpdate(msg)
	case confirmPage:
		m, cmd = m.ConfirmUpdate(msg)
	case finalSubPage:
		m, cmd = m.FinalSubUpdate(msg)
	case finalPage:
		m, cmd = m.FinalUpdate(msg)
	}

	var headerCmd tea.Cmd
	m, headerCmd = m.HeaderUpdate(msg)
	cmds = append(cmds, headerCmd)

	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.hasMenu = m.page == shopPage ||
		(m.page == accountPage && m.state.apps.editing == false)
		// m.page == aboutPage ||
		// m.page == faqPage

	m.checkout = m.page == cartPage ||
		m.page == subscribePage ||
		m.page == paymentPage ||
		m.page == shippingPage ||
		m.page == confirmPage

	if m.switched {
		m.switched = false
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.size == undersized {
		return m.ResizeView()
	}

	switch m.page {
	case splashPage:
		return m.SplashView()
	case menuPage:
		return m.MenuView()
	default:
		header := m.HeaderView()
		footer := m.FooterView()
		breadcrumbs := m.BreadcrumbsView()

		// Get content based on current page
		content := m.getContent()

		height := m.heightContainer
		height -= lipgloss.Height(header)
		// if breadcrumbs != "" {
		height -= lipgloss.Height(breadcrumbs)
		// }
		height -= lipgloss.Height(footer)

		body := m.theme.Base().Width(m.widthContainer).Height(height).Render(content)
		// bodyHeight := lipgloss.Height(body)
		// if bodyHeight < height {
		// 	body += lipgloss.NewStyle().Height(height - bodyHeight).Render(" ")
		// }

		items := []string{}
		items = append(items, header)
		// if breadcrumbs != "" {
		items = append(items, breadcrumbs)
		// }
		items = append(items, body)
		items = append(items, footer)

		child := lipgloss.JoinVertical(
			lipgloss.Left,
			items...,
		)

		return m.renderer.Place(
			m.viewportWidth,
			m.viewportHeight,
			lipgloss.Center,
			lipgloss.Center,
			m.theme.Base().
				MaxWidth(m.widthContainer).
				MaxHeight(m.heightContainer).
				Render(child),
		)
	}
}

func (m model) getContent() string {
	page := "unknown"
	switch m.page {
	case shopPage:
		page = m.ShopView()
	case cartPage:
		page = m.CartView()
	case subscribePage:
		page = m.SubscribeView()
	case paymentPage:
		page = m.PaymentView()
	case shippingPage:
		page = m.ShippingView(m.widthContent-2, false)
	case confirmPage:
		page = m.ConfirmView()
	case finalSubPage:
		page = m.FinalSubView()
	case finalPage:
		page = m.FinalView()
	case accountPage:
		page = m.AccountView()
	}
	return page
}

var modifiedKeyMap = viewport.KeyMap{
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("pgdn", "page down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "page up"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "½ page up"),
	),
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "½ page down"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "down"),
	),
}
