package initcmd

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// pickerModel lets the user choose a service type from the registry.
type pickerModel struct {
	items    []ServiceType
	filtered []int // indices into items
	cursor   int
	filter   textinput.Model
}

func newPickerModel() pickerModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 40
	ti.Width = 30

	m := pickerModel{
		items:  Registry,
		filter: ti,
	}
	m.applyFilter()
	return m
}

func (m *pickerModel) applyFilter() {
	query := strings.ToLower(m.filter.Value())
	m.filtered = m.filtered[:0]
	for i, svc := range m.items {
		if query == "" || strings.Contains(strings.ToLower(svc.Label), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m pickerModel) Init() tea.Cmd { return textinput.Blink }

func (m pickerModel) Update(msg tea.Msg) (pickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			if len(m.filtered) > 0 {
				picked := m.items[m.filtered[m.cursor]]
				return m, func() tea.Msg { return pickedServiceMsg{svcType: picked} }
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m pickerModel) View() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("Pick Service Type"))
	b.WriteByte('\n')

	for i, idx := range m.filtered {
		cursor := "  "
		if i == m.cursor {
			cursor = styleCursor.Render("> ")
		}
		label := m.items[idx].Label
		if i == m.cursor {
			label = styleSelected.Render(label)
		}
		b.WriteString(cursor + label + "\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(styleSubtle.Render("  No matches."))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(m.filter.View())
	b.WriteByte('\n')
	b.WriteString(styleSubtle.Render("↑↓ navigate • enter select • esc back"))

	return b.String()
}

// pickedServiceMsg is emitted when the user selects a service type.
type pickedServiceMsg struct {
	svcType ServiceType
}
