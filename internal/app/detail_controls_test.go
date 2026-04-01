package app

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lazybeads/internal/config"
	"lazybeads/internal/models"
)

func TestExecuteCustomCommand_UsesInjectedRunner(t *testing.T) {
	m := New()
	m.selected = &models.Task{ID: "lazybeads-123", Title: "Demo"}

	called := 0
	rendered := ""
	m.customRunner = func(command string) error {
		called++
		rendered = command
		return nil
	}

	m.executeCustomCommand(config.CustomCommand{Command: "echo {{.ID}}"})

	if called != 1 {
		t.Fatalf("expected injected runner once, got %d", called)
	}
	if rendered != "echo lazybeads-123" {
		t.Fatalf("unexpected rendered command: %q", rendered)
	}
}

func TestHandleDetailKeys_ReservedDotCommaBypassCustom(t *testing.T) {
	m := newDetailTestModel(strings.Repeat("line\n", 80))
	m.detailScrollStep = 10
	m.customCommands = []config.CustomCommand{
		{Key: ".", Context: "detail", Command: "echo dot"},
		{Key: ",", Context: "global", Command: "echo comma"},
	}

	called := 0
	m.customRunner = func(string) error {
		called++
		return nil
	}

	m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	if m.detail.YOffset != 10 {
		t.Fatalf("expected '.' to scroll detail viewport down by 10, got %d", m.detail.YOffset)
	}
	if called != 0 {
		t.Fatalf("expected reserved '.' to bypass custom command, got %d calls", called)
	}

	m.detail.SetYOffset(12)
	m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{','}})
	if m.detail.YOffset != 2 {
		t.Fatalf("expected ',' to scroll detail viewport up to offset 2, got %d", m.detail.YOffset)
	}
	if called != 0 {
		t.Fatalf("expected reserved ',' to bypass custom command, got %d calls", called)
	}
}

func TestHandleDetailKeys_ReservedDotCommaClampToViewport(t *testing.T) {
	m := newDetailTestModel(strings.Repeat("line\n", 80))
	m.detailScrollStep = 10

	maxOffset := 0
	for {
		prev := m.detail.YOffset
		m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
		if m.detail.YOffset == prev {
			maxOffset = prev
			break
		}
	}
	if maxOffset <= 0 {
		t.Fatal("expected '.' to find a scrollable viewport bottom")
	}

	m.detail.SetYOffset(maxOffset - 5)

	m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	if m.detail.YOffset != maxOffset {
		t.Fatalf("expected '.' scrolling to clamp at viewport bottom, got %d want %d", m.detail.YOffset, maxOffset)
	}

	m.detail.SetYOffset(5)
	m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{','}})
	if m.detail.YOffset != 0 {
		t.Fatalf("expected ',' to clamp back to top, got %d", m.detail.YOffset)
	}
}

func TestHandleDetailKeys_NonReservedExecutesCustom(t *testing.T) {
	m := newDetailTestModel(strings.Repeat("line\n", 40))
	m.customCommands = []config.CustomCommand{{Key: "z", Context: "detail", Command: "echo z"}}

	called := 0
	m.customRunner = func(string) error {
		called++
		return nil
	}

	m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	if called != 1 {
		t.Fatalf("expected non-reserved key to execute custom command once, got %d", called)
	}
	if m.detail.YOffset != 0 {
		t.Fatalf("expected non-reserved key not to scroll detail viewport, got offset %d", m.detail.YOffset)
	}
}

func TestDetailWheelUsesViewportPathOnce(t *testing.T) {
	msg := tea.MouseMsg{X: 0, Y: 0, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress}

	direct := newDetailTestModel(strings.Repeat("line\n", 120))
	direct.detail, _ = direct.detail.Update(msg)
	expected := direct.detail.YOffset
	if expected == 0 {
		t.Fatal("expected direct viewport update to scroll")
	}

	m := newDetailTestModel(strings.Repeat("line\n", 120))
	updated, _ := m.Update(msg)
	got := updated.(Model)

	if got.detail.YOffset != expected {
		t.Fatalf("expected detail wheel offset %d, got %d", expected, got.detail.YOffset)
	}
}

func TestBuildHelpItems_AliasAwareCollisionPreservesDetailScroll(t *testing.T) {
	keys := New().keys
	items := buildHelpItems(keys, []config.CustomCommand{
		{Key: "shift+tab", Description: "custom prev", Context: "global"},
		{Key: ".", Description: "custom detail", Context: "detail"},
	})

	if helpItemPresent(items, "h/left/shift+tab", "prev view") {
		t.Fatal("expected alias collision to hide prev view binding")
	}
	if !helpItemPresent(items, ".", "detail down") {
		t.Fatal("expected reserved detail down binding to remain visible")
	}
	if !helpItemPresent(items, ".", "custom detail") {
		t.Fatal("expected colliding custom command to remain visible")
	}
}

func TestViewDetailOverlay_ShowsDotCommaWheelHints(t *testing.T) {
	m := newDetailTestModel("detail")
	m.width = 70
	m.height = 20

	view := m.viewDetailOverlay()
	for _, want := range []string{".: down", ",: up", "wheel: scroll"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected detail overlay to contain %q, got %q", want, view)
		}
	}
}

func TestRenderStatusBar_DetailModeFilterMarkerCompact(t *testing.T) {
	m := newDetailTestModel("detail")
	m.filterQuery = "urgent"

	status := m.renderStatusBar()
	for _, want := range []string{".:", ",:", "wheel:", "/:urgent"} {
		if !strings.Contains(status, want) {
			t.Fatalf("expected detail status to contain %q, got %q", want, status)
		}
	}
	if strings.Contains(status, "results") {
		t.Fatalf("expected compact detail filter marker without results summary, got %q", status)
	}
}

func newDetailTestModel(description string) Model {
	m := newModelWithConfig(&config.Config{DetailScrollStep: 10, DetailContentColorMode: "alternate"})
	m.mode = ViewDetail
	m.width = 100
	m.height = 30
	m.selected = &models.Task{
		ID:          "lazybeads-123",
		Title:       "Task",
		Status:      "open",
		Priority:    2,
		Type:        "feature",
		Description: description,
		CreatedAt:   time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
	}
	m.updateSizes()
	m.updateDetailContent()
	return m
}

func TestNewModelWithConfig_UsesConfiguredDetailScrollStep(t *testing.T) {
	m := newModelWithConfig(&config.Config{DetailScrollStep: 7, DetailContentColorMode: "gray"})
	if m.detailScrollStep != 7 {
		t.Fatalf("expected configured detail scroll step 7, got %d", m.detailScrollStep)
	}

	defaultModel := newModelWithConfig(&config.Config{})
	if defaultModel.detailScrollStep != 10 {
		t.Fatalf("expected default detail scroll step 10, got %d", defaultModel.detailScrollStep)
	}
}

func helpItemPresent(items []helpItem, key, desc string) bool {
	for _, item := range items {
		if item.key == key && item.desc == desc {
			return true
		}
	}
	return false
}
