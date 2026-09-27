package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/alireza/work-today/internal/model"
	"github.com/alireza/work-today/internal/storage"
)

// Run executes a CLI subcommand using standard stdout and stderr.
func Run(args []string) error {
	return RunWithIO(args, os.Stdout, os.Stderr)
}

// RunWithIO executes a CLI subcommand writing to the provided writers.
func RunWithIO(args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		return errors.New("no command specified")
	}

	cmd := strings.ToLower(args[0])
	subArgs := args[1:]

	switch cmd {
	case "add", "+":
		return runAdd(subArgs, out)
	case "list", "ls":
		return runList(out)
	case "status":
		return runStatus(out)
	case "done":
		return runDone(subArgs, out)
	case "export":
		return runExport(out)
	case "help", "-h", "--help":
		printHelp(out)
		return nil
	default:
		return fmt.Errorf("unknown command %q. Run 'today help' for usage", cmd)
	}
}

func runAdd(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: today add [-p high|med|low] <title>")
	}

	priority := model.PriorityNone
	titleWords := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-p" || arg == "--priority" {
			if i+1 >= len(args) {
				return errors.New("missing priority value after -p/--priority")
			}
			i++
			priority = model.ParsePriority(args[i])
		} else if strings.HasPrefix(arg, "-p=") || strings.HasPrefix(arg, "--priority=") {
			parts := strings.SplitN(arg, "=", 2)
			priority = model.ParsePriority(parts[1])
		} else {
			titleWords = append(titleWords, arg)
		}
	}

	title := strings.TrimSpace(strings.Join(titleWords, " "))
	if title == "" {
		return errors.New("task title cannot be empty")
	}

	doc, err := storage.Load()
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	now := time.Now()
	task := model.Task{
		ID:        uuid.NewString(),
		Title:     title,
		Status:    model.StatusTodo,
		Priority:  priority,
		CreatedAt: now,
		UpdatedAt: now,
	}

	doc.Tasks = append(doc.Tasks, task)
	if err := storage.Save(doc); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	pBadge := ""
	if priority != "" {
		pBadge = fmt.Sprintf(" [%s]", model.PriorityLabel(priority))
	}
	fmt.Fprintf(out, "Added: %s%s\n", title, pBadge)
	return nil
}

func runList(out io.Writer) error {
	doc, err := storage.Load()
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	dateStr := doc.Date
	if parsed, err := time.Parse("2006-01-02", doc.Date); err == nil {
		dateStr = parsed.Format("Mon, 02 Jan 2006")
	}

	done, total := 0, len(doc.Tasks)
	for _, t := range doc.Tasks {
		if t.Status == model.StatusDone {
			done++
		}
	}

	fmt.Fprintf(out, "Work Today (%s) · %d/%d done\n", dateStr, done, total)
	if len(doc.Tasks) == 0 {
		fmt.Fprintln(out, "No tasks yet. Add one with: today add \"title\"")
		return nil
	}

	for i, t := range doc.Tasks {
		pBadge := ""
		if t.Priority != "" {
			pBadge = fmt.Sprintf("[%s] ", model.PriorityLabel(t.Priority))
		}
		fmt.Fprintf(out, "%2d. %s %s%s\n", i+1, model.StatusGlyph(t.Status), pBadge, t.Title)
	}
	return nil
}

func runStatus(out io.Writer) error {
	doc, err := storage.Load()
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	if len(doc.Tasks) == 0 {
		fmt.Fprintln(out, "No tasks")
		return nil
	}

	done := 0
	for _, t := range doc.Tasks {
		if t.Status == model.StatusDone {
			done++
		}
	}

	if done == len(doc.Tasks) {
		fmt.Fprintf(out, "All done! (%d/%d)\n", done, len(doc.Tasks))
	} else {
		fmt.Fprintf(out, "%d/%d done\n", done, len(doc.Tasks))
	}
	return nil
}

func runDone(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: today done <task-number>")
	}

	idx, err := strconv.Atoi(args[0])
	if err != nil || idx < 1 {
		return fmt.Errorf("invalid task number: %q (must be a positive integer)", args[0])
	}

	doc, err := storage.Load()
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	if idx > len(doc.Tasks) {
		return fmt.Errorf("task number %d out of range (total %d tasks)", idx, len(doc.Tasks))
	}

	task := &doc.Tasks[idx-1]
	task.Status = model.StatusDone
	task.UpdatedAt = time.Now()

	if err := storage.Save(doc); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	fmt.Fprintf(out, "Completed: #%d %s\n", idx, task.Title)
	return nil
}

func runExport(out io.Writer) error {
	doc, err := storage.Load()
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	fmt.Fprint(out, model.ExportMarkdown(doc))
	return nil
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `work-today: minimalist daily task manager

USAGE:
  today                       Launch interactive TUI
  today add [-p PRIORITY] ... Add a task for today
  today list, today ls        Display today's task list
  today status                Print one-line status summary (e.g. for tmux)
  today done <number>         Mark task by number as completed
  today export                Print today's tasks as Markdown checklist
  today help                  Display this help message

PRIORITY LEVELS:
  high, h, 1                  High priority ([HIGH])
  med, medium, m, 2           Medium priority ([MED])
  low, l, 3                   Low priority ([LOW])

TUI KEYBOARD SHORTCUTS:
  j, k, ↑, ↓                  Navigate tasks
  J, K, Shift+↑, Shift+↓      Move selected task down / up (reorder)
  p                           Cycle task priority (None → HIGH → MED → LOW)
  Enter, Space                Cycle task status (TODO → DOING → DONE)
  a                           Add task
  e                           Edit task title
  d                           Delete task (prompts confirmation)
  u                           Undo last action
  U, Ctrl+r                   Redo last undone action
  c, y                        Copy Markdown summary to clipboard
  ?                           Toggle quick help
  q, Ctrl+c                   Quit`)
}
