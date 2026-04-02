package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"lazybeads/internal/beads"
	"lazybeads/internal/config"
	"lazybeads/internal/models"
	"lazybeads/internal/ui"
)

// ViewMode represents the current view
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
	ViewComments
	ViewForm
	ViewHelp
	ViewConfirm
	ViewEditTitle
	ViewEditStatus
	ViewEditPriority
	ViewEditType
	ViewFilter
)

const (
	wideModeMinWidth   = 80
	panelDefaultMax    = 80
	panelMinWidthChars = 10
	detailMinWidth     = 10
	panelWidthStep     = 5
)

const formFieldCount = 8

var panelStatusOrder = map[string]int{
	"in_progress": 0,
	"open":        1,
	"closed":      2,
}

type editorField string

const (
	editorFieldDescription editorField = "description"
	editorFieldNotes       editorField = "notes"
	editorFieldDesign      editorField = "design"
	editorFieldAcceptance  editorField = "acceptance_criteria"
)

// PanelFocus represents which panel is focused
type PanelFocus int

const (
	FocusInProgress PanelFocus = iota
	FocusOpen
	FocusClosed
	panelCount
)

// taskItem wraps a Task for the list component
type taskItem struct {
	task models.Task
}

func (t taskItem) Title() string {
	return t.task.Title
}

func (t taskItem) Description() string {
	return t.task.ID
}

func (t taskItem) FilterValue() string {
	return t.task.Title + " " + t.task.ID
}

// Model is the main application state
type Model struct {
	client *beads.Client
	keys   ui.KeyMap
	help   help.Model

	// Data
	tasks    []models.Task
	selected *models.Task

	// UI state
	mode                  ViewMode
	focusedPanel          PanelFocus
	width                 int
	height                int
	err                   error
	panelWidth            int
	detailWidth           int
	panelAdjust           int
	initialLoadInProgress bool

	// Panels (3 vertically stacked)
	inProgressPanel PanelModel
	openPanel       PanelModel
	closedPanel     PanelModel

	// Components
	detail             viewport.Model
	commentsView       viewport.Model
	helpList           list.Model
	filterText         textinput.Model
	helpItems          []helpItem
	initialLoadSpinner spinner.Model

	// Form state
	formTitle        textinput.Model
	formDesc         textarea.Model
	formNotes        textarea.Model
	formDesign       textarea.Model
	formAcceptance   textarea.Model
	formPriority     int
	formType         string
	formFocus        int
	formSubmitBounds formBounds
	editing          bool
	editingID        string
	editorField      editorField
	editorTargetID   string
	editorTargetForm bool

	// Confirmation
	confirmMsg    string
	confirmAction func() tea.Cmd

	// Modal state for field editing
	modal ui.Modal

	// Filter state
	filterQuery      string
	searchMode       bool            // true when inline search is active
	searchInput      textinput.Model // text input for inline search in status bar
	helpFilterInput  textinput.Model
	helpFilterQuery  string
	helpFilterActive bool
	helpFilterPrev   string

	// Status message (flash notification)
	statusMsg string

	// Comments timeline state
	commentsByIssue    map[string][]models.Comment
	commentsError      map[string]string
	commentsLoaded     map[string]bool
	commentsLoading    map[string]bool
	commentsReturnMode ViewMode

	// Custom commands from config
	customCommands []config.CustomCommand
	customRunner   func(string) error

	// Detail content styling
	detailContentColorMode string
	detailScrollStep       int
}

type formBounds struct {
	X int
	Y int
	W int
	H int
}

// New creates a new application model
func New() Model {
	// Load config (ignore errors, use empty config)
	cfg, _ := config.Load()
	return newModelWithConfig(cfg)
}

func newModelWithConfig(cfg *config.Config) Model {
	// Initialize help
	h := help.New()
	h.ShowAll = false

	// Initialize 3 panels
	inProgressPanel := NewPanel("In Progress")
	inProgressPanel.SetFocus(true) // Start with in progress focused
	openPanel := NewPanel("Open")
	closedPanel := NewPanel("Closed")
	closedPanel.SetCollapsed(true) // Start collapsed since not focused

	// Initialize detail and comments viewports
	vp := viewport.New(0, 0)
	commentsVP := viewport.New(0, 0)

	// Initialize help viewport
	// Initialize filter input (legacy - can be removed)
	filter := textinput.New()
	filter.Placeholder = "Search tasks..."
	filter.CharLimit = 100

	// Initialize inline search input for status bar
	searchInput := textinput.New()
	searchInput.Prompt = ""
	searchInput.CharLimit = 100
	searchInput.Width = 30

	helpFilterInput := textinput.New()
	helpFilterInput.Prompt = ""
	helpFilterInput.CharLimit = 100
	helpFilterInput.Width = 30

	// Initialize form inputs
	formTitle := textinput.New()
	formTitle.Prompt = ""
	formTitle.Placeholder = "Enter a brief, descriptive title for this task"
	formTitle.CharLimit = 200

	formDesc := textarea.New()
	formDesc.Prompt = ""
	formDesc.Placeholder = "Add description or context (optional)"
	formDesc.CharLimit = 1000
	formDesc.ShowLineNumbers = false
	formDesc.FocusedStyle.Base = ui.FormInputFocusedStyle
	formDesc.BlurredStyle.Base = ui.FormInputStyle

	formNotes := textarea.New()
	formNotes.Prompt = ""
	formNotes.Placeholder = "Add notes (optional)"
	formNotes.CharLimit = 1000
	formNotes.ShowLineNumbers = false
	formNotes.FocusedStyle.Base = ui.FormInputFocusedStyle
	formNotes.BlurredStyle.Base = ui.FormInputStyle

	formDesign := textarea.New()
	formDesign.Prompt = ""
	formDesign.Placeholder = "Add design notes (optional)"
	formDesign.CharLimit = 1000
	formDesign.ShowLineNumbers = false
	formDesign.FocusedStyle.Base = ui.FormInputFocusedStyle
	formDesign.BlurredStyle.Base = ui.FormInputStyle

	formAcceptance := textarea.New()
	formAcceptance.Prompt = ""
	formAcceptance.Placeholder = "Add acceptance criteria (optional)"
	formAcceptance.CharLimit = 1000
	formAcceptance.ShowLineNumbers = false
	formAcceptance.FocusedStyle.Base = ui.FormInputFocusedStyle
	formAcceptance.BlurredStyle.Base = ui.FormInputStyle

	var customCmds []config.CustomCommand
	detailContentColorMode := "alternate"
	detailScrollStep := 10
	if cfg != nil {
		customCmds = cfg.CustomCommands
		detailContentColorMode = cfg.DetailContentColorMode
		if cfg.DetailScrollStep > 0 {
			detailScrollStep = cfg.DetailScrollStep
		}
	}

	// Build key map with custom commands
	keys := ui.DefaultKeyMap()
	keys.CustomCommands = buildCustomCommandBindings(customCmds)

	// Build help list
	helpItems := buildHelpItems(keys, customCmds)
	helpList := newHelpList(helpItems)
	initialLoadSpinner := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(ui.HelpKeyStyle))

	return Model{
		client:                 beads.NewClient(),
		keys:                   keys,
		help:                   h,
		mode:                   ViewList,
		initialLoadInProgress:  true,
		focusedPanel:           FocusInProgress,
		inProgressPanel:        inProgressPanel,
		openPanel:              openPanel,
		closedPanel:            closedPanel,
		detail:                 vp,
		commentsView:           commentsVP,
		helpList:               helpList,
		filterText:             filter,
		helpItems:              helpItems,
		initialLoadSpinner:     initialLoadSpinner,
		searchInput:            searchInput,
		helpFilterInput:        helpFilterInput,
		formTitle:              formTitle,
		formDesc:               formDesc,
		formNotes:              formNotes,
		formDesign:             formDesign,
		formAcceptance:         formAcceptance,
		formPriority:           2,
		formType:               "feature",
		commentsByIssue:        make(map[string][]models.Comment),
		commentsError:          make(map[string]string),
		commentsLoaded:         make(map[string]bool),
		commentsLoading:        make(map[string]bool),
		customCommands:         customCmds,
		customRunner:           defaultCustomCommandRunner,
		detailContentColorMode: detailContentColorMode,
		detailScrollStep:       detailScrollStep,
	}
}

// buildCustomCommandBindings creates key bindings from custom commands
func buildCustomCommandBindings(cmds []config.CustomCommand) []key.Binding {
	var bindings []key.Binding
	for _, cmd := range cmds {
		bindings = append(bindings, key.NewBinding(
			key.WithKeys(cmd.Key),
			key.WithHelp(cmd.Key, cmd.Description),
		))
	}
	return bindings
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadTasks(), pollTick(), m.initialLoadSpinner.Tick)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()

	case tea.KeyMsg:
		// Global key handling - intercept before components
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			// Only quit from list view
			if m.mode == ViewList {
				return m, tea.Quit
			}
		case "esc":
			// If in search mode, exit search mode and clear filter
			if m.searchMode {
				m.searchMode = false
				m.searchInput.Blur()
				m.filterQuery = ""
				m.searchInput.SetValue("")
				m.distributeTasks()
				return m, nil
			}
			if m.mode == ViewHelp && m.helpFilterActive {
				cmd := m.handleHelpKeys(msg)
				return m, cmd
			}
			if m.mode == ViewHelp && !m.helpFilterActive {
				m.clearHelpFilter()
			}
			if m.mode == ViewComments {
				m.mode = m.commentsReturnMode
				return m, nil
			}
			// Escape goes back to list, never quits
			if m.mode != ViewList {
				m.mode = ViewList
				return m, nil
			}
			// In list mode, clear filter if active
			if m.filterQuery != "" {
				m.filterQuery = ""
				m.distributeTasks()
				return m, nil
			}
			return m, nil
		}

		prevMode := m.mode
		prevSearchMode := m.searchMode
		cmd := m.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// If mode changed or search mode was just activated, don't pass key to child components
		if m.mode != prevMode || (m.searchMode && !prevSearchMode) {
			return m, tea.Batch(cmds...)
		}

	case tea.MouseMsg:
		if m.mode == ViewForm {
			if cmd := m.handleFormMouse(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case tasksLoadedMsg:
		m.initialLoadInProgress = false
		if msg.err != nil {
			m.err = msg.err
		}
		if msg.tasks != nil {
			m.tasks = msg.tasks
			applyBlockingDepth(m.tasks)
			m.distributeTasks()
		}

	case taskCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.mode = ViewList
			cmds = append(cmds, m.loadTasks())
		}

	case taskUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		cmds = append(cmds, m.loadTasks())

	case taskClosedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.mode = ViewList
		cmds = append(cmds, m.loadTasks())

	case taskDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.mode = ViewList
		cmds = append(cmds, m.loadTasks())

	case editorFinishedMsg:
		targetID := m.editorTargetID
		field := m.editorField
		targetForm := m.editorTargetForm
		m.editorTargetID = ""
		m.editorField = ""
		m.editorTargetForm = false

		if msg.err != nil {
			m.err = msg.err
			if targetForm {
				m.mode = ViewForm
			} else {
				m.mode = ViewList
			}
			break
		}

		if targetForm {
			m.applyEditorContentToForm(field, msg.content)
			m.mode = ViewForm
			break
		}

		if targetID != "" {
			return m, func() tea.Msg {
				opts := beads.UpdateOptions{}
				switch field {
				case editorFieldDescription:
					opts.Description = msg.content
				case editorFieldNotes:
					opts.Notes = msg.content
				case editorFieldDesign:
					opts.Design = msg.content
				case editorFieldAcceptance:
					opts.AcceptanceCriteria = msg.content
				}
				err := m.client.Update(targetID, opts)
				return taskUpdatedMsg{err: err}
			}
		}

		m.mode = ViewList

	case tickMsg:
		// Periodic refresh - reload tasks and schedule next tick
		cmds = append(cmds, m.loadTasks(), pollTick())

	case clipboardCopiedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = "Copied!"
			cmds = append(cmds, tea.Tick(statusFlashDuration, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			}))
		}

	case clearStatusMsg:
		m.statusMsg = ""

	case commentsLoadedMsg:
		m.commentsLoading[msg.issueID] = false
		if msg.err != nil {
			m.commentsLoaded[msg.issueID] = false
			m.commentsError[msg.issueID] = msg.err.Error()
		} else {
			m.commentsLoaded[msg.issueID] = true
			m.commentsByIssue[msg.issueID] = msg.comments
			delete(m.commentsError, msg.issueID)
		}
		if m.mode == ViewComments && m.selected != nil && m.selected.ID == msg.issueID {
			m.commentsView.GotoTop()
			m.updateCommentsContent()
		}
	}

	if m.initialLoadInProgress {
		var spinnerCmd tea.Cmd
		m.initialLoadSpinner, spinnerCmd = m.initialLoadSpinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	// Update child components
	switch m.mode {
	case ViewList:
		// If in search mode, update the search input
		if m.searchMode {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			cmds = append(cmds, cmd)
			// Update filter query in real-time
			m.filterQuery = m.searchInput.Value()
			m.distributeTasks()
		} else {
			// Update the focused panel
			var cmd tea.Cmd
			switch m.focusedPanel {
			case FocusInProgress:
				m.inProgressPanel, cmd = m.inProgressPanel.Update(msg)
			case FocusOpen:
				m.openPanel, cmd = m.openPanel.Update(msg)
			case FocusClosed:
				m.closedPanel, cmd = m.closedPanel.Update(msg)
			}
			cmds = append(cmds, cmd)
		}
		// Sync selected item with detail panel
		m.selected = m.getSelectedTask()
	case ViewDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
	case ViewComments:
		var cmd tea.Cmd
		m.commentsView, cmd = m.commentsView.Update(msg)
		cmds = append(cmds, cmd)
	case ViewForm:
		cmds = append(cmds, m.updateForm(msg))
	case ViewEditTitle:
		// Update text input in modal
		var cmd tea.Cmd
		m.modal.Input, cmd = m.modal.Input.Update(msg)
		cmds = append(cmds, cmd)
	case ViewFilter:
		// Update text input in modal for filter
		var cmd tea.Cmd
		m.modal.Input, cmd = m.modal.Input.Update(msg)
		cmds = append(cmds, cmd)
	case ViewHelp:
		// Avoid list handling for key messages; handled in handleHelpKeys
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			if m.helpFilterActive {
				var inputCmd tea.Cmd
				m.helpFilterInput, inputCmd = m.helpFilterInput.Update(msg)
				cmds = append(cmds, inputCmd)
			}
			var cmd tea.Cmd
			m.helpList, cmd = m.helpList.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateSizes() {
	// Reserve space for help bar (1 line) + margins
	contentHeight := m.height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Determine how many panels are visible
	visiblePanels := m.getVisiblePanels()
	numPanels := len(visiblePanels)
	if numPanels == 0 {
		numPanels = 1 // Shouldn't happen, but avoid division by zero
	}

	// Account for newlines between panels when joined vertically
	// JoinVertical adds (numPanels - 1) newlines between panels
	joinedHeight := contentHeight - (numPanels - 1)
	if joinedHeight < numPanels {
		joinedHeight = numPanels // Minimum 1 line per panel
	}

	commentsOverlayWidth := m.width - 4
	if commentsOverlayWidth < 1 {
		commentsOverlayWidth = 1
	}
	commentsOverlayHeight := m.height - 6
	if commentsOverlayHeight < 1 {
		commentsOverlayHeight = 1
	}
	commentsInnerWidth := commentsOverlayWidth - ui.OverlayStyle.GetHorizontalFrameSize()
	if commentsInnerWidth < 1 {
		commentsInnerWidth = 1
	}
	commentsInnerHeight := commentsOverlayHeight - ui.OverlayStyle.GetVerticalFrameSize()
	if commentsInnerHeight < 1 {
		commentsInnerHeight = 1
	}

	// Wide mode: panels on left, detail on right
	var panelWidth int
	if m.width >= wideModeMinWidth {
		panelWidth, m.detailWidth = m.wideLayoutWidths()
		m.panelWidth = panelWidth
		m.detail.Width = maxInt(m.detailWidth-4, 1)
		m.detail.Height = contentHeight - 2
	} else {
		// Narrow mode: full width panels stacked
		panelWidth = m.width - 2
		m.panelWidth = panelWidth
		m.detailWidth = 0
		m.detail.Width = m.width - 4
		m.detail.Height = contentHeight - 2
	}
	m.commentsView.Width = commentsInnerWidth
	m.commentsView.Height = commentsInnerHeight

	// Check if Closed panel is collapsed (only when not focused)
	closedCollapsed := m.closedPanel.IsCollapsed()
	collapsedHeight := 3 // Collapsed panel takes 3 lines (top border + 1 content + bottom border)

	// Calculate available height for expanded panels
	availableHeight := joinedHeight
	numExpandedPanels := numPanels
	if closedCollapsed {
		availableHeight -= collapsedHeight
		numExpandedPanels--
	}
	if numExpandedPanels < 1 {
		numExpandedPanels = 1
	}

	// Calculate panel heights - distribute evenly with remainder going to first panels
	panelHeight := availableHeight / numExpandedPanels
	remainder := availableHeight % numExpandedPanels
	if panelHeight < 4 {
		panelHeight = 4
	}

	// Distribute heights to visible panels
	expandedPanelIndex := 0
	for _, panel := range visiblePanels {
		switch panel {
		case FocusInProgress:
			h := panelHeight
			if expandedPanelIndex < remainder {
				h++
			}
			m.inProgressPanel.SetSize(panelWidth, h)
			expandedPanelIndex++
		case FocusOpen:
			h := panelHeight
			if expandedPanelIndex < remainder {
				h++
			}
			m.openPanel.SetSize(panelWidth, h)
			expandedPanelIndex++
		case FocusClosed:
			if closedCollapsed {
				m.closedPanel.SetSize(panelWidth, collapsedHeight)
			} else {
				h := panelHeight
				if expandedPanelIndex < remainder {
					h++
				}
				m.closedPanel.SetSize(panelWidth, h)
				expandedPanelIndex++
			}
		}
	}

	// Set size 0 for hidden panels (In Progress when empty)
	if !m.isInProgressVisible() {
		m.inProgressPanel.SetSize(panelWidth, 0)
	}

	// Update form input widths for placeholder text display
	formWidth := m.width - 24 // Account for padding and borders
	if formWidth < 20 {
		formWidth = 20
	}
	m.formTitle.Width = formWidth
	m.formDesc.SetWidth(formWidth)
	m.formNotes.SetWidth(formWidth)
	m.formDesign.SetWidth(formWidth)
	m.formAcceptance.SetWidth(formWidth)
	m.updateFormTextAreaHeights()

	// Update help list size
	// Help view: title (2 lines) + content + help bar (1 line)
	helpWidth, helpHeight := helpModalSize(m.width, m.height)
	listHeight := helpHeight - 3
	if listHeight < 1 {
		listHeight = 1
	}
	m.helpList.SetSize(helpWidth-2, listHeight)
	helpInputWidth := helpWidth - 10
	if helpInputWidth < 10 {
		helpInputWidth = 10
	}
	m.helpFilterInput.Width = helpInputWidth
}

func (m *Model) wideLayoutWidths() (panelWidth int, detailWidth int) {
	defaultPanel := m.width / 2
	if defaultPanel > panelDefaultMax {
		defaultPanel = panelDefaultMax
	}
	if defaultPanel < panelMinWidthChars {
		defaultPanel = panelMinWidthChars
	}

	panelWidth = defaultPanel + m.panelAdjust
	if panelWidth < panelMinWidthChars {
		panelWidth = panelMinWidthChars
	}
	maxPanel := m.width - detailMinWidth
	if maxPanel < panelMinWidthChars {
		maxPanel = panelMinWidthChars
	}
	if panelWidth > maxPanel {
		panelWidth = maxPanel
	}

	detailWidth = m.width - panelWidth
	if detailWidth < detailMinWidth {
		detailWidth = detailMinWidth
		panelWidth = m.width - detailWidth
	}

	m.panelAdjust = panelWidth - defaultPanel
	return panelWidth, detailWidth
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *Model) distributeTasks() {
	var inProgress, open, closed []models.Task
	unknownStatusCounts := make(map[string]int)
	unknownStatusIDs := make(map[string][]string)
	filterLower := strings.ToLower(m.filterQuery)

	filteredTasks := make([]models.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		if isTombstoneTask(task) {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}
	stabilizeTaskOrder(filteredTasks)

	for _, t := range filteredTasks {
		// Apply filter if set
		if filterLower != "" {
			titleLower := strings.ToLower(t.Title)
			idLower := strings.ToLower(t.ID)
			if !strings.Contains(titleLower, filterLower) && !strings.Contains(idLower, filterLower) {
				continue
			}
		}
		switch t.Status {
		case "in_progress":
			inProgress = append(inProgress, t)
		case "open":
			open = append(open, t)
		case "closed":
			closed = append(closed, t)
		default:
			statusKey := t.Status
			if strings.TrimSpace(statusKey) == "" {
				statusKey = "<empty>"
			}
			unknownStatusCounts[statusKey]++
			unknownStatusIDs[statusKey] = append(unknownStatusIDs[statusKey], t.ID)
			open = append(open, t)
		}
	}

	// Sort closed tasks by ClosedAt (most recently closed first)
	sort.Slice(closed, func(i, j int) bool {
		// Tasks with ClosedAt come before those without
		if closed[i].ClosedAt == nil && closed[j].ClosedAt == nil {
			return taskOrderLess(closed[i], closed[j])
		}
		if closed[i].ClosedAt == nil {
			return false
		}
		if closed[j].ClosedAt == nil {
			return true
		}
		// Most recently closed first (descending order)
		if closed[i].ClosedAt.Equal(*closed[j].ClosedAt) {
			return taskOrderLess(closed[i], closed[j])
		}
		return closed[i].ClosedAt.After(*closed[j].ClosedAt)
	})

	inProgress = orderTasksByBlockingTree(inProgress)
	open = orderTasksByBlockingTree(open)
	inProgress = groupTasksByEpic(inProgress, m.tasks)
	open = groupTasksByEpic(open, m.tasks)
	closed = groupTasksByEpic(closed, m.tasks)

	if len(unknownStatusCounts) > 0 {
		if err := buildUnknownStatusPlacementError(unknownStatusCounts, unknownStatusIDs); err != nil {
			if m.err == nil {
				m.err = err
			}
		}
	}

	m.inProgressPanel.SetTasks(inProgress)
	m.openPanel.SetTasks(open)
	m.closedPanel.SetTasks(closed)

	// If In Progress panel disappears while focused, move focus to Open panel
	if m.focusedPanel == FocusInProgress && len(inProgress) == 0 {
		m.inProgressPanel.SetFocus(false)
		m.focusedPanel = FocusOpen
		m.openPanel.SetFocus(true)
		m.selected = m.getSelectedTask()
	}

	// Recalculate sizes since panel visibility may have changed
	m.updateSizes()
}

func buildUnknownStatusPlacementError(counts map[string]int, idsByStatus map[string][]string) error {
	if len(counts) == 0 {
		return nil
	}

	statuses := make([]string, 0, len(counts))
	for status := range counts {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	parts := make([]string, 0, len(statuses))
	for _, status := range statuses {
		ids := append([]string(nil), idsByStatus[status]...)
		sort.Strings(ids)
		parts = append(parts, fmt.Sprintf("%s (%d): %s", status, counts[status], strings.Join(ids, ", ")))
	}

	return fmt.Errorf("tasks with unsupported status shown in Open panel: %s", strings.Join(parts, " | "))
}

func isTombstoneTask(task models.Task) bool {
	status := strings.TrimSpace(strings.ToLower(task.Status))
	taskType := strings.TrimSpace(strings.ToLower(task.Type))
	return status == "tombstone" || taskType == "tombstone"
}

func stabilizeTaskOrder(tasks []models.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		return taskOrderLess(tasks[i], tasks[j])
	})
}

func taskOrderLess(a, b models.Task) bool {
	aStatusRank, aKnown := panelStatusOrder[a.Status]
	bStatusRank, bKnown := panelStatusOrder[b.Status]
	if !aKnown {
		aStatusRank = len(panelStatusOrder)
	}
	if !bKnown {
		bStatusRank = len(panelStatusOrder)
	}
	if aStatusRank != bStatusRank {
		return aStatusRank < bStatusRank
	}

	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	if !a.UpdatedAt.Equal(b.UpdatedAt) {
		return a.UpdatedAt.After(b.UpdatedAt)
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	return a.ID < b.ID
}

func (m *Model) getSelectedTask() *models.Task {
	switch m.focusedPanel {
	case FocusInProgress:
		if task := m.inProgressPanel.SelectedTask(); task != nil {
			return task
		}
	case FocusOpen:
		if task := m.openPanel.SelectedTask(); task != nil {
			return task
		}
	case FocusClosed:
		if task := m.closedPanel.SelectedTask(); task != nil {
			return task
		}
	}
	return m.selected
}

// isInProgressVisible returns true if the In Progress panel should be shown
func (m *Model) isInProgressVisible() bool {
	return m.inProgressPanel.TaskCount() > 0
}

// getVisiblePanels returns the list of currently visible panel focus values
func (m *Model) getVisiblePanels() []PanelFocus {
	var panels []PanelFocus
	if m.isInProgressVisible() {
		panels = append(panels, FocusInProgress)
	}
	panels = append(panels, FocusOpen)
	panels = append(panels, FocusClosed)
	return panels
}

func (m *Model) cyclePanelFocus(direction int) {
	visiblePanels := m.getVisiblePanels()
	if len(visiblePanels) == 0 {
		return
	}

	// Track if we're leaving the Closed panel
	wasClosedFocused := m.focusedPanel == FocusClosed

	// Clear focus from current panel
	switch m.focusedPanel {
	case FocusInProgress:
		m.inProgressPanel.SetFocus(false)
	case FocusOpen:
		m.openPanel.SetFocus(false)
	case FocusClosed:
		m.closedPanel.SetFocus(false)
	}

	// Find current panel index in visible panels
	currentIdx := -1
	for i, p := range visiblePanels {
		if p == m.focusedPanel {
			currentIdx = i
			break
		}
	}

	// If current panel is not visible (e.g., In Progress disappeared), start from first visible
	if currentIdx == -1 {
		currentIdx = 0
	}

	// Cycle to next visible panel
	newIdx := (currentIdx + direction + len(visiblePanels)) % len(visiblePanels)
	m.focusedPanel = visiblePanels[newIdx]

	// Set focus on new panel
	switch m.focusedPanel {
	case FocusInProgress:
		m.inProgressPanel.SetFocus(true)
	case FocusOpen:
		m.openPanel.SetFocus(true)
	case FocusClosed:
		m.closedPanel.SetFocus(true)
	}

	// Handle Closed panel collapse/expand
	nowClosedFocused := m.focusedPanel == FocusClosed
	if wasClosedFocused && !nowClosedFocused {
		// Leaving Closed panel - collapse it
		m.closedPanel.SetCollapsed(true)
		m.updateSizes()
	} else if !wasClosedFocused && nowClosedFocused {
		// Entering Closed panel - expand it
		m.closedPanel.SetCollapsed(false)
		m.updateSizes()
	}

	// Update selected task for detail panel
	m.selected = m.getSelectedTask()
}
