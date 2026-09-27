# work-today (`today`)

[![Go Reference](https://pkg.go.dev/badge/github.com/aazerra/work-today.svg)](https://pkg.go.dev/github.com/aazerra/work-today)
[![Go Report Card](https://goreportcard.com/badge/github.com/aazerra/work-today)](https://goreportcard.com/report/github.com/aazerra/work-today)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

An opinionated, minimalist, keyboard-driven daily task manager for the terminal. Built with Go and the [Charm](https://charm.sh) ecosystem ([Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), [Bubbles](https://github.com/charmbracelet/bubbles)).

Focus strictly on **what needs to get done today**—no endless backlog hoarding.

```text
 work today
 Mon, 27 Sep 2026  ·  2/4 done  ·  ~/.local/share/work-today/tasks.json
──────────────────────────────────────────────────────────────────────────
  [x] DONE         Update project documentation
› [~] DOING [HIGH] Refactor persistence engine
  [ ] TODO  [MED]  Write CLI integration tests
  [ ] TODO         Plan tomorrow's standup

 a add · e edit · d del · p prio · J/K move · u undo · c copy · ↵ cycle · q quit
 · priority: HIGH
```

---

## ✨ Features

- **Daily Focus & Automatic Rollover**:
  - Focus exclusively on today's tasks.
  - When the calendar date rolls over, yesterday's completed tasks are cleared out while open tasks (`TODO`, `DOING`) carry forward automatically.
- **Dual Mode (TUI & Headless CLI)**:
  - Run `today` for an interactive, full-screen terminal UI.
  - Run `today add "..."`, `today list`, `today status`, or `today done 1` directly from your shell or scripts without launching the TUI.
- **Priority Management**:
  - Tag tasks as **HIGH**, **MED**, or **LOW** with clear, color-coded badges.
  - Cycle priorities in the TUI with <kbd>p</kbd> or specify via CLI flag `-p high`.
- **Keyboard Task Reordering**:
  - Move tasks up/down with <kbd>K</kbd> / <kbd>J</kbd> or <kbd>Shift</kbd>+<kbd>↑</kbd> / <kbd>Shift</kbd>+<kbd>↓</kbd>.
- **Multi-Level Undo & Redo**:
  - Accidental deletion or edit? Press <kbd>u</kbd> to undo (up to 50 snapshot states).
  - Press <kbd>U</kbd> or <kbd>Ctrl+R</kbd> to redo.
- **Clipboard & Standup Export**:
  - Press <kbd>c</kbd> or <kbd>y</kbd> in the TUI to copy a Markdown standup checklist to your system clipboard.
  - Run `today export` in shell to stream Markdown to `stdout`.
- **XDG Base Directory Compliance & Legacy Fallback**:
  - Stores data in `$XDG_DATA_HOME/work-today/tasks.json` (defaults to `~/.local/share/work-today/tasks.json`).
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
| <kbd>a</kbd> | Add new task |
| <kbd>e</kbd> | Edit selected task title |
| <kbd>d</kbd> | Delete selected task (prompts confirmation) |
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
# Add a simple task
today add "Review pull requests"

# Add with priority (-p high, med, low or h, m, l)
today add -p high "Fix critical server crash"
today add -p med "Draft weekly release notes"

# Multi-word arguments don't require quotes
today add Buy groceries on the way home
```

#### List Today's Tasks
```bash
today list
# or
today ls
```
Output:
```text
Work Today (Mon, 27 Sep 2026) · 1/3 done
 1. [x] Buy groceries on the way home
 2. [ ] [HIGH] Fix critical server crash
 3. [ ] [MED] Draft weekly release notes
```

#### Complete a Task
```bash
today done 2
# Output: Completed: #2 Fix critical server crash
```

#### One-Line Status Summary
```bash
today status
# Output: 2/3 done
# (or "All done! (3/3)" / "No tasks")
```

#### Export to Markdown (Daily Standup)
```bash
today export
```
Output:
```markdown
## Work Today (Mon, 27 Sep 2026) · 2/3 done

- [x] Buy groceries on the way home
- [x] [HIGH] Fix critical server crash
- [ ] [MED] Draft weekly release notes
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
   - Uses `$XDG_DATA_HOME/work-today/tasks.json` if `$XDG_DATA_HOME` is set.
   - Otherwise defaults to `$HOME/.local/share/work-today/tasks.json`.
3. **Legacy Fallback**:
   - If `$HOME/.work_today.json` exists from previous versions and no XDG file exists yet, it preserves and updates the legacy file seamlessly.

---

## 💡 Shell & Status Bar Integrations

### Tmux Status Bar
Add today's task progress to your `.tmux.conf`:
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
```

### Terminal Greeting / MOTD
See today's tasks every time you open a new shell:
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
