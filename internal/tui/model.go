package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dropalltables/canvas/internal/api"
)

var keys = struct {
	Up, Down, Done, ToggleDone, ToggleLocked, Open, Quit key.Binding
}{
	Up:           key.NewBinding(key.WithKeys("k", "up")),
	Down:         key.NewBinding(key.WithKeys("j", "down")),
	Done:         key.NewBinding(key.WithKeys("d", "x", " ")),
	ToggleDone:   key.NewBinding(key.WithKeys("D")),
	ToggleLocked: key.NewBinding(key.WithKeys("L")),
	Open:         key.NewBinding(key.WithKeys("enter")),
	Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c")),
}

type Model struct {
	assignments []api.Assignment
	cursor      int
	err         error
	client      *api.Client
	showDone    bool
	showLocked  bool
	baseURL     string
	descCache   map[int]string
	width       int
}

type errMsg error
type doneMsg int

func New(client *api.Client, assignments []api.Assignment, descCache map[int]string) Model {
	return Model{
		assignments: assignments,
		client:      client,
		baseURL:     client.BaseURL(),
		descCache:   descCache,
		width:       80,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case errMsg:
		m.err = msg

	case doneMsg:
		for i := range m.assignments {
			if m.assignments[i].PlannableID == int(msg) {
				m.assignments[i].Completed = true
				break
			}
		}
		visible := m.visibleAssignments()
		if m.cursor >= len(visible) && len(visible) > 0 {
			m.cursor = len(visible) - 1
		}

	case tea.KeyMsg:
		visible := m.visibleAssignments()
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Up) && m.cursor > 0:
			m.cursor--
		case key.Matches(msg, keys.Down) && m.cursor < len(visible)-1:
			m.cursor++
		case key.Matches(msg, keys.Done) && m.cursor < len(visible) && !visible[m.cursor].Completed:
			return m, m.markDone(visible[m.cursor])
		case key.Matches(msg, keys.ToggleDone):
			m.showDone = !m.showDone
			visible = m.visibleAssignments()
			if m.cursor >= len(visible) && len(visible) > 0 {
				m.cursor = len(visible) - 1
			} else if len(visible) == 0 {
				m.cursor = 0
			}
		case key.Matches(msg, keys.ToggleLocked):
			m.showLocked = !m.showLocked
			visible = m.visibleAssignments()
			if m.cursor >= len(visible) && len(visible) > 0 {
				m.cursor = len(visible) - 1
			} else if len(visible) == 0 {
				m.cursor = 0
			}
		case key.Matches(msg, keys.Open) && m.cursor < len(visible) && visible[m.cursor].HTMLURL != "":
			openBrowser(m.baseURL + visible[m.cursor].HTMLURL)
		}
	}
	return m, nil
}

func (m Model) markDone(a api.Assignment) tea.Cmd {
	return func() tea.Msg {
		var overrideID *int
		if a.Override != nil {
			overrideID = &a.Override.ID
		}
		if err := m.client.MarkDone(a.PlannableType, a.PlannableID, overrideID); err != nil {
			return errMsg(err)
		}
		return doneMsg(a.PlannableID)
	}
}

func (m Model) visibleAssignments() []api.Assignment {
	var visible []api.Assignment
	for _, a := range m.assignments {
		if !m.showDone && a.Completed {
			continue
		}
		if !m.showLocked && a.Locked {
			continue
		}
		visible = append(visible, a)
	}
	return visible
}
