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
	"charm.land/bubbles/v2/spinner"
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
	help     help.Model
	spinner  spinner.Model
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
	fileListReady  bool

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

	h := help.New()
	h.ShowAll = false

	s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	s.Style = lipgloss.NewStyle().Foreground(colorCyan)

	return AppModel{
		input:          ti,
		help:           h,
		spinner:        s,
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

		viewportHeight := m.height - 6
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

	case tea.KeyPressMsg:
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
			m.statusText = ""
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
			if m.activeTab == tabTasks {
				m.viewport.SetContent(m.renderContent())
			}
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

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing...")
	}

	tabs := []string{"Daily Note", "Tasks", "Files"}
	tabBar := RenderTabBar(tabs, m.activeTab, m.width)

	// Main content area
	var mainContent string
	if m.activeTab == tabFiles && m.fileListReady {
		mainContent = m.fileList.View()
	} else {
		mainContent = borderStyle.
			Width(m.width - 2).
			Render(m.viewport.View())
	}

	// Status bar
	statusContent := m.buildStatusBar()
	statusBar := statusBarStyle.Width(m.width).Render(statusContent)

	// Help bar
	helpBar := m.help.View(m.keys)

	var result string
	if m.activeTab == tabFiles {
		// Files tab: no input line, just list
		result = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			mainContent,
			statusBar,
			helpBar,
		)
	} else {
		// Notes/Tasks tabs: include input and separator
		sep := separatorStyle.Render(strings.Repeat("─", m.width))
		inputLine := inputPromptStyle.Render("> ") + m.input.View()
		result = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			mainContent,
			sep,
			inputLine,
			statusBar,
			helpBar,
		)
	}

	v := tea.NewView(result)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
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

	if m.statusText != "" {
		return prefix + m.statusText
	}

	switch m.activeTab {
	case tabNotes:
		words := len(regexp.MustCompile(`\S+`).FindAllString(m.fileContent, -1))
		sections := len(regexp.MustCompile(`(?m)^## `).FindAllString(m.fileContent, -1))
		filename := filepath.Base(m.currentFile)
		return fmt.Sprintf("%s %s | %d words | %d sections", prefix, filename, words, sections)
	case tabTasks:
		pending := len(filterPending(m.tasks))
		completed := len(filterCompleted(m.tasks))
		total := len(m.tasks)
		return fmt.Sprintf("%s Tasks | %d pending | %d completed | %d total", prefix, pending, completed, total)
	case tabFiles:
		return prefix + " Files | Type to filter, Enter to open"
	default:
		return prefix
	}
}

// Run starts the TUI application.
func Run(vaultPath, filePath, fileContent string) error {
	model := NewApp(vaultPath, filePath, fileContent)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}
