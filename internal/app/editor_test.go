package app

import (
	"strings"
	"testing"
)

func TestWrapEditorContent_RoundTripsBulkSections(t *testing.T) {
	t.Parallel()

	input := editorValues{
		Description:        "desc\nline 2",
		Notes:              "notes",
		Design:             "",
		AcceptanceCriteria: "acceptance",
	}

	rendered := wrapEditorContent(input, "lazybeads-9oc0", false)
	for _, want := range []string{
		"# Description",
		"# Notes",
		"# Design",
		"# Acceptance Criteria",
		editorSectionStart(editorFieldDescription, true),
		editorSectionStart(editorFieldNotes, true),
		editorSectionStart(editorFieldDesign, true),
		editorSectionStart(editorFieldAcceptance, true),
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered editor content to include %q", want)
		}
	}

	parsed, err := parseEditorContent(rendered)
	if err != nil {
		t.Fatalf("parseEditorContent failed: %v", err)
	}

	if parsed != input {
		t.Fatalf("parsed editor values mismatch: got %#v want %#v", parsed, input)
	}
}

func TestParseEditorCommand_SupportsEditorFlags(t *testing.T) {
	t.Parallel()

	args, err := parseEditorCommand("nvim -f")
	if err != nil {
		t.Fatalf("parseEditorCommand failed: %v", err)
	}
	if len(args) != 2 || args[0] != "nvim" || args[1] != "-f" {
		t.Fatalf("unexpected editor args: %#v", args)
	}

	quotedArgs, err := parseEditorCommand(`"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" --wait`)
	if err != nil {
		t.Fatalf("parseEditorCommand with quotes failed: %v", err)
	}
	if len(quotedArgs) != 2 || quotedArgs[0] != "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" || quotedArgs[1] != "--wait" {
		t.Fatalf("unexpected quoted editor args: %#v", quotedArgs)
	}

	emptyArgArgs, err := parseEditorCommand(`emacsclient -c -a ""`)
	if err != nil {
		t.Fatalf("parseEditorCommand with empty quoted arg failed: %v", err)
	}
	if len(emptyArgArgs) != 4 || emptyArgArgs[0] != "emacsclient" || emptyArgArgs[1] != "-c" || emptyArgArgs[2] != "-a" || emptyArgArgs[3] != "" {
		t.Fatalf("unexpected empty-arg editor args: %#v", emptyArgArgs)
	}
}

func TestWrapEditorContent_PreservesTrailingNewlines(t *testing.T) {
	t.Parallel()

	input := editorValues{AcceptanceCriteria: "line 1\n\n"}
	parsed, err := parseEditorContent(wrapEditorContent(input, "lazybeads-9oc0", false))
	if err != nil {
		t.Fatalf("parseEditorContent failed: %v", err)
	}
	if parsed.AcceptanceCriteria != input.AcceptanceCriteria {
		t.Fatalf("expected trailing newlines to round-trip, got %q want %q", parsed.AcceptanceCriteria, input.AcceptanceCriteria)
	}
}

func TestWrapEditorContent_EscapesMarkerLookingContent(t *testing.T) {
	t.Parallel()

	input := editorValues{
		Description: `<!-- lb:start notes -->
<!-- lb:end notes -->
\already escaped`,
	}

	parsed, err := parseEditorContent(wrapEditorContent(input, "lazybeads-9oc0", false))
	if err != nil {
		t.Fatalf("parseEditorContent failed: %v", err)
	}
	if parsed.Description != input.Description {
		t.Fatalf("expected marker-like content to round-trip, got %q want %q", parsed.Description, input.Description)
	}
}

func TestParseEditorContent_RejectsNestedSectionStart(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		editorSectionStart(editorFieldDescription, true),
		"description",
		editorSectionStart(editorFieldNotes, true),
		"notes",
		editorSectionEnd(editorFieldNotes),
		editorSectionStart(editorFieldDesign, true),
		editorSectionEnd(editorFieldDesign),
		editorSectionStart(editorFieldAcceptance, true),
		editorSectionEnd(editorFieldAcceptance),
	}, "\n")

	if _, err := parseEditorContent(content); err == nil {
		t.Fatal("expected parseEditorContent to reject a new section before the prior one closes")
	}
}

func TestParseEditorContent_RejectsDuplicateSection(t *testing.T) {
	t.Parallel()

	content := strings.Join([]string{
		editorSectionStart(editorFieldDescription, true),
		"description",
		editorSectionEnd(editorFieldDescription),
		editorSectionStart(editorFieldDescription, true),
		"description again",
		editorSectionEnd(editorFieldDescription),
		editorSectionStart(editorFieldNotes, true),
		editorSectionEnd(editorFieldNotes),
		editorSectionStart(editorFieldDesign, true),
		editorSectionEnd(editorFieldDesign),
		editorSectionStart(editorFieldAcceptance, true),
		editorSectionEnd(editorFieldAcceptance),
	}, "\n")

	if _, err := parseEditorContent(content); err == nil {
		t.Fatal("expected parseEditorContent to reject duplicate sections")
	}
}

func TestUpdate_EditorFinishedMsgRestoresFormFocusAndFields(t *testing.T) {
	m := New()
	m.mode = ViewForm
	m.formFocus = 6
	m.editorTargetForm = true
	m.editorReturnFocus = 6

	updated, _ := m.Update(editorFinishedMsg{values: editorValues{
		Description:        "new description",
		Notes:              "new notes",
		Design:             "new design",
		AcceptanceCriteria: "new acceptance",
	}})
	got := updated.(Model)

	if got.mode != ViewForm {
		t.Fatalf("expected form mode after editor update, got %v", got.mode)
	}
	if got.formDesc.Value() != "new description" {
		t.Fatalf("expected description to update, got %q", got.formDesc.Value())
	}
	if got.formNotes.Value() != "new notes" {
		t.Fatalf("expected notes to update, got %q", got.formNotes.Value())
	}
	if got.formDesign.Value() != "new design" {
		t.Fatalf("expected design to update, got %q", got.formDesign.Value())
	}
	if got.formAcceptance.Value() != "new acceptance" {
		t.Fatalf("expected acceptance to update, got %q", got.formAcceptance.Value())
	}
	if got.formFocus != 6 {
		t.Fatalf("expected focus 6 to be restored, got %d", got.formFocus)
	}
}
