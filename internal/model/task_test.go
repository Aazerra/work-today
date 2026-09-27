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
