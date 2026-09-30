package model_test

import (
	"strings"
	"testing"

	"github.com/aazerra/work-today/internal/model"
)

func TestNextStatusCycles(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{model.StatusTodo, model.StatusInProgress},
		{model.StatusInProgress, model.StatusDone},
		{model.StatusDone, model.StatusTodo},
		{"unknown", model.StatusTodo},
	}
	for _, tc := range cases {
		if got := model.NextStatus(tc.in); got != tc.want {
			t.Fatalf("NextStatus(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNextPriorityCycles(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{model.PriorityNone, model.PriorityHigh},
		{model.PriorityHigh, model.PriorityMedium},
		{model.PriorityMedium, model.PriorityLow},
		{model.PriorityLow, model.PriorityNone},
		{"unknown", model.PriorityNone},
	}
	for _, tc := range cases {
		if got := model.NextPriority(tc.in); got != tc.want {
			t.Fatalf("NextPriority(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPriorityLabel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{model.PriorityHigh, "HIGH"},
		{model.PriorityMedium, "MED"},
		{model.PriorityLow, "LOW"},
		{model.PriorityNone, ""},
		{"unknown", ""},
	}
	for _, tc := range cases {
		if got := model.PriorityLabel(tc.in); got != tc.want {
			t.Fatalf("PriorityLabel(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParsePriority(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"high", model.PriorityHigh},
		{"H", model.PriorityHigh},
		{"1", model.PriorityHigh},
		{"med", model.PriorityMedium},
		{"medium", model.PriorityMedium},
		{"M", model.PriorityMedium},
		{"2", model.PriorityMedium},
		{"low", model.PriorityLow},
		{"l", model.PriorityLow},
		{"3", model.PriorityLow},
		{"", model.PriorityNone},
		{"none", model.PriorityNone},
		{"unknown", model.PriorityNone},
	}
	for _, tc := range cases {
		if got := model.ParsePriority(tc.in); got != tc.want {
			t.Fatalf("ParsePriority(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestExportMarkdown(t *testing.T) {
	doc := &model.Document{
		Date: "2026-09-27",
		Tasks: []model.Task{
			{Title: "Task 1", Status: model.StatusDone},
			{Title: "Task 2", Status: model.StatusInProgress, Priority: model.PriorityHigh},
			{Title: "Task 3", Status: model.StatusTodo, Priority: model.PriorityLow},
		},
	}

	out := model.ExportMarkdown(doc)
	if !strings.Contains(out, "1/3 done") {
		t.Fatalf("expected progress in header: %s", out)
	}
	if !strings.Contains(out, "- [x] Task 1") {
		t.Fatalf("expected done task: %s", out)
	}
	if !strings.Contains(out, "- [~] [HIGH] Task 2") {
		t.Fatalf("expected in progress task with high priority: %s", out)
	}
	if !strings.Contains(out, "- [ ] [LOW] Task 3") {
		t.Fatalf("expected todo task with low priority: %s", out)
	}

	emptyOut := model.ExportMarkdown(&model.Document{Date: "2026-09-27"})
	if !strings.Contains(emptyOut, "No tasks for today") {
		t.Fatalf("expected empty message: %s", emptyOut)
	}
}

func TestExtractTagsAndContexts(t *testing.T) {
	title := "Review PR #42 for +auth and +backend, then call @client and check @home."
	tags, contexts := model.ExtractTagsAndContexts(title)

	expectedTags := map[string]bool{"42": true, "auth": true, "backend": true}
	expectedCtx := map[string]bool{"client": true, "home": true}

	if len(tags) != len(expectedTags) {
		t.Fatalf("expected %d tags, got %d (%v)", len(expectedTags), len(tags), tags)
	}
	for _, tag := range tags {
		if !expectedTags[tag] {
			t.Fatalf("unexpected tag %q", tag)
		}
	}

	if len(contexts) != len(expectedCtx) {
		t.Fatalf("expected %d contexts, got %d (%v)", len(expectedCtx), len(contexts), contexts)
	}
	for _, ctx := range contexts {
		if !expectedCtx[ctx] {
			t.Fatalf("unexpected context %q", ctx)
		}
	}
}

func TestTaskNormalize(t *testing.T) {
	task := model.Task{
		Title: "Ship release +v2 @work",
		Tags:  []string{"release"},
	}
	task.Normalize()

	if len(task.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", task.Tags)
	}
	if len(task.Contexts) != 1 || task.Contexts[0] != "work" {
		t.Fatalf("expected context 'work', got %v", task.Contexts)
	}
}

func TestMatchesFilter(t *testing.T) {
	task := model.Task{
		Title:    "Write integration tests for auth",
		Priority: model.PriorityHigh,
		Tags:     []string{"backend", "api"},
		Contexts: []string{"work"},
	}

	cases := []struct {
		query string
		want  bool
	}{
		{"", true},
		{"tests", true},
		{"TESTS", true},
		{"backend", true},
		{"+backend", true},
		{"api", true},
		{"#api", true},
		{"work", true},
		{"@work", true},
		{"high", true},
		{"frontend", false},
		{"@home", false},
		{"low", false},
	}

	for _, tc := range cases {
		if got := task.MatchesFilter(tc.query); got != tc.want {
			t.Fatalf("MatchesFilter(%q)=%v, want %v", tc.query, got, tc.want)
		}
	}
}
