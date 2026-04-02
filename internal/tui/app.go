package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/config"
	"github.com/caneppelevitor/obsidian-cli/internal/content"
	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

const (
	tabNotes = 0
	tabTasks = 1
)

// AppModel is the root Bubble Tea model.
type AppModel struct {
	// Sub-models
	input    textinput.Model
	viewport viewport.Model

	// State
	activeTab      int
	vaultPath      string
	currentFile    string
	fileContent    string
	lastInserted   int // 1-based line for auto-scroll
	eisenhowerTags map[string]string
	tasks          []tasks.Task
	statusText     string

	// Layout
	width, height int
	ready         bool
}

// NewApp creates a new application model.
func NewApp(vaultPath, filePath, fileContent string) AppModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Focus()
	ti.CharLimit = 500

	tags, _ := config.GetEisenhowerTags()

	return AppModel{
		input:          ti,
		activeTab:      tabNotes,
		vaultPath:      vaultPath,
		currentFile:    filePath,
		fileContent:    fileContent,
		eisenhowerTags: tags,
		statusText:     "",
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		loadTasksCmd(m.vaultPath),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		viewportHeight := m.height - 5
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(
				viewport.WithWidth(m.width-2),
				viewport.WithHeight(viewportHeight),
			)
			m.viewport.SetContent(m.renderContent())
			m.ready = true
		} else {
			m.viewport.SetWidth(m.width - 2)
			m.viewport.SetHeight(viewportHeight)
			m.viewport.SetContent(m.renderContent())
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			m.viewport.SetContent(m.renderContent())
			if m.activeTab == tabTasks {
				cmds = append(cmds, loadTasksCmd(m.vaultPath))
			}
			return m, tea.Batch(cmds...)

		case "enter":
			value := m.input.Value()
			if strings.TrimSpace(value) != "" {
				cmd := m.handleInput(value)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			m.input.Reset()
			m.viewport.SetContent(m.renderContent())
			return m, tea.Batch(cmds...)

		case "escape":
			m.input.Reset()
			return m, nil
		}

	case SavedMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Error saving: %v", msg.Err)
		} else {
			// Auto-scroll to inserted line after save
			if m.lastInserted > 0 {
				vpHeight := m.viewport.Height()
				yOffset := m.viewport.YOffset()
				insertedIdx := m.lastInserted - 1

				// Scroll if inserted line is outside visible area
				if insertedIdx < yOffset || insertedIdx >= yOffset+vpHeight {
					target := insertedIdx - vpHeight/2
					if target < 0 {
						target = 0
					}
					m.viewport.SetYOffset(target)
				}
				m.lastInserted = 0
			}
			m.viewport.SetContent(m.renderContent())
		}

	case FileLoadedMsg:
		if msg.Err == nil {
			m.currentFile = msg.Path
			m.fileContent = msg.Content
			m.viewport.SetContent(m.renderContent())
		}

	case TasksLoadedMsg:
		if msg.Err == nil {
			m.tasks = msg.Tasks
			if m.activeTab == tabTasks {
				m.viewport.SetContent(m.renderContent())
			}
		}

	case TaskCompletedMsg:
		if msg.Err == nil {
			cmds = append(cmds, loadTasksCmd(m.vaultPath))
		} else {
			m.statusText = fmt.Sprintf("Error: %v", msg.Err)
		}

	case LoggedMsg:
		// Silent

	case FileListMsg:
		if len(msg.Files) == 0 {
			m.statusText = "No markdown files found"
		} else {
			var sb strings.Builder
			sb.WriteString("\nFiles in vault:\n")
			sb.WriteString(strings.Repeat("─", 50) + "\n")
			for i, f := range msg.Files {
				sb.WriteString(fmt.Sprintf("%3d. %s\n", i+1, f))
			}
			sb.WriteString(strings.Repeat("─", 50) + "\n")
			sb.WriteString("\nUse /open <number or name> to open a file")
			m.fileContent = sb.String()
			m.viewport.SetContent(m.renderContent())
		}

	case StatusMsg:
		m.statusText = msg.Text
	}

	// Update text input
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	if inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing...")
	}

	tabs := []string{"Daily Note", "Tasks"}
	tabBar := RenderTabBar(tabs, m.activeTab, m.width)

	// Viewport with border
	vpContent := borderStyle.
		Width(m.width - 2).
		Render(m.viewport.View())

	// Input line
	inputLine := inputPromptStyle.Render("> ") + m.input.View()

	// Status bar
	statusContent := m.buildStatusBar()
	statusBar := statusBarStyle.Width(m.width).Render(statusContent)

	// Separator
	sep := separatorStyle.Render(strings.Repeat("─", m.width))

	result := lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		vpContent,
		sep,
		inputLine,
		statusBar,
	)

	v := tea.NewView(result)
	v.AltScreen = true
	return v
}

// handleInput processes user input (content or slash commands).
func (m *AppModel) handleInput(input string) tea.Cmd {
	if strings.HasPrefix(input, "/") {
		return m.handleSlashCommand(input)
	}

	if m.activeTab == tabTasks {
		if num, err := strconv.Atoi(strings.TrimSpace(input)); err == nil {
			return m.handleTaskCompletion(num)
		}
	}

	return m.handleContentInput(input)
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
		return m.handleFileList()

	case strings.HasPrefix(cmd, "open "):
		return m.handleOpenFile(strings.TrimSpace(cmd[5:]))

	default:
		m.statusText = fmt.Sprintf("Unknown command: /%s", cmd)
		return nil
	}
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

func (m *AppModel) handleFileList() tea.Cmd {
	return func() tea.Msg {
		files, err := vault.ListMarkdownFiles(m.vaultPath)
		if err != nil {
			return StatusMsg{Text: fmt.Sprintf("Error listing files: %v", err)}
		}
		return FileListMsg{Files: files}
	}
}

func (m *AppModel) handleOpenFile(target string) tea.Cmd {
	return func() tea.Msg {
		files, err := vault.ListMarkdownFiles(m.vaultPath)
		if err != nil {
			return StatusMsg{Text: fmt.Sprintf("Error listing files: %v", err)}
		}

		var filename string
		if num, parseErr := strconv.Atoi(target); parseErr == nil && num > 0 && num <= len(files) {
			filename = files[num-1]
		} else {
			// Search by name
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

// renderContent builds the viewport content based on active tab.
func (m AppModel) renderContent() string {
	if m.activeTab == tabNotes {
		return m.renderNotesContent()
	}
	return m.renderTasksContent()
}

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

type indexedTask struct {
	task  tasks.Task
	index int
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

func (m AppModel) buildStatusBar() string {
	if m.activeTab == tabNotes {
		words := len(regexp.MustCompile(`\S+`).FindAllString(m.fileContent, -1))
		sections := len(regexp.MustCompile(`(?m)^## `).FindAllString(m.fileContent, -1))
		filename := filepath.Base(m.currentFile)
		return fmt.Sprintf(" %s | %d words | %d sections", filename, words, sections)
	}

	pending := len(filterPending(m.tasks))
	completed := len(filterCompleted(m.tasks))
	total := len(m.tasks)
	return fmt.Sprintf(" Tasks | %d pending | %d completed | %d total", pending, completed, total)
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
  Tab        Switch between Daily Note and Tasks tabs
  Ctrl+C     Exit
  Escape     Clear input

In Tasks tab:
  Type a number and press Enter to complete that task
`

// Run starts the TUI application.
func Run(vaultPath, filePath, fileContent string) error {
	model := NewApp(vaultPath, filePath, fileContent)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}
