package app

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lazybeads/internal/models"
)

func findLineContaining(view, needle string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

func TestViewStaysLoadingUntilFirstTasksLoadedMsg(t *testing.T) {
	t.Parallel()

	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	got := updated.(Model)

	if !got.initialLoadInProgress {
		t.Fatal("expected initial load gate to remain active before first tasksLoadedMsg")
	}
	view := got.View()
	if view == "Loading..." {
		t.Fatalf("expected initial layout with loading overlay before first tasksLoadedMsg, got %q", view)
	}
	if !strings.Contains(view, "Loading tasks") {
		t.Fatalf("expected loading modal before first tasksLoadedMsg, got %q", view)
	}
	if !strings.Contains(view, "Fetching tasks from beads") {
		t.Fatalf("expected loading modal body before first tasksLoadedMsg, got %q", view)
	}
	if !strings.Contains(view, "(no tasks)") {
		t.Fatalf("expected base layout behind loading modal, got %q", view)
	}
	if !strings.Contains(view, "\x1b[2m") {
		t.Fatalf("expected dimmed background while loading, got %q", view)
	}

	updated, _ = got.Update(tasksLoadedMsg{tasks: []models.Task{}})
	got = updated.(Model)

	if got.initialLoadInProgress {
		t.Fatal("expected initial load gate to clear after first tasksLoadedMsg")
	}
	view = got.View()
	if view == "Loading..." {
		t.Fatal("expected normal view after first tasksLoadedMsg")
	}
	if !strings.Contains(view, "(no tasks)") {
		t.Fatalf("expected empty-state panels after initial load, got %q", view)
	}
	if strings.Contains(view, "Loading...") {
		t.Fatalf("did not expect loading text after initial load, got %q", view)
	}
}

func TestInitialLoadSpinnerRespondsToTickMessages(t *testing.T) {
	t.Parallel()

	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	got := updated.(Model)

	before := got.initialLoadSpinner.View()
	updated, cmd := got.Update(got.initialLoadSpinner.Tick())
	got = updated.(Model)

	if !got.initialLoadInProgress {
		t.Fatal("expected spinner tick to keep initial load active")
	}
	if cmd == nil {
		t.Fatal("expected spinner tick to schedule the next animation step")
	}
	after := got.initialLoadSpinner.View()
	if before == after {
		t.Fatalf("expected spinner view to change after tick, before=%q after=%q", before, after)
	}
	if !strings.Contains(got.View(), "Fetching tasks from beads") {
		t.Fatalf("expected loading modal to remain visible after spinner tick, got %q", got.View())
	}
}

func TestTickDoesNotReenterInitialLoadGate(t *testing.T) {
	t.Parallel()

	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated, _ = updated.(Model).Update(tasksLoadedMsg{tasks: []models.Task{}})
	updated, _ = updated.(Model).Update(tickMsg{})
	got := updated.(Model)

	if got.initialLoadInProgress {
		t.Fatal("expected periodic refresh to leave initial load gate disabled")
	}
	view := got.View()
	if view == "Loading..." {
		t.Fatal("expected periodic refresh to keep normal empty-state view")
	}
	if strings.Contains(view, "Loading tasks") {
		t.Fatalf("did not expect initial-load modal during refresh, got %q", view)
	}
	if !strings.Contains(view, "(no tasks)") {
		t.Fatalf("expected empty-state panels during refresh, got %q", view)
	}
}

func TestInitialLoadOverlayStaysDimmedInNarrowTerminal(t *testing.T) {
	t.Parallel()

	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	got := updated.(Model)

	view := got.View()
	if !strings.Contains(view, "Loading tasks") {
		t.Fatalf("expected loading modal in narrow terminal, got %q", view)
	}

	titleLine := findLineContaining(view, "Loading tasks")
	if titleLine == "" {
		t.Fatalf("expected a rendered title line in %q", view)
	}

	titleIndex := strings.Index(titleLine, "Loading tasks")
	if titleIndex == -1 {
		t.Fatalf("expected loading title in line %q", titleLine)
	}
	if !strings.Contains(titleLine[:titleIndex], "\x1b[2m") {
		t.Fatalf("expected left side of overlay row to remain dimmed, got %q", titleLine)
	}
	if !strings.Contains(titleLine[titleIndex+len("Loading tasks"):], "\x1b[2m") {
		t.Fatalf("expected right side of overlay row to remain dimmed, got %q", titleLine)
	}

	updated, _ = got.Update(tasksLoadedMsg{tasks: []models.Task{}})
	got = updated.(Model)
	view = got.View()
	if strings.Contains(view, "Loading tasks") {
		t.Fatalf("did not expect loading modal after initial load in narrow terminal, got %q", view)
	}
}

func TestTasksLoadedErrorClearsInitialLoadGate(t *testing.T) {
	t.Parallel()

	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated, _ = updated.(Model).Update(tasksLoadedMsg{err: errors.New("boom")})
	got := updated.(Model)

	if got.initialLoadInProgress {
		t.Fatal("expected initial load gate to clear after load error")
	}
	view := got.View()
	if view == "Loading..." {
		t.Fatal("expected error state instead of loading view")
	}
	if !strings.Contains(view, "Error: boom") {
		t.Fatalf("expected error message in view, got %q", view)
	}
	if strings.Contains(view, "Loading tasks") {
		t.Fatalf("did not expect loading modal after error, got %q", view)
	}
}
