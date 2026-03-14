package app

import (
	"testing"

	"lazybeads/internal/models"
)

func TestIsTombstoneTask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		task models.Task
		want bool
	}{
		{
			name: "status tombstone",
			task: models.Task{Status: "tombstone", Type: "task"},
			want: true,
		},
		{
			name: "type tombstone",
			task: models.Task{Status: "open", Type: "tombstone"},
			want: true,
		},
		{
			name: "status casing and whitespace",
			task: models.Task{Status: "  TombStone  ", Type: "task"},
			want: true,
		},
		{
			name: "normal task",
			task: models.Task{Status: "open", Type: "feature"},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isTombstoneTask(tc.task)
			if got != tc.want {
				t.Fatalf("isTombstoneTask() = %v, want %v", got, tc.want)
			}
		})
	}
}
