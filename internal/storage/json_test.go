package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aazerra/work-today/internal/model"
	"github.com/aazerra/work-today/internal/storage"
)

func TestPathResolution(t *testing.T) {
	t.Run("defaults to XDG path when clean", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("HOME", dir)
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("WORK_TODAY_PATH", "")

		want := filepath.Join(dir, ".local", "share", "work-today", "tasks.json")
		if got := storage.Path(); got != want {
			t.Fatalf("Path()=%q, want %q", got, want)
		}
	})

	t.Run("uses XDG_DATA_HOME when set", func(t *testing.T) {
		homeDir := t.TempDir()
		xdgDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("XDG_DATA_HOME", xdgDir)
		t.Setenv("WORK_TODAY_PATH", "")

		want := filepath.Join(xdgDir, "work-today", "tasks.json")
		if got := storage.Path(); got != want {
			t.Fatalf("Path()=%q, want %q", got, want)
		}
	})

	t.Run("ignores relative XDG_DATA_HOME per XDG spec", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("XDG_DATA_HOME", "relative/path")
		t.Setenv("WORK_TODAY_PATH", "")

		want := filepath.Join(homeDir, ".local", "share", "work-today", "tasks.json")
		if got := storage.Path(); got != want {
			t.Fatalf("Path()=%q, want %q", got, want)
		}
	})

	t.Run("falls back to legacy ~/.work_today.json if it exists", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("HOME", dir)
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("WORK_TODAY_PATH", "")

		legacyFile := filepath.Join(dir, ".work_today.json")
		if err := os.WriteFile(legacyFile, []byte(`{"date":"2026-01-01","tasks":[]}`), 0644); err != nil {
			t.Fatalf("write legacy file: %v", err)
		}

		if got := storage.Path(); got != legacyFile {
			t.Fatalf("Path()=%q, want legacy %q", got, legacyFile)
		}
	})

	t.Run("prefers existing XDG file over legacy fallback", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("HOME", dir)
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("WORK_TODAY_PATH", "")

		legacyFile := filepath.Join(dir, ".work_today.json")
		if err := os.WriteFile(legacyFile, []byte(`{"date":"2026-01-01","tasks":[]}`), 0644); err != nil {
			t.Fatalf("write legacy file: %v", err)
		}

		xdgFile := filepath.Join(dir, ".local", "share", "work-today", "tasks.json")
		if err := os.MkdirAll(filepath.Dir(xdgFile), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(xdgFile, []byte(`{"date":"2026-01-01","tasks":[]}`), 0644); err != nil {
			t.Fatalf("write xdg file: %v", err)
		}

		if got := storage.Path(); got != xdgFile {
			t.Fatalf("Path()=%q, want xdg %q", got, xdgFile)
		}
	})

	t.Run("WORK_TODAY_PATH takes highest priority", func(t *testing.T) {
		dir := t.TempDir()
		customFile := filepath.Join(dir, "custom.json")
		t.Setenv("HOME", dir)
		t.Setenv("XDG_DATA_HOME", dir)
		t.Setenv("WORK_TODAY_PATH", customFile)

		if got := storage.Path(); got != customFile {
			t.Fatalf("Path()=%q, want custom %q", got, customFile)
		}
	})
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", "")

	now := time.Now().UTC()
	doc := &model.Document{
		Date: time.Now().Format("2006-01-02"),
		Tasks: []model.Task{
			{
				ID:        "t1",
				Title:     "Ship feature",
				Status:    model.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	if err := storage.Save(doc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	wantPath := filepath.Join(dir, ".local", "share", "work-today", "tasks.json")
	if storage.Path() != wantPath {
		t.Fatalf("Path()=%q, want %q", storage.Path(), wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected file at %s: %v", wantPath, err)
	}

	got, err := storage.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].Title != "Ship feature" {
		t.Fatalf("unexpected document: %+v", got)
	}
}

func TestSaveAndLoadLegacyFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", "")

	legacyFile := filepath.Join(dir, ".work_today.json")
	now := time.Now().UTC()
	initialDoc := &model.Document{
		Date: time.Now().Format("2006-01-02"),
		Tasks: []model.Task{
			{
				ID:        "legacy-1",
				Title:     "Existing legacy task",
				Status:    model.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	// Write directly to legacy path to simulate an existing legacy user installation
	if err := storage.Save(initialDoc); err != nil {
		t.Fatalf("initial save: %v", err)
	}
	// Move the file to the legacy path to emulate pre-existing legacy state
	xdgFile := filepath.Join(dir, ".local", "share", "work-today", "tasks.json")
	if err := os.Rename(xdgFile, legacyFile); err != nil {
		t.Fatalf("rename to legacy: %v", err)
	}

	if storage.Path() != legacyFile {
		t.Fatalf("Path()=%q, want legacy %q", storage.Path(), legacyFile)
	}

	got, err := storage.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].Title != "Existing legacy task" {
		t.Fatalf("unexpected doc: %+v", got)
	}

	// Update task and verify it saves back to legacy file
	got.Tasks[0].Title = "Updated legacy task"
	if err := storage.Save(got); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(xdgFile); !os.IsNotExist(err) {
		t.Fatalf("xdg file should not be created when using legacy fallback")
	}

	reloaded, err := storage.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Tasks[0].Title != "Updated legacy task" {
		t.Fatalf("unexpected title: %q", reloaded.Tasks[0].Title)
	}
}

func TestLoadCarriesForwardOpenTasks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", "")

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	doc := &model.Document{
		Date: yesterday,
		Tasks: []model.Task{
			{ID: "1", Title: "open", Status: model.StatusTodo},
			{ID: "2", Title: "done", Status: model.StatusDone},
			{ID: "3", Title: "doing", Status: model.StatusInProgress},
		},
	}
	if err := storage.Save(doc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := storage.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Date != time.Now().Format("2006-01-02") {
		t.Fatalf("date=%q, want today", got.Date)
	}
	if len(got.Tasks) != 2 {
		t.Fatalf("expected 2 carried tasks, got %d (%+v)", len(got.Tasks), got.Tasks)
	}
	for _, task := range got.Tasks {
		if task.Status == model.StatusDone {
			t.Fatalf("done task should not carry forward: %+v", task)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", "")

	got, err := storage.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Date != time.Now().Format("2006-01-02") || len(got.Tasks) != 0 {
		t.Fatalf("unexpected empty doc: %+v", got)
	}
}

func TestRolloverArchivesCompletedTasks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", "")

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	doc := &model.Document{
		Date: yesterday,
		Tasks: []model.Task{
			{ID: "1", Title: "open task", Status: model.StatusTodo},
			{ID: "2", Title: "finished task", Status: model.StatusDone},
		},
	}
	if err := storage.Save(doc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err := storage.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	archives, err := storage.LoadArchives(0)
	if err != nil {
		t.Fatalf("LoadArchives: %v", err)
	}
	if len(archives) != 1 {
		t.Fatalf("expected 1 archive file, got %d", len(archives))
	}
	if archives[0].Date != yesterday {
		t.Fatalf("expected archive date %s, got %s", yesterday, archives[0].Date)
	}
	if len(archives[0].Tasks) != 1 || archives[0].Tasks[0].Title != "finished task" {
		t.Fatalf("unexpected archived tasks: %+v", archives[0].Tasks)
	}
}

func TestClearCompleted(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", "")

	now := time.Now().Format("2006-01-02")
	doc := &model.Document{
		Date: now,
		Tasks: []model.Task{
			{ID: "1", Title: "Todo task", Status: model.StatusTodo},
			{ID: "2", Title: "Done task", Status: model.StatusDone},
		},
	}
	if err := storage.Save(doc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cleared, err := storage.ClearCompleted()
	if err != nil {
		t.Fatalf("ClearCompleted: %v", err)
	}
	if len(cleared) != 1 || cleared[0].Title != "Done task" {
		t.Fatalf("expected 1 cleared task, got %+v", cleared)
	}

	active, err := storage.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(active.Tasks) != 1 || active.Tasks[0].Title != "Todo task" {
		t.Fatalf("expected only Todo task remaining in active list, got %+v", active.Tasks)
	}

	archives, err := storage.LoadArchives(0)
	if err != nil {
		t.Fatalf("LoadArchives: %v", err)
	}
	if len(archives) != 1 || len(archives[0].Tasks) != 1 {
		t.Fatalf("expected 1 task in archive, got %+v", archives)
	}
}

func TestCalculateStreak(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", "")

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")

	// Archive yesterday and 2 days ago
	_ = storage.ArchiveTasks(yesterday, []model.Task{{ID: "y1", Title: "Done yesterday", Status: model.StatusDone}})
	_ = storage.ArchiveTasks(twoDaysAgo, []model.Task{{ID: "y2", Title: "Done 2 days ago", Status: model.StatusDone}})

	// No tasks completed today yet -> ongoing streak is 2 (from yesterday and day before)
	streak := storage.CalculateStreak()
	if streak != 2 {
		t.Fatalf("expected ongoing streak 2, got %d", streak)
	}

	// Complete a task today -> streak becomes 3
	todayDoc := &model.Document{
		Date: now.Format("2006-01-02"),
		Tasks: []model.Task{
			{ID: "t1", Title: "Done today", Status: model.StatusDone},
		},
	}
	_ = storage.Save(todayDoc)

	streak = storage.CalculateStreak()
	if streak != 3 {
		t.Fatalf("expected streak 3, got %d", streak)
	}
}
