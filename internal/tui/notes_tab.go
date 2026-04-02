package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/content"
	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

func (m AppModel) renderNotesContent() string {
	if m.fileContent == "" {
		return "File is empty"
	}

	lines := strings.Split(m.fileContent, "\n")
	var rendered []string

	for i, line := range lines {
		lineNum := lineNumberStyle.Render(fmt.Sprintf("%3d │", i+1))
		styledLine := StyleMarkdownLine(line, m.eisenhowerTags)
		rendered = append(rendered, lineNum+" "+styledLine)
	}

	// Add cheat sheet if there's spare space
	viewportHeight := m.viewport.Height()
	spareLines := viewportHeight - len(rendered)
	if spareLines >= 8 {
		cheatSheet := []string{
			"",
			cheatSheetStyle.Render("  Quick Reference:"),
			cheatSheetStyle.Render("    []  text   →  Tasks"),
			cheatSheetStyle.Render("    -   text   →  Ideas"),
			cheatSheetStyle.Render("    ?   text   →  Questions"),
			cheatSheetStyle.Render("    !   text   →  Insights"),
			cheatSheetStyle.Render("    /help      →  Show commands"),
			cheatSheetStyle.Render("    /save      →  Save file"),
		}
		blankLines := spareLines - len(cheatSheet)
		for i := 0; i < blankLines; i++ {
			rendered = append(rendered, "")
		}
		rendered = append(rendered, cheatSheet...)
	}

	return strings.Join(rendered, "\n")
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
		m.viewport.SetContent(m.renderContent())
		return nil

	case cmd == "view", cmd == "files":
		m.activeTab = tabFiles
		m.viewport.SetContent(m.renderContent())
		return m.loadFileList()

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
		if num, parseErr := fmt.Sscanf(target, "%d", new(int)); parseErr == nil && num == 1 {
			n := 0
			fmt.Sscanf(target, "%d", &n)
			if n > 0 && n <= len(files) {
				filename = files[n-1]
			}
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
  Ctrl+C     Exit
  Escape     Clear input

In Tasks tab:
  Type a number and press Enter to complete that task

In Files tab:
  Type to filter, Enter to open, Esc to go back
`
