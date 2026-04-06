package initcmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// formModel collects field values (env var names) for a chosen channel type.
type formModel struct {
	svcType   ChannelType
	nameInput textinput.Model // channel name
	inputs    []textinput.Model
	fields    []ChannelField
	focusIdx  int // 0 = name, 1..n = fields
	err       string
	existing  map[string]bool // existing channel names (for uniqueness check)
}

func newFormModel(svcType ChannelType, existing map[string]bool) formModel {
	nameInput := textinput.New()
	nameInput.Placeholder = svcType.Name
	nameInput.CharLimit = 40
	nameInput.Width = 40
	nameInput.Focus()

	var inputs []textinput.Model
	for _, f := range svcType.Fields {
		ti := textinput.New()
		ti.Placeholder = f.Placeholder
		ti.CharLimit = 80
		ti.Width = 40
		if f.EnvVar != "" {
			ti.SetValue(f.EnvVar)
		}
		inputs = append(inputs, ti)
	}

	return formModel{
		svcType:   svcType,
		nameInput: nameInput,
		inputs:    inputs,
		fields:    svcType.Fields,
		focusIdx:  0,
		existing:  existing,
	}
}

func (m formModel) Init() tea.Cmd { return textinput.Blink }

func (m *formModel) focusCurrent() {
	m.nameInput.Blur()
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	if m.focusIdx == 0 {
		m.nameInput.Focus()
	} else if m.focusIdx-1 < len(m.inputs) {
		m.inputs[m.focusIdx-1].Focus()
	}
}

func (m formModel) totalFields() int {
	return 1 + len(m.inputs) // name + fields
}

func (m formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.err = ""
			if m.focusIdx < m.totalFields()-1 {
				m.focusIdx++
				m.focusCurrent()
			}
			return m, nil
		case "shift+tab", "up":
			m.err = ""
			if m.focusIdx > 0 {
				m.focusIdx--
				m.focusCurrent()
			}
			return m, nil
		case "enter":
			// If not on the last field, advance
			if m.focusIdx < m.totalFields()-1 {
				m.focusIdx++
				m.focusCurrent()
				return m, nil
			}
			// On last field — validate and submit
			return m, m.trySubmit()
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	if m.focusIdx == 0 {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		idx := m.focusIdx - 1
		m.inputs[idx], cmd = m.inputs[idx].Update(msg)
	}
	return m, cmd
}

func (m *formModel) trySubmit() tea.Cmd {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		name = m.svcType.Name
	}
	if m.existing[name] {
		m.err = fmt.Sprintf("Channel name %q already exists", name)
		return nil
	}

	// Build values map keyed by field label
	vals := make(map[string]string)
	for i, f := range m.fields {
		v := strings.TrimSpace(m.inputs[i].Value())
		if v == "" {
			v = f.EnvVar
		}
		vals[f.Label] = v
	}

	url := m.svcType.BuildURL(vals)

	// Build params from IsParam fields using actual Shoutrrr param keys
	params := make(map[string]string)
	for i, f := range m.fields {
		if f.IsParam {
			v := strings.TrimSpace(m.inputs[i].Value())
			if v == "" {
				v = f.EnvVar
			}
			key := f.ParamKey
			if key == "" {
				key = f.Label
			}
			params[key] = envRef(v)
		}
	}

	svc := addedChannel{
		name:     name,
		typeName: m.svcType.Name,
		url:      url,
		params:   params,
	}
	return func() tea.Msg { return formSubmitMsg{channel: svc} }
}

func (m formModel) View() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render(m.svcType.Label))
	b.WriteByte('\n')

	// Channel name
	label := "Name for this channel:"
	if m.focusIdx == 0 {
		label = styleHighlight.Render(label)
	}
	b.WriteString(label + "\n")
	b.WriteString(m.nameInput.View() + "\n")
	b.WriteString(styleSubtle.Render("(must be unique, default: "+m.svcType.Name+")") + "\n\n")

	// Fields
	for i, f := range m.fields {
		purpose := "Env var for "
		if f.EnvVar == "" {
			purpose = ""
		}
		label := purpose + f.Label + ":"
		if m.focusIdx == i+1 {
			label = styleHighlight.Render(label)
		}
		b.WriteString(label + "\n")
		b.WriteString(m.inputs[i].View() + "\n\n")
	}

	if m.err != "" {
		b.WriteString(styleError.Render(m.err) + "\n")
	}

	b.WriteString(styleSubtle.Render("tab/↑↓ navigate fields • enter submit • esc back"))

	return b.String()
}

// formSubmitMsg is emitted when the form is completed.
type formSubmitMsg struct {
	channel addedChannel
}
