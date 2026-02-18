package app

import (
	"errors"
	"testing"

	"lazybeads/internal/models"
)

func TestCommentsLoadedMsg_ErrorDoesNotMarkLoaded(t *testing.T) {
	m := New()
	updated, _ := m.Update(commentsLoadedMsg{
		issueID: "lazybeads-test",
		err:     errors.New("transient failure"),
	})
	got := updated.(Model)

	if got.commentsLoaded["lazybeads-test"] {
		t.Fatal("expected commentsLoaded to remain false after error")
	}
	if got.commentsError["lazybeads-test"] == "" {
		t.Fatal("expected error to be recorded for failed comments load")
	}
}

func TestCommentsLoadedMsg_SuccessMarksLoaded(t *testing.T) {
	m := New()
	updated, _ := m.Update(commentsLoadedMsg{
		issueID: "lazybeads-test",
		comments: []models.Comment{
			{ID: 1, IssueID: "lazybeads-test", Author: "tester", Text: "ok"},
		},
	})
	got := updated.(Model)

	if !got.commentsLoaded["lazybeads-test"] {
		t.Fatal("expected commentsLoaded to be true after successful load")
	}
	if got.commentsError["lazybeads-test"] != "" {
		t.Fatal("expected comments error to be cleared after successful load")
	}
	if len(got.commentsByIssue["lazybeads-test"]) != 1 {
		t.Fatalf("expected one stored comment, got %d", len(got.commentsByIssue["lazybeads-test"]))
	}
}
