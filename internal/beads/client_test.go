package beads

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// These tests require bd to be installed and run in a beads-initialized directory
// Skip if not in a valid environment

func skipIfNoBeads(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(".beads"); os.IsNotExist(err) {
		// Try parent directories up to 3 levels
		for _, dir := range []string{"..", "../..", "../../.."} {
			if _, err := os.Stat(dir + "/.beads"); err == nil {
				if err := os.Chdir(dir); err == nil {
					return
				}
			}
		}
		t.Skip("No .beads directory found, skipping integration test")
	}
}

func TestClient_IsInitialized(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	if !client.IsInitialized() {
		t.Error("Expected IsInitialized to return true in beads directory")
	}
}

func TestClient_List(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	tasks, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	t.Logf("Found %d tasks", len(tasks))
	for _, task := range tasks {
		t.Logf("  - %s: %s (status=%s, priority=%d, type=%s)",
			task.ID, task.Title, task.Status, task.Priority, task.Type)
	}
}

func TestClient_ListOpen(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	tasks, err := client.ListOpen()
	if err != nil {
		t.Fatalf("ListOpen failed: %v", err)
	}

	t.Logf("Found %d open tasks", len(tasks))
	for _, task := range tasks {
		if task.Status != "open" {
			t.Errorf("Expected status 'open', got '%s' for task %s", task.Status, task.ID)
		}
	}
}

func TestClient_Ready(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	tasks, err := client.Ready()
	if err != nil {
		t.Fatalf("Ready failed: %v", err)
	}

	t.Logf("Found %d ready tasks", len(tasks))
}

func TestClient_Show(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	// First get a task ID from list
	tasks, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(tasks) == 0 {
		t.Skip("No tasks to show")
	}

	task, err := client.Show(tasks[0].ID)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if task.ID != tasks[0].ID {
		t.Errorf("Expected ID %s, got %s", tasks[0].ID, task.ID)
	}

	t.Logf("Showed task: %s - %s", task.ID, task.Title)
}

func TestClient_CreateAndDelete(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	// Create a test task
	task, err := client.Create(CreateOptions{
		Title:       "Test task from client_test.go",
		Description: "This is a test task",
		Type:        "task",
		Priority:    3,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	t.Logf("Created task: %s - %s", task.ID, task.Title)

	if task.Title != "Test task from client_test.go" {
		t.Errorf("Expected title 'Test task from client_test.go', got '%s'", task.Title)
	}
	if task.Priority != 3 {
		t.Errorf("Expected priority 3, got %d", task.Priority)
	}
	if task.Type != "task" {
		t.Errorf("Expected type 'task', got '%s'", task.Type)
	}

	// Clean up - delete the task
	err = client.Delete(task.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	t.Log("Deleted test task")
}

func TestClient_Update(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	// Create a test task
	task, err := client.Create(CreateOptions{
		Title:    "Update test task",
		Type:     "task",
		Priority: 2,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer client.Delete(task.ID)

	// Update the task
	newPriority := 1
	err = client.Update(task.ID, UpdateOptions{
		Status:   "in_progress",
		Priority: &newPriority,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify the update
	updated, err := client.Show(task.ID)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if updated.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", updated.Status)
	}
	if updated.Priority != 1 {
		t.Errorf("Expected priority 1, got %d", updated.Priority)
	}

	t.Log("Update test passed")
}

func TestClient_Close(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	// Create a test task
	task, err := client.Create(CreateOptions{
		Title:    "Close test task",
		Type:     "task",
		Priority: 3,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer client.Delete(task.ID)

	// Close the task
	err = client.Close(task.ID, "Test completed")
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify the close
	closed, err := client.Show(task.ID)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if closed.Status != "closed" {
		t.Errorf("Expected status 'closed', got '%s'", closed.Status)
	}

	t.Log("Close test passed")
}

func TestClient_Comments_EmptyList(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	task, err := client.Create(CreateOptions{
		Title:    "Comments empty test task",
		Type:     "task",
		Priority: 3,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer client.Delete(task.ID)

	comments, err := client.Comments(task.ID)
	if err != nil {
		t.Fatalf("Comments failed: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("Expected zero comments, got %d", len(comments))
	}
}

func TestClient_Comments_WithComment(t *testing.T) {
	skipIfNoBeads(t)
	client := NewClient()

	task, err := client.Create(CreateOptions{
		Title:    "Comments populated test task",
		Type:     "task",
		Priority: 3,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer client.Delete(task.ID)

	commentText := "This is a timeline comment from client tests."
	cmd := exec.Command("bd", "comments", "add", task.ID, commentText)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Adding comment failed: %v", err)
	}

	comments, err := client.Comments(task.ID)
	if err != nil {
		t.Fatalf("Comments failed: %v", err)
	}
	if len(comments) == 0 {
		t.Fatal("Expected at least one comment, got zero")
	}

	got := comments[len(comments)-1]
	if got.IssueID != task.ID {
		t.Fatalf("Expected issue ID %s, got %s", task.ID, got.IssueID)
	}
	if got.Text != commentText {
		t.Fatalf("Expected text %q, got %q", commentText, got.Text)
	}
	if got.Author == "" {
		t.Fatal("Expected non-empty author")
	}
	if got.ID == 0 {
		t.Fatal("Expected non-zero comment ID")
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("Expected non-zero created_at timestamp")
	}
}

type stubCommandResult struct {
	output string
	err    error
}

func newStubClient(t *testing.T, results map[string]stubCommandResult, calls *[]string) *Client {
	t.Helper()

	return &Client{
		commandOutput: func(args ...string) ([]byte, error) {
			t.Helper()

			key := strings.Join(args, " ")
			*calls = append(*calls, key)

			result, ok := results[key]
			if !ok {
				return nil, fmt.Errorf("unexpected command: %s", key)
			}

			if result.err != nil {
				return nil, result.err
			}

			return []byte(result.output), nil
		},
	}
}

func TestClient_List_FallsBackToLegacyCommand(t *testing.T) {
	var calls []string
	client := newStubClient(t, map[string]stubCommandResult{
		"list --json --flat --all": {err: errors.New("unknown flag: --flat")},
		"list --json --all":        {output: `[{"id":"task-1","title":"Task 1","status":"open","issue_type":"task"}]`},
	}, &calls)

	tasks, err := client.List("--all")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "task-1" {
		t.Fatalf("expected task id task-1, got %s", tasks[0].ID)
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 command calls, got %d", len(calls))
	}
	if calls[0] != "list --json --flat --all" {
		t.Fatalf("expected first call to preferred list command, got %s", calls[0])
	}
	if calls[1] != "list --json --all" {
		t.Fatalf("expected fallback legacy call, got %s", calls[1])
	}
}

func TestClient_List_ResolvesAmbiguousEntryViaShow(t *testing.T) {
	var calls []string
	client := newStubClient(t, map[string]stubCommandResult{
		"list --json --flat": {output: `[{"id":"task-2","title":"Task 2"}]`},
		"show task-2 --json": {output: `[{"id":"task-2","title":"Task 2","status":"in_progress","issue_type":"bug"}]`},
	}, &calls)

	tasks, err := client.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Status != "in_progress" {
		t.Fatalf("expected status in_progress, got %s", tasks[0].Status)
	}
	if tasks[0].Type != "bug" {
		t.Fatalf("expected type bug, got %s", tasks[0].Type)
	}

	if len(calls) != 2 {
		t.Fatalf("expected list + show calls, got %v", calls)
	}
}

func TestClient_Ready_ParsesEnvelopeAndTypeAlias(t *testing.T) {
	var calls []string
	client := newStubClient(t, map[string]stubCommandResult{
		"ready --json --plain": {
			output: `{"issues":[{"id":"task-3","title":"Task 3","status":"open","type":"feature","owner":"owner@example.com"}]}`,
		},
	}, &calls)

	tasks, err := client.Ready()
	if err != nil {
		t.Fatalf("Ready failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Type != "feature" {
		t.Fatalf("expected type feature, got %s", tasks[0].Type)
	}
	if tasks[0].Assignee != "owner@example.com" {
		t.Fatalf("expected assignee owner@example.com, got %s", tasks[0].Assignee)
	}
}

func TestClient_Update_IncludesEmptyBulkEditFields(t *testing.T) {
	var calls []string
	client := newStubClient(t, map[string]stubCommandResult{
		"update task-4 -d  --notes notes --design  --acceptance acceptance": {},
	}, &calls)

	description := ""
	notes := "notes"
	design := ""
	acceptance := "acceptance"
	err := client.Update("task-4", UpdateOptions{
		Description:        &description,
		Notes:              &notes,
		Design:             &design,
		AcceptanceCriteria: &acceptance,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 command call, got %d", len(calls))
	}
	if calls[0] != "update task-4 -d  --notes notes --design  --acceptance acceptance" {
		t.Fatalf("unexpected command call: %q", calls[0])
	}
}
