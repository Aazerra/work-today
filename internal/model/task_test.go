package model_test

import (
	"testing"

	"github.com/alireza/work-today/internal/model"
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
