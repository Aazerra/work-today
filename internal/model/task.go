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

// Task priority constants.
const (
	PriorityNone   = ""
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// ValidPriorities lists priorities in cycle order: none → high → medium → low → none.
var ValidPriorities = []string{PriorityNone, PriorityHigh, PriorityMedium, PriorityLow}

// Task is a single work item for the day.
type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Document is the on-disk shape of tasks storage (e.g. XDG tasks.json or legacy ~/.work_today.json).
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

// NextPriority returns the next priority in the cycle: none → high → medium → low → none.
func NextPriority(current string) string {
	for i, p := range ValidPriorities {
		if p == current {
			return ValidPriorities[(i+1)%len(ValidPriorities)]
		}
	}
	return PriorityNone
}

// PriorityLabel returns a short display label for a priority.
func PriorityLabel(priority string) string {
	switch priority {
	case PriorityHigh:
		return "HIGH"
	case PriorityMedium:
		return "MED"
	case PriorityLow:
		return "LOW"
	default:
		return ""
	}
}
