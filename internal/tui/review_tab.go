package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// loadReviewPreviewIfNeeded returns a command to load preview if the cursor changed.
func (m *AppModel) loadReviewPreviewIfNeeded() tea.Cmd {
	if len(m.reviewItems) == 0 || m.reviewCursor >= len(m.reviewItems) {
		return nil
	}
	name := m.reviewItems[m.reviewCursor].Name
	if name == m.reviewLastPreviewed {
		return nil
	}
	return loadReviewPreviewCmd(m.vaultRootPath, name)
}

// renderReviewList renders the review queue as a split-pane: list left, preview right.
func (m AppModel) renderReviewList() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	itemStyle := lipgloss.NewStyle().Foreground(colorText)
	selectedStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)

	totalHeight := m.height - 7
	if totalHeight < 5 {
		totalHeight = 5
	}
	listWidth := (m.width - 4) / 2
	rightWidth := m.width - 4 - listWidth - 1 // 1 for divider

	// Left pane: review list
	var listLines []string
	listLines = append(listLines, titleStyle.Render(fmt.Sprintf(" Review Queue (%d pending)", len(m.reviewItems))))
	listLines = append(listLines, "")

	for i, item := range m.reviewItems {
		cursor := "  "
		style := itemStyle
		if i == m.reviewCursor {
			cursor = "▸ "
			style = selectedStyle
		}
		line := fmt.Sprintf(" %s%s", cursor, item.Name)
		if len(line) > listWidth-1 {
			line = line[:listWidth-4] + "..."
		}
		listLines = append(listLines, style.Render(line))
	}

	if len(m.reviewItems) == 0 {
		listLines = append(listLines, dimStyle.Render(" No pending drafts"))
	}

	leftContent := strings.Join(listLines, "\n")
	leftPane := lipgloss.NewStyle().
		Width(listWidth).
		Height(totalHeight).
		Render(leftContent)

	// Vertical divider
	vDivider := lipgloss.NewStyle().
		Foreground(colorSurface1).
		Width(1).
		Height(totalHeight).
		Render(strings.Repeat("│\n", totalHeight))

	// Right pane: preview
	previewLabel := lipgloss.NewStyle().Foreground(colorOverlay).Render("─Preview")
	previewTopBorder := lipgloss.NewStyle().Foreground(colorSurface1).
		Render("─") + previewLabel +
		lipgloss.NewStyle().Foreground(colorSurface1).
			Render(strings.Repeat("─", max(0, rightWidth-lipgloss.Width("─Preview")-1)))

	var previewContent string
	if m.reviewPreviewContent != "" {
		displayContent := stripFrontmatter(m.reviewPreviewContent)
		previewWidth := rightWidth - 2
		if previewWidth < 30 {
			previewWidth = 30
		}
		renderer, err := newGlamourRenderer(previewWidth, true)
		if err == nil {
			rendered, err := renderer.Render(displayContent)
			if err == nil {
				previewContent = rendered
			} else {
				previewContent = displayContent
			}
		} else {
			previewContent = displayContent
		}
	} else if len(m.reviewItems) > 0 {
		previewContent = dimStyle.Render(" Loading preview...")
	}

	// Truncate preview to fit height
	previewLines := strings.Split(previewContent, "\n")
	maxPreviewLines := totalHeight - 2
	if len(previewLines) > maxPreviewLines {
		previewLines = previewLines[:maxPreviewLines]
	}
	previewContent = strings.Join(previewLines, "\n")

	rightPane := previewTopBorder + "\n" +
		lipgloss.NewStyle().
			Width(rightWidth).
			Height(totalHeight - 1).
			Render(previewContent)

	inner := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, vDivider, rightPane)

	return activeBorderStyle.
		Width(m.width - 2).
		Render(inner)
}
