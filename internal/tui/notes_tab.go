package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/charmbracelet/glamour"

	"github.com/caneppelevitor/obsidian-cli/internal/content"
	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

// renderNotesView renders the daily note in view mode using Glamour.
func (m AppModel) renderNotesView() string {
	if m.fileContent == "" {
		return "File is empty"
	}

	// Strip YAML frontmatter from display
	displayContent := stripFrontmatter(m.fileContent)

	width := m.width - 6
	if width < 40 {
		width = 40
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return m.renderNotesFallback()
	}

	rendered, err := renderer.Render(displayContent)
	if err != nil {
		return m.renderNotesFallback()
	}

	return rendered
}

// stripFrontmatter removes YAML frontmatter (--- blocks) from content for display.
func stripFrontmatter(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inFrontmatter := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter && i < 5 {
				// Start of frontmatter (must be near top)
				inFrontmatter = true
				continue
			} else if inFrontmatter {
				// End of frontmatter
				inFrontmatter = false
				continue
			}
		}
		if inFrontmatter {
			continue
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// renderNotesFallback renders with manual line-by-line styling (used if Glamour fails).
func (m AppModel) renderNotesFallback() string {
	lines := strings.Split(m.fileContent, "\n")
	var rendered []string

	for i, line := range lines {
		lineNum := lineNumberStyle.Render(fmt.Sprintf("%3d │", i+1))
		styledLine := StyleMarkdownLine(line, m.eisenhowerTags)
		rendered = append(rendered, lineNum+" "+styledLine)
	}

	return strings.Join(rendered, "\n")
}

// renderNotesContent dispatches to view or edit rendering.
func (m AppModel) renderNotesContent() string {
	if m.editMode {
		return "" // textarea renders directly in View(), not through viewport
	}

	rendered := m.renderNotesView()

	// Add cheat sheet if there's spare space
	viewportHeight := m.viewport.Height()
	renderedLines := strings.Count(rendered, "\n") + 1
	spareLines := viewportHeight - renderedLines
	if spareLines >= 6 {
		cheatSheet := []string{
			"",
			cheatSheetStyle.Render("  Quick Input:"),
			cheatSheetStyle.Render("    []  text → Tasks    -  text → Ideas"),
			cheatSheetStyle.Render("    ?   text → Questions !  text → Insights"),
			cheatSheetStyle.Render("    e → edit mode    /help → commands"),
		}
		rendered += strings.Join(cheatSheet, "\n")
	}

	return rendered
}

func (m *AppModel) handleContentInput(input string) tea.Cmd {
	parsed := content.ParseContentInput(input)

	if parsed == nil {
		result := content.AddContent(m.fileContent, strings.TrimSpace(input), "append")
		m.fileContent = result.NewContent
		m.lastInserted = result.InsertedLine
		return saveFileCmd(m.currentFile, m.fileContent)
	}

	result := content.AddToSection(m.fileContent, parsed.Section, parsed.FormattedContent)
	if result == nil {
		result = content.AddContent(m.fileContent, parsed.FormattedContent, "append")
	}

	m.fileContent = result.NewContent
	m.lastInserted = result.InsertedLine

	return tea.Batch(
		saveFileCmd(m.currentFile, m.fileContent),
		logEntryCmd(m.vaultPath, m.currentFile, parsed.RawContent, parsed.LogType),
	)
}

func (m *AppModel) handleSlashCommand(input string) tea.Cmd {
	cmd := strings.TrimPrefix(input, "/")
	cmd = strings.ToLower(strings.TrimSpace(cmd))

	switch {
	case cmd == "save":
		return saveFileCmd(m.currentFile, m.fileContent)

	case cmd == "exit":
		return tea.Quit

	case cmd == "daily":
		return loadFileCmd(m.currentFile)

	case cmd == "help":
		m.fileContent = helpText
		m.viewport.SetContent(m.renderNotesContent())
		return nil

	case cmd == "view", cmd == "files":
		m.activeTab = tabFiles
		if !m.fileListReady {
			m.loading = true
			return tea.Batch(m.loadFileList(), m.spinner.Tick)
		}
		return nil

	case strings.HasPrefix(cmd, "open "):
		return m.handleOpenFile(strings.TrimSpace(cmd[5:]))

	default:
		m.statusText = fmt.Sprintf("Unknown command: /%s", cmd)
		return nil
	}
}

func (m *AppModel) handleOpenFile(target string) tea.Cmd {
	return func() tea.Msg {
		files, err := vault.ListMarkdownFiles(m.vaultPath)
		if err != nil {
			return StatusMsg{Text: fmt.Sprintf("Error listing files: %v", err)}
		}

		var filename string
		var n int
		if _, scanErr := fmt.Sscanf(target, "%d", &n); scanErr == nil && n > 0 && n <= len(files) {
			filename = files[n-1]
		}
		if filename == "" {
			for _, f := range files {
				if f == target || strings.Contains(f, target) {
					filename = f
					break
				}
			}
		}

		if filename == "" {
			return StatusMsg{Text: fmt.Sprintf("File not found: %s", target)}
		}

		filePath := filepath.Join(m.vaultPath, filename)
		data, err := vault.ReadFile(filePath)
		if err != nil {
			return StatusMsg{Text: fmt.Sprintf("Error reading %s: %v", filename, err)}
		}
		return FileLoadedMsg{Content: data, Path: filePath}
	}
}

// enterEditMode switches to textarea editing.
func (m *AppModel) enterEditMode() tea.Cmd {
	m.editMode = true
	m.editor.SetValue(m.fileContent)

	viewportHeight := m.height - 5
	if viewportHeight < 5 {
		viewportHeight = 5
	}
	m.editor.SetWidth(m.width - 4)
	m.editor.SetHeight(viewportHeight)

	return m.editor.Focus()
}

// exitEditMode saves and returns to view mode.
func (m *AppModel) exitEditMode() tea.Cmd {
	m.editMode = false
	m.fileContent = m.editor.Value()
	m.editor.Blur()
	m.viewport.SetContent(m.renderNotesContent())
	return saveFileCmd(m.currentFile, m.fileContent)
}

// renderEditModeIndicator returns the edit mode label for the border.
func renderEditModeIndicator() string {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("15")).
		Bold(true).
		Padding(0, 1).
		Render(" EDITING ")
}

const helpText = `
Commands:
  /save      Save current file
  /daily     Reload daily note
  /help      Show this help
  /exit      Exit the application

Content Prefixes:
  []  text   Add a task (checkbox)
  -   text   Add an idea (bullet)
  ?   text   Add a question (bullet)
  !   text   Add an insight (bullet)

Navigation:
  Tab        Switch between tabs
  e          Enter edit mode (Daily Note)
  Ctrl+C     Exit
  Escape     Clear input / exit edit mode

In Edit Mode:
  Full cursor editing of your note
  Esc        Save and exit edit mode
  Ctrl+S     Save without exiting

In Tasks tab:
  j/k        Navigate tasks
  Enter      Complete selected task

In Files tab:
  Type to filter, Enter to open
`
