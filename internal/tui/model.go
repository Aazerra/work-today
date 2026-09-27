package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/alireza/work-today/internal/model"
	"github.com/alireza/work-today/internal/storage"
)

// Mode describes which interaction layer is active.
type Mode int

const (
	ModeList Mode = iota
	ModeAdd
	ModeEdit
	ModeConfirmDelete
)

type errMsg struct{ err error }
type savedMsg struct{}

// Model is the Bubble Tea application state.
type Model struct {
	doc      *model.Document
	cursor   int
	mode     Mode
	input    textinput.Model
	status   string
	width    int
	height   int
	quitting bool
	keys     keyMap
}

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Add    key.Binding
	Edit   key.Binding
	Delete key.Binding
	Toggle key.Binding
	Quit   key.Binding
	Help   key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Add:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Edit:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Toggle: key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("↵/space", "cycle status")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
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
		default:
			return m.updateList(msg)
		}
	}

	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.doc.Tasks)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Add):
		m.mode = ModeAdd
		m.input.SetValue("")
		m.input.Placeholder = "New task title"
		m.input.Focus()
		m.status = ""
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Edit):
		if len(m.doc.Tasks) == 0 {
			m.status = "nothing to edit"
			return m, nil
		}
		m.mode = ModeEdit
		m.input.SetValue(m.doc.Tasks[m.cursor].Title)
		m.input.Placeholder = "Edit task title"
		m.input.CursorEnd()
		m.input.Focus()
		m.status = ""
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Delete):
		if len(m.doc.Tasks) == 0 {
			m.status = "nothing to delete"
			return m, nil
		}
		m.mode = ModeConfirmDelete
		m.status = ""
		return m, nil

	case key.Matches(msg, m.keys.Toggle):
		if len(m.doc.Tasks) == 0 {
			return m, nil
		}
		t := &m.doc.Tasks[m.cursor]
		t.Status = model.NextStatus(t.Status)
		t.UpdatedAt = time.Now()
		return m, m.persist()

	case key.Matches(msg, m.keys.Help):
		m.status = "a add · e edit · d delete · ↵ cycle · j/k move · q quit"
	}
	return m, nil
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

		now := time.Now()
		if m.mode == ModeAdd {
			m.doc.Tasks = append(m.doc.Tasks, model.Task{
				ID:        uuid.NewString(),
				Title:     title,
				Status:    model.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			})
			m.cursor = len(m.doc.Tasks) - 1
			m.status = "task added"
		} else if m.mode == ModeEdit && len(m.doc.Tasks) > 0 {
			m.doc.Tasks[m.cursor].Title = title
			m.doc.Tasks[m.cursor].UpdatedAt = now
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
		if len(m.doc.Tasks) > 0 {
			idx := m.cursor
			m.doc.Tasks = append(m.doc.Tasks[:idx], m.doc.Tasks[idx+1:]...)
			if m.cursor >= len(m.doc.Tasks) && m.cursor > 0 {
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
