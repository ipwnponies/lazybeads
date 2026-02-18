package app

import (
	"strings"
	"testing"
	"time"

	"lazybeads/internal/models"
)

func TestRenderCommentsTimeline_States(t *testing.T) {
	if got := renderCommentsTimeline(nil, true, "", 40); got != "Loading comments..." {
		t.Fatalf("unexpected loading text: %q", got)
	}
	if got := renderCommentsTimeline(nil, false, "boom", 40); !strings.Contains(got, "Failed to load comments: boom") {
		t.Fatalf("unexpected error text: %q", got)
	}
	if got := renderCommentsTimeline(nil, false, "", 40); got != "No comments." {
		t.Fatalf("unexpected empty text: %q", got)
	}
}

func TestRenderCommentsTimeline_SortsNewestFirst(t *testing.T) {
	oldTime := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC)
	comments := []models.Comment{
		{ID: 1, Author: "A", Text: "older", CreatedAt: oldTime},
		{ID: 2, Author: "B", Text: "newer", CreatedAt: newTime},
	}

	got := renderCommentsTimeline(comments, false, "", 60)
	newPos := strings.Index(got, "newer")
	oldPos := strings.Index(got, "older")
	if newPos == -1 || oldPos == -1 {
		t.Fatalf("expected both comment texts in output, got: %q", got)
	}
	if newPos > oldPos {
		t.Fatalf("expected newer comment to appear first, got: %q", got)
	}
}
