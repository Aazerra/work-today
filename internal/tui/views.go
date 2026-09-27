package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/alireza/work-today/internal/model"
	"github.com/alireza/work-today/internal/storage"
)

var (
	colorMuted  = lipgloss.Color("245")
	colorAccent = lipgloss.Color("39")
	colorTodo   = lipgloss.Color("252")
	colorDoing  = lipgloss.Color("214")
	colorDone   = lipgloss.Color("114")
	colorDanger = lipgloss.Color("203")
	colorBorder = lipgloss.Color("238")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	headerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(colorBorder).
			Padding(0, 1).
			MarginBottom(1)

	listStyle = lipgloss.NewStyle().
			Padding(0, 1)

	cursorMarkStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	todoGlyphStyle = lipgloss.NewStyle().
			Foreground(colorTodo)

	doingGlyphStyle = lipgloss.NewStyle().
			Foreground(colorDoing).
			Bold(true)

	doneGlyphStyle = lipgloss.NewStyle().
			Foreground(colorDone)

	todoTitleStyle = lipgloss.NewStyle().
			Foreground(colorTodo)

	doingTitleStyle = lipgloss.NewStyle().
			Foreground(colorDoing)

	doneTitleStyle = lipgloss.NewStyle().
			Foreground(colorDone).
			Strikethrough(true)

	todoLabelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(5)

	doingLabelStyle = lipgloss.NewStyle().
			Foreground(colorDoing).
			Bold(true).
			Width(5)

	doneLabelStyle = lipgloss.NewStyle().
			Foreground(colorDone).
			Width(5)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	promptBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1).
			MarginTop(1)

	promptLabelStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	dangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	emptyStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Padding(1, 2)
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(),
		m.renderBody(),
		m.renderFooter(),
	)
}

func (m Model) renderHeader() string {
	date := m.doc.Date
	if parsed, err := time.Parse("2006-01-02", date); err == nil {
		date = parsed.Format("Mon, 02 Jan 2006")
	}

	done, total := 0, len(m.doc.Tasks)
	for _, t := range m.doc.Tasks {
		if t.Status == model.StatusDone {
			done++
		}
	}

	title := titleStyle.Render("work today")
	meta := subtitleStyle.Render(fmt.Sprintf("%s  ·  %d/%d done  ·  %s",
		date, done, total, storage.Path()))

	return headerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, meta),
	)
}

func (m Model) renderBody() string {
	list := listStyle.Render(m.renderTaskList())

	switch m.mode {
	case ModeAdd:
		box := promptBoxStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				promptLabelStyle.Render("Add task"),
				m.input.View(),
			),
		)
		return lipgloss.JoinVertical(lipgloss.Left, list, box)
	case ModeEdit:
		box := promptBoxStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				promptLabelStyle.Render("Edit task"),
				m.input.View(),
			),
		)
		return lipgloss.JoinVertical(lipgloss.Left, list, box)
	case ModeConfirmDelete:
		title := ""
		if len(m.doc.Tasks) > 0 {
			title = m.doc.Tasks[m.cursor].Title
		}
		msg := listStyle.Render(dangerStyle.Render(fmt.Sprintf("Delete %q? (y/n)", title)))
		return lipgloss.JoinVertical(lipgloss.Left, list, msg)
	default:
		return list
	}
}

func (m Model) renderTaskList() string {
	if len(m.doc.Tasks) == 0 {
		return emptyStyle.Render("No tasks yet. Press a to add one.")
	}

	lines := make([]string, 0, len(m.doc.Tasks))
	for i, t := range m.doc.Tasks {
		lines = append(lines, m.renderTaskRow(i, t))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderTaskRow(i int, t model.Task) string {
	var glyphStyle, labelStyle, titleStyle lipgloss.Style
	switch t.Status {
	case model.StatusInProgress:
		glyphStyle, labelStyle, titleStyle = doingGlyphStyle, doingLabelStyle, doingTitleStyle
	case model.StatusDone:
		glyphStyle, labelStyle, titleStyle = doneGlyphStyle, doneLabelStyle, doneTitleStyle
	default:
		glyphStyle, labelStyle, titleStyle = todoGlyphStyle, todoLabelStyle, todoTitleStyle
	}

	cursor := "  "
	if i == m.cursor {
		cursor = cursorMarkStyle.Render("› ")
	} else {
		cursor = "  "
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		cursor,
		glyphStyle.Render(model.StatusGlyph(t.Status)),
		" ",
		labelStyle.Render(model.StatusLabel(t.Status)),
		"  ",
		titleStyle.Render(t.Title),
	)
}

func (m Model) renderFooter() string {
	help := "a add · e edit · d delete · ↵/space cycle · j/k navigate · esc cancel · q quit"
	if m.mode == ModeAdd || m.mode == ModeEdit {
		help = "enter confirm · esc cancel"
	} else if m.mode == ModeConfirmDelete {
		help = "y confirm · n/esc cancel"
	}

	parts := []string{helpStyle.Render(help)}
	if m.status != "" {
		parts = append(parts, statusStyle.Render("· "+m.status))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
