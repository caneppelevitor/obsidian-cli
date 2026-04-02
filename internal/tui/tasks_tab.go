package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
)

// renderTasksContent renders the tasks view with colored Eisenhower tags.
func (m AppModel) renderTasksContent() string {
	pending := filterPending(m.tasks)
	completed := filterCompleted(m.tasks)

	if len(pending) == 0 {
		return "\n  No pending tasks!\n\n  Create some tasks in your daily note using [] prefix"
	}

	var sb strings.Builder

	// Summary line
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf(" %d pending", len(pending))))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(" | "))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorGreen).Render(
		fmt.Sprintf("%d completed", len(completed))))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(" | "))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(
		fmt.Sprintf("%d total\n", len(m.tasks))))

	// Tag legend
	tagOrder := []string{"#do", "#delegate", "#schedule", "#eliminate"}
	var legendParts []string
	for _, tag := range tagOrder {
		colorCode, ok := m.eisenhowerTags[tag]
		if !ok {
			continue
		}
		count := 0
		for _, t := range pending {
			if strings.Contains(t.Content, tag) {
				count++
			}
		}
		if count > 0 {
			tagStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorCode)).
				Bold(true)
			legendParts = append(legendParts, tagStyle.Render(fmt.Sprintf("%s(%d)", tag, count)))
		}
	}
	if len(legendParts) > 0 {
		sb.WriteString(" " + strings.Join(legendParts, "  ") + "\n")
	}

	sb.WriteString(lipgloss.NewStyle().Foreground(colorCyan).Render(strings.Repeat("─", m.width-6)) + "\n")

	// Render each task with colored tags
	for i, task := range pending {
		isSelected := i == m.taskCursor

		numStr := fmt.Sprintf("%2d", i+1)
		icon := "○"
		source := task.SourceFile

		// Apply Eisenhower tag colors to content
		content := task.Content
		for tag, colorCode := range m.eisenhowerTags {
			if strings.Contains(content, tag) {
				tagStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color(colorCode)).
					Bold(true)
				content = strings.ReplaceAll(content, tag, tagStyle.Render(tag))
			}
		}

		if isSelected {
			// Highlighted row
			selStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("15")).
				Bold(true)
			numRendered := selStyle.Render(numStr)
			iconRendered := selStyle.Render(" " + icon + " ")

			// For selected row, we still want tag colors to show
			sourceRendered := lipgloss.NewStyle().
				Foreground(colorGray).
				Render(fmt.Sprintf(" (%s)", source))

			sb.WriteString(fmt.Sprintf(" %s%s %s%s\n", numRendered, iconRendered, content, sourceRendered))
		} else {
			numRendered := lipgloss.NewStyle().Foreground(colorGray).Render(numStr)
			iconRendered := lipgloss.NewStyle().Foreground(colorRed).Render(" " + icon + " ")
			sourceRendered := lipgloss.NewStyle().Foreground(colorGray).
				Render(fmt.Sprintf(" (%s)", source))

			sb.WriteString(fmt.Sprintf(" %s%s %s%s\n", numRendered, iconRendered, content, sourceRendered))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(cheatSheetStyle.Render(
		fmt.Sprintf("  j/k navigate · Enter complete · %d tasks", len(pending))))

	return sb.String()
}

// handleTaskTableSelection completes the task at the current cursor.
func (m *AppModel) handleTaskTableSelection() tea.Cmd {
	pending := filterPending(m.tasks)
	if m.taskCursor < 0 || m.taskCursor >= len(pending) {
		return nil
	}

	target := pending[m.taskCursor]
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

// moveTaskCursor moves the cursor up or down.
func (m *AppModel) moveTaskCursor(delta int) {
	pending := filterPending(m.tasks)
	if len(pending) == 0 {
		return
	}
	m.taskCursor += delta
	if m.taskCursor < 0 {
		m.taskCursor = 0
	}
	if m.taskCursor >= len(pending) {
		m.taskCursor = len(pending) - 1
	}
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
