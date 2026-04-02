package ui

import (
	"strings"
	"testing"
)

func TestRenderOverlayKeepsBackgroundDimmedAroundOverlay(t *testing.T) {
	t.Parallel()

	background := strings.Join([]string{
		"abcdefghij",
		"abcdefghij",
		"abcdefghij",
	}, "\n")
	overlay := strings.Join([]string{
		"XYZ",
	}, "\n")

	rendered := RenderOverlay(background, overlay, 10, 3)
	lines := strings.Split(rendered, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	wantOverlayRow := Dimmed("abc") + "\x1b[0mXYZ\x1b[0m" + Dimmed("ghij")
	if lines[1] != wantOverlayRow {
		t.Fatalf("expected overlay row %q, got %q", wantOverlayRow, lines[1])
	}

	if lines[0] != Dimmed("abcdefghij") {
		t.Fatalf("expected top row to stay dimmed, got %q", lines[0])
	}
	if lines[2] != Dimmed("abcdefghij") {
		t.Fatalf("expected bottom row to stay dimmed, got %q", lines[2])
	}
}
