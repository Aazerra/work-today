package model

import (
	"fmt"
	"strings"
	"time"
)

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

// ParsePriority normalizes a priority string (e.g. "h", "high", "M", "med") to a valid priority constant.
func ParsePriority(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "h", "high", "1":
		return PriorityHigh
	case "m", "med", "medium", "2":
		return PriorityMedium
	case "l", "low", "3":
		return PriorityLow
	default:
		return PriorityNone
	}
}

// ExportMarkdown formats a document's tasks as a clean Markdown checklist.
func ExportMarkdown(doc *Document) string {
	if doc == nil {
		return ""
	}
	dateStr := doc.Date
	if parsed, err := time.Parse("2006-01-02", doc.Date); err == nil {
		dateStr = parsed.Format("Mon, 02 Jan 2006")
	}

	done, total := 0, len(doc.Tasks)
	for _, t := range doc.Tasks {
		if t.Status == StatusDone {
			done++
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Work Today (%s) · %d/%d done\n\n", dateStr, done, total))
	if len(doc.Tasks) == 0 {
		sb.WriteString("_No tasks for today._\n")
		return sb.String()
	}

	for _, t := range doc.Tasks {
		glyph := "[ ]"
		switch t.Status {
		case StatusInProgress:
			glyph = "[~]"
		case StatusDone:
			glyph = "[x]"
		}

		pBadge := ""
		if t.Priority != "" {
			pBadge = fmt.Sprintf("[%s] ", PriorityLabel(t.Priority))
		}

		sb.WriteString(fmt.Sprintf("- %s %s%s\n", glyph, pBadge, t.Title))
	}
	return sb.String()
}
