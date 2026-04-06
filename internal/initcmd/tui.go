package initcmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sznuper/sznuper/internal/config"
)

type screen int

const (
	screenList screen = iota
	screenPicker
	screenForm
	screenConfirm
)

// Model is the root bubbletea model for `sznuper init`.
type Model struct {
	screen  screen
	list    listModel
	picker  pickerModel
	form    formModel
	confirm confirmModel

	cfg   *config.Config
	path  string
	err   string
	done  bool
	wrote bool
}

// NewModel creates the root TUI model.
// Channels inherited from --from are shown pre-populated; cfg is the working config.
func NewModel(cfg *config.Config, path string) Model {
	var channels []addedChannel
	for name, ch := range cfg.Channels {
		channels = append(channels, addedChannel{
			name:     name,
			typeName: "base",
			url:      ch.URL,
			params:   ch.Params,
		})
	}
	return Model{
		screen: screenList,
		list:   newListModel(channels),
		cfg:    cfg,
		path:   path,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global quit
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "ctrl+c":
			m.done = true
			return m, tea.Quit
		case "esc":
			return m.handleEsc()
		}
	}

	switch m.screen {
	case screenList:
		return m.updateList(msg)
	case screenPicker:
		return m.updatePicker(msg)
	case screenForm:
		return m.updateForm(msg)
	case screenConfirm:
		return m.updateConfirm(msg)
	}
	return m, nil
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenList:
		m.done = true
		return m, tea.Quit
	case screenPicker, screenForm:
		m.screen = screenList
		return m, nil
	case screenConfirm:
		m.screen = screenList
		return m, nil
	}
	return m, nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actionAddMsg:
		m.screen = screenPicker
		m.picker = newPickerModel()
		return m, m.picker.Init()
	case actionFinishMsg:
		if len(m.list.channels) == 0 {
			m.err = "Add at least one notification channel before saving"
			return m, nil
		}
		m.buildConfig()
		m.confirm = newConfirmModel(m.cfg, m.path)
		m.screen = screenConfirm
		return m, nil
	case actionDeleteMsg:
		if msg.index < len(m.list.channels) {
			m.list.channels = append(m.list.channels[:msg.index], m.list.channels[msg.index+1:]...)
			if m.list.cursor >= len(m.list.channels)+2 {
				m.list.cursor = len(m.list.channels) + 1
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pickedServiceMsg:
		existing := m.existingNames()
		m.form = newFormModel(msg.svcType, existing)
		m.screen = screenForm
		return m, m.form.Init()
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case formSubmitMsg:
		m.list.channels = append(m.list.channels, msg.channel)
		m.list.cursor = len(m.list.channels) // point to [a]dd
		m.screen = screenList
		return m, nil
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case confirmWriteMsg:
		m.buildConfig()
		if err := config.Write(m.cfg, m.path); err != nil {
			m.err = fmt.Sprintf("Error writing config: %v", err)
			m.screen = screenList
			return m, nil
		}
		m.wrote = true
		m.done = true
		return m, tea.Quit
	case confirmCancelMsg:
		m.screen = screenList
		return m, nil
	}

	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.done && m.wrote {
		return styleSuccess.Render(fmt.Sprintf("✓ Config written to %s\n", m.path))
	}
	if m.done {
		return ""
	}

	var view string
	switch m.screen {
	case screenList:
		view = m.list.View()
	case screenPicker:
		view = m.picker.View()
	case screenForm:
		view = m.form.View()
	case screenConfirm:
		view = m.confirm.View()
	}

	if m.err != "" {
		view += "\n" + styleError.Render(m.err)
	}

	return view + "\n"
}

func (m *Model) buildConfig() {
	existing := make(map[string]bool)
	for name := range m.cfg.Channels {
		existing[name] = true
	}

	if m.cfg.Channels == nil {
		m.cfg.Channels = make(map[string]config.Channel)
	}

	var added []string
	for _, svc := range m.list.channels {
		ch := config.Channel{URL: svc.url}
		if len(svc.params) > 0 {
			ch.Params = make(map[string]string)
			for k, v := range svc.params {
				ch.Params[k] = v
			}
		}
		m.cfg.Channels[svc.name] = ch
		if !existing[svc.name] {
			added = append(added, svc.name)
		}
	}

	for i := range m.cfg.Alerts {
		for _, name := range added {
			m.cfg.Alerts[i].Notify = append(m.cfg.Alerts[i].Notify,
				config.NotifyTarget{Channel: name})
		}
	}
}

func (m Model) existingNames() map[string]bool {
	names := make(map[string]bool)
	for _, svc := range m.list.channels {
		names[svc.name] = true
	}
	return names
}

// Run launches the TUI and returns the final model.
func Run(cfg *config.Config, path string) error {
	m := NewModel(cfg, path)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	fm, ok := finalModel.(Model)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}
	if fm.err != "" {
		return fmt.Errorf("%s", fm.err)
	}
	if !fm.wrote {
		return fmt.Errorf("cancelled")
	}
	return nil
}
