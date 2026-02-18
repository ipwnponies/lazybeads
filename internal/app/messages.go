package app

import (
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lazybeads/internal/beads"
	"lazybeads/internal/models"
)

const pollInterval = 2 * time.Second
const statusFlashDuration = 1 * time.Second

// tasksLoadedMsg is sent when tasks are loaded
type tasksLoadedMsg struct {
	tasks []models.Task
	err   error
}

// taskCreatedMsg is sent when a task is created
type taskCreatedMsg struct {
	task *models.Task
	err  error
}

// taskUpdatedMsg is sent when a task is updated
type taskUpdatedMsg struct {
	err error
}

// taskClosedMsg is sent when a task is closed
type taskClosedMsg struct {
	err error
}

// taskDeletedMsg is sent when a task is deleted
type taskDeletedMsg struct {
	err error
}

// editorFinishedMsg is sent when external editor completes
type editorFinishedMsg struct {
	content string
	err     error
}

// clipboardCopiedMsg is sent when text is copied to clipboard
type clipboardCopiedMsg struct {
	text string
	err  error
}

// commentsLoadedMsg is sent when issue comments are loaded.
type commentsLoadedMsg struct {
	issueID  string
	comments []models.Comment
	err      error
}

// clearStatusMsg clears the status flash message
type clearStatusMsg struct{}

// tickMsg triggers periodic refresh
type tickMsg time.Time

// pollTick creates a command that ticks for polling
func pollTick() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// loadTasks creates a command to load all tasks
func (m Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		// Load all tasks so we can distribute them to the 3 panels
		tasks, err := m.client.List("--all")
		if err != nil {
			return tasksLoadedMsg{tasks: tasks, err: err}
		}

		tasks, err = enrichDeferredTasks(tasks, m.client)
		tasks, err = enrichBlockedTasks(tasks, m.client, err)
		tasks, err = enrichEpicTasks(tasks, m.client, err)
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

// loadComments creates a command to load comments for a single issue.
func (m Model) loadComments(issueID string) tea.Cmd {
	return func() tea.Msg {
		comments, err := m.client.Comments(issueID)
		return commentsLoadedMsg{issueID: issueID, comments: comments, err: err}
	}
}

func enrichDeferredTasks(tasks []models.Task, client *beads.Client) ([]models.Task, error) {
	deferred, err := client.List("--deferred")
	if err != nil {
		return tasks, err
	}
	if len(deferred) == 0 {
		return tasks, nil
	}

	indexByID := make(map[string]int, len(tasks))
	for i, task := range tasks {
		indexByID[task.ID] = i
	}

	var firstErr error
	for _, task := range deferred {
		// TODO: Remove this bd show fetch once `bd list --json` includes defer_until.
		fullTask, err := client.Show(task.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to load deferred task %s: %w", task.ID, err)
			}
			continue
		}

		if idx, ok := indexByID[fullTask.ID]; ok {
			tasks[idx] = *fullTask
		} else {
			tasks = append(tasks, *fullTask)
		}
	}

	return tasks, firstErr
}

func enrichBlockedTasks(tasks []models.Task, client *beads.Client, prevErr error) ([]models.Task, error) {
	blocked, err := client.Blocked()
	if err != nil {
		return tasks, errors.Join(prevErr, err)
	}
	if len(blocked) == 0 {
		return tasks, prevErr
	}

	indexByID := make(map[string]int, len(tasks))
	for i, task := range tasks {
		indexByID[task.ID] = i
	}

	for _, task := range blocked {
		if idx, ok := indexByID[task.ID]; ok {
			tasks[idx].BlockedBy = task.BlockedBy
		} else {
			tasks = append(tasks, task)
		}
	}

	return tasks, prevErr
}

func enrichEpicTasks(tasks []models.Task, client *beads.Client, prevErr error) ([]models.Task, error) {
	epicByID := make(map[string]models.Task)
	for _, task := range tasks {
		if task.Type == "epic" {
			epicByID[task.ID] = task
		}
	}
	if len(epicByID) == 0 {
		return tasks, prevErr
	}

	childToEpic := make(map[string]string)
	var firstErr error
	for epicID := range epicByID {
		children, err := client.ListDependencies(epicID, "up", "parent-child")
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to load epic children for %s: %w", epicID, err)
			}
			continue
		}
		for _, child := range children {
			if child.ID == "" {
				continue
			}
			if _, exists := childToEpic[child.ID]; !exists {
				childToEpic[child.ID] = epicID
			}
		}
	}
	if len(childToEpic) == 0 {
		return tasks, errors.Join(prevErr, firstErr)
	}

	indexByID := make(map[string]int, len(tasks))
	for i, task := range tasks {
		indexByID[task.ID] = i
	}

	for childID, epicID := range childToEpic {
		idx, ok := indexByID[childID]
		if !ok {
			continue
		}
		tasks[idx].EpicParentID = epicID
		if epic, ok := epicByID[epicID]; ok {
			tasks[idx].EpicParentTitle = epic.Title
		}
	}

	return tasks, errors.Join(prevErr, firstErr)
}
