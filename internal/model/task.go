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

// Subtask is a child checklist item under a Task.
type Subtask struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Task is a single work item for the day.
type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Contexts  []string  `json:"contexts,omitempty"`
	Subtasks  []Subtask `json:"subtasks,omitempty"`
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
		for _, st := range t.Subtasks {
			stGlyph := "[ ]"
			if st.Done {
				stGlyph = "[x]"
			}
			sb.WriteString(fmt.Sprintf("  - %s %s\n", stGlyph, st.Title))
		}
	}
	return sb.String()
}

// SubtaskProgress returns the count of completed subtasks and total subtasks.
func (t Task) SubtaskProgress() (done int, total int) {
	total = len(t.Subtasks)
	for _, st := range t.Subtasks {
		if st.Done {
			done++
		}
	}
	return done, total
}

// ExtractTagsAndContexts inspects a string and extracts all +project/#tag tags and @context contexts.
func ExtractTagsAndContexts(s string) (tags []string, contexts []string) {
	seenTags := make(map[string]bool)
	seenCtx := make(map[string]bool)

	for _, word := range strings.Fields(s) {
		clean := strings.TrimRight(word, ",.;:!?)]}\"'")
		if len(clean) > 1 {
			if clean[0] == '+' || clean[0] == '#' {
				tag := strings.ToLower(clean[1:])
				if tag != "" && !seenTags[tag] {
					seenTags[tag] = true
					tags = append(tags, tag)
				}
			} else if clean[0] == '@' {
				ctx := strings.ToLower(clean[1:])
				if ctx != "" && !seenCtx[ctx] {
					seenCtx[ctx] = true
					contexts = append(contexts, ctx)
				}
			}
		}
	}
	return tags, contexts
}

// Normalize ensures tags and contexts embedded in the title are extracted into the Task's metadata slices.
func (t *Task) Normalize() {
	titleTags, titleCtx := ExtractTagsAndContexts(t.Title)
	seenTags := make(map[string]bool)
	for _, tag := range t.Tags {
		clean := strings.ToLower(strings.TrimSpace(tag))
		if clean != "" {
			seenTags[clean] = true
		}
	}
	for _, tag := range titleTags {
		if !seenTags[tag] {
			t.Tags = append(t.Tags, tag)
			seenTags[tag] = true
		}
	}

	seenCtx := make(map[string]bool)
	for _, c := range t.Contexts {
		clean := strings.ToLower(strings.TrimSpace(c))
		if clean != "" {
			seenCtx[clean] = true
		}
	}
	for _, c := range titleCtx {
		if !seenCtx[c] {
			t.Contexts = append(t.Contexts, c)
			seenCtx[c] = true
		}
	}
}

// MatchesFilter checks whether a task matches a search or tag/context query.
func (t Task) MatchesFilter(query string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return true
	}

	titleLower := strings.ToLower(t.Title)
	if strings.Contains(titleLower, q) {
		return true
	}

	// Match tag prefix e.g. "+backend" or "backend"
	tagQuery := strings.TrimPrefix(strings.TrimPrefix(q, "+"), "#")
	for _, tag := range t.Tags {
		if strings.Contains(tag, tagQuery) {
			return true
		}
	}

	// Match context prefix e.g. "@work" or "work"
	ctxQuery := strings.TrimPrefix(q, "@")
	for _, ctx := range t.Contexts {
		if strings.Contains(ctx, ctxQuery) {
			return true
		}
	}

	// Match priority
	if t.Priority != "" && strings.Contains(strings.ToLower(t.Priority), q) {
		return true
	}

	// Match subtasks
	for _, st := range t.Subtasks {
		if strings.Contains(strings.ToLower(st.Title), q) {
			return true
		}
	}

	return false
}
