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
	// Account for border (2 top/bottom) and side padding (4 for │ + space on each side)
	contentHeight := height - 4 // top border + bottom border + some padding
	if contentHeight < 1 {
		contentHeight = 1
	}
	contentWidth := width - 6 // side borders + padding
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

// View renders the panel with title embedded in the top border
func (p PanelModel) View() string {
	width := p.width - 2
	height := p.height - 2
	if width < 10 {
		width = 10
	}
	if height < 3 {
		height = 3
	}

	// Choose colors based on focus
	var borderColor, titleColor lipgloss.Color
	if p.focused {
		borderColor = ui.ColorPrimary
		titleColor = ui.ColorPrimary
	} else {
		borderColor = ui.ColorBorder
		titleColor = ui.ColorMuted
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(titleColor)

	// Build title with count
	titleText := fmt.Sprintf(" %s (%d) ", p.title, len(p.tasks))

	// Truncate title if too long
	maxTitleLen := width - 4 // Leave room for corners and some border
	if len(titleText) > maxTitleLen {
		if maxTitleLen > 3 {
			titleText = titleText[:maxTitleLen-3] + "..."
		} else {
			titleText = ""
		}
	}

	// Build top border: ╭─ Title ─────────╮
	remainingWidth := width - len(titleText) - 2 // -2 for corners
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	topBorder := borderStyle.Render("╭─") +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat("─", remainingWidth) + "╮")

	// Build content area
	contentWidth := width - 2 // -2 for side borders
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeight := height - 2 // -2 for top/bottom borders
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render the task list or empty message
	var contentLines []string
	if len(p.tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true)
		emptyMsg := emptyStyle.Render("(no tasks)")
		// Pad to content width
		padded := emptyMsg + strings.Repeat(" ", max(0, contentWidth-lipgloss.Width(emptyMsg)))
		contentLines = append(contentLines, padded)
	} else {
		// Get the list view and split into lines
		listView := p.list.View()
		contentLines = strings.Split(listView, "\n")
	}

	// Pad or truncate content lines to fit
	var middleRows []string
	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		} else {
			line = ""
		}
		// Ensure line fits within content width (with padding)
		lineWidth := lipgloss.Width(line)
		if lineWidth < contentWidth {
			line = line + strings.Repeat(" ", contentWidth-lineWidth)
		}
		// Add side borders
		row := borderStyle.Render("│") + " " + line + " " + borderStyle.Render("│")
		middleRows = append(middleRows, row)
	}

	// Build bottom border: ╰───────────────────╯
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", width-2) + "╯")

	// Combine all parts
	var result strings.Builder
	result.WriteString(topBorder + "\n")
	for _, row := range middleRows {
		result.WriteString(row + "\n")
	}
	result.WriteString(bottomBorder)

	return result.String()
}
