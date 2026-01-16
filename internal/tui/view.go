package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dropalltables/canvas/internal/api"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	doneStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	dateStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	descStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Faint(true)
)

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Canvas Assignments"))
	b.WriteString(" ")
	b.WriteString(dateStyle.Render(m.baseURL))
	b.WriteString("\n\n")

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

	descWidth := m.width - 8
	if descWidth < 30 {
		descWidth = 30
	}

	for i, a := range visible {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		checkbox := "[ ]"
		if a.Completed {
			checkbox = "[x]"
		} else if a.Locked {
			checkbox = "[!]"
		}
		dueStr := ""
		if a.DueAt != nil {
			dueStr = a.DueAt.Format("Jan 02 15:04")
		}

		line := fmt.Sprintf("%s%s %s  %s", cursor, checkbox, dueStr, a.Plannable.Title)
		if a.Completed || a.Locked {
			line = doneStyle.Render(line)
		} else if i == m.cursor {
			line = selectedStyle.Render(line)
		} else {
			line = fmt.Sprintf("%s%s %s  %s", cursor, checkbox, dateStyle.Render(dueStr), a.Plannable.Title)
		}
		b.WriteString(line + "\n")

		if i == m.cursor {
			if desc := m.descCache[a.PlannableID]; desc != "" {
				b.WriteString(m.renderDesc(desc, descWidth))
			}
		}
	}

	b.WriteString("\n")
	toggleDone := "show done"
	if m.showDone {
		toggleDone = "hide done"
	}
	toggleLocked := "show locked"
	if m.showLocked {
		toggleLocked = "hide locked"
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf("j/k: navigate  enter: open  d/x/space: mark done  D: %s  L: %s  q: quit", toggleDone, toggleLocked)))
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderDesc(desc string, width int) string {
	var b strings.Builder
	for _, line := range strings.Split(desc, "\n") {
		if strings.HasPrefix(line, "- ") {
			b.WriteString("    " + descStyle.Render(line) + "\n")
		} else {
			wrapped := wrapText(line, width)
			for _, w := range strings.Split(wrapped, "\n") {
				b.WriteString("    " + descStyle.Render(w) + "\n")
			}
		}
	}
	return b.String()
}
