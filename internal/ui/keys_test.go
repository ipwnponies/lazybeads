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
