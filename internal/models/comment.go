package models

import "time"

// Comment represents a timeline comment for a beads issue.
type Comment struct {
	ID        int       `json:"id"`
	IssueID   string    `json:"issue_id"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
