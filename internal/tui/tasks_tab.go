package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
)

// renderTasksTab renders the complete split-pane tasks view with Eisenhower matrix.
func (m AppModel) renderTasksTab() string {
	pending := filterPending(m.tasks)
	completed := filterCompleted(m.tasks)

	if len(pending) == 0 && len(completed) == 0 {
		content := RenderEmptyState(
			"No pending tasks",
			"Add tasks in your daily note using [] prefix\nor press Tab to switch to Daily Note",
			m.width-4,
			m.height-7,
		)
		return activeBorderStyle.
			Width(m.width - 2).
			Render(content)
	}

	totalHeight := m.height - 7 // tab(2) + outer border(2) + status(1) + padding
	leftWidth := (m.width - 4) * 55 / 100
	rightWidth := m.width - 4 - leftWidth - 1 // outer border(2) + divider(1)

	if rightWidth < 20 {
		// Narrow terminal: fall back to full-width task list
		content := m.renderTaskList(m.width-4, totalHeight)
		return activeBorderStyle.
			Width(m.width - 2).
			Render(content)
	}

	// Left pane: task list + completed section
	leftContent := m.renderTaskList(leftWidth, totalHeight)
	leftPane := lipgloss.NewStyle().
		Width(leftWidth).
		Height(totalHeight).
		Render(leftContent)

	// Vertical divider
	vDivider := lipgloss.NewStyle().
		Foreground(colorSurface1).
		Width(1).
		Height(totalHeight).
		Render(strings.Repeat("│\n", totalHeight))

	// Right pane: Eisenhower matrix
	rightContent := m.renderEisenhowerMatrix(rightWidth, totalHeight)
	rightPane := lipgloss.NewStyle().
		Width(rightWidth).
		Height(totalHeight).
		Render(rightContent)

	inner := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, vDivider, rightPane)

	return activeBorderStyle.
		Width(m.width - 2).
		Render(inner)
}

// renderTaskList renders the pending task list with completed tasks below.
func (m AppModel) renderTaskList(width, height int) string {
	pending := filterPending(m.tasks)
	completed := filterCompleted(m.tasks)

	var sb strings.Builder

	// Summary line with progress bar
	total := len(m.tasks)
	completionPercent := 0.0
	if total > 0 {
		completionPercent = float64(len(completed)) / float64(total)
	}

	pendingStyle := lipgloss.NewStyle().Bold(true).Foreground(colorText)
	doneStyle := lipgloss.NewStyle().Foreground(colorGreen)

	sb.WriteString(" " + pendingStyle.Render(fmt.Sprintf("%d pending", len(pending))))
	sb.WriteString("  ")
	sb.WriteString(doneStyle.Render(fmt.Sprintf("%d done", len(completed))))
	sb.WriteString("  ")
	sb.WriteString(m.progress.ViewAs(completionPercent))
	sb.WriteString("\n")

	// Divider
	sb.WriteString(dimStyle.Render(" " + strings.Repeat("─", width-2)) + "\n")

	linesUsed := 2

	// Calculate available space for pending vs completed
	completedHeader := 1 // divider line
	maxCompletedLines := len(completed)
	if maxCompletedLines > 5 {
		maxCompletedLines = 5 // cap completed display
	}

	availableForPending := height - linesUsed
	if len(completed) > 0 {
		// Reserve space for completed section
		completedSpace := completedHeader + 1 + maxCompletedLines // header + label + items
		availableForPending = height - linesUsed - completedSpace
		if availableForPending < 3 {
			availableForPending = 3
		}
	}

	// Render pending tasks
	sourceWidth := 10
	taskWidth := width - 12 - sourceWidth // num(3) + icon(3) + padding + source

	for i, task := range pending {
		if i >= availableForPending {
			remaining := len(pending) - i
			sb.WriteString(dimStyle.Render(fmt.Sprintf("  ... +%d more", remaining)) + "\n")
			linesUsed++
			break
		}

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
			content = content[:taskWidth-1] + "…"
		}

		// Source (just the date part, trimmed)
		source := task.SourceFile
		if len(source) > sourceWidth {
			source = source[:sourceWidth]
		}

		numStr := fmt.Sprintf("%2d", i+1)
		icon := " ○ "
		sourceStr := lipgloss.NewStyle().
			Foreground(colorOverlay).
			Render(fmt.Sprintf("%*s", sourceWidth, source))

		if isSelected {
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
		linesUsed++
	}

	// Completed tasks section
	if len(completed) > 0 {
		// Fill gap before completed section
		gap := height - linesUsed - 1 - maxCompletedLines - 1
		if gap > 0 {
			sb.WriteString(strings.Repeat("\n", gap))
		}

		// Completed divider
		completedLabel := lipgloss.NewStyle().Foreground(colorOverlay).Render("─Completed")
		divLine := lipgloss.NewStyle().Foreground(colorSurface1).
			Render("─") + completedLabel +
			lipgloss.NewStyle().Foreground(colorSurface1).
				Render(strings.Repeat("─", max(0, width-lipgloss.Width("─Completed")-1)))
		sb.WriteString(divLine + "\n")

		for i, task := range completed {
			if i >= maxCompletedLines {
				remaining := len(completed) - i
				sb.WriteString(dimStyle.Render(fmt.Sprintf("  ... +%d more", remaining)) + "\n")
				break
			}

			content := task.Content
			// Truncate
			if len(content) > taskWidth {
				content = content[:taskWidth-1] + "…"
			}

			checkStyle := lipgloss.NewStyle().Foreground(colorGreen)
			contentStyle := lipgloss.NewStyle().Foreground(colorOverlay).Strikethrough(true)

			sb.WriteString(fmt.Sprintf(" %s %s\n",
				checkStyle.Render(" ✓"),
				contentStyle.Render(content)))
		}
	}

	return sb.String()
}

// renderEisenhowerMatrix renders a 2x2 Eisenhower matrix with tasks sorted into quadrants.
func (m AppModel) renderEisenhowerMatrix(width, height int) string {
	pending := filterPending(m.tasks)

	// Sort tasks into quadrants
	quadrants := map[string][]quadrantTask{
		"#do":        {},
		"#schedule":  {},
		"#delegate":  {},
		"#eliminate": {},
	}
	var untagged []quadrantTask

	for i, task := range pending {
		tag := extractEisenhowerTag(task.Content)
		qt := quadrantTask{content: task.Content, index: i}
		if tag != "" {
			quadrants[tag] = append(quadrants[tag], qt)
		} else {
			untagged = append(untagged, qt)
		}
	}

	// Matrix header
	matrixLabel := lipgloss.NewStyle().Foreground(colorOverlay).Render("─Eisenhower Matrix")
	header := lipgloss.NewStyle().Foreground(colorSurface1).
		Render("─") + matrixLabel +
		lipgloss.NewStyle().Foreground(colorSurface1).
			Render(strings.Repeat("─", max(0, width-lipgloss.Width("─Eisenhower Matrix")-1)))

	// Calculate quadrant sizes
	halfWidth := (width - 1) / 2 // -1 for middle divider
	quadrantHeight := (height - 3) / 2 // -1 header, -1 mid divider, -1 for bottom labels
	if quadrantHeight < 3 {
		quadrantHeight = 3
	}

	// Render each quadrant
	topLeft := renderQuadrant("#do", "Do", quadrants["#do"], m.taskCursor, halfWidth, quadrantHeight)
	topRight := renderQuadrant("#schedule", "Schedule", quadrants["#schedule"], m.taskCursor, width-halfWidth-1, quadrantHeight)
	botLeft := renderQuadrant("#delegate", "Delegate", quadrants["#delegate"], m.taskCursor, halfWidth, quadrantHeight)
	botRight := renderQuadrant("#eliminate", "Eliminate", quadrants["#eliminate"], m.taskCursor, width-halfWidth-1, quadrantHeight)

	// Mid-column divider
	midVert := lipgloss.NewStyle().Foreground(colorSurface1).Render("│")

	// Build rows
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topLeft, midVert, topRight)
	botRow := lipgloss.JoinHorizontal(lipgloss.Top, botLeft, midVert, botRight)

	// Horizontal mid-divider
	midDiv := lipgloss.NewStyle().Foreground(colorSurface1).
		Render(strings.Repeat("─", halfWidth) + "┼" + strings.Repeat("─", width-halfWidth-1))

	var sb strings.Builder
	sb.WriteString(header + "\n")
	sb.WriteString(topRow + "\n")
	sb.WriteString(midDiv + "\n")
	sb.WriteString(botRow)

	// Untagged count at bottom
	if len(untagged) > 0 {
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render(fmt.Sprintf(" %d tasks without tag", len(untagged))))
	}

	return sb.String()
}

type quadrantTask struct {
	content string
	index   int
}

// renderQuadrant renders a single Eisenhower quadrant.
func renderQuadrant(tag, label string, qTasks []quadrantTask, cursor int, width, height int) string {
	dc, ok := eisenhowerDisplayColors[tag]
	if !ok {
		dc = colorOverlay
	}

	tagStyle := lipgloss.NewStyle().Foreground(dc).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(colorOverlay)

	// Header: colored label + count
	headerLine := " " + tagStyle.Render(label)
	if len(qTasks) > 0 {
		headerLine += " " + countStyle.Render(fmt.Sprintf("(%d)", len(qTasks)))
	}

	var lines []string
	lines = append(lines, headerLine)

	taskLines := height - 1 // -1 for header
	if len(qTasks) == 0 {
		lines = append(lines, dimStyle.Render("  ─"))
	} else {
		for i, qt := range qTasks {
			if i >= taskLines-1 && i < len(qTasks)-1 {
				remaining := len(qTasks) - i
				lines = append(lines, dimStyle.Render(fmt.Sprintf("  +%d more", remaining)))
				break
			}

			// Clean content: remove the tag from display since quadrant already shows it
			content := qt.content
			content = strings.ReplaceAll(content, tag, "")
			content = strings.TrimSpace(content)

			// Truncate
			maxContentWidth := width - 4
			if len(content) > maxContentWidth {
				content = content[:maxContentWidth-1] + "…"
			}

			prefix := "  · "
			if qt.index == cursor {
				prefix = lipgloss.NewStyle().Foreground(colorBlue).Bold(true).Render(" ▸ ")
				content = lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(content)
			} else {
				content = lipgloss.NewStyle().Foreground(colorSubtext).Render(content)
			}

			lines = append(lines, prefix+content)
		}
	}

	// Pad to height
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(strings.Join(lines, "\n"))
}

// renderTasksContent returns task content for viewport (fallback).
func (m AppModel) renderTasksContent() string {
	return m.renderTaskList(m.width-4, m.height-7)
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
