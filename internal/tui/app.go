package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/config"
	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
)

const (
	tabNotes = 0
	tabTasks = 1
	tabFiles = 2
)

// AppModel is the root Bubble Tea model.
type AppModel struct {
	// Sub-models
	input    textinput.Model
	viewport viewport.Model
	editor   textarea.Model
	help     help.Model
	spinner  spinner.Model
	progress progress.Model
	fileList list.Model
	keys     KeyMap

	// State
	activeTab      int
	vaultPath      string
	currentFile    string
	fileContent    string
	lastInserted   int
	eisenhowerTags map[string]string
	tasks          []tasks.Task
	statusText     string
	loading        bool
	editMode       bool
	taskCursor     int
	fileListReady  bool

	// Layout
	width, height int
	ready         bool
}

// NewApp creates a new application model.
func NewApp(vaultPath, filePath, fileContent string) AppModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Focus()
	ti.CharLimit = 500

	tags, _ := config.GetEisenhowerTags()

	h := help.New()
	h.ShowAll = false

	s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	s.Style = lipgloss.NewStyle().Foreground(colorBlue)

	ed := textarea.New()
	ed.Prompt = "│ "
	ed.ShowLineNumbers = true
	ed.CharLimit = 0
	ed.Blur()

	prog := progress.New(
		progress.WithWidth(20),
		progress.WithoutPercentage(),
	)
	prog.FullColor = colorGreen
	prog.EmptyColor = colorSurface1

	return AppModel{
		input:          ti,
		editor:         ed,
		help:           h,
		spinner:        s,
		progress:       prog,
		keys:           DefaultKeyMap(),
		activeTab:      tabNotes,
		vaultPath:      vaultPath,
		currentFile:    filePath,
		fileContent:    fileContent,
		eisenhowerTags: tags,
	}
}

func (m AppModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(
		loadTasksCmd(m.vaultPath),
		m.spinner.Tick,
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)

		viewportHeight := m.height - 5 // tab bar + border(2) + sep/input + status
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

		if m.fileListReady {
			m.fileList.SetSize(m.width-2, viewportHeight)
		}
		if m.editMode {
			m.editor.SetWidth(m.width - 4)
			m.editor.SetHeight(viewportHeight)
		}

	case tea.KeyPressMsg:
		// Edit mode: forward everything to textarea except Esc and Ctrl+S
		if m.editMode {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case msg.String() == "escape":
				cmd := m.exitEditMode()
				if cmd != nil {
					m.loading = true
					cmds = append(cmds, cmd, m.spinner.Tick)
				}
				return m, tea.Batch(cmds...)
			case msg.String() == "ctrl+s":
				m.fileContent = m.editor.Value()
				m.loading = true
				cmds = append(cmds, saveFileCmd(m.currentFile, m.fileContent), m.spinner.Tick)
				m.statusText = "Saved"
				return m, tea.Batch(cmds...)
			default:
				var edCmd tea.Cmd
				m.editor, edCmd = m.editor.Update(msg)
				if edCmd != nil {
					cmds = append(cmds, edCmd)
				}
				return m, tea.Batch(cmds...)
			}
		}

		// Tasks tab: j/k navigation + Enter to complete
		if m.activeTab == tabTasks {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Tab):
				m.activeTab = (m.activeTab + 1) % 3
				m.viewport.SetContent(m.renderContent())
				if m.activeTab == tabFiles && !m.fileListReady {
					m.loading = true
					cmds = append(cmds, m.loadFileList(), m.spinner.Tick)
				}
				return m, tea.Batch(cmds...)
			case key.Matches(msg, m.keys.Submit):
				cmd := m.handleTaskTableSelection()
				if cmd != nil {
					m.loading = true
					cmds = append(cmds, cmd, m.spinner.Tick)
				}
				return m, tea.Batch(cmds...)
			case msg.String() == "j" || msg.String() == "down":
				m.moveTaskCursor(1)
				m.viewport.SetContent(m.renderContent())
				return m, nil
			case msg.String() == "k" || msg.String() == "up":
				m.moveTaskCursor(-1)
				m.viewport.SetContent(m.renderContent())
				return m, nil
			}
			return m, tea.Batch(cmds...)
		}

		// If files tab is active and not filtering, handle tab-level keys
		if m.activeTab == tabFiles && m.fileListReady {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Tab):
				m.activeTab = (m.activeTab + 1) % 3
				m.viewport.SetContent(m.renderContent())
				if m.activeTab == tabTasks {
					m.loading = true
					cmds = append(cmds, loadTasksCmd(m.vaultPath), m.spinner.Tick)
				}
				return m, tea.Batch(cmds...)
			case msg.String() == "enter":
				cmd := m.handleFileSelection()
				if cmd != nil {
					m.loading = true
					cmds = append(cmds, cmd, m.spinner.Tick)
				}
				return m, tea.Batch(cmds...)
			}

			// Forward all other keys to the list model
			var listCmd tea.Cmd
			m.fileList, listCmd = m.fileList.Update(msg)
			if listCmd != nil {
				cmds = append(cmds, listCmd)
			}
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			m.activeTab = (m.activeTab + 1) % 3
			m.viewport.SetContent(m.renderContent())
			if m.activeTab == tabTasks {
				m.loading = true
				cmds = append(cmds, loadTasksCmd(m.vaultPath), m.spinner.Tick)
			} else if m.activeTab == tabFiles && !m.fileListReady {
				m.loading = true
				cmds = append(cmds, m.loadFileList(), m.spinner.Tick)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Submit):
			value := m.input.Value()
			if strings.TrimSpace(value) != "" {
				cmd := m.handleInput(value)
				if cmd != nil {
					m.loading = true
					cmds = append(cmds, cmd, m.spinner.Tick)
				}
			}
			m.input.Reset()
			m.viewport.SetContent(m.renderContent())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Clear):
			m.input.Reset()
			return m, nil

		case msg.String() == "e":
			// Enter edit mode only when input is empty and on notes tab
			if m.activeTab == tabNotes && m.input.Value() == "" {
				cmd := m.enterEditMode()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}
		}

	case tea.MouseClickMsg:
		if msg.Y == 0 {
			// Tab bar click detection
			if msg.X < 14 {
				m.activeTab = tabNotes
			} else if msg.X < 22 {
				m.activeTab = tabTasks
				m.loading = true
				cmds = append(cmds, loadTasksCmd(m.vaultPath), m.spinner.Tick)
			} else {
				m.activeTab = tabFiles
				if !m.fileListReady {
					m.loading = true
					cmds = append(cmds, m.loadFileList(), m.spinner.Tick)
				}
			}
			m.viewport.SetContent(m.renderContent())
			return m, tea.Batch(cmds...)
		}

	case spinner.TickMsg:
		if m.loading {
			var spinCmd tea.Cmd
			m.spinner, spinCmd = m.spinner.Update(msg)
			cmds = append(cmds, spinCmd)
		}

	case SavedMsg:
		m.loading = false
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Error saving: %v", msg.Err)
		} else {
			m.statusText = lipgloss.NewStyle().Foreground(colorGreen).Render("✓ Saved")
			if m.lastInserted > 0 {
				vpHeight := m.viewport.Height()
				yOffset := m.viewport.YOffset()
				insertedIdx := m.lastInserted - 1
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
		m.loading = false
		if msg.Err == nil {
			m.currentFile = msg.Path
			m.fileContent = msg.Content
			m.activeTab = tabNotes
			m.viewport.SetContent(m.renderContent())
		}

	case TasksLoadedMsg:
		m.loading = false
		if msg.Err == nil {
			m.tasks = msg.Tasks
			pending := filterPending(m.tasks)
			if m.taskCursor >= len(pending) {
				m.taskCursor = max(0, len(pending)-1)
			}
			m.viewport.SetContent(m.renderContent())
		}

	case TaskCompletedMsg:
		if msg.Err == nil {
			m.loading = true
			cmds = append(cmds, loadTasksCmd(m.vaultPath), m.spinner.Tick)
		} else {
			m.loading = false
			m.statusText = fmt.Sprintf("Error: %v", msg.Err)
		}

	case LoggedMsg:
		// Silent

	case FileListMsg:
		m.loading = false
		if len(msg.Files) > 0 {
			viewportHeight := m.height - 6
			if viewportHeight < 1 {
				viewportHeight = 10
			}
			m.fileList = newFileList(msg.Files, m.width-2, viewportHeight)
			m.fileListReady = true
		} else {
			m.statusText = "No markdown files found"
		}

	case StatusMsg:
		m.statusText = msg.Text

	case tea.FocusMsg:
		// Terminal regained focus — no action needed

	case tea.BlurMsg:
		// Terminal lost focus — auto-save if we have content
		if m.currentFile != "" && m.fileContent != "" {
			if m.editMode {
				m.fileContent = m.editor.Value()
			}
			cmds = append(cmds, saveFileCmd(m.currentFile, m.fileContent))
		}
	}

	// Update text input (only when not on files tab)
	if m.activeTab != tabFiles {
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)
		if inputCmd != nil {
			cmds = append(cmds, inputCmd)
		}
	}

	// Update viewport (only when not on files tab)
	if m.activeTab != tabFiles {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		if vpCmd != nil {
			cmds = append(cmds, vpCmd)
		}
	}

	// Update file list when on files tab
	if m.activeTab == tabFiles && m.fileListReady {
		var listCmd tea.Cmd
		m.fileList, listCmd = m.fileList.Update(msg)
		if listCmd != nil {
			cmds = append(cmds, listCmd)
		}
	}

	// Update editor when in edit mode (for cursor blink etc.)
	if m.editMode {
		var edCmd tea.Cmd
		m.editor, edCmd = m.editor.Update(msg)
		if edCmd != nil {
			cmds = append(cmds, edCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing...")
	}

	tabs := []string{"Daily Note", "Tasks", "Files"}
	tabBar := RenderTabBar(tabs, m.activeTab, m.width)

	// Main content area with active/inactive borders
	var mainContent string
	switch {
	case m.activeTab == tabNotes && m.editMode:
		title := BorderWithTitle("EDITING: "+filepath.Base(m.currentFile), m.width-2, true)
		body := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			BorderForeground(colorBlue).
			Width(m.width - 2).
			Render(m.editor.View())
		mainContent = title + "\n" + body
	case m.activeTab == tabFiles && m.fileListReady:
		mainContent = m.fileList.View()
	default:
		border := activeBorderStyle
		vpContent := m.viewport.View()

		// Show centered spinner for initial heavy loads
		if m.loading && m.activeTab == tabTasks && len(m.tasks) == 0 {
			viewportHeight := m.height - 5
			if viewportHeight < 3 {
				viewportHeight = 3
			}
			vpContent = lipgloss.Place(
				m.width-4, viewportHeight,
				lipgloss.Center, lipgloss.Center,
				m.spinner.View()+" Loading tasks...",
			)
		}

		mainContent = border.
			Width(m.width - 2).
			Render(vpContent)
	}

	// Status bar
	statusBar := m.buildStatusBar()

	var result string
	if m.activeTab == tabNotes && !m.editMode {
		// Notes view mode: input line with mode pill
		sep := separatorStyle.Render(strings.Repeat("─", m.width))
		modePill := DetectInputMode(m.input.Value())
		inputLine := " " + modePill + " " + inputPromptStyle.Render("> ") + m.input.View()
		result = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			mainContent,
			sep,
			inputLine,
			statusBar,
		)
	} else if m.editMode {
		editStatus := statusBarStyle.Width(m.width).Render(
			" " + renderEditModeIndicator() + "  esc save & exit · ctrl+s save · ctrl+c quit")
		result = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			mainContent,
			editStatus,
		)
	} else {
		result = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			mainContent,
			statusBar,
		)
	}

	v := tea.NewView(result)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.ReportFocus = true
	v.WindowTitle = "Obsidian: " + filepath.Base(m.currentFile)
	return v
}

// handleInput routes user input to the appropriate handler.
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

// renderContent builds the viewport content based on active tab.
func (m AppModel) renderContent() string {
	switch m.activeTab {
	case tabNotes:
		return m.renderNotesContent()
	case tabTasks:
		return m.renderTasksContent()
	default:
		return ""
	}
}

func (m AppModel) buildStatusBar() string {
	prefix := ""
	if m.loading {
		prefix = m.spinner.View() + " "
	}

	var left, right string

	if m.statusText != "" {
		left = prefix + m.statusText
	} else {
		switch m.activeTab {
		case tabNotes:
			words := len(regexp.MustCompile(`\S+`).FindAllString(m.fileContent, -1))
			sections := len(regexp.MustCompile(`(?m)^## `).FindAllString(m.fileContent, -1))
			filename := filepath.Base(m.currentFile)
			left = fmt.Sprintf("%s %s  %d words  %d sections", prefix, filename, words, sections)
			right = "e edit · tab switch · /help "
		case tabTasks:
			left = prefix + " j/k navigate · Enter complete"
			right = "tab switch "
		case tabFiles:
			left = prefix + " Type to filter · Enter open"
			right = "tab switch "
		default:
			left = prefix
		}
	}

	leftRendered := statusBarStyle.Render(left)
	rightRendered := statusBarDimStyle.Render(right)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	filler := statusBarStyle.Render(strings.Repeat(" ", gap))

	return leftRendered + filler + rightRendered
}

// Run starts the TUI application.
func Run(vaultPath, filePath, fileContent string) error {
	model := NewApp(vaultPath, filePath, fileContent)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}
