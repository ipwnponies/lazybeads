package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazybeads/internal/models"
	"lazybeads/internal/ui"
)

// PanelModel represents a single panel showing a filtered list of tasks
type PanelModel struct {
	title    string
	tasks    []models.Task
	selected int
	focused  bool
	width    int
	height   int
	list     list.Model
}

// panelDelegate is a custom delegate for rendering task items in panels
type panelDelegate struct {
	listWidth int
}

func newPanelDelegate() panelDelegate {
	return panelDelegate{}
}

func (d panelDelegate) Height() int                             { return 1 }
func (d panelDelegate) Spacing() int                            { return 0 }
func (d panelDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d panelDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	t, ok := item.(taskItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	icon := t.task.StatusIcon()
	priority := t.task.PriorityString()
	title := t.task.Title

	width := m.Width()
	if width <= 0 {
		width = 40
	}

	if isSelected {
		line := fmt.Sprintf(" %s %s %s", icon, priority, title)
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("#2a4a6d")).
			Bold(true).
			Width(width)
		fmt.Fprint(w, style.Render(line))
	} else {
		iconStyle := ui.StatusStyle(t.task.Status)
		priorityStyle := ui.PriorityStyle(t.task.Priority)

		line := fmt.Sprintf(" %s %s %s",
			iconStyle.Render(icon),
			priorityStyle.Render(priority),
			title)
		fmt.Fprint(w, line)
	}
}

// NewPanel creates a new panel with the given title
func NewPanel(title string) PanelModel {
	delegate := newPanelDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)

	return PanelModel{
		title:    title,
		tasks:    []models.Task{},
		selected: 0,
		focused:  false,
		list:     l,
	}
}

// SetTasks updates the panel's task list
func (p *PanelModel) SetTasks(tasks []models.Task) {
	p.tasks = tasks
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{task: t}
	}
	p.list.SetItems(items)
}

// SetSize updates the panel dimensions
func (p *PanelModel) SetSize(width, height int) {
	p.width = width
	p.height = height
	// Account for border (2) and title line (1)
	contentHeight := height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	contentWidth := width - 4 // Account for border and padding
	if contentWidth < 10 {
		contentWidth = 10
	}
	p.list.SetSize(contentWidth, contentHeight)
}

// SetFocus sets whether this panel is focused
func (p *PanelModel) SetFocus(focused bool) {
	p.focused = focused
}

// IsFocused returns whether this panel is focused
func (p PanelModel) IsFocused() bool {
	return p.focused
}

// SelectedTask returns the currently selected task, if any
func (p PanelModel) SelectedTask() *models.Task {
	if len(p.tasks) == 0 {
		return nil
	}
	idx := p.list.Index()
	if idx >= 0 && idx < len(p.tasks) {
		return &p.tasks[idx]
	}
	return nil
}

// TaskCount returns the number of tasks in this panel
func (p PanelModel) TaskCount() int {
	return len(p.tasks)
}

// Update handles messages for the panel
func (p PanelModel) Update(msg tea.Msg) (PanelModel, tea.Cmd) {
	if !p.focused {
		return p, nil
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// HandleKey handles key navigation within the panel
func (p *PanelModel) HandleKey(msg tea.KeyMsg, keys ui.KeyMap) bool {
	if !p.focused {
		return false
	}

	switch {
	case key.Matches(msg, keys.Up):
		p.list.CursorUp()
		return true
	case key.Matches(msg, keys.Down):
		p.list.CursorDown()
		return true
	case key.Matches(msg, keys.Top):
		p.list.Select(0)
		return true
	case key.Matches(msg, keys.Bottom):
		if len(p.tasks) > 0 {
			p.list.Select(len(p.tasks) - 1)
		}
		return true
	case key.Matches(msg, keys.PageUp):
		for i := 0; i < 10; i++ {
			p.list.CursorUp()
		}
		return true
	case key.Matches(msg, keys.PageDown):
		for i := 0; i < 10; i++ {
			p.list.CursorDown()
		}
		return true
	}
	return false
}

// View renders the panel
func (p PanelModel) View() string {
	// Choose style based on focus
	var panelStyle lipgloss.Style
	var titleColor lipgloss.Color
	if p.focused {
		panelStyle = ui.FocusedPanelStyle
		titleColor = ui.ColorPrimary
	} else {
		panelStyle = ui.PanelStyle
		titleColor = ui.ColorMuted
	}

	// Build title with count
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(titleColor)
	title := titleStyle.Render(fmt.Sprintf("%s (%d)", p.title, len(p.tasks)))

	// Build content
	var content strings.Builder
	content.WriteString(title + "\n")

	if len(p.tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true)
		content.WriteString(emptyStyle.Render("  (no tasks)"))
	} else {
		content.WriteString(p.list.View())
	}

	return panelStyle.
		Width(p.width - 2).
		Height(p.height - 2).
		Render(content.String())
}
