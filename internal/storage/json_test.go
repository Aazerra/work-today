package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alireza/work-today/internal/model"
	"github.com/alireza/work-today/internal/storage"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

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

	wantPath := filepath.Join(dir, ".work_today.json")
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

func TestLoadCarriesForwardOpenTasks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

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

	got, err := storage.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Date != time.Now().Format("2006-01-02") || len(got.Tasks) != 0 {
		t.Fatalf("unexpected empty doc: %+v", got)
	}
}
