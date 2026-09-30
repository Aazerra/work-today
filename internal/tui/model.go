package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/aazerra/work-today/internal/model"
	"github.com/aazerra/work-today/internal/storage"
)

// Mode describes which interaction layer is active.
type Mode int

const (
	ModeList Mode = iota
	ModeAdd
	ModeEdit
	ModeConfirmDelete
	ModeFilter
)

type errMsg struct{ err error }
type savedMsg struct{}

// Model is the Bubble Tea application state.
type Model struct {
	doc       *model.Document
	undoStack [][]model.Task
	redoStack [][]model.Task
	cursor    int
	mode      Mode
	input     textinput.Model
	filter    string
	status    string
	streak    int
	width     int
	height    int
	quitting  bool
	keys      keyMap
}

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	MoveUp    key.Binding
	MoveDown  key.Binding
	Add       key.Binding
	Edit      key.Binding
	Delete    key.Binding
	Toggle    key.Binding
	Priority  key.Binding
	Filter    key.Binding
	ClearDone key.Binding
	Undo      key.Binding
	Redo      key.Binding
	Copy      key.Binding
	Quit      key.Binding
	Help      key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		MoveUp:    key.NewBinding(key.WithKeys("K", "shift+up"), key.WithHelp("K", "move up")),
		MoveDown:  key.NewBinding(key.WithKeys("J", "shift+down"), key.WithHelp("J", "move down")),
		Add:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Edit:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Toggle:    key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("↵/space", "cycle status")),
		Priority:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "priority")),
		Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		ClearDone: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "clear completed")),
		Undo:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undo")),
		Redo:      key.NewBinding(key.WithKeys("U", "ctrl+r"), key.WithHelp("U", "redo")),
		Copy:      key.NewBinding(key.WithKeys("c", "y"), key.WithHelp("c/y", "copy markdown")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

func (m Model) visibleIndices() []int {
	if m.filter == "" {
		indices := make([]int, len(m.doc.Tasks))
		for i := range m.doc.Tasks {
			indices[i] = i
		}
		return indices
	}
	var indices []int
	for i, t := range m.doc.Tasks {
		if t.MatchesFilter(m.filter) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m Model) withUndo() Model {
	snapshot := make([]model.Task, len(m.doc.Tasks))
	copy(snapshot, m.doc.Tasks)
	m.undoStack = append(m.undoStack, snapshot)
	if len(m.undoStack) > 50 {
		m.undoStack = m.undoStack[1:]
	}
	m.redoStack = nil
	return m
}

// New creates a Model loaded from local storage.
func New() (Model, error) {
	doc, err := storage.Load()
	if err != nil {
		return Model{}, err
	}

	ti := textinput.New()
	ti.Placeholder = "What needs doing?"
	ti.CharLimit = 200
	ti.Width = 60
	ti.Prompt = "› "

	m := Model{
		doc:    doc,
		cursor: 0,
		mode:   ModeList,
		input:  ti,
		keys:   defaultKeys(),
		streak: storage.CalculateStreak(),
	}
	if len(doc.Tasks) > 0 {
		m.cursor = 0
	}
	return m, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = max(20, msg.Width-10)
		return m, nil

	case errMsg:
		m.status = fmt.Sprintf("error: %v", msg.err)
		return m, nil

	case savedMsg:
		m.status = "saved"
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case ModeAdd, ModeEdit:
			return m.updateInput(msg)
		case ModeConfirmDelete:
			return m.updateConfirm(msg)
		case ModeFilter:
			return m.updateFilter(msg)
		default:
			return m.updateList(msg)
		}
	}

	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.visibleIndices()

	switch {
	case msg.Type == tea.KeyEsc && m.filter != "":
		m.filter = ""
		m.status = "filter cleared"
		if m.cursor >= len(m.doc.Tasks) && m.cursor > 0 {
			m.cursor = len(m.doc.Tasks) - 1
		}
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(visible)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Filter):
		m.mode = ModeFilter
		m.input.SetValue(m.filter)
		m.input.Placeholder = "Filter by tag/context/text (esc to clear)"
		m.input.CursorEnd()
		m.input.Focus()
		m.status = ""
		return m, textinput.Blink

	case key.Matches(msg, m.keys.ClearDone):
		hasDone := false
		for _, t := range m.doc.Tasks {
			if t.Status == model.StatusDone {
				hasDone = true
				break
			}
		}
		if !hasDone {
			m.status = "no completed tasks to clear"
			return m, nil
		}
		m = m.withUndo()
		cleared, err := storage.ClearCompleted()
		if err != nil {
			m.status = fmt.Sprintf("clear error: %v", err)
			return m, nil
		}
		doc, err := storage.Load()
		if err != nil {
			m.status = fmt.Sprintf("reload error: %v", err)
			return m, nil
		}
		m.doc = doc
		if m.cursor >= len(m.doc.Tasks) && m.cursor > 0 {
			m.cursor = len(m.doc.Tasks) - 1
		}
		m.status = fmt.Sprintf("cleared %d completed task(s) to archive", len(cleared))
		m.streak = storage.CalculateStreak()
		return m, nil

	case key.Matches(msg, m.keys.Add):
		m.mode = ModeAdd
		m.input.SetValue("")
		m.input.Placeholder = "New task title (+tag, @context supported)"
		m.input.Focus()
		m.status = ""
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Edit):
		if len(visible) == 0 {
			m.status = "nothing to edit"
			return m, nil
		}
		realIdx := visible[m.cursor]
		m.mode = ModeEdit
		m.input.SetValue(m.doc.Tasks[realIdx].Title)
		m.input.Placeholder = "Edit task title"
		m.input.CursorEnd()
		m.input.Focus()
		m.status = ""
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Delete):
		if len(visible) == 0 {
			m.status = "nothing to delete"
			return m, nil
		}
		m.mode = ModeConfirmDelete
		m.status = ""
		return m, nil

	case key.Matches(msg, m.keys.Toggle):
		if len(visible) == 0 {
			return m, nil
		}
		realIdx := visible[m.cursor]
		m = m.withUndo()
		t := &m.doc.Tasks[realIdx]
		t.Status = model.NextStatus(t.Status)
		t.UpdatedAt = time.Now()
		m.streak = storage.CalculateStreak()
		return m, m.persist()

	case key.Matches(msg, m.keys.Priority):
		if len(visible) == 0 {
			m.status = "nothing to prioritize"
			return m, nil
		}
		realIdx := visible[m.cursor]
		m = m.withUndo()
		t := &m.doc.Tasks[realIdx]
		t.Priority = model.NextPriority(t.Priority)
		t.UpdatedAt = time.Now()
		if t.Priority == model.PriorityNone {
			m.status = "priority cleared"
		} else {
			m.status = fmt.Sprintf("priority: %s", model.PriorityLabel(t.Priority))
		}
		return m, m.persist()

	case key.Matches(msg, m.keys.MoveUp):
		if len(visible) == 0 {
			return m, nil
		}
		if m.cursor > 0 {
			realIdx := visible[m.cursor]
			prevIdx := visible[m.cursor-1]
			m = m.withUndo()
			m.doc.Tasks[realIdx], m.doc.Tasks[prevIdx] = m.doc.Tasks[prevIdx], m.doc.Tasks[realIdx]
			m.cursor--
			m.status = "task moved up"
			return m, m.persist()
		}
		m.status = "already at top"
		return m, nil

	case key.Matches(msg, m.keys.MoveDown):
		if len(visible) == 0 {
			return m, nil
		}
		if m.cursor < len(visible)-1 {
			realIdx := visible[m.cursor]
			nextIdx := visible[m.cursor+1]
			m = m.withUndo()
			m.doc.Tasks[realIdx], m.doc.Tasks[nextIdx] = m.doc.Tasks[nextIdx], m.doc.Tasks[realIdx]
			m.cursor++
			m.status = "task moved down"
			return m, m.persist()
		}
		m.status = "already at bottom"
		return m, nil

	case key.Matches(msg, m.keys.Undo):
		if len(m.undoStack) == 0 {
			m.status = "nothing to undo"
			return m, nil
		}
		current := make([]model.Task, len(m.doc.Tasks))
		copy(current, m.doc.Tasks)
		m.redoStack = append(m.redoStack, current)

		last := m.undoStack[len(m.undoStack)-1]
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
		m.doc.Tasks = last

		newVis := m.visibleIndices()
		if m.cursor >= len(newVis) && m.cursor > 0 {
			m.cursor = len(newVis) - 1
		}
		m.streak = storage.CalculateStreak()
		m.status = "action undone"
		return m, m.persist()

	case key.Matches(msg, m.keys.Redo):
		if len(m.redoStack) == 0 {
			m.status = "nothing to redo"
			return m, nil
		}
		current := make([]model.Task, len(m.doc.Tasks))
		copy(current, m.doc.Tasks)
		m.undoStack = append(m.undoStack, current)

		next := m.redoStack[len(m.redoStack)-1]
		m.redoStack = m.redoStack[:len(m.redoStack)-1]
		m.doc.Tasks = next

		newVis := m.visibleIndices()
		if m.cursor >= len(newVis) && m.cursor > 0 {
			m.cursor = len(newVis) - 1
		}
		m.streak = storage.CalculateStreak()
		m.status = "action redone"
		return m, m.persist()

	case key.Matches(msg, m.keys.Copy):
		if len(m.doc.Tasks) == 0 {
			m.status = "nothing to copy"
			return m, nil
		}
		md := model.ExportMarkdown(m.doc)
		if err := clipboard.WriteAll(md); err != nil {
			m.status = "clipboard copy failed"
			return m, nil
		}
		m.status = "copied markdown to clipboard"
		return m, nil

	case key.Matches(msg, m.keys.Help):
		m.status = "a add · e edit · d del · p prio · J/K move · / filter · C clear · u undo · c copy · q quit"
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = ModeList
		m.input.Blur()
		m.filter = ""
		m.status = "filter cleared"
		m.cursor = 0
		return m, nil

	case tea.KeyEnter:
		m.mode = ModeList
		m.input.Blur()
		m.filter = strings.TrimSpace(m.input.Value())
		m.cursor = 0
		if m.filter == "" {
			m.status = "filter cleared"
		} else {
			m.status = fmt.Sprintf("filtered by %q", m.filter)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filter = strings.TrimSpace(m.input.Value())
	m.cursor = 0
	return m, cmd
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = ModeList
		m.input.Blur()
		m.status = "cancelled"
		return m, nil

	case tea.KeyEnter:
		title := strings.TrimSpace(m.input.Value())
		if title == "" {
			m.status = "title cannot be empty"
			return m, nil
		}

		m = m.withUndo()
		now := time.Now()
		if m.mode == ModeAdd {
			task := model.Task{
				ID:        uuid.NewString(),
				Title:     title,
				Status:    model.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			}
			task.Normalize()
			m.doc.Tasks = append(m.doc.Tasks, task)
			m.cursor = len(m.doc.Tasks) - 1
			m.status = "task added"
		} else if m.mode == ModeEdit && len(m.doc.Tasks) > 0 {
			visible := m.visibleIndices()
			realIdx := m.cursor
			if len(visible) > 0 && m.cursor < len(visible) {
				realIdx = visible[m.cursor]
			}
			m.doc.Tasks[realIdx].Title = title
			m.doc.Tasks[realIdx].UpdatedAt = now
			m.doc.Tasks[realIdx].Normalize()
			m.status = "task updated"
		}

		m.mode = ModeList
		m.input.Blur()
		return m, m.persist()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		visible := m.visibleIndices()
		if len(visible) > 0 && m.cursor < len(visible) {
			realIdx := visible[m.cursor]
			m = m.withUndo()
			m.doc.Tasks = append(m.doc.Tasks[:realIdx], m.doc.Tasks[realIdx+1:]...)
			newVis := m.visibleIndices()
			if m.cursor >= len(newVis) && m.cursor > 0 {
				m.cursor--
			}
			m.status = "task deleted"
		}
		m.mode = ModeList
		return m, m.persist()
	case "n", "esc":
		m.mode = ModeList
		m.status = "delete cancelled"
		return m, nil
	}
	return m, nil
}

func (m Model) persist() tea.Cmd {
	doc := *m.doc
	tasks := make([]model.Task, len(m.doc.Tasks))
	copy(tasks, m.doc.Tasks)
	doc.Tasks = tasks

	return func() tea.Msg {
		if err := storage.Save(&doc); err != nil {
			return errMsg{err}
		}
		return savedMsg{}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
