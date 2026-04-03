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

// FileItem implements list.Item for the file/directory browser.
type FileItem struct {
	name  string
	isDir bool
}

func (f FileItem) FilterValue() string { return f.name }

func (f FileItem) Title() string {
	if f.name == ".." {
		return "↩ .."
	}
	if f.isDir {
		return "📁 " + f.name
	}
	return "   " + f.name
}

func (f FileItem) Description() string { return "" }

// newFileList creates a list.Model for directory browsing.
func newFileList(entries []vault.DirEntry, currentDir string, atRoot bool, width, height int) list.Model {
	var items []list.Item

	// Add parent directory entry when not at vault root
	if !atRoot {
		items = append(items, FileItem{name: "..", isDir: true})
	}

	for _, e := range entries {
		items = append(items, FileItem{name: e.Name, isDir: e.IsDir})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false

	l := list.New(items, delegate, width, height)
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	// Show current directory as title
	if currentDir == "" || currentDir == "." {
		l.Title = "/"
	} else {
		l.Title = "/" + currentDir
	}

	return l
}

// AllFilesMsg is sent with all markdown files for fuzzy search mode.
type AllFilesMsg struct {
	Files []string
}

// loadAllFiles loads all markdown files recursively for fuzzy search.
func (m *AppModel) loadAllFiles() tea.Cmd {
	rootPath := m.vaultRootPath
	return func() tea.Msg {
		files, err := vault.ListMarkdownFiles(rootPath)
		if err != nil {
			return StatusMsg{Text: fmt.Sprintf("Error listing files: %v", err)}
		}
		return AllFilesMsg{Files: files}
	}
}

// loadFileList loads the directory contents asynchronously.
func (m *AppModel) loadFileList() tea.Cmd {
	dir := m.currentDir
	rootPath := m.vaultRootPath

	return func() tea.Msg {
		fullPath := filepath.Join(rootPath, dir)
		entries, err := vault.ListDirectory(fullPath)
		if err != nil {
			return StatusMsg{Text: fmt.Sprintf("Error listing directory: %v", err)}
		}
		return FileListMsg{Entries: entries, Dir: dir}
	}
}

// navigateToDir changes the current directory and reloads the file list.
func (m *AppModel) navigateToDir(dirName string) tea.Cmd {
	if dirName == ".." {
		// Go up one level
		if m.currentDir == "" || m.currentDir == "." {
			return nil // already at root
		}
		m.currentDir = filepath.Dir(m.currentDir)
		if m.currentDir == "." {
			m.currentDir = ""
		}
	} else {
		// Go into subdirectory
		if m.currentDir == "" {
			m.currentDir = dirName
		} else {
			m.currentDir = filepath.Join(m.currentDir, dirName)
		}
	}

	// Reset preview state
	m.filePreviewContent = ""
	m.filePreviewName = ""
	m.lastPreviewedFile = ""
	m.fileListReady = false
	m.loading = true

	return tea.Batch(m.loadFileList(), m.spinner.Tick)
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

	// Don't preview ".." entry
	if fileItem.name == ".." {
		return nil
	}

	name := fileItem.name
	currentDir := m.currentDir
	rootPath := m.vaultRootPath
	isDir := fileItem.isDir
	fuzzyMode := m.fileFuzzyMode

	return func() tea.Msg {
		var fullPath string
		if fuzzyMode {
			// In fuzzy mode, name is already relative to rootPath
			fullPath = filepath.Join(rootPath, name)
		} else if currentDir == "" {
			fullPath = filepath.Join(rootPath, name)
		} else {
			fullPath = filepath.Join(rootPath, currentDir, name)
		}

		if isDir {
			// For directories, show a summary
			folders, mdFiles, err := vault.CountDirContents(fullPath)
			if err != nil {
				return FilePreviewMsg{Name: name, Content: "Error reading directory"}
			}

			stat, _ := os.Stat(fullPath)
			modTime := ""
			if stat != nil {
				modTime = stat.ModTime().Format("2006-01-02 15:04")
			}

			return FilePreviewMsg{
				Name:      name + "/",
				Content:   fmt.Sprintf("Directory with %d folders and %d markdown files", folders, mdFiles),
				ModTime:   modTime,
				IsDir:     true,
				DirStats:  dirStats{Folders: folders, Files: mdFiles},
			}
		}

		content, err := vault.ReadFile(fullPath)
		if err != nil {
			return FilePreviewMsg{Name: name, Content: "Error reading file"}
		}

		meta := extractFileMetadata(fullPath, content)
		return FilePreviewMsg{
			Name:      name,
			Content:   content,
			WordCount: meta.WordCount,
			LineCount: meta.LineCount,
			ModTime:   meta.ModTime,
			Size:      meta.Size,
			Sections:  meta.Sections,
			Tags:      meta.Tags,
		}
	}
}

// handleFileSelection opens the selected file or navigates into a directory.
func (m *AppModel) handleFileSelection() tea.Cmd {
	selected := m.fileList.SelectedItem()
	if selected == nil {
		return nil
	}

	fileItem, ok := selected.(FileItem)
	if !ok {
		return nil
	}

	// Directory navigation (not in fuzzy mode)
	if fileItem.isDir && !m.fileFuzzyMode {
		return m.navigateToDir(fileItem.name)
	}

	// Open file in file viewer
	var filePath string
	if m.fileFuzzyMode {
		filePath = filepath.Join(m.vaultRootPath, fileItem.name)
	} else if m.currentDir == "" {
		filePath = filepath.Join(m.vaultRootPath, fileItem.name)
	} else {
		filePath = filepath.Join(m.vaultRootPath, m.currentDir, fileItem.name)
	}

	return func() tea.Msg {
		data, err := vault.ReadFile(filePath)
		return FileViewLoadedMsg{Content: data, Path: filePath, Err: err}
	}
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

	// For directories, show a simple centered summary
	if m.filePreviewMeta.IsDir {
		stats := m.filePreviewMeta.DirStats
		folderIcon := lipgloss.NewStyle().Foreground(colorYellow).Render("📁")
		title := lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(m.filePreviewName)
		detail := dimStyle.Render(fmt.Sprintf("%d folders · %d files", stats.Folders, stats.Files))
		content := lipgloss.JoinVertical(lipgloss.Center, "", folderIcon, "", title, "", detail)
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
	}

	content := stripFrontmatter(m.filePreviewContent)

	// Truncate source before rendering
	srcLines := strings.Split(content, "\n")
	if len(srcLines) > height {
		srcLines = srcLines[:height]
		content = strings.Join(srcLines, "\n")
	}

	renderer, err := newGlamourRenderer(width, false)
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

	// For directories, show directory-specific metadata
	if m.filePreviewMeta.IsDir {
		stats := m.filePreviewMeta.DirStats
		if m.filePreviewMeta.ModTime != "" {
			lines = append(lines, labelStyle.Render(" Modified  ")+valueStyle.Render(m.filePreviewMeta.ModTime))
		}
		lines = append(lines, labelStyle.Render(" Folders   ")+valueStyle.Render(fmt.Sprintf("%d", stats.Folders)))
		lines = append(lines, labelStyle.Render(" MD Files  ")+valueStyle.Render(fmt.Sprintf("%d", stats.Files)))
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render(" Enter to browse"))

		if len(lines) > height {
			lines = lines[:height]
		}
		return strings.Join(lines, "\n")
	}

	if m.filePreviewMeta.ModTime != "" {
		lines = append(lines, labelStyle.Render(" Modified ")+valueStyle.Render(m.filePreviewMeta.ModTime))
	}
	if m.filePreviewMeta.Size != "" {
		lines = append(lines, labelStyle.Render(" Size     ")+valueStyle.Render(m.filePreviewMeta.Size))
	}
	lines = append(lines, labelStyle.Render(" Words    ")+valueStyle.Render(fmt.Sprintf("%d", m.filePreviewMeta.WordCount)))
	lines = append(lines, labelStyle.Render(" Lines    ")+valueStyle.Render(fmt.Sprintf("%d", m.filePreviewMeta.LineCount)))

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

// renderFileViewContent renders the file being viewed full-screen with Glamour.
func (m AppModel) renderFileViewContent() string {
	if m.fileViewContent == "" {
		return "File is empty"
	}

	content := stripFrontmatter(m.fileViewContent)

	width := m.width - 6
	if width < 40 {
		width = 40
	}

	renderer, err := newGlamourRenderer(width, true)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	return rendered
}

// enterFileEditMode switches to textarea editing in Files tab.
func (m *AppModel) enterFileEditMode() tea.Cmd {
	m.fileEditMode = true
	m.editor.SetValue(m.fileViewContent)

	viewportHeight := m.height - 5
	if viewportHeight < 5 {
		viewportHeight = 5
	}
	m.editor.SetWidth(m.width - 4)
	m.editor.SetHeight(viewportHeight)

	return m.editor.Focus()
}

// exitFileEditMode saves and returns to file view mode.
func (m *AppModel) exitFileEditMode() tea.Cmd {
	m.fileEditMode = false
	m.fileViewContent = m.editor.Value()
	m.editor.Blur()
	m.viewport.SetContent(m.renderFileViewContent())
	return saveFileCmd(m.fileViewPath, m.fileViewContent)
}

// exitFileViewMode returns to the file browser.
func (m *AppModel) exitFileViewMode() {
	m.fileViewMode = false
	m.fileViewPath = ""
	m.fileViewContent = ""
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

// filePreviewMetadata holds metadata about the previewed file or directory.
type filePreviewMetadata struct {
	WordCount int
	LineCount int
	ModTime   string
	Size      string
	Sections  []string
	Tags      []string
	IsDir     bool
	DirStats  dirStats
}

type dirStats struct {
	Folders int
	Files   int
}

// extractFileMetadata gathers word count, line count, sections, tags, and file stats
// from a file's content and path. Pure extraction — no rendering.
func extractFileMetadata(fullPath, content string) filePreviewMetadata {
	stat, _ := os.Stat(fullPath)
	modTime := ""
	size := ""
	if stat != nil {
		modTime = stat.ModTime().Format("2006-01-02 15:04")
		size = formatFileSize(stat.Size())
	}

	words := len(regexp.MustCompile(`\S+`).FindAllString(content, -1))
	lines := strings.Count(content, "\n") + 1

	var sections []string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## ") {
			sections = append(sections, strings.TrimPrefix(line, "## "))
		}
	}

	var tags []string
	seen := map[string]bool{}
	tagRe := regexp.MustCompile(`#\w+`)
	for _, match := range tagRe.FindAllString(content, -1) {
		if !seen[match] {
			seen[match] = true
			tags = append(tags, match)
		}
	}

	return filePreviewMetadata{
		WordCount: words,
		LineCount: lines,
		ModTime:   modTime,
		Size:      size,
		Sections:  sections,
		Tags:      tags,
	}
}

// newGlamourRenderer creates a configured Glamour markdown renderer.
func newGlamourRenderer(width int, preserveNewLines bool) (*glamour.TermRenderer, error) {
	opts := []glamour.TermRendererOption{
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	}
	if preserveNewLines {
		opts = append(opts, glamour.WithPreservedNewLines())
	}
	return glamour.NewTermRenderer(opts...)
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
