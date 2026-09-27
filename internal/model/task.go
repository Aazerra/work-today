package model

import "time"

// Task status constants.
const (
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
)

// ValidStatuses lists statuses in cycle order: todo → in_progress → done → todo.
var ValidStatuses = []string{StatusTodo, StatusInProgress, StatusDone}

// Task is a single work item for the day.
type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Document is the on-disk shape of ~/.work_today.json.
type Document struct {
	Date  string `json:"date"` // YYYY-MM-DD local date the file belongs to
	Tasks []Task `json:"tasks"`
}

// NextStatus returns the next status in the cycle.
func NextStatus(current string) string {
	for i, s := range ValidStatuses {
		if s == current {
			return ValidStatuses[(i+1)%len(ValidStatuses)]
		}
	}
	return StatusTodo
}

// StatusLabel returns a short display label for a status.
func StatusLabel(status string) string {
	switch status {
	case StatusTodo:
		return "TODO"
	case StatusInProgress:
		return "DOING"
	case StatusDone:
		return "DONE"
	default:
		return status
	}
}

// StatusGlyph returns a checkbox-style glyph for a status.
func StatusGlyph(status string) string {
	switch status {
	case StatusTodo:
		return "[ ]"
	case StatusInProgress:
		return "[~]"
	case StatusDone:
		return "[x]"
	default:
		return "[?]"
	}
}
