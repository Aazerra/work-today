package tui_test

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alireza/work-today/internal/model"
	"github.com/alireza/work-today/internal/storage"
	"github.com/alireza/work-today/internal/tui"
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
