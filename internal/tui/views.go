package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/aazerra/work-today/internal/model"
	"github.com/aazerra/work-today/internal/storage"
)

var (
	colorMuted          = lipgloss.Color("245")
	colorAccent         = lipgloss.Color("39")
	colorTodo           = lipgloss.Color("252")
	colorDoing          = lipgloss.Color("214")
	colorDone           = lipgloss.Color("114")
	colorDanger         = lipgloss.Color("203")
	colorBorder         = lipgloss.Color("238")
	colorPriorityHigh   = lipgloss.Color("196")
	colorPriorityMedium = lipgloss.Color("214")
	colorPriorityLow    = lipgloss.Color("75")
	colorTag            = lipgloss.Color("141") // soft purple
	colorContext        = lipgloss.Color("43")  // teal / cyan

	priorityHighStyle = lipgloss.NewStyle().
				Foreground(colorPriorityHigh).
				Bold(true)

	priorityMediumStyle = lipgloss.NewStyle().
				Foreground(colorPriorityMedium)

	priorityLowStyle = lipgloss.NewStyle().
				Foreground(colorPriorityLow)

	tagStyle = lipgloss.NewStyle().
			Foreground(colorTag)

	contextStyle = lipgloss.NewStyle().
			Foreground(colorContext)

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

	treeBranchStyle = lipgloss.NewStyle().
			Foreground(colorBorder)

	foldGlyphStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	progressBadgeStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	subtaskTitleTodoStyle = lipgloss.NewStyle().
				Foreground(colorTodo)

	subtaskTitleDoneStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Strikethrough(true)
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

	streak := m.streak
	if streak == 0 {
		streak = storage.CalculateStreak()
	}
	streakStr := ""
	if streak > 0 {
		streakStr = fmt.Sprintf("  ·  🔥 %d day streak", streak)
	}

	filterStr := ""
	if m.filter != "" {
		filterStr = fmt.Sprintf("  ·  filter: %q", m.filter)
	}

	title := titleStyle.Render("work today")
	meta := subtitleStyle.Render(fmt.Sprintf("%s  ·  %d/%d done%s%s  ·  %s",
		date, done, total, streakStr, filterStr, storage.Path()))

	return headerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, meta),
	)
}

func (m Model) renderBody() string {
	list := listStyle.Render(m.renderTaskList())
	items := m.visibleItems()

	switch m.mode {
	case ModeAdd:
		box := promptBoxStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				promptLabelStyle.Render("Add task (+tag, @context supported)"),
				m.input.View(),
			),
		)
		return lipgloss.JoinVertical(lipgloss.Left, list, box)

	case ModeAddSubtask:
		parentTitle := ""
		if len(items) > 0 && m.cursor < len(items) {
			parentTitle = m.doc.Tasks[items[m.cursor].TaskIndex].Title
		}
		box := promptBoxStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				promptLabelStyle.Render(fmt.Sprintf("Add subtask to %q", parentTitle)),
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

	case ModeEditSubtask:
		box := promptBoxStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				promptLabelStyle.Render("Edit subtask"),
				m.input.View(),
			),
		)
		return lipgloss.JoinVertical(lipgloss.Left, list, box)

	case ModeFilter:
		box := promptBoxStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				promptLabelStyle.Render("Filter tasks (enter apply · esc clear)"),
				m.input.View(),
			),
		)
		return lipgloss.JoinVertical(lipgloss.Left, list, box)

	case ModeConfirmDelete:
		title := ""
		if len(items) > 0 && m.cursor < len(items) {
			title = m.doc.Tasks[items[m.cursor].TaskIndex].Title
		}
		msg := listStyle.Render(dangerStyle.Render(fmt.Sprintf("Delete task %q? (y/n)", title)))
		return lipgloss.JoinVertical(lipgloss.Left, list, msg)

	case ModeConfirmDeleteSubtask:
		title := ""
		if len(items) > 0 && m.cursor < len(items) {
			it := items[m.cursor]
			if it.SubtaskIndex >= 0 {
				title = m.doc.Tasks[it.TaskIndex].Subtasks[it.SubtaskIndex].Title
			}
		}
		msg := listStyle.Render(dangerStyle.Render(fmt.Sprintf("Delete subtask %q? (y/n)", title)))
		return lipgloss.JoinVertical(lipgloss.Left, list, msg)

	default:
		return list
	}
}

func (m Model) renderTaskList() string {
	items := m.visibleItems()
	if len(items) == 0 {
		if m.filter != "" {
			return emptyStyle.Render(fmt.Sprintf("No tasks matching %q. Press esc to clear filter.", m.filter))
		}
		return emptyStyle.Render("No tasks yet. Press a to add one.")
	}

	lines := make([]string, 0, len(items))
	for displayIdx, it := range items {
		if it.SubtaskIndex == -1 {
			lines = append(lines, m.renderTaskRow(displayIdx, it.TaskIndex, m.doc.Tasks[it.TaskIndex]))
		} else {
			lines = append(lines, m.renderSubtaskRow(displayIdx, it.TaskIndex, it.SubtaskIndex))
		}
	}
	return strings.Join(lines, "\n")
}

func renderPriorityBadge(priority string, done bool) string {
	if priority == "" {
		return ""
	}
	var style lipgloss.Style
	if done {
		style = lipgloss.NewStyle().Foreground(colorMuted)
	} else {
		switch priority {
		case model.PriorityHigh:
			style = priorityHighStyle
		case model.PriorityMedium:
			style = priorityMediumStyle
		case model.PriorityLow:
			style = priorityLowStyle
		default:
			style = subtitleStyle
		}
	}
	return style.Render(fmt.Sprintf("[%s]", model.PriorityLabel(priority))) + " "
}

func renderTaskTitle(t model.Task) string {
	isDone := t.Status == model.StatusDone
	var baseTitleStyle lipgloss.Style
	switch t.Status {
	case model.StatusInProgress:
		baseTitleStyle = doingTitleStyle
	case model.StatusDone:
		baseTitleStyle = doneTitleStyle
	default:
		baseTitleStyle = todoTitleStyle
	}

	if isDone {
		res := baseTitleStyle.Render(t.Title)
		titleLower := strings.ToLower(t.Title)
		var extras []string
		for _, tag := range t.Tags {
			if !strings.Contains(titleLower, "+"+tag) && !strings.Contains(titleLower, "#"+tag) {
				extras = append(extras, baseTitleStyle.Render("+"+tag))
			}
		}
		for _, ctx := range t.Contexts {
			if !strings.Contains(titleLower, "@"+ctx) {
				extras = append(extras, baseTitleStyle.Render("@"+ctx))
			}
		}
		if len(extras) > 0 {
			res += " " + strings.Join(extras, " ")
		}
		return res
	}

	words := strings.Split(t.Title, " ")
	renderedWords := make([]string, len(words))
	for i, w := range words {
		clean := strings.TrimRight(w, ",.;:!?)]}\"'")
		punct := w[len(clean):]
		if len(clean) > 1 {
			if clean[0] == '+' || clean[0] == '#' {
				renderedWords[i] = tagStyle.Render(clean) + baseTitleStyle.Render(punct)
				continue
			} else if clean[0] == '@' {
				renderedWords[i] = contextStyle.Render(clean) + baseTitleStyle.Render(punct)
				continue
			}
		}
		renderedWords[i] = baseTitleStyle.Render(w)
	}

	res := strings.Join(renderedWords, " ")

	titleLower := strings.ToLower(t.Title)
	var extras []string
	for _, tag := range t.Tags {
		if !strings.Contains(titleLower, "+"+tag) && !strings.Contains(titleLower, "#"+tag) {
			extras = append(extras, tagStyle.Render("+"+tag))
		}
	}
	for _, ctx := range t.Contexts {
		if !strings.Contains(titleLower, "@"+ctx) {
			extras = append(extras, contextStyle.Render("@"+ctx))
		}
	}
	if len(extras) > 0 {
		res += " " + strings.Join(extras, " ")
	}

	return res
}

func (m Model) renderTaskRow(displayIdx int, taskIdx int, t model.Task) string {
	var glyphStyle, labelStyle lipgloss.Style
	switch t.Status {
	case model.StatusInProgress:
		glyphStyle, labelStyle = doingGlyphStyle, doingLabelStyle
	case model.StatusDone:
		glyphStyle, labelStyle = doneGlyphStyle, doneLabelStyle
	default:
		glyphStyle, labelStyle = todoGlyphStyle, todoLabelStyle
	}

	cursor := "  "
	if displayIdx == m.cursor {
		cursor = cursorMarkStyle.Render("› ")
	}

	foldGlyph := "  "
	if len(t.Subtasks) > 0 {
		if m.collapsed[t.ID] {
			foldGlyph = foldGlyphStyle.Render("▸ ")
		} else {
			foldGlyph = foldGlyphStyle.Render("▾ ")
		}
	}

	pBadge := renderPriorityBadge(t.Priority, t.Status == model.StatusDone)
	title := renderTaskTitle(t)

	progress := ""
	if len(t.Subtasks) > 0 {
		done, total := t.SubtaskProgress()
		progress = " " + progressBadgeStyle.Render(fmt.Sprintf("(%d/%d)", done, total))
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		cursor,
		foldGlyph,
		glyphStyle.Render(model.StatusGlyph(t.Status)),
		" ",
		labelStyle.Render(model.StatusLabel(t.Status)),
		"  ",
		pBadge,
		title,
		progress,
	)
}

func (m Model) renderSubtaskRow(displayIdx int, taskIdx int, subtaskIdx int) string {
	parent := m.doc.Tasks[taskIdx]
	st := parent.Subtasks[subtaskIdx]

	cursor := "  "
	if displayIdx == m.cursor {
		cursor = cursorMarkStyle.Render("› ")
	}

	isLast := subtaskIdx == len(parent.Subtasks)-1
	branchStr := "   ├─ "
	if isLast {
		branchStr = "   └─ "
	}
	branch := treeBranchStyle.Render(branchStr)

	glyph := todoGlyphStyle.Render("[ ]")
	title := subtaskTitleTodoStyle.Render(st.Title)
	if st.Done {
		glyph = doneGlyphStyle.Render("[x]")
		title = subtaskTitleDoneStyle.Render(st.Title)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		cursor,
		branch,
		glyph,
		" ",
		title,
	)
}

func (m Model) renderFooter() string {
	help := "a add · s subtask · tab fold · e edit · d del · p prio · J/K move · / filter · C clear · u undo · c copy · ↵ toggle · q quit"
	if m.mode == ModeAdd || m.mode == ModeEdit || m.mode == ModeAddSubtask || m.mode == ModeEditSubtask {
		help = "enter confirm · esc cancel"
	} else if m.mode == ModeConfirmDelete || m.mode == ModeConfirmDeleteSubtask {
		help = "y confirm · n/esc cancel"
	} else if m.mode == ModeFilter {
		help = "enter apply · esc clear"
	}

	parts := []string{helpStyle.Render(help)}
	if m.status != "" {
		parts = append(parts, statusStyle.Render("· "+m.status))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
