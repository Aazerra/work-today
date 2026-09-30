package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aazerra/work-today/internal/model"
	"github.com/aazerra/work-today/internal/storage"
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
		return runList(subArgs, out)
	case "status":
		return runStatus(out)
	case "done":
		return runDone(subArgs, out)
	case "clear":
		return runClear(out)
	case "history", "log":
		return runHistory(subArgs, out)
	case "stats":
		return runStats(out)
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
		return errors.New("usage: today add [-p high|med|low] [-t tag] [-c context] <title>")
	}

	priority := model.PriorityNone
	var tags []string
	var contexts []string
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
		} else if arg == "-t" || arg == "--tag" {
			if i+1 >= len(args) {
				return errors.New("missing tag value after -t/--tag")
			}
			i++
			tags = append(tags, strings.TrimPrefix(strings.TrimPrefix(args[i], "+"), "#"))
		} else if strings.HasPrefix(arg, "-t=") || strings.HasPrefix(arg, "--tag=") {
			parts := strings.SplitN(arg, "=", 2)
			tags = append(tags, strings.TrimPrefix(strings.TrimPrefix(parts[1], "+"), "#"))
		} else if arg == "-c" || arg == "--context" {
			if i+1 >= len(args) {
				return errors.New("missing context value after -c/--context")
			}
			i++
			contexts = append(contexts, strings.TrimPrefix(args[i], "@"))
		} else if strings.HasPrefix(arg, "-c=") || strings.HasPrefix(arg, "--context=") {
			parts := strings.SplitN(arg, "=", 2)
			contexts = append(contexts, strings.TrimPrefix(parts[1], "@"))
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
		Tags:      tags,
		Contexts:  contexts,
		CreatedAt: now,
		UpdatedAt: now,
	}
	task.Normalize()

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

func runList(args []string, out io.Writer) error {
	doc, err := storage.Load()
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	filter := strings.TrimSpace(strings.Join(args, " "))

	dateStr := doc.Date
	if parsed, err := time.Parse("2006-01-02", doc.Date); err == nil {
		dateStr = parsed.Format("Mon, 02 Jan 2006")
	}

	done, total := 0, 0
	type matchItem struct {
		origIndex int
		task      model.Task
	}
	var matched []matchItem

	for i, t := range doc.Tasks {
		if t.Status == model.StatusDone {
			done++
		}
		total++
		if t.MatchesFilter(filter) {
			matched = append(matched, matchItem{origIndex: i + 1, task: t})
		}
	}

	if filter != "" {
		fmt.Fprintf(out, "Work Today (%s) · matching %q (%d/%d tasks)\n", dateStr, filter, len(matched), total)
	} else {
		fmt.Fprintf(out, "Work Today (%s) · %d/%d done\n", dateStr, done, total)
	}

	if len(doc.Tasks) == 0 {
		fmt.Fprintln(out, "No tasks yet. Add one with: today add \"title\"")
		return nil
	}

	if len(matched) == 0 {
		fmt.Fprintf(out, "No tasks matching %q\n", filter)
		return nil
	}

	for _, item := range matched {
		t := item.task
		pBadge := ""
		if t.Priority != "" {
			pBadge = fmt.Sprintf("[%s] ", model.PriorityLabel(t.Priority))
		}
		fmt.Fprintf(out, "%2d. %s %s%s\n", item.origIndex, model.StatusGlyph(t.Status), pBadge, t.Title)
	}
	return nil
}

func runStatus(out io.Writer) error {
	doc, err := storage.Load()
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	streak := storage.CalculateStreak()
	streakStr := ""
	if streak > 0 {
		streakStr = fmt.Sprintf(" (🔥 %d day streak)", streak)
	}

	if len(doc.Tasks) == 0 {
		if streak > 0 {
			fmt.Fprintf(out, "No tasks%s\n", streakStr)
		} else {
			fmt.Fprintln(out, "No tasks")
		}
		return nil
	}

	done := 0
	for _, t := range doc.Tasks {
		if t.Status == model.StatusDone {
			done++
		}
	}

	if done == len(doc.Tasks) {
		fmt.Fprintf(out, "All done! (%d/%d)%s\n", done, len(doc.Tasks), streakStr)
	} else {
		fmt.Fprintf(out, "%d/%d done%s\n", done, len(doc.Tasks), streakStr)
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

func runClear(out io.Writer) error {
	cleared, err := storage.ClearCompleted()
	if err != nil {
		return fmt.Errorf("clear completed: %w", err)
	}
	if len(cleared) == 0 {
		fmt.Fprintln(out, "No completed tasks to clear.")
		return nil
	}
	fmt.Fprintf(out, "Cleared %d completed task(s) to archive.\n", len(cleared))
	return nil
}

func runHistory(args []string, out io.Writer) error {
	days := 7
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "yesterday", "1":
			days = 1
		case "all", "0":
			days = 0
		default:
			if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
				days = n
			}
		}
	}

	archives, err := storage.LoadArchives(days)
	if err != nil {
		return fmt.Errorf("load archives: %w", err)
	}

	if len(archives) == 0 {
		fmt.Fprintln(out, "No archived tasks found.")
		return nil
	}

	fmt.Fprintf(out, "Completed Tasks History (last %d days with activity):\n\n", len(archives))
	for _, doc := range archives {
		dateStr := doc.Date
		if parsed, err := time.Parse("2006-01-02", doc.Date); err == nil {
			dateStr = parsed.Format("Mon, 02 Jan 2006")
		}
		fmt.Fprintf(out, "📅 %s · %d tasks completed\n", dateStr, len(doc.Tasks))
		for _, t := range doc.Tasks {
			pBadge := ""
			if t.Priority != "" {
				pBadge = fmt.Sprintf("[%s] ", model.PriorityLabel(t.Priority))
			}
			fmt.Fprintf(out, "   • %s%s\n", pBadge, t.Title)
		}
		fmt.Fprintln(out)
	}
	return nil
}

func runStats(out io.Writer) error {
	streak := storage.CalculateStreak()

	archives, _ := storage.LoadArchives(30)
	past7DaysCount := 0
	past30DaysCount := 0

	tagCounts := make(map[string]int)
	ctxCounts := make(map[string]int)

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7).Format("2006-01-02")

	for _, doc := range archives {
		past30DaysCount += len(doc.Tasks)
		if doc.Date >= sevenDaysAgo {
			past7DaysCount += len(doc.Tasks)
		}
		for _, t := range doc.Tasks {
			t.Normalize()
			for _, tag := range t.Tags {
				tagCounts["+"+tag]++
			}
			for _, ctx := range t.Contexts {
				ctxCounts["@"+ctx]++
			}
		}
	}

	doc, err := storage.Load()
	todayDone := 0
	todayTotal := 0
	if err == nil {
		todayTotal = len(doc.Tasks)
		for _, t := range doc.Tasks {
			if t.Status == model.StatusDone {
				todayDone++
				past7DaysCount++
				past30DaysCount++
			}
			t.Normalize()
			for _, tag := range t.Tags {
				tagCounts["+"+tag]++
			}
			for _, ctx := range t.Contexts {
				ctxCounts["@"+ctx]++
			}
		}
	}

	fmt.Fprintln(out, "📊 Productivity Stats:")
	if streak > 0 {
		fmt.Fprintf(out, "  Streak:            🔥 %d days in a row\n", streak)
	} else {
		fmt.Fprintln(out, "  Streak:            0 days (complete a task to start!)")
	}
	fmt.Fprintf(out, "  Today:             %d/%d completed\n", todayDone, todayTotal)
	fmt.Fprintf(out, "  Last 7 Days:       %d completed\n", past7DaysCount)
	fmt.Fprintf(out, "  Last 30 Days:      %d completed\n", past30DaysCount)

	if len(tagCounts) > 0 || len(ctxCounts) > 0 {
		fmt.Fprintf(out, "\n🏷️  Top Tags & Contexts:\n")
		printTopCounts(out, tagCounts, 5)
		printTopCounts(out, ctxCounts, 5)
	}
	return nil
}

func printTopCounts(out io.Writer, counts map[string]int, limit int) {
	type kv struct {
		key   string
		count int
	}
	var list []kv
	for k, v := range counts {
		list = append(list, kv{key: k, count: v})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].count > list[j].count
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	for _, item := range list {
		fmt.Fprintf(out, "    %-16s (%d)\n", item.key, item.count)
	}
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
  today add [-p P] [-t T] ... Add a task for today (+tag, @context supported)
  today list [filter]         Display today's task list (filtered by tag/context/text)
  today status                Print one-line status summary (with streak)
  today done <number>         Mark task by number as completed
  today clear                 Archive and clear completed tasks from today's list
  today history [days]        View completed task history across past days
  today stats                 View completion streaks and top tags/contexts
  today export                Print today's tasks as Markdown checklist
  today help                  Display this help message

PRIORITY LEVELS:
  high, h, 1                  High priority ([HIGH])
  med, medium, m, 2           Medium priority ([MED])
  low, l, 3                   Low priority ([LOW])

TAGS & CONTEXTS:
  +project                    Tag project (e.g. +backend, +frontend)
  @context                    Tag context (e.g. @work, @home, @errands)
  #tag                        General tag (e.g. #meeting, #p1)

TUI KEYBOARD SHORTCUTS:
  j, k, ↑, ↓                  Navigate tasks
  J, K, Shift+↑, Shift+↓      Move selected task down / up (reorder)
  p                           Cycle task priority (None → HIGH → MED → LOW)
  /                           Search & filter tasks by tag/context/text
  Enter, Space                Cycle task status (TODO → DOING → DONE)
  a                           Add task
  e                           Edit task title
  d                           Delete task (prompts confirmation)
  C                           Clear completed tasks to archive
  u                           Undo last action (up to 50 levels)
  U, Ctrl+r                   Redo last undone action
  c, y                        Copy Markdown summary to clipboard
  ?                           Toggle quick help
  q, Ctrl+c                   Quit`)
}
