package cli_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aazerra/work-today/internal/cli"
	"github.com/aazerra/work-today/internal/model"
	"github.com/aazerra/work-today/internal/storage"
)

func setupTestStorage(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("WORK_TODAY_PATH", filepath.Join(dir, "tasks.json"))
}

func TestCliAddAndList(t *testing.T) {
	setupTestStorage(t)

	var out, errOut bytes.Buffer

	// 1. Add simple task
	err := cli.RunWithIO([]string{"add", "First task"}, &out, &errOut)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out.String(), "Added: First task") {
		t.Fatalf("unexpected add output: %s", out.String())
	}

	// 2. Add task with priority flag
	out.Reset()
	err = cli.RunWithIO([]string{"add", "-p", "high", "Critical issue"}, &out, &errOut)
	if err != nil {
		t.Fatalf("add with priority: %v", err)
	}
	if !strings.Contains(out.String(), "[HIGH]") {
		t.Fatalf("expected [HIGH] in output: %s", out.String())
	}

	// 3. List tasks
	out.Reset()
	err = cli.RunWithIO([]string{"list"}, &out, &errOut)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	listOutput := out.String()
	if !strings.Contains(listOutput, "1. [ ] First task") {
		t.Fatalf("expected First task in list: %s", listOutput)
	}
	if !strings.Contains(listOutput, "2. [ ] [HIGH] Critical issue") {
		t.Fatalf("expected Critical issue in list: %s", listOutput)
	}
}

func TestCliStatusAndDone(t *testing.T) {
	setupTestStorage(t)

	var out, errOut bytes.Buffer

	// Initial status when empty
	err := cli.RunWithIO([]string{"status"}, &out, &errOut)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out.String(), "No tasks") {
		t.Fatalf("expected 'No tasks', got: %s", out.String())
	}

	// Add 2 tasks
	_ = cli.RunWithIO([]string{"add", "Task 1"}, &out, &errOut)
	_ = cli.RunWithIO([]string{"add", "Task 2"}, &out, &errOut)

	// Status: 0/2 done
	out.Reset()
	_ = cli.RunWithIO([]string{"status"}, &out, &errOut)
	if !strings.Contains(out.String(), "0/2 done") {
		t.Fatalf("expected 0/2 done, got: %s", out.String())
	}

	// Mark task 1 done
	out.Reset()
	err = cli.RunWithIO([]string{"done", "1"}, &out, &errOut)
	if err != nil {
		t.Fatalf("done: %v", err)
	}
	if !strings.Contains(out.String(), "Completed: #1 Task 1") {
		t.Fatalf("unexpected done output: %s", out.String())
	}

	// Status: 1/2 done
	out.Reset()
	_ = cli.RunWithIO([]string{"status"}, &out, &errOut)
	if !strings.Contains(out.String(), "1/2 done") {
		t.Fatalf("expected 1/2 done, got: %s", out.String())
	}

	// Check underlying storage
	doc, err := storage.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if doc.Tasks[0].Status != model.StatusDone {
		t.Fatalf("task 1 should be StatusDone, got %s", doc.Tasks[0].Status)
	}
}

func TestCliExport(t *testing.T) {
	setupTestStorage(t)

	var out, errOut bytes.Buffer
	_ = cli.RunWithIO([]string{"add", "-p", "high", "Important thing"}, &out, &errOut)

	out.Reset()
	err := cli.RunWithIO([]string{"export"}, &out, &errOut)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(out.String(), "- [ ] [HIGH] Important thing") {
		t.Fatalf("expected markdown checklist, got: %s", out.String())
	}
}

func TestCliTagsAndFiltering(t *testing.T) {
	setupTestStorage(t)

	var out, errOut bytes.Buffer

	// Add tasks with inline tags and contexts
	_ = cli.RunWithIO([]string{"add", "Fix backend bug +auth @work"}, &out, &errOut)
	_ = cli.RunWithIO([]string{"add", "Buy groceries @home +errands"}, &out, &errOut)
	_ = cli.RunWithIO([]string{"add", "-t", "auth", "-c", "work", "Code review"}, &out, &errOut)

	// Filter by tag +auth
	out.Reset()
	err := cli.RunWithIO([]string{"list", "+auth"}, &out, &errOut)
	if err != nil {
		t.Fatalf("list +auth: %v", err)
	}
	res := out.String()
	if !strings.Contains(res, "Fix backend bug") || !strings.Contains(res, "Code review") {
		t.Fatalf("expected auth tasks in list: %s", res)
	}
	if strings.Contains(res, "Buy groceries") {
		t.Fatalf("unexpected task in auth list: %s", res)
	}

	// Filter by context @home
	out.Reset()
	err = cli.RunWithIO([]string{"list", "@home"}, &out, &errOut)
	if err != nil {
		t.Fatalf("list @home: %v", err)
	}
	res = out.String()
	if !strings.Contains(res, "Buy groceries") {
		t.Fatalf("expected groceries in home list: %s", res)
	}
	if strings.Contains(res, "Fix backend bug") {
		t.Fatalf("unexpected task in home list: %s", res)
	}
}

func TestCliHistoryAndClear(t *testing.T) {
	setupTestStorage(t)

	var out, errOut bytes.Buffer

	_ = cli.RunWithIO([]string{"add", "Task to finish"}, &out, &errOut)
	_ = cli.RunWithIO([]string{"done", "1"}, &out, &errOut)

	// Clear completed tasks
	out.Reset()
	err := cli.RunWithIO([]string{"clear"}, &out, &errOut)
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if !strings.Contains(out.String(), "Cleared 1 completed task(s) to archive") {
		t.Fatalf("unexpected clear output: %s", out.String())
	}

	// View history
	out.Reset()
	err = cli.RunWithIO([]string{"history"}, &out, &errOut)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if !strings.Contains(out.String(), "Task to finish") {
		t.Fatalf("expected finished task in history: %s", out.String())
	}
}

func TestCliStats(t *testing.T) {
	setupTestStorage(t)

	var out, errOut bytes.Buffer

	_ = cli.RunWithIO([]string{"add", "Task +dev @office"}, &out, &errOut)
	_ = cli.RunWithIO([]string{"done", "1"}, &out, &errOut)

	out.Reset()
	err := cli.RunWithIO([]string{"stats"}, &out, &errOut)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	statsOut := out.String()
	if !strings.Contains(statsOut, "Productivity Stats") {
		t.Fatalf("expected Productivity Stats in output: %s", statsOut)
	}
	if !strings.Contains(statsOut, "+dev") || !strings.Contains(statsOut, "@office") {
		t.Fatalf("expected tags and contexts in stats output: %s", statsOut)
	}
}

func TestCliErrors(t *testing.T) {
	setupTestStorage(t)

	var out, errOut bytes.Buffer

	// Empty add
	err := cli.RunWithIO([]string{"add"}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected error for empty add")
	}

	// Done out of range
	err = cli.RunWithIO([]string{"done", "99"}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected error for done out of range")
	}

	// Unknown command
	err = cli.RunWithIO([]string{"nonexistent"}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
}
