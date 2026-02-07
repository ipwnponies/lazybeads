package app

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazybeads/internal/models"
	"lazybeads/internal/ui"
)

// PanelModel represents a single panel showing a filtered list of tasks
type PanelModel struct {
	title     string
	tasks     []models.Task
	selected  int
	focused   bool
	collapsed bool
	width     int
	height    int
	list      list.Model
}

// panelDelegate is a custom delegate for rendering task items in panels
type panelDelegate struct {
	listWidth int
	focused   bool
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

	width := m.Width()
	if width <= 0 {
		width = 40
	}

	line := formatTaskLine(t.task, width, isSelected, d.focused)
	fmt.Fprint(w, line)
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
	// Content area: panel width minus borders and padding (│ + space on each side = 4)
	contentWidth := width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}
	// Content height: panel height minus top and bottom borders (2 lines)
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}
	p.list.SetSize(contentWidth, contentHeight)
}

// SetFocus sets whether this panel is focused
func (p *PanelModel) SetFocus(focused bool) {
	p.focused = focused
	// Update delegate so it knows whether to show selection highlight
	p.list.SetDelegate(panelDelegate{focused: focused})
}

// IsFocused returns whether this panel is focused
func (p PanelModel) IsFocused() bool {
	return p.focused
}

// SetCollapsed sets whether this panel is collapsed to a single line
func (p *PanelModel) SetCollapsed(collapsed bool) {
	p.collapsed = collapsed
}

// IsCollapsed returns whether this panel is collapsed
func (p PanelModel) IsCollapsed() bool {
	return p.collapsed
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

	// Don't pass key messages to list - we handle navigation in HandleKey
	// This prevents double-processing of j/k which causes cursor to skip items
	if _, isKey := msg.(tea.KeyMsg); isKey {
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
	// If collapsed, render a single-line view
	if p.collapsed {
		return p.viewCollapsed()
	}

	// Use the full allocated width/height
	width := p.width
	height := p.height
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

	// Truncate title if too long (use lipgloss.Width for proper display width)
	maxTitleLen := width - 6 // Leave room for corners (╭─ and ─╮) and some border
	if lipgloss.Width(titleText) > maxTitleLen {
		// Truncate with ellipsis
		for lipgloss.Width(titleText) > maxTitleLen-3 && len(titleText) > 0 {
			titleText = titleText[:len(titleText)-1]
		}
		titleText = titleText + "..."
	}

	// Build top border: ╭─ Title ─────────╮
	// Use lipgloss.Width for proper character width calculation
	titleDisplayWidth := lipgloss.Width(titleText)
	remainingWidth := width - titleDisplayWidth - 4 // -4 for "╭─" and "─╮"
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	topBorder := borderStyle.Render("╭─") +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat("─", remainingWidth)+"─╮")

	// Build content area
	contentWidth := width - 4 // -4 for side borders and padding (│ + space on each side)
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

	// Pad or truncate content lines to fit the full height
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
		} else if lineWidth > contentWidth {
			// Truncate if too long
			line = lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth).Render(line)
		}
		// Add side borders with single space padding
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

// viewCollapsed renders a collapsed view with full border and one task line
func (p PanelModel) viewCollapsed() string {
	width := p.width
	if width < 10 {
		width = 10
	}

	// Use muted colors for unfocused collapsed panel
	borderColor := ui.ColorBorder
	titleColor := ui.ColorMuted
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(titleColor)

	// Build title with count
	titleText := fmt.Sprintf(" %s (%d) ", p.title, len(p.tasks))

	// Build top border: ╭─ Closed (5) ─────────╮
	titleDisplayWidth := lipgloss.Width(titleText)
	remainingWidth := width - titleDisplayWidth - 4 // -4 for "╭─" and "─╮"
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	topBorder := borderStyle.Render("╭─") +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat("─", remainingWidth)+"─╮")

	// Build content line with first task
	contentWidth := width - 4 // -4 for side borders and padding
	if contentWidth < 1 {
		contentWidth = 1
	}

	var contentLine string
	if len(p.tasks) > 0 {
		task := p.tasks[0]
		contentLine = formatTaskLine(task, contentWidth, false, false)
	} else {
		emptyStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true)
		contentLine = emptyStyle.Render("(no tasks)")
	}

	// Pad content to fill width
	lineWidth := lipgloss.Width(contentLine)
	if lineWidth < contentWidth {
		contentLine = contentLine + strings.Repeat(" ", contentWidth-lineWidth)
	}

	// Add side borders
	middleRow := borderStyle.Render("│") + " " + contentLine + " " + borderStyle.Render("│")

	// Build bottom border: ╰───────────────────╯
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", width-2) + "╯")

	return topBorder + "\n" + middleRow + "\n" + bottomBorder
}

func formatTaskLine(task models.Task, width int, isSelected bool, focused bool) string {
	priority := task.PriorityString()
	issueID := shortenIssueID(task.ID)
	title := task.Title
	treePrefix := task.TreePrefix

	var suffix string
	now := time.Now()
	deferred := task.IsDeferred(now)
	blocked := task.IsBlocked()
	stateMarker := " "
	if blocked {
		stateMarker = "⛔"
	} else if deferred {
		stateMarker = "⏳"
	}
	markerWidth := 2
	markerPad := markerWidth - lipgloss.Width(stateMarker)
	if markerPad < 0 {
		markerPad = 0
	}
	markerText := stateMarker + strings.Repeat(" ", markerPad)
	markerStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	if deferred || blocked {
		var parts []string
		if deferred {
			parts = append(parts, fmt.Sprintf("(%s)", formatRelativeTime(*task.DeferUntil, now)))
		}
		if blocked && task.TreePrefix == "" {
			blockedIDs := make([]string, 0, len(task.BlockedBy))
			for _, blockedID := range task.BlockedBy {
				blockedIDs = append(blockedIDs, shortenIssueID(blockedID))
			}
			blockedIDsText := strings.Join(blockedIDs, ", ")
			parts = append(parts, fmt.Sprintf("(blocked by %s)", blockedIDsText))
		}
		suffix = " " + strings.Join(parts, " ")
	}
	stateStyles := defaultStateStyles()
	deferredStyle := deferredStyleConfig(stateStyles["deferred"])

	// Calculate available width for title (account for priority, issue ID, spaces, and suffix)
	// Format: " P# issue-id title (:timer: in 5m)"
	prefixWidth := lipgloss.Width(fmt.Sprintf(" %s %s %s ", markerText, priority, issueID))
	suffixWidth := lipgloss.Width(suffix)
	maxTitleWidth := width - prefixWidth - suffixWidth
	if maxTitleWidth < 0 {
		maxTitleWidth = 0
	}

	treePrefixWidth := lipgloss.Width(treePrefix)
	titleWidthBudget := maxTitleWidth - treePrefixWidth
	if titleWidthBudget < 0 {
		titleWidthBudget = 0
	}
	title = truncateTitle(title, titleWidthBudget)
	displayTitle := treePrefix + title

	remainingWidth := width - prefixWidth - lipgloss.Width(displayTitle)
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	if lipgloss.Width(suffix) > remainingWidth {
		trimmed := truncateTitle(strings.TrimPrefix(suffix, " "), remainingWidth)
		if trimmed == "" {
			suffix = ""
		} else {
			suffix = " " + trimmed
		}
	}

	if isSelected && focused {
		// Show highlight only when panel is focused
		line := fmt.Sprintf(" %s %s %s %s%s", markerText, priority, issueID, displayTitle, suffix)
		bgColor := lipgloss.Color("#2a4a6d")
		fgColor := lipgloss.Color("15")
		faint := false
		if deferred {
			bgColor = deferredStyle.FocusedBackground
			fgColor = deferredStyle.FocusedForeground
			faint = true
		}
		style := lipgloss.NewStyle().
			Foreground(fgColor).
			Background(bgColor).
			Bold(true).
			Faint(faint).
			Width(width)
		return style.Render(line)
	}

	priorityStyle := ui.PriorityStyle(task.Priority)
	idStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	titleStyle := lipgloss.NewStyle().Foreground(ui.ColorWhite)
	suffixStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)

	if deferred {
		priorityStyle = priorityStyle.Faint(true)
		idStyle = idStyle.Faint(true)
		titleStyle = titleStyle.Faint(true)
		suffixStyle = suffixStyle.Faint(true)
	}

	line := fmt.Sprintf(" %s %s %s %s%s",
		markerStyle.Render(markerText),
		priorityStyle.Render(priority),
		idStyle.Render(issueID),
		titleStyle.Render(displayTitle),
		suffixStyle.Render(suffix))

	style := lipgloss.NewStyle().Width(width).MaxWidth(width)
	return style.Render(line)
}

func truncateTitle(title string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(title) <= maxWidth {
		return title
	}
	for lipgloss.Width(title+"...") > maxWidth && len(title) > 0 {
		title = title[:len(title)-1]
	}
	return title + "..."
}

func shortenIssueID(id string) string {
	lastDash := strings.LastIndex(id, "-")
	if lastDash <= 0 || lastDash >= len(id)-1 {
		return id
	}
	return id[lastDash+1:]
}

func formatRelativeTime(target time.Time, now time.Time) string {
	delta := target.Sub(now)
	future := delta >= 0
	abs := delta
	if abs < 0 {
		abs = -abs
	}

	seconds := abs.Seconds()
	if seconds < 1 {
		seconds = 1
	}

	var value int
	var unit string
	switch {
	case abs.Hours() >= 48:
		value = int(math.Ceil(abs.Hours() / 24))
		unit = "d"
	case abs.Minutes() >= 90:
		value = int(math.Ceil(abs.Hours()))
		unit = "h"
	case abs.Seconds() >= 90:
		value = int(math.Ceil(abs.Minutes()))
		unit = "m"
	default:
		value = int(math.Ceil(seconds))
		unit = "s"
	}

	if future {
		return fmt.Sprintf("in %d%s", value, unit)
	}
	return fmt.Sprintf("%d%s ago", value, unit)
}

type deferStyleConfig struct {
	FocusedBackground lipgloss.Color
	FocusedForeground lipgloss.Color
	MarkerColor       lipgloss.Color
}

type deferredStyleName string

const (
	deferredStyleAmber deferredStyleName = "amber"
	deferredStyleSlate deferredStyleName = "slate"
)

type stateStyleMap map[string]deferredStyleName

func defaultStateStyles() stateStyleMap {
	return stateStyleMap{
		"deferred": deferredStyleSlate,
		"blocked":  deferredStyleAmber,
	}
}

func deferredStyleConfig(styleName deferredStyleName) deferStyleConfig {
	switch styleName {
	case deferredStyleAmber:
		return deferStyleConfig{
			FocusedBackground: lipgloss.Color("#5a4a00"),
			FocusedForeground: lipgloss.Color("15"),
			MarkerColor:       lipgloss.Color("#c9a400"),
		}
	case deferredStyleSlate:
		return deferStyleConfig{
			FocusedBackground: lipgloss.Color("#3a3f45"),
			FocusedForeground: lipgloss.Color("15"),
			MarkerColor:       lipgloss.Color("#9aa0a6"),
		}
	default:
		return deferStyleConfig{
			FocusedBackground: lipgloss.Color("#3a3f45"),
			FocusedForeground: lipgloss.Color("15"),
			MarkerColor:       lipgloss.Color("#9aa0a6"),
		}
	}
}
