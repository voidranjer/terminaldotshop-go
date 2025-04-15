package tui

import (
	"embed"
	"encoding/json"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed faq.json
var jsonData embed.FS

type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

func LoadFaqs() []FAQ {
	data, err := jsonData.ReadFile("faq.json")
	if err != nil {
		log.Fatalf("Failed to read embedded file: %s", err)
	}
	var faqs []FAQ
	if err := json.Unmarshal(data, &faqs); err != nil {
		log.Fatalf("Failed to unmarshal JSON: %s", err)
	}
	return faqs
}

func (m model) FaqSwitch() (model, tea.Cmd) {
	m = m.SwitchPage(faqPage)
	m.state.footer.commands = []footerCommand{
		{key: "↑↓", value: "scroll"},
		{key: "c", value: "cart"},
	}
	return m, nil
}

func (m model) FaqView(totalWidth int) string {
	var faqs []string
	for _, faq := range m.faqs {
		faqs = append(
			faqs,
			m.theme.TextAccent().Render(wordWrap(faq.Question, totalWidth)),
		)
		faqs = append(
			faqs,
			m.theme.Base().Render(wordWrap(faq.Answer, totalWidth)),
		)
		faqs = append(faqs, "")
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		faqs...,
	)
}
