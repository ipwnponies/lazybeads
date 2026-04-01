package ui

import "testing"

func TestDefaultKeyMap_ViewCommentsBinding(t *testing.T) {
	keys := DefaultKeyMap()
	help := keys.ViewComments.Help()
	if help.Key != "m" {
		t.Fatalf("expected key help 'm', got %q", help.Key)
	}
	if help.Desc != "comments" {
		t.Fatalf("expected desc 'comments', got %q", help.Desc)
	}
}

func TestDefaultKeyMap_DetailScrollBindings(t *testing.T) {
	keys := DefaultKeyMap()

	if help := keys.DetailScrollUp.Help(); help.Key != "," || help.Desc != "detail up" {
		t.Fatalf("unexpected detail up help: %#v", help)
	}
	if help := keys.DetailScrollDown.Help(); help.Key != "." || help.Desc != "detail down" {
		t.Fatalf("unexpected detail down help: %#v", help)
	}

	fullHelp := keys.FullHelp()
	seenUp := false
	seenDown := false
	for _, group := range fullHelp {
		for _, binding := range group {
			help := binding.Help()
			if help.Key == "," && help.Desc == "detail up" {
				seenUp = true
			}
			if help.Key == "." && help.Desc == "detail down" {
				seenDown = true
			}
		}
	}

	if !seenUp || !seenDown {
		t.Fatalf("expected detail scroll bindings in FullHelp, saw up=%v down=%v", seenUp, seenDown)
	}
}
