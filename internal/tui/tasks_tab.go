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
		return RenderEmptyState(
			"No pending tasks",
			"Add tasks in your daily note using [] prefix\nor press Tab to switch to Daily Note",
			m.width-4,
			m.viewport.Height(),
		)
	}

	var sb strings.Builder

	// Summary line
	pendingStyle := lipgloss.NewStyle().Bold(true).Foreground(colorText)
	doneStyle := lipgloss.NewStyle().Foreground(colorGreen)
	totalStyle := lipgloss.NewStyle().Foreground(colorOverlay)

	sb.WriteString(" " + pendingStyle.Render(fmt.Sprintf("%d pending", len(pending))))
	sb.WriteString("  ")
	sb.WriteString(doneStyle.Render(fmt.Sprintf("%d done", len(completed))))
	sb.WriteString("  ")
	sb.WriteString(totalStyle.Render(fmt.Sprintf("%d total", len(m.tasks))))

	// Tag legend — right side
	tagOrder := []string{"#do", "#delegate", "#schedule", "#eliminate"}
	var legendParts []string
	for _, tag := range tagOrder {
		dc, ok := eisenhowerDisplayColors[tag]
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
				Foreground(dc).
				Bold(true)
			legendParts = append(legendParts, tagStyle.Render(fmt.Sprintf("%s(%d)", tag, count)))
		}
	}
	if len(legendParts) > 0 {
		legend := strings.Join(legendParts, " ")
		summaryWidth := lipgloss.Width(sb.String())
		legendWidth := lipgloss.Width(legend)
		gap := m.width - 6 - summaryWidth - legendWidth
		if gap < 2 {
			gap = 2
		}
		sb.WriteString(strings.Repeat(" ", gap))
		sb.WriteString(legend)
	}
	sb.WriteString("\n")

	// Divider
	sb.WriteString(dimStyle.Render(" " + strings.Repeat("─", m.width-6)) + "\n")

	// Calculate column widths
	sourceWidth := 10
	taskWidth := m.width - 14 - sourceWidth // num(4) + icon(3) + padding + source

	// Render each task
	for i, task := range pending {
		isSelected := i == m.taskCursor

		// Task content with Eisenhower tag colors
		content := task.Content
		for tag := range m.eisenhowerTags {
			if strings.Contains(content, tag) {
				dc, ok := eisenhowerDisplayColors[tag]
				if !ok {
					dc = colorOverlay
				}
				tagStyle := lipgloss.NewStyle().
					Foreground(dc).
					Bold(true)
				content = strings.ReplaceAll(content, tag, tagStyle.Render(tag))
			}
		}

		// Truncate content if too long
		if lipgloss.Width(content) > taskWidth {
			// Simple truncation (keeps ANSI codes intact enough)
			content = content[:taskWidth-1] + "…"
		}

		// Source (just the date part, trimmed)
		source := task.SourceFile
		if len(source) > sourceWidth {
			source = source[:sourceWidth]
		}

		numStr := fmt.Sprintf(" %2d", i+1)
		icon := " ○ "
		sourceStr := lipgloss.NewStyle().
			Foreground(colorOverlay).
			Render(fmt.Sprintf("%*s", sourceWidth, source))

		if isSelected {
			// Selected row: accent marker + subtle background
			marker := lipgloss.NewStyle().
				Foreground(colorBlue).
				Bold(true).
				Render("▸")
			numRendered := lipgloss.NewStyle().
				Foreground(colorBlue).
				Bold(true).
				Render(numStr)
			iconRendered := lipgloss.NewStyle().
				Foreground(colorBlue).
				Render(icon)

			sb.WriteString(fmt.Sprintf("%s%s%s%s  %s\n",
				marker, numRendered, iconRendered, content, sourceStr))
		} else {
			numRendered := lipgloss.NewStyle().Foreground(colorOverlay).Render(numStr)
			iconRendered := lipgloss.NewStyle().Foreground(colorOverlay).Render(icon)

			sb.WriteString(fmt.Sprintf(" %s%s%s  %s\n",
				numRendered, iconRendered, content, sourceStr))
		}
	}

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
