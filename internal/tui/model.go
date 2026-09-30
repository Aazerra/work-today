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
	ModeAddSubtask
	ModeEditSubtask
	ModeConfirmDelete
	ModeConfirmDeleteSubtask
	ModeFilter
)

type errMsg struct{ err error }
type savedMsg struct{}

// ItemRef identifies an item currently visible in the tree (parent Task or child Subtask).
type ItemRef struct {
	TaskIndex    int
	SubtaskIndex int // -1 for parent task, >= 0 for subtask
}

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
	collapsed map[string]bool
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
	AddSub    key.Binding
	Collapse  key.Binding
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
		AddSub:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "add subtask")),
		Collapse:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "fold/unfold")),
		Edit:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Toggle:    key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("↵/space", "toggle")),
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

func cloneTasks(tasks []model.Task) []model.Task {
	res := make([]model.Task, len(tasks))
	for i, t := range tasks {
		res[i] = t
		if len(t.Tags) > 0 {
			res[i].Tags = make([]string, len(t.Tags))
			copy(res[i].Tags, t.Tags)
		}
		if len(t.Contexts) > 0 {
			res[i].Contexts = make([]string, len(t.Contexts))
			copy(res[i].Contexts, t.Contexts)
		}
		if len(t.Subtasks) > 0 {
			res[i].Subtasks = make([]model.Subtask, len(t.Subtasks))
			copy(res[i].Subtasks, t.Subtasks)
		}
	}
	return res
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

func (m Model) visibleItems() []ItemRef {
	var items []ItemRef
	for i, t := range m.doc.Tasks {
		if m.filter != "" && !t.MatchesFilter(m.filter) {
			continue
		}
		items = append(items, ItemRef{TaskIndex: i, SubtaskIndex: -1})
		if !m.collapsed[t.ID] {
			for j, st := range t.Subtasks {
				if m.filter == "" || strings.Contains(strings.ToLower(st.Title), strings.ToLower(m.filter)) || strings.Contains(strings.ToLower(t.Title), strings.ToLower(m.filter)) {
					items = append(items, ItemRef{TaskIndex: i, SubtaskIndex: j})
				}
			}
		}
	}
	return items
}

func (m Model) withUndo() Model {
	snapshot := cloneTasks(m.doc.Tasks)
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
		doc:       doc,
		cursor:    0,
		mode:      ModeList,
		input:     ti,
		collapsed: make(map[string]bool),
		keys:      defaultKeys(),
		streak:    storage.CalculateStreak(),
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
		case ModeAdd, ModeEdit, ModeAddSubtask, ModeEditSubtask:
			return m.updateInput(msg)
		case ModeConfirmDelete, ModeConfirmDeleteSubtask:
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
	items := m.visibleItems()

	switch {
	case msg.Type == tea.KeyEsc && m.filter != "":
		m.filter = ""
		m.status = "filter cleared"
		newItems := m.visibleItems()
		if m.cursor >= len(newItems) && m.cursor > 0 {
			m.cursor = len(newItems) - 1
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
		if m.cursor < len(items)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Collapse):
		if len(items) == 0 {
			return m, nil
		}
		item := items[m.cursor]
		parent := &m.doc.Tasks[item.TaskIndex]
		if len(parent.Subtasks) == 0 {
			m.status = "no subtasks to fold"
			return m, nil
		}
		isCollapsed := m.collapsed[parent.ID]
		m.collapsed[parent.ID] = !isCollapsed
		if isCollapsed {
			m.status = fmt.Sprintf("unfolded %q", parent.Title)
		} else {
			m.status = fmt.Sprintf("folded %q", parent.Title)
		}
		// place cursor on the parent task
		newItems := m.visibleItems()
		for idx, it := range newItems {
			if it.TaskIndex == item.TaskIndex && it.SubtaskIndex == -1 {
				m.cursor = idx
				break
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.AddSub):
		if len(items) == 0 {
			m.status = "no task to add subtask to"
			return m, nil
		}
		item := items[m.cursor]
		parent := m.doc.Tasks[item.TaskIndex]
		m.mode = ModeAddSubtask
		m.input.SetValue("")
		m.input.Placeholder = fmt.Sprintf("Subtask for %q", parent.Title)
		m.input.Focus()
		m.status = ""
		return m, textinput.Blink

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
		newItems := m.visibleItems()
		if m.cursor >= len(newItems) && m.cursor > 0 {
			m.cursor = len(newItems) - 1
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
		if len(items) == 0 {
			m.status = "nothing to edit"
			return m, nil
		}
		item := items[m.cursor]
		if item.SubtaskIndex == -1 {
			m.mode = ModeEdit
			m.input.SetValue(m.doc.Tasks[item.TaskIndex].Title)
			m.input.Placeholder = "Edit task title"
		} else {
			m.mode = ModeEditSubtask
			m.input.SetValue(m.doc.Tasks[item.TaskIndex].Subtasks[item.SubtaskIndex].Title)
			m.input.Placeholder = "Edit subtask title"
		}
		m.input.CursorEnd()
		m.input.Focus()
		m.status = ""
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Delete):
		if len(items) == 0 {
			m.status = "nothing to delete"
			return m, nil
		}
		item := items[m.cursor]
		if item.SubtaskIndex == -1 {
			m.mode = ModeConfirmDelete
		} else {
			m.mode = ModeConfirmDeleteSubtask
		}
		m.status = ""
		return m, nil

	case key.Matches(msg, m.keys.Toggle):
		if len(items) == 0 {
			return m, nil
		}
		item := items[m.cursor]
		m = m.withUndo()
		parent := &m.doc.Tasks[item.TaskIndex]
		if item.SubtaskIndex == -1 {
			parent.Status = model.NextStatus(parent.Status)
			parent.UpdatedAt = time.Now()
		} else {
			st := &parent.Subtasks[item.SubtaskIndex]
			st.Done = !st.Done
			st.UpdatedAt = time.Now()
			parent.UpdatedAt = time.Now()
		}
		m.streak = storage.CalculateStreak()
		return m, m.persist()

	case key.Matches(msg, m.keys.Priority):
		if len(items) == 0 {
			m.status = "nothing to prioritize"
			return m, nil
		}
		item := items[m.cursor]
		m = m.withUndo()
		parent := &m.doc.Tasks[item.TaskIndex]
		parent.Priority = model.NextPriority(parent.Priority)
		parent.UpdatedAt = time.Now()
		if parent.Priority == model.PriorityNone {
			m.status = "priority cleared"
		} else {
			m.status = fmt.Sprintf("priority: %s", model.PriorityLabel(parent.Priority))
		}
		return m, m.persist()

	case key.Matches(msg, m.keys.MoveUp):
		if len(items) == 0 {
			return m, nil
		}
		item := items[m.cursor]
		if item.SubtaskIndex == -1 {
			if item.TaskIndex > 0 {
				m = m.withUndo()
				prevTaskIdx := item.TaskIndex - 1
				m.doc.Tasks[item.TaskIndex], m.doc.Tasks[prevTaskIdx] = m.doc.Tasks[prevTaskIdx], m.doc.Tasks[item.TaskIndex]
				newItems := m.visibleItems()
				for idx, it := range newItems {
					if it.TaskIndex == prevTaskIdx && it.SubtaskIndex == -1 {
						m.cursor = idx
						break
					}
				}
				m.status = "task moved up"
				return m, m.persist()
			}
			m.status = "already at top"
			return m, nil
		} else {
			if item.SubtaskIndex > 0 {
				m = m.withUndo()
				parent := &m.doc.Tasks[item.TaskIndex]
				prevSubIdx := item.SubtaskIndex - 1
				parent.Subtasks[item.SubtaskIndex], parent.Subtasks[prevSubIdx] = parent.Subtasks[prevSubIdx], parent.Subtasks[item.SubtaskIndex]
				parent.UpdatedAt = time.Now()
				newItems := m.visibleItems()
				for idx, it := range newItems {
					if it.TaskIndex == item.TaskIndex && it.SubtaskIndex == prevSubIdx {
						m.cursor = idx
						break
					}
				}
				m.status = "subtask moved up"
				return m, m.persist()
			}
			m.status = "subtask already at top"
			return m, nil
		}

	case key.Matches(msg, m.keys.MoveDown):
		if len(items) == 0 {
			return m, nil
		}
		item := items[m.cursor]
		if item.SubtaskIndex == -1 {
			if item.TaskIndex < len(m.doc.Tasks)-1 {
				m = m.withUndo()
				nextTaskIdx := item.TaskIndex + 1
				m.doc.Tasks[item.TaskIndex], m.doc.Tasks[nextTaskIdx] = m.doc.Tasks[nextTaskIdx], m.doc.Tasks[item.TaskIndex]
				newItems := m.visibleItems()
				for idx, it := range newItems {
					if it.TaskIndex == nextTaskIdx && it.SubtaskIndex == -1 {
						m.cursor = idx
						break
					}
				}
				m.status = "task moved down"
				return m, m.persist()
			}
			m.status = "already at bottom"
			return m, nil
		} else {
			parent := &m.doc.Tasks[item.TaskIndex]
			if item.SubtaskIndex < len(parent.Subtasks)-1 {
				m = m.withUndo()
				nextSubIdx := item.SubtaskIndex + 1
				parent.Subtasks[item.SubtaskIndex], parent.Subtasks[nextSubIdx] = parent.Subtasks[nextSubIdx], parent.Subtasks[item.SubtaskIndex]
				parent.UpdatedAt = time.Now()
				newItems := m.visibleItems()
				for idx, it := range newItems {
					if it.TaskIndex == item.TaskIndex && it.SubtaskIndex == nextSubIdx {
						m.cursor = idx
						break
					}
				}
				m.status = "subtask moved down"
				return m, m.persist()
			}
			m.status = "subtask already at bottom"
			return m, nil
		}

	case key.Matches(msg, m.keys.Undo):
		if len(m.undoStack) == 0 {
			m.status = "nothing to undo"
			return m, nil
		}
		m.redoStack = append(m.redoStack, cloneTasks(m.doc.Tasks))
		last := m.undoStack[len(m.undoStack)-1]
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
		m.doc.Tasks = last

		newItems := m.visibleItems()
		if m.cursor >= len(newItems) && m.cursor > 0 {
			m.cursor = len(newItems) - 1
		}
		m.streak = storage.CalculateStreak()
		m.status = "action undone"
		return m, m.persist()

	case key.Matches(msg, m.keys.Redo):
		if len(m.redoStack) == 0 {
			m.status = "nothing to redo"
			return m, nil
		}
		m.undoStack = append(m.undoStack, cloneTasks(m.doc.Tasks))
		next := m.redoStack[len(m.redoStack)-1]
		m.redoStack = m.redoStack[:len(m.redoStack)-1]
		m.doc.Tasks = next

		newItems := m.visibleItems()
		if m.cursor >= len(newItems) && m.cursor > 0 {
			m.cursor = len(newItems) - 1
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
		m.status = "a add · s subtask · tab fold · e edit · d del · p prio · J/K move · / filter · C clear · u undo · c copy · q quit"
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
		items := m.visibleItems()

		switch m.mode {
		case ModeAdd:
			task := model.Task{
				ID:        uuid.NewString(),
				Title:     title,
				Status:    model.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			}
			task.Normalize()
			m.doc.Tasks = append(m.doc.Tasks, task)
			m.status = "task added"
			newItems := m.visibleItems()
			m.cursor = len(newItems) - 1

		case ModeAddSubtask:
			if len(items) > 0 && m.cursor < len(items) {
				item := items[m.cursor]
				parent := &m.doc.Tasks[item.TaskIndex]
				sub := model.Subtask{
					ID:        uuid.NewString(),
					Title:     title,
					Done:      false,
					CreatedAt: now,
					UpdatedAt: now,
				}
				parent.Subtasks = append(parent.Subtasks, sub)
				parent.UpdatedAt = now
				delete(m.collapsed, parent.ID) // ensure expanded
				m.status = "subtask added"
				newItems := m.visibleItems()
				for idx, it := range newItems {
					if it.TaskIndex == item.TaskIndex && it.SubtaskIndex == len(parent.Subtasks)-1 {
						m.cursor = idx
						break
					}
				}
			}

		case ModeEdit:
			if len(items) > 0 && m.cursor < len(items) {
				item := items[m.cursor]
				m.doc.Tasks[item.TaskIndex].Title = title
				m.doc.Tasks[item.TaskIndex].UpdatedAt = now
				m.doc.Tasks[item.TaskIndex].Normalize()
				m.status = "task updated"
			}

		case ModeEditSubtask:
			if len(items) > 0 && m.cursor < len(items) {
				item := items[m.cursor]
				if item.SubtaskIndex >= 0 {
					parent := &m.doc.Tasks[item.TaskIndex]
					parent.Subtasks[item.SubtaskIndex].Title = title
					parent.Subtasks[item.SubtaskIndex].UpdatedAt = now
					parent.UpdatedAt = now
					m.status = "subtask updated"
				}
			}
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
		items := m.visibleItems()
		if len(items) > 0 && m.cursor < len(items) {
			item := items[m.cursor]
			m = m.withUndo()
			if m.mode == ModeConfirmDeleteSubtask && item.SubtaskIndex >= 0 {
				parent := &m.doc.Tasks[item.TaskIndex]
				subIdx := item.SubtaskIndex
				parent.Subtasks = append(parent.Subtasks[:subIdx], parent.Subtasks[subIdx+1:]...)
				parent.UpdatedAt = time.Now()
				m.status = "subtask deleted"
			} else {
				taskIdx := item.TaskIndex
				m.doc.Tasks = append(m.doc.Tasks[:taskIdx], m.doc.Tasks[taskIdx+1:]...)
				m.status = "task deleted"
			}
			newItems := m.visibleItems()
			if m.cursor >= len(newItems) && m.cursor > 0 {
				m.cursor = len(newItems) - 1
			}
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
	doc.Tasks = cloneTasks(m.doc.Tasks)

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
