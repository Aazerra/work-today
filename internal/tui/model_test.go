package tui_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aazerra/work-today/internal/model"
	"github.com/aazerra/work-today/internal/storage"
	"github.com/aazerra/work-today/internal/tui"
)

func setupTestModel(t *testing.T, tasks []model.Task) tui.Model {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", filepath.Join(dir, "tasks.json"))

	now := time.Now()
	doc := &model.Document{
		Date:  now.Format("2006-01-02"),
		Tasks: tasks,
	}
	if err := storage.Save(doc); err != nil {
		t.Fatalf("setup storage: %v", err)
	}

	m, err := tui.New()
	if err != nil {
		t.Fatalf("tui.New: %v", err)
	}
	return m
}

func sendKey(m tea.Model, keyStr string) tea.Model {
	var msg tea.KeyMsg
	switch keyStr {
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	default:
		runes := []rune(keyStr)
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: runes}
	}
	newM, cmd := m.Update(msg)
	if cmd != nil {
		_ = cmd()
	}
	return newM
}

func TestReorderTasks(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: "1", Title: "Task 1", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
		{ID: "2", Title: "Task 2", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
		{ID: "3", Title: "Task 3", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
	}

	m := setupTestModel(t, tasks)

	// Move first task down (J)
	m = sendKey(m, "J").(tui.Model)

	doc, err := storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Title != "Task 2" || doc.Tasks[1].Title != "Task 1" {
		t.Fatalf("expected Task 1 and Task 2 swapped, got %s, %s", doc.Tasks[0].Title, doc.Tasks[1].Title)
	}

	// Move task 1 up (K)
	m = sendKey(m, "K").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Title != "Task 1" || doc.Tasks[1].Title != "Task 2" {
		t.Fatalf("expected Task 1 back at top, got %s, %s", doc.Tasks[0].Title, doc.Tasks[1].Title)
	}

	// MoveUp at top boundary should not panic or error
	m = sendKey(m, "K").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Title != "Task 1" {
		t.Fatalf("expected Task 1 still at top, got %s", doc.Tasks[0].Title)
	}
}

func TestPriorityCycling(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: "1", Title: "Task 1", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
	}

	m := setupTestModel(t, tasks)

	// Press 'p': None -> High
	m = sendKey(m, "p").(tui.Model)
	doc, err := storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Priority != model.PriorityHigh {
		t.Fatalf("expected PriorityHigh, got %q", doc.Tasks[0].Priority)
	}

	// Press 'p': High -> Medium
	m = sendKey(m, "p").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Priority != model.PriorityMedium {
		t.Fatalf("expected PriorityMedium, got %q", doc.Tasks[0].Priority)
	}

	// Press 'p': Medium -> Low
	m = sendKey(m, "p").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Priority != model.PriorityLow {
		t.Fatalf("expected PriorityLow, got %q", doc.Tasks[0].Priority)
	}

	// Press 'p': Low -> None
	m = sendKey(m, "p").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Priority != model.PriorityNone {
		t.Fatalf("expected PriorityNone, got %q", doc.Tasks[0].Priority)
	}
}

func TestViewsRendering(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: "1", Title: "Urgent Task", Status: model.StatusTodo, Priority: model.PriorityHigh, CreatedAt: now, UpdatedAt: now},
		{ID: "2", Title: "Done Task", Status: model.StatusDone, Priority: model.PriorityLow, CreatedAt: now, UpdatedAt: now},
	}

	m := setupTestModel(t, tasks)
	view := m.View()

	if view == "" {
		t.Fatalf("expected non-empty view")
	}
}

func TestUndoAndRedo(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: "1", Title: "Task 1", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
		{ID: "2", Title: "Task 2", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
	}

	m := setupTestModel(t, tasks)

	// 1. Delete task 1 ('d' -> 'y')
	m = sendKey(m, "d").(tui.Model)
	m = sendKey(m, "y").(tui.Model)

	doc, err := storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(doc.Tasks) != 1 || doc.Tasks[0].Title != "Task 2" {
		t.Fatalf("expected only Task 2 remaining, got %+v", doc.Tasks)
	}

	// 2. Undo deletion ('u')
	m = sendKey(m, "u").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(doc.Tasks) != 2 || doc.Tasks[0].Title != "Task 1" {
		t.Fatalf("expected Task 1 restored, got %+v", doc.Tasks)
	}

	// 3. Redo deletion ('U')
	m = sendKey(m, "U").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(doc.Tasks) != 1 || doc.Tasks[0].Title != "Task 2" {
		t.Fatalf("expected Task 1 deleted again, got %+v", doc.Tasks)
	}

	// 4. Undo back to 2 tasks
	m = sendKey(m, "u").(tui.Model)

	// 5. Toggle status (Enter) -> Todo to InProgress
	m = sendKey(m, "enter").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Status != model.StatusInProgress {
		t.Fatalf("expected StatusInProgress, got %s", doc.Tasks[0].Status)
	}

	// 6. Undo status toggle
	m = sendKey(m, "u").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Status != model.StatusTodo {
		t.Fatalf("expected StatusTodo restored, got %s", doc.Tasks[0].Status)
	}
}

func typeString(m tea.Model, s string) tea.Model {
	for _, r := range s {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func TestFilterMode(t *testing.T) {
	now := time.Now()
	t1 := model.Task{ID: "1", Title: "Fix backend bug +auth @work", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now}
	t1.Normalize()
	t2 := model.Task{ID: "2", Title: "Design landing page +design @home", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now}
	t2.Normalize()
	tasks := []model.Task{t1, t2}

	m := setupTestModel(t, tasks)

	// Press '/' to enter filter mode
	m = sendKey(m, "/").(tui.Model)

	// Type "+auth" and press Enter
	m = typeString(m, "+auth").(tui.Model)
	m = sendKey(m, "enter").(tui.Model)

	view := m.View()
	if !strings.Contains(view, "Fix backend bug") {
		t.Fatalf("expected view to contain filtered task 'Fix backend bug', got:\n%s", view)
	}
	if strings.Contains(view, "Design landing page") {
		t.Fatalf("expected view NOT to contain 'Design landing page', got:\n%s", view)
	}

	// Press 'esc' to clear filter
	m = sendKey(m, "esc").(tui.Model)
	view = m.View()
	if !strings.Contains(view, "Fix backend bug") || !strings.Contains(view, "Design landing page") {
		t.Fatalf("expected both tasks to be visible after clearing filter, got:\n%s", view)
	}
}

func TestClearCompleted(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: "1", Title: "Task 1 Done", Status: model.StatusDone, CreatedAt: now, UpdatedAt: now},
		{ID: "2", Title: "Task 2 Pending", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
	}

	m := setupTestModel(t, tasks)

	// Press 'C' to clear completed
	m = sendKey(m, "C").(tui.Model)

	doc, err := storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(doc.Tasks) != 1 || doc.Tasks[0].Title != "Task 2 Pending" {
		t.Fatalf("expected only pending task remaining, got %+v", doc.Tasks)
	}

	// Verify archived
	archives, err := storage.LoadArchives(10)
	if err != nil {
		t.Fatalf("load archives: %v", err)
	}
	if len(archives) == 0 || len(archives[0].Tasks) != 1 || archives[0].Tasks[0].Title != "Task 1 Done" {
		t.Fatalf("expected Task 1 Done in archives, got %+v", archives)
	}

	// Undo clear ('u')
	m = sendKey(m, "u").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load after undo: %v", err)
	}
	if len(doc.Tasks) != 2 {
		t.Fatalf("expected 2 tasks restored on undo, got %d", len(doc.Tasks))
	}
}

func TestStreakDisplay(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: "1", Title: "Today Task", Status: model.StatusDone, CreatedAt: now, UpdatedAt: now},
	}

	m := setupTestModel(t, tasks)

	// Streak should be 1 day streak
	view := m.View()
	if !strings.Contains(view, "1 day streak") {
		t.Fatalf("expected view to contain streak indicator '1 day streak', got:\n%s", view)
	}
}

func TestTuiSubtasks(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: "1", Title: "Parent Task", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
	}

	m := setupTestModel(t, tasks)

	// 1. Add subtask 's' -> type "Write specs" -> enter
	m = sendKey(m, "s").(tui.Model)
	m = typeString(m, "Write specs").(tui.Model)
	m = sendKey(m, "enter").(tui.Model)

	doc, err := storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(doc.Tasks[0].Subtasks) != 1 || doc.Tasks[0].Subtasks[0].Title != "Write specs" {
		t.Fatalf("expected subtask 'Write specs', got %+v", doc.Tasks[0].Subtasks)
	}

	// View should contain parent progress badge (0/1) and subtask
	view := m.View()
	if !strings.Contains(view, "(0/1)") {
		t.Fatalf("expected (0/1) in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Write specs") {
		t.Fatalf("expected 'Write specs' in view, got:\n%s", view)
	}

	// 2. Add second subtask
	m = sendKey(m, "s").(tui.Model)
	m = typeString(m, "Implement code").(tui.Model)
	m = sendKey(m, "enter").(tui.Model)

	// Cursor is now on second subtask. Move up 'k' to first subtask
	m = sendKey(m, "up").(tui.Model)

	// Toggle first subtask 'enter'
	m = sendKey(m, "enter").(tui.Model)

	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load after toggle: %v", err)
	}
	if !doc.Tasks[0].Subtasks[0].Done {
		t.Fatalf("expected first subtask to be done")
	}

	// View should show (1/2)
	view = m.View()
	if !strings.Contains(view, "(1/2)") {
		t.Fatalf("expected (1/2) in view, got:\n%s", view)
	}

	// 3. Move cursor up to parent task and press 'tab' to fold
	m = sendKey(m, "up").(tui.Model)
	m = sendKey(m, "tab").(tui.Model)

	// Subtasks should now be hidden in view
	view = m.View()
	if strings.Contains(view, "Write specs") {
		t.Fatalf("expected subtasks to be folded/hidden in view, got:\n%s", view)
	}
	if !strings.Contains(view, "▸") {
		t.Fatalf("expected fold glyph ▸ in view, got:\n%s", view)
	}

	// Press 'tab' to unfold
	m = sendKey(m, "tab").(tui.Model)
	view = m.View()
	if !strings.Contains(view, "Write specs") {
		t.Fatalf("expected subtasks to be visible after unfolding, got:\n%s", view)
	}

	// 4. Test undo: undo toggle -> first subtask not done
	m = sendKey(m, "u").(tui.Model)
	doc, err = storage.Load()
	if err != nil {
		t.Fatalf("load after undo: %v", err)
	}
	if doc.Tasks[0].Subtasks[0].Done {
		t.Fatalf("expected first subtask undone to not done")
	}
}
