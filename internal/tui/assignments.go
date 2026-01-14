package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dropalltables/canvas/internal/api"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	itemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("170"))
	doneStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	dateStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Done     key.Binding
	ToggleDone key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/up", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/down", "down"),
	),
	Done: key.NewBinding(
		key.WithKeys("d", " "),
		key.WithHelp("d/space", "mark done"),
	),
	ToggleDone: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "toggle show done"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type Model struct {
	assignments []api.Assignment
	cursor      int
	loading     bool
	err         error
	client      *api.Client
	showDone    bool
	baseURL     string
}

type assignmentsMsg []api.Assignment
type errMsg error
type doneMsg int

func New(client *api.Client) Model {
	return Model{
		loading: true,
		client:  client,
		baseURL: client.BaseURL(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchAssignments
}

func (m Model) fetchAssignments() tea.Msg {
	assignments, err := m.client.GetAssignments(14, 30)
	if err != nil {
		return errMsg(err)
	}
	return assignmentsMsg(assignments)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case assignmentsMsg:
		m.assignments = msg
		m.loading = false
		return m, nil

	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil

	case doneMsg:
		for i := range m.assignments {
			if m.assignments[i].PlannableID == int(msg) {
				m.assignments[i].Completed = true
				break
			}
		}
		return m, nil

	case tea.KeyMsg:
		visible := m.visibleAssignments()

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, keys.Down):
			if m.cursor < len(visible)-1 {
				m.cursor++
			}

		case key.Matches(msg, keys.Done):
			if len(visible) > 0 && m.cursor < len(visible) {
				a := visible[m.cursor]
				if !a.Completed {
					return m, m.markDone(a)
				}
			}

		case key.Matches(msg, keys.ToggleDone):
			m.showDone = !m.showDone
			m.cursor = 0
		}
	}

	return m, nil
}

func (m Model) markDone(a api.Assignment) tea.Cmd {
	return func() tea.Msg {
		err := m.client.MarkDone(a.PlannableType, a.PlannableID)
		if err != nil {
			return errMsg(err)
		}
		return doneMsg(a.PlannableID)
	}
}

func (m Model) visibleAssignments() []api.Assignment {
	if m.showDone {
		return m.assignments
	}

	var visible []api.Assignment
	for _, a := range m.assignments {
		if !a.Completed {
			visible = append(visible, a)
		}
	}
	return visible
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Canvas Assignments"))
	b.WriteString(" ")
	b.WriteString(dateStyle.Render(m.baseURL))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("  Loading...\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v\n", m.err)))
		if apiErr, ok := m.err.(*api.APIError); ok && apiErr.IsUnauthorized() {
			b.WriteString(errorStyle.Render("  Run 'canvas auth login' to authenticate\n"))
		}
		return b.String()
	}

	visible := m.visibleAssignments()

	if len(visible) == 0 {
		b.WriteString("  No assignments\n")
	}

	for i, a := range visible {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := "[ ]"
		if a.Completed {
			checkbox = "[x]"
		}

		dueStr := ""
		if a.DueAt != nil {
			dueStr = a.DueAt.Format("Jan 02 15:04")
		}

		title := a.Plannable.Title

		var line string
		if a.Completed {
			line = fmt.Sprintf("%s%s %s  %s", cursor, checkbox, dueStr, title)
			line = doneStyle.Render(line)
		} else if i == m.cursor {
			line = fmt.Sprintf("%s%s %s  %s", cursor, checkbox, dueStr, title)
			line = selectedStyle.Render(line)
		} else {
			line = fmt.Sprintf("%s%s %s  %s", cursor, checkbox, dateStyle.Render(dueStr), title)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	showDoneText := "show done"
	if m.showDone {
		showDoneText = "hide done"
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf("j/k: navigate  d/space: mark done  D: %s  q: quit", showDoneText)))
	b.WriteString("\n")

	return b.String()
}

func Run(client *api.Client) error {
	p := tea.NewProgram(New(client))
	_, err := p.Run()
	return err
}
