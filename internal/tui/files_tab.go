package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"

	"github.com/charmbracelet/glamour"

	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

// FileItem implements list.Item and list.DefaultItem for the file browser.
type FileItem struct {
	name string
}

func (f FileItem) FilterValue() string { return f.name }
func (f FileItem) Title() string       { return f.name }
func (f FileItem) Description() string { return "" }

// newFileList creates a list.Model for file browsing.
func newFileList(files []string, width, height int) list.Model {
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = FileItem{name: f}
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false

	l := list.New(items, delegate, width, height)
	l.Title = "Vault Files"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return l
}

// loadFileList loads the file list asynchronously.
func (m *AppModel) loadFileList() tea.Cmd {
	return func() tea.Msg {
		files, err := vault.ListMarkdownFiles(m.vaultPath)
		if err != nil {
			return StatusMsg{Text: fmt.Sprintf("Error listing files: %v", err)}
		}
		return FileListMsg{Files: files}
	}
}

// loadFilePreview loads preview content for the selected file.
func (m *AppModel) loadFilePreview() tea.Cmd {
	selected := m.fileList.SelectedItem()
	if selected == nil {
		return nil
	}

	fileItem, ok := selected.(FileItem)
	if !ok {
		return nil
	}

	name := fileItem.name
	vaultPath := m.vaultPath

	return func() tea.Msg {
		filePath := filepath.Join(vaultPath, name)

		content, err := vault.ReadFile(filePath)
		if err != nil {
			return FilePreviewMsg{Name: name, Content: "Error reading file"}
		}

		// Gather metadata
		stat, _ := os.Stat(filePath)
		modTime := ""
		size := ""
		if stat != nil {
			modTime = stat.ModTime().Format("2006-01-02 15:04")
			size = formatFileSize(stat.Size())
		}

		words := len(regexp.MustCompile(`\S+`).FindAllString(content, -1))
		lines := strings.Count(content, "\n") + 1

		// Extract sections
		var sections []string
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "## ") {
				sections = append(sections, strings.TrimPrefix(line, "## "))
			}
		}

		// Extract tags
		var tags []string
		tagRe := regexp.MustCompile(`#\w+`)
		for _, match := range tagRe.FindAllString(content, -1) {
			if match != "#daily" && match != "#inbox" {
				// Deduplicate
				found := false
				for _, t := range tags {
					if t == match {
						found = true
						break
					}
				}
				if !found {
					tags = append(tags, match)
				}
			}
		}
		// Always include #daily and #inbox if present
		if strings.Contains(content, "#daily") {
			tags = append([]string{"#daily"}, tags...)
		}
		if strings.Contains(content, "#inbox") {
			tags = append(tags, "#inbox")
		}

		return FilePreviewMsg{
			Name:      name,
			Content:   content,
			WordCount: words,
			LineCount: lines,
			ModTime:   modTime,
			Size:      size,
			Sections:  sections,
			Tags:      tags,
		}
	}
}

// handleFileSelection opens the selected file from the list.
func (m *AppModel) handleFileSelection() tea.Cmd {
	selected := m.fileList.SelectedItem()
	if selected == nil {
		return nil
	}

	fileItem, ok := selected.(FileItem)
	if !ok {
		return nil
	}

	filePath := filepath.Join(m.vaultPath, fileItem.name)
	m.activeTab = tabNotes
	return loadFileCmd(filePath)
}

// renderFilePreview renders the Glamour preview of the selected file.
func (m AppModel) renderFilePreview(width, height int) string {
	if width < 10 {
		width = 10
	}
	if height < 3 {
		height = 3
	}

	if m.filePreviewContent == "" {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			dimStyle.Render("Select a file to preview"))
	}

	content := stripFrontmatter(m.filePreviewContent)

	// Truncate source before rendering
	srcLines := strings.Split(content, "\n")
	if len(srcLines) > height {
		srcLines = srcLines[:height]
		content = strings.Join(srcLines, "\n")
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	// Hard-truncate rendered output to fit height
	outLines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	if len(outLines) > height {
		outLines = outLines[:height-1]
		outLines = append(outLines, dimStyle.Render("  ..."))
	}

	return strings.Join(outLines, "\n")
}

// renderFileDetails renders the metadata panel for the selected file.
func (m AppModel) renderFileDetails(width, height int) string {
	if m.filePreviewName == "" {
		return dimStyle.Render("  No file selected")
	}

	labelStyle := lipgloss.NewStyle().Foreground(colorOverlay)
	valueStyle := lipgloss.NewStyle().Foreground(colorSubtext)
	nameStyle := lipgloss.NewStyle().Foreground(colorText).Bold(true)

	var lines []string

	lines = append(lines, nameStyle.Render(" "+m.filePreviewName))
	lines = append(lines, "")

	if m.filePreviewMeta.ModTime != "" {
		lines = append(lines, labelStyle.Render(" Modified ")+ valueStyle.Render(m.filePreviewMeta.ModTime))
	}
	if m.filePreviewMeta.Size != "" {
		lines = append(lines, labelStyle.Render(" Size     ")+ valueStyle.Render(m.filePreviewMeta.Size))
	}
	lines = append(lines, labelStyle.Render(" Words    ")+ valueStyle.Render(fmt.Sprintf("%d", m.filePreviewMeta.WordCount)))
	lines = append(lines, labelStyle.Render(" Lines    ")+ valueStyle.Render(fmt.Sprintf("%d", m.filePreviewMeta.LineCount)))

	if len(m.filePreviewMeta.Sections) > 0 && len(lines) < height-3 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render(" Sections"))
		for _, s := range m.filePreviewMeta.Sections {
			if len(lines) >= height-1 {
				break
			}
			lines = append(lines, lipgloss.NewStyle().Foreground(colorTeal).Render(" · ")+valueStyle.Render(s))
		}
	}

	if len(m.filePreviewMeta.Tags) > 0 && len(lines) < height-2 {
		lines = append(lines, "")
		tagLine := " "
		for i, t := range m.filePreviewMeta.Tags {
			tagLine += lipgloss.NewStyle().Foreground(colorLavender).Render(t)
			if i < len(m.filePreviewMeta.Tags)-1 {
				tagLine += " "
			}
		}
		lines = append(lines, tagLine)
	}

	// Truncate to height
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// renderFilesTab renders the complete split-pane files view.
func (m AppModel) renderFilesTab() string {
	totalHeight := m.height - 7 // tab(2) + outer border(2) + status(1) + padding
	listWidth := (m.width - 4) * 55 / 100 // account for outer border
	rightWidth := m.width - 4 - listWidth - 1 // outer border(2) + divider(1)

	if rightWidth < 25 {
		return m.fileList.View()
	}

	previewHeight := totalHeight * 65 / 100
	detailsHeight := totalHeight - previewHeight - 1 // 1 for horizontal divider

	if previewHeight < 5 {
		previewHeight = 5
	}
	if detailsHeight < 4 {
		detailsHeight = 4
	}

	// Left pane: file list, constrained to its width/height
	leftPane := lipgloss.NewStyle().
		Width(listWidth).
		Height(totalHeight).
		Render(m.fileList.View())

	// Vertical divider between left and right
	vDivider := lipgloss.NewStyle().
		Foreground(colorSurface1).
		Width(1).
		Height(totalHeight).
		Render(strings.Repeat("│\n", totalHeight))

	// Right top: preview content
	previewLabel := lipgloss.NewStyle().Foreground(colorOverlay).Render("─Preview")
	previewTopBorder := lipgloss.NewStyle().Foreground(colorSurface1).
		Render("─") + previewLabel +
		lipgloss.NewStyle().Foreground(colorSurface1).
			Render(strings.Repeat("─", max(0, rightWidth-lipgloss.Width("─Preview")-1)))

	previewContent := m.renderFilePreview(rightWidth-1, previewHeight-1)
	previewPane := previewTopBorder + "\n" +
		lipgloss.NewStyle().
			Width(rightWidth).
			Height(previewHeight - 1).
			Render(previewContent)

	// Horizontal divider between preview and details
	detailsLabel := lipgloss.NewStyle().Foreground(colorOverlay).Render("─Details")
	hDivider := lipgloss.NewStyle().Foreground(colorSurface1).
		Render("─") + detailsLabel +
		lipgloss.NewStyle().Foreground(colorSurface1).
			Render(strings.Repeat("─", max(0, rightWidth-lipgloss.Width("─Details")-1)))

	// Right bottom: details content
	detailsContent := m.renderFileDetails(rightWidth-1, detailsHeight)
	detailsPane := hDivider + "\n" +
		lipgloss.NewStyle().
			Width(rightWidth).
			Height(detailsHeight).
			Render(detailsContent)

	// Compose right side
	rightPane := lipgloss.JoinVertical(lipgloss.Left, previewPane, detailsPane)

	inner := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, vDivider, rightPane)

	return activeBorderStyle.
		Width(m.width - 2).
		Render(inner)
}

// filePreviewMeta holds metadata about the previewed file.
type filePreviewMetadata struct {
	WordCount int
	LineCount int
	ModTime   string
	Size      string
	Sections  []string
	Tags      []string
}

func formatFileSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}
