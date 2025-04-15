package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/terminaldotshop/terminal-sdk-go"
	"github.com/terminaldotshop/terminal-sdk-go/option"
	"github.com/terminaldotshop/terminal/go/pkg/api"
	"github.com/terminaldotshop/terminal/go/pkg/resource"
)

// CreateSDKClient creates a Terminal SDK client with the given context, token, and region
func (m model) CreateSDKClient() *terminal.Client {
	options := []option.RequestOption{
		option.WithBaseURL(resource.Resource.Api.Url),
		option.WithBearerToken(m.accessToken),
		option.WithAppID("ssh"),
	}

	// Only add client IP header if in context
	// Region lookup will be performed server-side
	clientIP, _ := m.context.Value("client_ip").(*string)
	if clientIP != nil {
		options = append(options, option.WithHeader("x-terminal-ip", *clientIP))
	}
	if m.region != nil {
		options = append(options, option.WithHeader("x-terminal-region", string(*m.region)))
	}

	return terminal.NewClient(options...)
}

type SplashState struct {
	data  bool
	delay bool
}

type UserSignedInMsg struct {
	accessToken string
	client      *terminal.Client
}

type DelayCompleteMsg struct{}

func (m model) LoadCmds() []tea.Cmd {
	cmds := []tea.Cmd{}

	// Make sure the loading state shows for at least a couple seconds
	cmds = append(cmds, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return DelayCompleteMsg{}
	}))

	cmds = append(cmds, func() tea.Msg {
		response, err := m.client.View.Init(m.context)
		if err != nil {
			return err
		}
		return response.Data
	})

	return cmds
}

func (m model) IsLoadingComplete() bool {
	return m.state.splash.data &&
		m.state.splash.delay
}

func (m model) SplashInit() tea.Cmd {
	cmd := func() tea.Msg {
		token, err := api.FetchUserToken(m.fingerprint)
		if err != nil {
			return err
		}

		m.accessToken = token.AccessToken

		// Create the client with the region
		client := m.CreateSDKClient()

		return UserSignedInMsg{
			accessToken: token.AccessToken,
			client:      client,
		}
	}

	disableMouseCmd := func() tea.Msg {
		return tea.DisableMouse()
	}

	return tea.Batch(m.CursorInit(), disableMouseCmd, cmd)
}

func (m model) SplashUpdate(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case UserSignedInMsg:
		m.client = msg.client
		m.accessToken = msg.accessToken
		return m, tea.Batch(m.LoadCmds()...)
	case DelayCompleteMsg:
		m.state.splash.delay = true
	case terminal.ViewInitResponseData:
		m.state.splash.data = true
	}

	if m.IsLoadingComplete() {
		return m.InitialDataLoaded()
	}
	return m, nil
}

func (m model) SplashView() string {
	var msg string
	if m.error != nil {
		msg = m.error.message
	} else {
		msg = ""
	}

	var hint string
	if m.error != nil {
		hint = lipgloss.JoinHorizontal(
			lipgloss.Center,
			m.theme.TextAccent().Bold(true).Render("esc"),
			" ",
			"quit",
		)
	} else {
		hint = ""
	}

	if m.error == nil {
		return lipgloss.Place(
			m.viewportWidth,
			m.viewportHeight,
			lipgloss.Center,
			lipgloss.Center,
			m.LogoView(),
		)
	}

	return lipgloss.Place(
		m.viewportWidth,
		m.viewportHeight,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Center,
			"",
			"",
			"",
			"",
			m.LogoView(),
			"",
			"",
			lipgloss.JoinHorizontal(
				lipgloss.Center,
				m.theme.TextError().Render(msg),
			),
			hint,
		),
	)
}
