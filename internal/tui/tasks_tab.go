package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
)

// buildTaskTable creates a table.Model from the current task list.
func (m *AppModel) buildTaskTable() {
	pending := filterPending(m.tasks)

	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Status", Width: 3},
		{Title: "Task", Width: max(m.width-38, 20)},
		{Title: "Source", Width: 12},
		{Title: "Tag", Width: 12},
	}

	rows := make([]table.Row, len(pending))
	for i, t := range pending {
		tag := extractEisenhowerTag(t.Content)
		content := t.Content
		// Strip the tag from content to avoid duplication
		if tag != "" {
			content = strings.ReplaceAll(content, tag, "")
			content = strings.TrimSpace(content)
		}
		rows[i] = table.Row{
			fmt.Sprintf("%d", i+1),
			"○",
			content,
			t.SourceFile,
			tag,
		}
	}

	viewportHeight := m.height - 8
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorCyan).
		Bold(true).
		Foreground(colorCyan)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Bold(true)
	s.Cell = s.Cell.Foreground(colorWhite)

	m.taskTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(viewportHeight),
		table.WithWidth(m.width-2),
	)
	m.taskTable.SetStyles(s)
	m.taskTableReady = true
}

// renderTasksHeader renders the summary line above the task table.
func (m AppModel) renderTasksHeader() string {
	pending := filterPending(m.tasks)
	completed := filterCompleted(m.tasks)

	if len(pending) == 0 {
		return "\n  No pending tasks!\n\n  Create some tasks in your daily note using [] prefix"
	}

	var sb strings.Builder

	// Summary
	summaryStyle := lipgloss.NewStyle().Bold(true)
	sb.WriteString(summaryStyle.Render(fmt.Sprintf(
		" %d pending", len(pending))))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(" | "))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorGreen).Render(
		fmt.Sprintf("%d completed", len(completed))))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(" | "))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(
		fmt.Sprintf("%d total", len(m.tasks))))
	sb.WriteString("\n")

	// Tag legend
	tagOrder := []string{"#do", "#delegate", "#schedule", "#eliminate"}
	var legendParts []string
	for _, tag := range tagOrder {
		colorCode, ok := m.eisenhowerTags[tag]
		if !ok {
			continue
		}
		tagStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCode)).
			Bold(true)
		count := 0
		for _, t := range pending {
			if strings.Contains(t.Content, tag) {
				count++
			}
		}
		if count > 0 {
			legendParts = append(legendParts, tagStyle.Render(fmt.Sprintf("%s(%d)", tag, count)))
		}
	}
	if len(legendParts) > 0 {
		sb.WriteString(" " + strings.Join(legendParts, "  ") + "\n")
	}

	return sb.String()
}

// handleTaskTableSelection completes the selected task from the table.
func (m *AppModel) handleTaskTableSelection() tea.Cmd {
	if !m.taskTableReady {
		return nil
	}

	selectedRow := m.taskTable.SelectedRow()
	if selectedRow == nil {
		return nil
	}

	// Parse the index from the first column
	idx := m.taskTable.Cursor()
	pending := filterPending(m.tasks)
	if idx < 0 || idx >= len(pending) {
		return nil
	}

	target := pending[idx]
	for i, t := range m.tasks {
		if t.Content == target.Content && !t.Completed {
			return completeTaskCmd(m.vaultPath, i, m.tasks)
		}
	}
	return nil
}

func (m *AppModel) handleTaskCompletion(num int) tea.Cmd {
	pendingTasks := filterPending(m.tasks)
	idx := num - 1
	if idx < 0 || idx >= len(pendingTasks) {
		m.statusText = fmt.Sprintf("Invalid task number: %d", num)
		return nil
	}

	target := pendingTasks[idx]
	for i, t := range m.tasks {
		if t.Content == target.Content && !t.Completed {
			return completeTaskCmd(m.vaultPath, i, m.tasks)
		}
	}
	return nil
}

func extractEisenhowerTag(content string) string {
	tags := []string{"#do", "#delegate", "#schedule", "#eliminate"}
	for _, tag := range tags {
		if strings.Contains(content, tag) {
			return tag
		}
	}
	return ""
}

func filterPending(taskList []tasks.Task) []tasks.Task {
	var result []tasks.Task
	for _, t := range taskList {
		if !t.Completed {
			result = append(result, t)
		}
	}
	return result
}

func filterCompleted(taskList []tasks.Task) []tasks.Task {
	var result []tasks.Task
	for _, t := range taskList {
		if t.Completed {
			result = append(result, t)
		}
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
