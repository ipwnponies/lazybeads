package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"lazybeads/internal/ui"
)

// View renders the application
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	switch m.mode {
	case ViewHelp:
		return m.viewHelp()
	case ViewConfirm:
		return m.viewConfirm()
	case ViewForm:
		return m.viewForm()
	case ViewDetail:
		if m.width < 80 {
			// Narrow mode: full screen detail
			return m.viewDetailOverlay()
		}
		return m.viewMain()
	case ViewEditTitle, ViewEditStatus, ViewEditPriority, ViewEditType, ViewFilter:
		return m.viewMainWithModal()
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	var b strings.Builder

	// Content area
	contentHeight := m.height - 2

	// Stack visible panels vertically
	var panelViews []string
	if m.isInProgressVisible() {
		panelViews = append(panelViews, m.inProgressPanel.View())
	}
	panelViews = append(panelViews, m.openPanel.View())
	panelViews = append(panelViews, m.closedPanel.View())
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, panelViews...)

	if m.width >= wideModeMinWidth {
		// Wide mode: panels on left, detail on right
		detailStyle := ui.PanelStyle
		if m.mode == ViewDetail {
			detailStyle = ui.FocusedPanelStyle
		}

		detailContent := ""
		if m.selected != nil {
			m.updateDetailContent()
			detailContent = m.detail.View()
		} else {
			detailContent = ui.HelpDescStyle.Render("Select a task to view details")
		}

		detailPanel := detailStyle.
			Width(m.detailWidth).
			Height(contentHeight - 2). // -2 for lipgloss border (top + bottom)
			Render(detailContent)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, detailPanel))
	} else {
		// Narrow mode: panels only
		b.WriteString(leftColumn)
	}

	b.WriteString("\n")

	// Error message if any
	if m.err != nil {
		b.WriteString(ui.ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
		m.err = nil
	}

	// Status bar (shows key bindings by default, search results when filtering)
	statusText := m.renderStatusBar()
	b.WriteString(ui.HelpBarStyle.Render(statusText))

	return b.String()
}

func (m Model) viewDetailOverlay() string {
	var b strings.Builder

	title := ui.TitleStyle.Render("Task Details")
	b.WriteString(title + "\n\n")

	m.updateDetailContent()
	content := ui.OverlayStyle.
		Width(m.width - 4).
		Height(m.height - 6).
		Render(m.detail.View())
	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(ui.HelpBarStyle.Render("enter/esc: back  ?: help"))

	return b.String()
}

func (m *Model) viewHelp() string {
	var b strings.Builder

	helpWidth, helpHeight := helpModalSize(m.width, m.height)
	listHeight := helpHeight - 3
	if listHeight < 1 {
		listHeight = 1
	}
	m.helpList.SetSize(helpWidth-2, listHeight)

	b.WriteString(ui.TitleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n")
	b.WriteString(m.helpList.View())
	b.WriteString("\n")

	helpParts := []string{
		ui.HelpKeyStyle.Render("j/k") + ":" + ui.HelpDescStyle.Render("move"),
		ui.HelpKeyStyle.Render("^u/^d") + ":" + ui.HelpDescStyle.Render("page"),
		ui.HelpKeyStyle.Render("enter") + ":" + ui.HelpDescStyle.Render("run"),
		ui.HelpKeyStyle.Render("/") + ":" + ui.HelpDescStyle.Render("filter"),
		ui.HelpKeyStyle.Render("q/?/esc") + ":" + ui.HelpDescStyle.Render("close"),
	}

	if m.helpFilterActive {
		filterPart := ui.HelpKeyStyle.Render("/: ") + m.helpFilterInput.View()
		helpParts = append(helpParts, filterPart)
	} else if m.helpFilterQuery != "" {
		filterPart := ui.HelpKeyStyle.Render("/: ") + ui.HelpDescStyle.Render(m.helpFilterQuery)
		helpParts = append(helpParts, filterPart)
	}

	total := len(m.helpItems)
	filtered := len(m.helpList.Items())
	helpParts = append(helpParts, ui.HelpDescStyle.Render(fmt.Sprintf("(%d of %d shortcuts)", filtered, total)))

	helpBar := strings.Join(helpParts, "  ")
	b.WriteString(ui.HelpBarStyle.Render(helpBar))

	helpBoxStyle := ui.OverlayStyle.Padding(0, 1)
	modal := helpBoxStyle.
		Width(helpWidth).
		Height(helpHeight).
		Render(b.String())

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

func (m Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Confirm") + "\n\n")
	b.WriteString(ui.OverlayStyle.Render(m.confirmMsg + "\n\n(y)es / (n)o"))

	return b.String()
}

func (m Model) viewMainWithModal() string {
	// Render the modal centered on screen
	return m.modal.View(m.width, m.height)
}

func (m Model) renderStatusBar() string {
	var parts []string

	// Show status message if present (flash notifications like "Copied!")
	if m.statusMsg != "" {
		parts = append(parts, ui.SuccessStyle.Render(m.statusMsg))
	}

	// When in search mode, show the search input
	if m.searchMode {
		// Search input with cursor
		searchPart := ui.HelpKeyStyle.Render("/: ") + m.searchInput.View()
		parts = append(parts, searchPart)

		// Live result counts
		inProgressCount := m.inProgressPanel.TaskCount()
		openCount := m.openPanel.TaskCount()
		closedCount := m.closedPanel.TaskCount()
		total := inProgressCount + openCount + closedCount

		resultsPart := ui.HelpDescStyle.Render(fmt.Sprintf("(%d results", total))
		if inProgressCount > 0 {
			resultsPart += ui.StatusStyle("in_progress").Render(fmt.Sprintf(": %d in progress", inProgressCount))
		}
		if openCount > 0 {
			resultsPart += ui.StatusStyle("open").Render(fmt.Sprintf(", %d open", openCount))
		}
		if closedCount > 0 {
			resultsPart += ui.HelpDescStyle.Render(fmt.Sprintf(", %d closed", closedCount))
		}
		resultsPart += ui.HelpDescStyle.Render(")")
		parts = append(parts, resultsPart)

		// Minimal key hints during search
		parts = append(parts, ui.HelpKeyStyle.Render("enter")+":"+ui.HelpDescStyle.Render("confirm"))
		parts = append(parts, ui.HelpKeyStyle.Render("esc")+":"+ui.HelpDescStyle.Render("clear"))
	} else if m.filterQuery != "" {
		// When filter is active (but not in search mode), show search results
		// Filter indicator
		filterPart := ui.HelpKeyStyle.Render("/") + ":" +
			ui.HelpDescStyle.Render(m.filterQuery)
		parts = append(parts, filterPart)

		// Search result counts
		inProgressCount := m.inProgressPanel.TaskCount()
		openCount := m.openPanel.TaskCount()
		closedCount := m.closedPanel.TaskCount()
		total := inProgressCount + openCount + closedCount

		resultsPart := ui.HelpDescStyle.Render(fmt.Sprintf("(%d results:", total))
		if inProgressCount > 0 {
			resultsPart += ui.StatusStyle("in_progress").Render(fmt.Sprintf(" %d in progress", inProgressCount))
		}
		if openCount > 0 {
			resultsPart += ui.StatusStyle("open").Render(fmt.Sprintf(" %d open", openCount))
		}
		if closedCount > 0 {
			resultsPart += ui.HelpDescStyle.Render(fmt.Sprintf(" %d closed", closedCount))
		}
		resultsPart += ui.HelpDescStyle.Render(")")
		parts = append(parts, resultsPart)

		// Minimal key bindings when filtering
		parts = append(parts, ui.HelpKeyStyle.Render("esc")+":"+ui.HelpDescStyle.Render("clear"))
	} else {
		// Default: show key bindings
		keys := []struct {
			key  string
			desc string
		}{
			{"j/k", "nav"},
			{"h/l, ←/→, tab/shift+tab", "panel"},
			{"/", "filter"},
			{"enter", "detail"},
			{"e/s/p/t/d/N/D/C", "edit"},
			{"y", "copy"},
			{"x", "delete"},
			{"?", "help"},
			{"q", "quit"},
		}

		for _, k := range keys {
			part := ui.HelpKeyStyle.Render(k.key) + ":" + ui.HelpDescStyle.Render(k.desc)
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, "  ")
}

func (m *Model) updateDetailContent() {
	if m.selected == nil {
		m.detail.SetContent("")
		return
	}

	t := m.selected
	var b strings.Builder

	b.WriteString(ui.DetailLabelStyle.Render("ID:"))
	b.WriteString(ui.DetailValueStyle.Render(t.ID))
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Title:"))
	b.WriteString(ui.DetailValueStyle.Render(t.Title))
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Status:"))
	b.WriteString(ui.StatusStyle(t.Status).Render(t.Status))
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Priority:"))
	b.WriteString(ui.PriorityStyle(t.Priority).Render(t.PriorityString()))
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Type:"))
	b.WriteString(ui.DetailValueStyle.Render(t.Type))
	b.WriteString("\n")

	if t.Assignee != "" {
		b.WriteString(ui.DetailLabelStyle.Render("Assignee:"))
		b.WriteString(ui.DetailValueStyle.Render(t.Assignee))
		b.WriteString("\n")
	}

	if len(t.Labels) > 0 {
		b.WriteString(ui.DetailLabelStyle.Render("Labels:"))
		b.WriteString(ui.DetailValueStyle.Render(strings.Join(t.Labels, ", ")))
		b.WriteString("\n")
	}

	if t.DueDate != nil {
		b.WriteString(ui.DetailLabelStyle.Render("Due:"))
		b.WriteString(ui.DetailValueStyle.Render(t.DueDate.Format("2006-01-02")))
		b.WriteString("\n")
	}

	if t.IsDeferred(time.Now()) {
		b.WriteString(ui.DetailLabelStyle.Render("Deferred:"))
		b.WriteString(ui.DetailValueStyle.Render("until " + t.DeferUntil.Format("2006-01-02")))
		b.WriteString("\n")
	}

	renderWrappedSection := func(label, value string) {
		if value == "" {
			return
		}
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render(label))
		b.WriteString("\n")
		descWidth := m.detail.Width - 2
		if descWidth < 20 {
			descWidth = 20
		}
		wrapped := lipgloss.NewStyle().Width(descWidth).Render(value)
		b.WriteString(wrapped)
		b.WriteString("\n")
	}

	renderWrappedSection("Description:", t.Description)
	renderWrappedSection("Notes:", t.Notes)
	renderWrappedSection("Design:", t.Design)
	renderWrappedSection("Acceptance Criteria:", t.AcceptanceCriteria)
	renderWrappedSection("Close Reason:", t.CloseReason)

	if len(t.BlockedBy) > 0 {
		taskTitles := make(map[string]string, len(m.tasks))
		for _, task := range m.tasks {
			taskTitles[task.ID] = task.Title
		}
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Blocked by:"))
		b.WriteString("\n")
		for _, id := range t.BlockedBy {
			line := id
			if title, ok := taskTitles[id]; ok && title != "" {
				line = fmt.Sprintf("%s %s", id, title)
			}
			b.WriteString("  - " + line + "\n")
		}
	}

	if len(t.Blocks) > 0 {
		b.WriteString("\n")
		b.WriteString(ui.DetailLabelStyle.Render("Blocks:"))
		b.WriteString("\n")
		for _, id := range t.Blocks {
			b.WriteString("  - " + id + "\n")
		}
	}

	// Timestamps section
	b.WriteString("\n")
	b.WriteString(ui.DetailLabelStyle.Render("Created:"))
	b.WriteString(ui.DetailValueStyle.Render(t.CreatedAt.Format("2006-01-02 15:04")))
	if t.CreatedBy != "" {
		b.WriteString(ui.HelpDescStyle.Render(" by " + t.CreatedBy))
	}
	b.WriteString("\n")

	b.WriteString(ui.DetailLabelStyle.Render("Updated:"))
	b.WriteString(ui.DetailValueStyle.Render(t.UpdatedAt.Format("2006-01-02 15:04")))
	b.WriteString("\n")

	if t.ClosedAt != nil {
		b.WriteString(ui.DetailLabelStyle.Render("Closed:"))
		b.WriteString(ui.DetailValueStyle.Render(t.ClosedAt.Format("2006-01-02 15:04")))
		b.WriteString("\n")
	}

	m.detail.SetContent(b.String())
}

func (m Model) viewForm() string {
	var b strings.Builder

	blocks, button, help := m.formViewBlocks()
	for _, block := range blocks {
		b.WriteString(block)
	}
	b.WriteString(button + "\n\n")
	b.WriteString("\n")
	b.WriteString(help)

	return b.String()
}

func (m Model) formViewBlocks() ([]string, string, string) {
	var blocks []string

	title := "New Task"
	if m.editing {
		title = "Edit Task"
	}
	blocks = append(blocks, ui.TitleStyle.Render(title)+"\n\n")

	// Title field
	titleLabel := ui.FormLabelStyle.Render("Title:")
	titleStyle := ui.FormInputStyle
	if m.formFocus == 0 {
		titleStyle = ui.FormInputFocusedStyle
	}
	titleInput := titleStyle.Width(m.width - 20).Render(m.formTitle.View())
	blocks = append(blocks, titleLabel+"\n"+titleInput+"\n\n")

	// Description field
	descLabel := ui.FormLabelStyle.Render("Description:")
	blocks = append(blocks, descLabel+"\n"+m.formDesc.View()+"\n\n")

	// Notes field
	notesLabel := ui.FormLabelStyle.Render("Notes:")
	blocks = append(blocks, notesLabel+"\n"+m.formNotes.View()+"\n\n")

	// Design field
	designLabel := ui.FormLabelStyle.Render("Design:")
	blocks = append(blocks, designLabel+"\n"+m.formDesign.View()+"\n\n")

	// Acceptance field
	acceptLabel := ui.FormLabelStyle.Render("Acceptance Criteria:")
	blocks = append(blocks, acceptLabel+"\n"+m.formAcceptance.View()+"\n\n")

	// Priority selector
	priLabel := ui.FormLabelStyle.Render("Priority:")
	priValue := ""
	for i := 0; i <= 4; i++ {
		style := ui.HelpDescStyle
		if i == m.formPriority {
			style = ui.PriorityStyle(i).Bold(true)
		}
		priValue += style.Render(fmt.Sprintf(" P%d ", i))
	}
	focusIndicator := ""
	if m.formFocus == 5 {
		focusIndicator = " <"
	}
	blocks = append(blocks, priLabel+priValue+focusIndicator+"\n\n")

	// Type selector
	typeLabel := ui.FormLabelStyle.Render("Type:")
	types := []string{"task", "bug", "feature", "epic", "chore"}
	typeValue := ""
	for _, t := range types {
		style := ui.HelpDescStyle
		if t == m.formType {
			style = ui.HelpKeyStyle
		}
		typeValue += style.Render(fmt.Sprintf(" %s ", t))
	}
	focusIndicator = ""
	if m.formFocus == 6 {
		focusIndicator = " <"
	}
	blocks = append(blocks, typeLabel+typeValue+focusIndicator+"\n\n")

	// Submit button
	buttonStyle := ui.FormButtonStyle
	if m.formFocus == 7 {
		buttonStyle = ui.FormButtonFocusedStyle
	}
	buttonText := "Submit"
	button := buttonStyle.Render(buttonText)

	help := ui.HelpBarStyle.Render("tab/shift+tab: next/prev field  ^e: edit in $EDITOR  alt+enter: focus submit  enter: newline/activate button  esc: cancel")
	return blocks, button, help
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
