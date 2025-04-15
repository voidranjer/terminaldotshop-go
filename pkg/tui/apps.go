package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/terminaldotshop/terminal-sdk-go"
	"github.com/terminaldotshop/terminal/go/pkg/tui/validate"
)

type AppAddedMsg struct {
	newApp terminal.AppNewResponseData
	apps   []terminal.App
}

type appInput struct {
	name        string
	redirectUri string
}

type appsState struct {
	selected   int
	deleting   *int
	editing    bool
	input      appInput
	form       *huh.Form
	submitting bool
	newApp     *terminal.AppNewResponseData
}

func (m model) nextApp() (model, tea.Cmd) {
	next := m.state.apps.selected + 1
	max := len(m.apps)
	if next > max {
		next = max
	}

	m.state.apps.selected = next
	return m, nil
}

func (m model) previousApp() (model, tea.Cmd) {
	next := m.state.apps.selected - 1
	if next < 0 {
		next = 0
	}

	m.state.apps.selected = next
	return m, nil
}

func (m model) createAppForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("name").
				Key("name").
				Value(&m.state.apps.input.name).
				Validate(validate.NotEmpty("name")),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("redirect uri").
				Key("redirectUri").
				Value(&m.state.apps.input.redirectUri).
				Validate(validate.NotEmpty("redirect uri")),
		),
	).
		WithTheme(m.theme.Form()).
		WithLayout(huh.LayoutColumns(2)).
		WithShowHelp(false)
}

func (m model) AppsFormUpdate(msg tea.Msg) (model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state.apps.editing = false
			return m, nil
		}

	case AppAddedMsg:
		m.state.apps.submitting = false
		m.state.apps.editing = false
		m.state.apps.newApp = &msg.newApp
		m.state.apps.input = appInput{}
		m.state.apps.form = m.createAppForm()

		m.apps = msg.apps
		return m, m.state.apps.form.Init()

	case error:
		m.state.apps.submitting = false
		m.state.apps.editing = false
		return m, nil
	}

	next, cmd := m.state.apps.form.Update(msg)
	m.state.apps.form = next.(*huh.Form)

	cmds = append(cmds, cmd)
	if !m.state.apps.submitting && m.state.apps.form.State == huh.StateCompleted {
		m.state.apps.submitting = true

		form := m.state.apps.form
		return m, func() tea.Msg {
			params := terminal.AppNewParams{
				Name:        terminal.F(form.GetString("name")),
				RedirectUri: terminal.F(form.GetString("redirectUri")),
			}
			response, err := m.client.App.New(m.context, params)
			if err != nil {
				return err
			}
			apps, err := m.client.App.List(m.context)
			if err != nil {
				return err
			}
			// if m.output != nil {
			// 	m.output.Copy(m.state.tokens.newToken.Token)
			// }
			return AppAddedMsg{
				newApp: response.Data,
				apps:   apps.Data,
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) AppsUpdate(msg tea.Msg) (model, tea.Cmd) {
	if m.state.apps.editing {
		return m.AppsFormUpdate(msg)
	}

	m.state.footer.commands = []footerCommand{
		{key: "↑/↓", value: "navigate"},
		{key: "x/del", value: "remove"},
		{key: "esc", value: "back"},
	}

	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "tab":
			if m.state.apps.deleting == nil {
				return m.nextApp()
			}
		case "k", "up", "shift+tab":
			if m.state.apps.deleting == nil {
				return m.previousApp()
			}
		case "delete", "d", "backspace", "x":
			if m.state.apps.deleting == nil {
				m.state.apps.deleting = &m.state.apps.selected
			}
			return m, nil
		case "y":
			if m.state.apps.deleting != nil {
				m.state.apps.deleting = nil
				_, err := m.client.App.Delete(m.context, m.apps[m.state.apps.selected].ID)
				if err != nil {
					return m, func() tea.Msg { return err }
				}
				if len(m.apps)-1 == 0 {
					m.state.account.focused = false
				}
				return m, func() tea.Msg {
					apps, err := m.client.App.List(m.context)
					if err != nil {
						return err
					}
					return apps.Data
				}
			}
			return m, nil
		case "n", "esc":
			m.state.apps.deleting = nil
			m.state.apps.editing = false
			return m, nil
		case "enter":
			if m.state.apps.deleting == nil && m.state.apps.selected == len(m.apps) {
				m.state.apps.editing = true
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) formatApp(app terminal.App) string {
	lines := []string{}
	lines = append(lines, m.theme.TextAccent().Render(app.Name))
	lines = append(lines, "redirect: "+app.RedirectUri)
	lines = append(lines, "client id: "+app.ID)

	if m.state.apps.newApp != nil && app.ID == m.state.apps.newApp.ID {
		lines = append(
			lines,
			"secret: "+m.theme.TextBrand().Bold(true).Render(m.state.apps.newApp.Secret),
		)
		lines = append(lines, "(will not be shown again)")
	} else {
		lines = append(lines, "secret: "+app.Secret)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) AppsView(totalWidth int, focused bool) string {
	base := m.theme.Base().Render
	accent := m.theme.TextAccent().Render

	apps := []string{}
	for i, app := range m.apps {
		content := m.formatApp(app)
		if m.state.apps.deleting != nil && *m.state.apps.deleting == i {
			content = accent("are you sure you want to remove?") + base(" (y/n)")
		}
		box := m.CreateBoxCustom(
			content,
			focused && i == m.state.apps.selected,
			totalWidth,
		)
		apps = append(apps, box)
	}

	newAppIndex := len(m.apps)
	newAppContent := "create app"
	if m.state.apps.submitting {
		newAppContent = m.theme.Base().Render("submitting app...")
	}

	newApp := m.CreateBoxCustom(
		m.formatListItemCustom(newAppContent, m.state.apps.selected == newAppIndex, totalWidth, false),
		focused && m.state.apps.selected == newAppIndex,
		totalWidth,
	)

	if m.state.apps.editing && !m.state.apps.submitting {
		newApp = m.state.apps.form.WithWidth(totalWidth).View()
	}

	apps = append(apps, newApp)
	appList := lipgloss.JoinVertical(lipgloss.Left, apps...)

	return m.theme.Base().Render(lipgloss.JoinVertical(
		lipgloss.Left,
		appList,
	))
}
