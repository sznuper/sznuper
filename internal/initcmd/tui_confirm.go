package initcmd

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sznuper/sznuper/internal/config"
)

// confirmModel shows a YAML preview and lets the user confirm or cancel.
type confirmModel struct {
	cfg      *config.Config
	path     string
	preview  string
	accepted bool
}

func newConfirmModel(cfg *config.Config, path string) confirmModel {
	data, _ := config.Marshal(cfg)
	return confirmModel{
		cfg:     cfg,
		path:    path,
		preview: string(data),
	}
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (confirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "enter":
			m.accepted = true
			return m, func() tea.Msg { return confirmWriteMsg{} }
		case "n":
			return m, func() tea.Msg { return confirmCancelMsg{} }
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("Confirm"))
	b.WriteByte('\n')
	b.WriteString("Config will be written to:\n")
	b.WriteString(styleHighlight.Render(m.path) + "\n")

	b.WriteString(stylePreview.Render(m.preview))
	b.WriteByte('\n')
	b.WriteByte('\n')

	b.WriteString(styleHighlight.Render("[y]") + " Write  " + styleHighlight.Render("[n]") + " Cancel")

	return b.String()
}

// Messages emitted by confirmModel.
type (
	confirmWriteMsg  struct{}
	confirmCancelMsg struct{}
)
