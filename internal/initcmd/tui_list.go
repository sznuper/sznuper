package initcmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// listModel shows currently added channels and offers add/finish actions.
type listModel struct {
	channels []addedChannel // channels added so far
	cursor   int            // 0..len(channels)+1 (add, finish)
}

type addedChannel struct {
	name     string
	typeName string // e.g. "telegram", or "base" for inherited
	url      string
	params   map[string]string
}

func newListModel(channels []addedChannel) listModel {
	return listModel{channels: channels, cursor: len(channels)}
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		total := len(m.channels) + 2 // +add +finish
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < total-1 {
				m.cursor++
			}
		case "a":
			return m, func() tea.Msg { return actionAddMsg{} }
		case "f":
			if len(m.channels) > 0 {
				return m, func() tea.Msg { return actionFinishMsg{} }
			}
		case "enter":
			addIdx := len(m.channels)
			finishIdx := addIdx + 1
			switch m.cursor {
			case addIdx:
				return m, func() tea.Msg { return actionAddMsg{} }
			case finishIdx:
				if len(m.channels) > 0 {
					return m, func() tea.Msg { return actionFinishMsg{} }
				}
			}
		case "delete", "backspace", "x":
			if m.cursor < len(m.channels) {
				return m, func() tea.Msg { return actionDeleteMsg{index: m.cursor} }
			}
		}
	}
	return m, nil
}

func (m listModel) View() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("Channels"))
	b.WriteByte('\n')

	if len(m.channels) == 0 {
		b.WriteString(styleSubtle.Render("  No channels added yet."))
		b.WriteByte('\n')
	}

	for i, svc := range m.channels {
		cursor := "  "
		if m.cursor == i {
			cursor = styleCursor.Render("> ")
		}
		label := styleSuccess.Render("✓") + " " + svc.name
		if svc.typeName != "" {
			label += styleSubtle.Render(fmt.Sprintf(" (%s)", svc.typeName))
		}
		b.WriteString(cursor + label + "\n")
	}

	b.WriteByte('\n')

	addIdx := len(m.channels)
	finishIdx := addIdx + 1

	// Add action
	cursor := "  "
	if m.cursor == addIdx {
		cursor = styleCursor.Render("> ")
	}
	b.WriteString(cursor + styleHighlight.Render("[a]") + " Add channel\n")

	// Finish action
	cursor = "  "
	if m.cursor == finishIdx {
		cursor = styleCursor.Render("> ")
	}
	finishStyle := styleHighlight
	if len(m.channels) == 0 {
		finishStyle = styleSubtle
	}
	b.WriteString(cursor + finishStyle.Render("[f]") + " Finish\n")

	b.WriteByte('\n')
	b.WriteString(styleSubtle.Render("↑↓/jk navigate • enter select • x delete • esc quit"))

	return b.String()
}

// Messages emitted by listModel.
type (
	actionAddMsg    struct{}
	actionFinishMsg struct{}
	actionDeleteMsg struct{ index int }
)
