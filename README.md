# work-today (`today`)

[![Go Reference](https://pkg.go.dev/badge/github.com/aazerra/work-today.svg)](https://pkg.go.dev/github.com/aazerra/work-today)
[![Go Report Card](https://goreportcard.com/badge/github.com/aazerra/work-today)](https://goreportcard.com/report/github.com/aazerra/work-today)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

An opinionated, minimalist, keyboard-driven daily task manager for the terminal. Built with Go and the [Charm](https://charm.sh) ecosystem ([Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), [Bubbles](https://github.com/charmbracelet/bubbles)).

Focus strictly on **what needs to get done today**—no endless backlog hoarding.

```text
 work today
 Mon, 28 Sep 2026  ·  2/4 done  ·  🔥 3 day streak  ·  ~/.local/share/work-today/tasks.json
────────────────────────────────────────────────────────────────────────────────────────
  [x] DONE         Ship database migration +ops @work
› [~] DOING [HIGH] Implement OAuth2 token refresh +auth @work
  [ ] TODO  [MED]  Draft weekly team summary @home
  [ ] TODO  [LOW]  Review pull requests +core

 a add · e edit · d del · p prio · J/K move · / filter · C clear · u undo · c copy · ↵ cycle · q quit
```

---

## ✨ Features

- **Daily Focus & Automatic Archive Rollover**:
  - Focus exclusively on today's tasks.
  - When the calendar date rolls over, yesterday's completed tasks are automatically archived to dated history snapshots (`YYYY-MM-DD.json`), while open tasks (`TODO`, `DOING`) carry forward smoothly.
- **Tags & Contexts**:
  - Organize tasks naturally with inline tags like `+project`, contexts like `@work`, or `#tag`.
  - Filter tasks instantly in CLI (`today list +auth`, `today list @work`) or in the interactive TUI with <kbd>/</kbd>.
  - Syntax highlighted in the TUI: tags in soft purple, contexts in cyan/teal.
  - Add via flags (`today add -t auth -c work "Task title"`) or directly in the title.
- **Daily Archive, History & Statistics**:
  - View past completed work with `today history [days]` (default: 7 days).
  - Track productivity insights with `today stats`: active streak, best streak, total archived tasks, and top projects/contexts.
  - Clear completed tasks from today's list on demand with `today clear` or <kbd>C</kbd> in the TUI.
- **Streak Tracking**:
  - Stay motivated with consecutive-day completion streaks (displayed as `🔥 X day streak` in both CLI status and TUI header).
- **Dual Mode (Interactive TUI & Headless CLI)**:
  - Run `today` for an interactive, full-screen terminal UI.
  - Run `today add`, `today list`, `today status`, `today done`, or `today stats` directly from your shell or scripts.
- **Priority Management**:
  - Categorize tasks as **HIGH**, **MED**, or **LOW** with clear, color-coded badges.
  - Cycle priorities in the TUI with <kbd>p</kbd> or specify via CLI flag `-p high`.
- **Keyboard Task Reordering**:
  - Move tasks up/down with <kbd>K</kbd> / <kbd>J</kbd> or <kbd>Shift</kbd>+<kbd>↑</kbd> / <kbd>Shift</kbd>+<kbd>↓</kbd>.
- **Multi-Level Undo & Redo**:
  - Accidental deletion, reordering, status change, or clear? Press <kbd>u</kbd> to undo (up to 50 snapshot states).
  - Press <kbd>U</kbd> or <kbd>Ctrl+R</kbd> to redo.
- **Clipboard & Standup Export**:
  - Press <kbd>c</kbd> or <kbd>y</kbd> in the TUI to copy a Markdown standup checklist to your system clipboard.
  - Run `today export` in shell to stream Markdown to `stdout`.
- **XDG Base Directory Compliance & Legacy Fallback**:
  - Stores data in `$XDG_DATA_HOME/work-today/tasks.json` (defaults to `~/.local/share/work-today/tasks.json`).
  - Archives saved to `$XDG_DATA_HOME/work-today/archive/YYYY-MM-DD.json`.
  - Automatically falls back to legacy `~/.work_today.json` if it already exists, avoiding data loss.
  - Explicit custom file override via `$WORK_TODAY_PATH`.
- **Atomic File Operations**:
  - Uses temporary-file sync and atomic POSIX rename to eliminate corrupt JSON files on sudden terminal exits or power cuts.

---

## 📦 Installation

### From Source (Go 1.22+)

```bash
go install github.com/aazerra/work-today/cmd/today@latest
```

Ensure `$GOPATH/bin` or `$HOME/go/bin` is in your `$PATH`.

### Clone & Build Locally

```bash
git clone https://github.com/aazerra/work-today.git
cd work-today
go build -o today ./cmd/today
sudo mv today /usr/local/bin/
```

---

## 🚀 Usage

### 1. Interactive TUI

Launch the full-screen terminal interface:

```bash
today
```

#### TUI Keyboard Shortcuts

| Key | Action |
| :--- | :--- |
| <kbd>↑</kbd> / <kbd>k</kbd> | Navigate up |
| <kbd>↓</kbd> / <kbd>j</kbd> | Navigate down |
| <kbd>K</kbd> / <kbd>Shift+↑</kbd> | **Move selected task up** (reorder) |
| <kbd>J</kbd> / <kbd>Shift+↓</kbd> | **Move selected task down** (reorder) |
| <kbd>Enter</kbd> / <kbd>Space</kbd> | **Cycle status** (`[ ] TODO` → `[~] DOING` → `[x] DONE`) |
| <kbd>p</kbd> | **Cycle priority** (`None` → `[HIGH]` → `[MED]` → `[LOW]`) |
| <kbd>/</kbd> | **Filter tasks** by tag, context, or keyword (<kbd>Esc</kbd> to clear) |
| <kbd>a</kbd> | Add new task (`+project` and `@context` supported inline) |
| <kbd>e</kbd> | Edit selected task title |
| <kbd>d</kbd> | Delete selected task (prompts confirmation) |
| <kbd>C</kbd> | **Clear completed tasks** (moves to archive, supports undo) |
| <kbd>u</kbd> | **Undo** last action (up to 50 levels) |
| <kbd>U</kbd> / <kbd>Ctrl+R</kbd> | **Redo** last undone action |
| <kbd>c</kbd> / <kbd>y</kbd> | **Copy** Markdown checklist to system clipboard |
| <kbd>?</kbd> | Show shortcut help bar |
| <kbd>q</kbd> / <kbd>Ctrl+C</kbd> | Quit application |

---

### 2. Headless CLI Subcommands

Use `today` directly from your shell, terminal aliases, or scripts without opening the full UI:

#### Add Tasks
```bash
# Add a task with inline tags and contexts
today add "Fix login bug +auth @work"

# Add with flags (-p priority, -t tag, -c context)
today add -p high -t backend -c work "Implement API rate limiting"

# Multi-word arguments don't require quotes
today add Buy groceries on the way home @errands
```

#### List Today's Tasks & Filtering
```bash
# List all tasks
today list

# Filter by project tag
today list +auth

# Filter by context
today list @work

# Filter by keyword
today list bug
```
Output:
```text
Work Today (Mon, 28 Sep 2026) · 1/3 done (filter: "+auth")
 1. [ ] [HIGH] Fix login bug +auth @work
```

#### Complete a Task
```bash
today done 1
# Output: Completed: #1 Fix login bug +auth @work
```

#### Clear Completed Tasks
```bash
today clear
# Output: Cleared 1 completed task(s) to archive (~/.local/share/work-today/archive/2026-09-28.json).
```

#### View History & Archives
```bash
# View completed tasks from the last 7 days (default)
today history

# View completed tasks from the last 30 days
today history 30
```
Output:
```text
Archive History (past 7 days):
[2026-09-28] (1 task)
  1. [x] Fix login bug +auth @work
[2026-09-27] (3 tasks)
  1. [x] Review pull requests +core
  2. [x] Database backup check +ops
  3. [x] Plan weekly sprint @work
```

#### Productivity Stats & Streaks
```bash
today stats
```
Output:
```text
=== Work Today Stats ===
Active Streak:    3 days 🔥
Total Archived:   14 tasks across 5 days

Top Projects/Tags:
  +auth: 5 tasks
  +ops:  4 tasks
  +core: 3 tasks

Top Contexts:
  @work: 10 tasks
  @home: 4 tasks
```

#### One-Line Status Summary
```bash
today status
# Output: 2/3 done (🔥 3 day streak)
# (or "All done! (3/3) (🔥 3 day streak)" / "No tasks")
```

#### Export to Markdown (Daily Standup)
```bash
today export
```
Output:
```markdown
## Work Today (Mon, 28 Sep 2026) · 2/3 done

- [x] Ship database migration +ops @work
- [~] [HIGH] Implement OAuth2 token refresh +auth @work
- [ ] [MED] Draft weekly team summary @home
```

Save directly to notes:
```bash
today export > ~/notes/standup-$(date +%F).md
```

---

## ⚙️ Configuration & Storage

`work-today` requires zero configuration out of the box. Storage locations are resolved in the following priority:

1. **`$WORK_TODAY_PATH`**: If set, uses this exact file path.
   ```bash
   export WORK_TODAY_PATH="$HOME/Dropbox/todo/tasks.json"
   ```
2. **XDG Data Directory**:
   - Tasks: `$XDG_DATA_HOME/work-today/tasks.json` (defaults to `~/.local/share/work-today/tasks.json`).
   - Archive: `$XDG_DATA_HOME/work-today/archive/YYYY-MM-DD.json`.
3. **Legacy Fallback**:
   - If `$HOME/.work_today.json` exists from previous versions and no XDG file exists yet, it preserves and updates the legacy file seamlessly.
   - Archive is stored in `$HOME/.work_today_archive/`.

---

## 💡 Shell & Status Bar Integrations

### Tmux Status Bar
Add today's task progress and streak to your `.tmux.conf`:
```tmux
set -g status-right "#(today status) | %H:%M"
```

### Shell Aliases
Add to your `~/.bashrc` or `~/.zshrc`:
```bash
alias t="today"
alias ta="today add"
alias tl="today list"
alias td="today done"
alias ts="today stats"
alias th="today history"
```

### Terminal Greeting / MOTD
See today's tasks and active streak every time you open a new terminal:
```bash
# Add to ~/.zshrc or ~/.bashrc:
today list
```

---

## 🛠️ Development & Testing

Run unit tests across all packages:
```bash
go test -v ./...
```

Run test suite with race detection:
```bash
go test -race ./...
```

Build binary:
```bash
go build -o today ./cmd/today
```

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
