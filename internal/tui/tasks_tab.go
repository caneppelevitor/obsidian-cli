package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
)

type indexedTask struct {
	task  tasks.Task
	index int
}

func (m AppModel) renderTasksContent() string {
	pendingTasks := filterPending(m.tasks)
	completedTasks := filterCompleted(m.tasks)

	if len(pendingTasks) == 0 {
		return "\nNo pending tasks!\n\nCreate some tasks in your daily note using [] prefix"
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(" %d pending | %d completed | %d total\n",
		len(pendingTasks), len(completedTasks), len(m.tasks)))
	sb.WriteString(lipgloss.NewStyle().Foreground(colorCyan).Render(strings.Repeat("─", 50)) + "\n")

	tagOrder := []string{"#do", "#delegate", "#schedule", "#eliminate"}
	groups := make(map[string][]indexedTask)
	var untagged []indexedTask

	for i, task := range pendingTasks {
		matched := false
		for _, tag := range tagOrder {
			if strings.Contains(task.Content, tag) {
				groups[tag] = append(groups[tag], indexedTask{task, i})
				matched = true
				break
			}
		}
		if !matched {
			untagged = append(untagged, indexedTask{task, i})
		}
	}

	for _, tag := range tagOrder {
		if tasksInGroup, ok := groups[tag]; ok && len(tasksInGroup) > 0 {
			colorCode := m.eisenhowerTags[tag]
			tagStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorCode)).
				Bold(true)
			sb.WriteString(fmt.Sprintf("\n%s\n", tagStyle.Render(fmt.Sprintf("%s (%d)", tag, len(tasksInGroup)))))
			for _, it := range tasksInGroup {
				sb.WriteString(m.renderTask(it))
			}
		}
	}

	if len(untagged) > 0 {
		sb.WriteString(fmt.Sprintf("\n%s\n",
			lipgloss.NewStyle().Foreground(colorGray).Bold(true).Render(
				fmt.Sprintf("Untagged (%d)", len(untagged)))))
		for _, it := range untagged {
			sb.WriteString(m.renderTask(it))
		}
	}

	sb.WriteString(fmt.Sprintf("\n%s",
		cheatSheetStyle.Render(fmt.Sprintf("Tip: Type a number (1-%d) and press Enter to complete that task", len(pendingTasks)))))

	return sb.String()
}

func (m AppModel) renderTask(it indexedTask) string {
	numStyle := lipgloss.NewStyle().Foreground(colorYellow)
	iconStyle := lipgloss.NewStyle().Foreground(colorRed)

	taskContent := it.task.Content
	for tag, colorCode := range m.eisenhowerTags {
		if strings.Contains(taskContent, tag) {
			tagStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorCode)).
				Bold(true)
			taskContent = strings.ReplaceAll(taskContent, tag, tagStyle.Render(tag))
		}
	}

	sourceStyle := lipgloss.NewStyle().Foreground(colorGray)

	return fmt.Sprintf("  %s %s %s %s\n",
		numStyle.Render(fmt.Sprintf("[%d]", it.index+1)),
		iconStyle.Render("○"),
		taskContent,
		sourceStyle.Render(fmt.Sprintf("(%s)", it.task.SourceFile)),
	)
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
