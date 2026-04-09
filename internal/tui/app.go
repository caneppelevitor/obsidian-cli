package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	"github.com/caneppelevitor/obsidian-cli/internal/content"
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
	vaultRootPath  string // root path for file browser (may differ from vaultPath)
	currentFile    string
	fileContent    string
	lastInserted   int
	eisenhowerTags map[string]string
	tasks          []tasks.Task
	statusText     string
	loading        bool
	editMode       bool
	showingHelp    bool
	taskCursor         int
	fileListReady      bool
	currentDir         string // current directory in file browser (relative to vaultRootPath)
	fileFuzzyMode      bool   // true = global fuzzy search, false = directory browse
	fileViewMode       bool   // true = viewing a file full-screen in Files tab
	fileViewPath       string // path of the file being viewed in Files tab
	fileViewContent    string // content of the file being viewed
	fileEditMode       bool   // true = editing the viewed file
	filePreviewContent string
	filePreviewName    string
	filePreviewMeta    filePreviewMetadata
	lastPreviewedFile  string

	// Compile & Status
	showingStatus        bool
	showingCompileSummary bool
	compiling            bool
	compileResult        *content.CompileResult
	vaultStatus          *content.VaultStatus
	lastCompileTime      *time.Time

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
	vaultRoot, _ := config.GetVaultRootPath()
	if vaultRoot == "" {
		vaultRoot = vaultPath
	}

	h := help.New()
	h.ShowAll = false

	s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	s.Style = lipgloss.NewStyle().Foreground(colorBlue)

	ed := textarea.New()
	ed.Prompt = "│ "
	ed.ShowLineNumbers = true
	ed.CharLimit = 0
	ed.Blur()

	// Customize editor styles: remove cursor line highlight, add text color
	edStyles := ed.Styles()
	edStyles.Focused.CursorLine = lipgloss.NewStyle()
	edStyles.Focused.CursorLineNumber = lipgloss.NewStyle().Foreground(colorBlue)
	edStyles.Focused.Text = lipgloss.NewStyle().Foreground(colorText)
	edStyles.Focused.LineNumber = lipgloss.NewStyle().Foreground(colorOverlay)
	edStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(colorSurface1)
	edStyles.Blurred.CursorLine = lipgloss.NewStyle()
	edStyles.Blurred.Text = lipgloss.NewStyle().Foreground(colorSubtext)
	edStyles.Blurred.LineNumber = lipgloss.NewStyle().Foreground(colorOverlay)
	edStyles.Blurred.Prompt = lipgloss.NewStyle().Foreground(colorSurface1)
	ed.SetStyles(edStyles)

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
		vaultRootPath:  vaultRoot,
		currentFile:    filePath,
		fileContent:    fileContent,
		eisenhowerTags: tags,
	}
}

func (m AppModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(
		loadTasksCmd(m.vaultPath),
		loadLastCompileTimeCmd(m.vaultRootPath),
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

		viewportHeight := m.height - 7 // tab(2) + border(2) + sep(1) + input(1) + status(1)
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
			listWidth := (m.width - 4) * 55 / 100
			if listWidth < 20 {
				listWidth = 20
			}
			m.fileList.SetSize(listWidth, viewportHeight-2)
		}
		if m.editMode || m.fileEditMode {
			m.editor.SetWidth(m.width - 4)
			m.editor.SetHeight(viewportHeight)
		}

	case tea.KeyPressMsg:
		// Dismiss overlays on any keypress
		if m.showingStatus {
			m.showingStatus = false
			return m, nil
		}
		if m.showingCompileSummary {
			m.showingCompileSummary = false
			return m, nil
		}

		// Edit mode: forward everything to textarea except Esc and Ctrl+S
		if m.editMode {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case msg.String() == "esc" || msg.String() == "escape":
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

		// File edit mode in Files tab
		if m.fileEditMode {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case msg.String() == "esc" || msg.String() == "escape":
				cmd := m.exitFileEditMode()
				if cmd != nil {
					m.loading = true
					cmds = append(cmds, cmd, m.spinner.Tick)
				}
				return m, tea.Batch(cmds...)
			case msg.String() == "ctrl+s":
				m.fileViewContent = m.editor.Value()
				m.loading = true
				cmds = append(cmds, saveFileCmd(m.fileViewPath, m.fileViewContent), m.spinner.Tick)
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

		// File view mode in Files tab
		if m.fileViewMode {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case msg.String() == "esc" || msg.String() == "escape":
				m.exitFileViewMode()
				return m, nil
			case msg.String() == "e":
				cmd := m.enterFileEditMode()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			case key.Matches(msg, m.keys.Tab):
				m.exitFileViewMode()
				m.activeTab = (m.activeTab + 1) % 3
				m.viewport.SetContent(m.renderContent())
				if m.activeTab == tabTasks {
					m.loading = true
					cmds = append(cmds, loadTasksCmd(m.vaultPath), m.spinner.Tick)
				}
				return m, tea.Batch(cmds...)
			default:
				// Forward scroll keys to viewport
				var vpCmd tea.Cmd
				m.viewport, vpCmd = m.viewport.Update(msg)
				if vpCmd != nil {
					cmds = append(cmds, vpCmd)
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
				m.fileFuzzyMode = false
				m.viewport.SetContent(m.renderContent())
				if m.activeTab == tabTasks {
					m.loading = true
					cmds = append(cmds, loadTasksCmd(m.vaultPath), m.spinner.Tick)
				}
				return m, tea.Batch(cmds...)
			case msg.String() == "/" && !m.fileFuzzyMode && !m.fileList.SettingFilter():
				// Enter global fuzzy search mode
				m.fileFuzzyMode = true
				m.loading = true
				cmds = append(cmds, m.loadAllFiles(), m.spinner.Tick)
				return m, tea.Batch(cmds...)
			case (msg.String() == "esc" || msg.String() == "escape") && m.fileFuzzyMode && !m.fileList.SettingFilter():
				// Exit fuzzy mode, return to directory browser
				m.fileFuzzyMode = false
				m.fileListReady = false
				m.lastPreviewedFile = ""
				m.filePreviewContent = ""
				m.filePreviewName = ""
				m.loading = true
				cmds = append(cmds, m.loadFileList(), m.spinner.Tick)
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

			// Load preview if selected file changed
			if selected := m.fileList.SelectedItem(); selected != nil {
				if fi, ok := selected.(FileItem); ok && fi.name != m.lastPreviewedFile {
					m.lastPreviewedFile = fi.name
					cmds = append(cmds, m.loadFilePreview())
				}
			}
			return m, tea.Batch(cmds...)
		}

		// Dismiss help view
		if m.showingHelp && (msg.String() == "esc" || msg.String() == "escape") {
			m.showingHelp = false
			m.viewport.SetContent(m.renderContent())
			return m, nil
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
			m.showingHelp = false
			value := m.input.Value()
			if strings.TrimSpace(value) != "" {
				cmd := m.handleInput(value)
				if cmd != nil {
					m.loading = true
					cmds = append(cmds, cmd, m.spinner.Tick)
				}
			}
			m.input.Reset()
			if !m.showingHelp {
				m.viewport.SetContent(m.renderContent())
			}
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
		if m.loading || m.compiling {
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
			if m.fileViewMode {
				m.viewport.SetContent(m.renderFileViewContent())
			} else {
				m.viewport.SetContent(m.renderContent())
			}
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
		m.currentDir = msg.Dir
		viewportHeight := m.height - 7
		if viewportHeight < 1 {
			viewportHeight = 10
		}
		listWidth := (m.width - 4) * 55 / 100
		if listWidth < 20 {
			listWidth = 20
		}
		atRoot := msg.Dir == "" || msg.Dir == "."
		m.fileList = newFileList(msg.Entries, msg.Dir, atRoot, listWidth, viewportHeight-2)
		m.fileListReady = true
		m.lastPreviewedFile = ""
		// Load preview for first item
		cmds = append(cmds, m.loadFilePreview())

	case AllFilesMsg:
		m.loading = false
		viewportHeight := m.height - 7
		if viewportHeight < 1 {
			viewportHeight = 10
		}
		listWidth := (m.width - 4) * 55 / 100
		if listWidth < 20 {
			listWidth = 20
		}
		items := make([]list.Item, len(msg.Files))
		for i, f := range msg.Files {
			items[i] = FileItem{name: f, isDir: false}
		}
		delegate := list.NewDefaultDelegate()
		delegate.SetHeight(1)
		delegate.SetSpacing(0)
		delegate.ShowDescription = false
		l := list.New(items, delegate, listWidth, viewportHeight-2)
		l.Title = "Search All Files"
		l.SetShowHelp(false)
		l.SetShowStatusBar(true)
		l.SetFilteringEnabled(true)
		m.fileList = l
		m.fileListReady = true
		m.fileFuzzyMode = true
		m.lastPreviewedFile = ""
		// Start filtering immediately
		m.fileList.FilterInput.Focus()

	case FileViewLoadedMsg:
		m.loading = false
		if msg.Err == nil {
			m.fileViewMode = true
			m.fileViewPath = msg.Path
			m.fileViewContent = msg.Content
			m.fileEditMode = false
			m.viewport.SetContent(m.renderFileViewContent())
		} else {
			m.statusText = fmt.Sprintf("Error: %v", msg.Err)
		}

	case FilePreviewMsg:
		m.filePreviewName = msg.Name
		m.filePreviewContent = msg.Content
		m.filePreviewMeta = filePreviewMetadata{
			WordCount: msg.WordCount,
			LineCount: msg.LineCount,
			ModTime:   msg.ModTime,
			Size:      msg.Size,
			Sections:  msg.Sections,
			Tags:      msg.Tags,
			IsDir:     msg.IsDir,
			DirStats:  msg.DirStats,
		}

	case LastCompileLoadedMsg:
		if msg.Err == nil {
			m.lastCompileTime = msg.Time
		}

	case VaultStatusMsg:
		m.loading = false
		if msg.Err == nil {
			m.vaultStatus = &msg.Status
			m.showingStatus = true
		} else {
			m.statusText = fmt.Sprintf("Error fetching status: %v", msg.Err)
		}

	case CompileDoneMsg:
		m.compiling = false
		m.loading = false
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Compile error: %v", msg.Err)
		} else if msg.ExitCode != 0 {
			errMsg := msg.Stderr
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
			m.statusText = fmt.Sprintf("Compile failed (exit %d): %s", msg.ExitCode, errMsg)
		} else {
			cmds = append(cmds,
				loadCompileResultCmd(m.vaultRootPath),
				loadLastCompileTimeCmd(m.vaultRootPath),
			)
		}

	case CompileResultMsg:
		if msg.Err == nil && msg.Result != nil {
			m.compileResult = msg.Result
			m.showingCompileSummary = true
		} else if msg.Err != nil {
			m.statusText = fmt.Sprintf("Error parsing compile results: %v", msg.Err)
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
	if m.editMode || m.fileEditMode {
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

	// Full-screen overlays
	if m.showingStatus {
		v := tea.NewView(m.renderStatusOverlay())
		v.AltScreen = true
		return v
	}
	if m.showingCompileSummary {
		v := tea.NewView(m.renderCompileSummary())
		v.AltScreen = true
		return v
	}

	tabs := []string{"Daily Note", "Tasks", "Files"}
	tabBar := RenderTabBar(tabs, m.activeTab, m.width)

	// Main content area with active/inactive borders
	var mainContent string
	switch {
	case m.compiling:
		viewportHeight := m.height - 5
		if viewportHeight < 3 {
			viewportHeight = 3
		}
		spinContent := lipgloss.Place(
			m.width-4, viewportHeight,
			lipgloss.Center, lipgloss.Center,
			m.spinner.View()+" Compiling vault...",
		)
		mainContent = activeBorderStyle.
			Width(m.width - 2).
			Render(spinContent)
	case m.activeTab == tabNotes && m.editMode:
		title := BorderWithTitle("EDITING: "+filepath.Base(m.currentFile), m.width-2, true)
		body := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			BorderForeground(colorBlue).
			Width(m.width - 2).
			Render(m.editor.View())
		mainContent = title + "\n" + body
	case m.activeTab == tabTasks:
		if m.loading && len(m.tasks) == 0 {
			viewportHeight := m.height - 5
			if viewportHeight < 3 {
				viewportHeight = 3
			}
			content := lipgloss.Place(
				m.width-4, viewportHeight,
				lipgloss.Center, lipgloss.Center,
				m.spinner.View()+" Loading tasks...",
			)
			mainContent = activeBorderStyle.
				Width(m.width - 2).
				Render(content)
		} else {
			mainContent = m.renderTasksTab()
		}
	case m.activeTab == tabFiles && m.fileEditMode:
		title := BorderWithTitle("EDITING: "+filepath.Base(m.fileViewPath), m.width-2, true)
		body := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			BorderForeground(colorBlue).
			Width(m.width - 2).
			Render(m.editor.View())
		mainContent = title + "\n" + body
	case m.activeTab == tabFiles && m.fileViewMode:
		border := activeBorderStyle
		vpContent := m.viewport.View()
		mainContent = border.
			Width(m.width - 2).
			Render(vpContent)
	case m.activeTab == tabFiles && m.fileListReady:
		mainContent = m.renderFilesTab()
	default:
		border := activeBorderStyle
		vpContent := m.viewport.View()

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
	} else if m.fileViewMode && !m.fileEditMode {
		// File view mode in Files tab
		fileViewStatus := statusBarStyle.Width(m.width).Render(
			" " + filepath.Base(m.fileViewPath) + "  e edit · esc back · tab switch")
		result = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			mainContent,
			fileViewStatus,
		)
	} else if m.editMode || m.fileEditMode {
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
		// Compile indicator
		compileAgo := formatDurationAgo(m.lastCompileTime)
		compileIndicator := fmt.Sprintf("Last compile: %s", compileAgo)
		if m.lastCompileTime == nil || time.Since(*m.lastCompileTime) > 7*24*time.Hour {
			compileIndicator = lipgloss.NewStyle().Foreground(colorYellow).
				Background(colorSurface0).Render(compileIndicator)
		}

		switch m.activeTab {
		case tabNotes:
			words := len(regexp.MustCompile(`\S+`).FindAllString(m.fileContent, -1))
			sections := len(regexp.MustCompile(`(?m)^## `).FindAllString(m.fileContent, -1))
			filename := filepath.Base(m.currentFile)
			left = fmt.Sprintf("%s %s  %d words  %d sections  %s", prefix, filename, words, sections, compileIndicator)
			right = "e edit · tab switch · /help "
		case tabTasks:
			left = prefix + " j/k navigate · Enter complete"
			right = "tab switch "
		case tabFiles:
			if m.fileFuzzyMode {
				left = prefix + " Type to search · Enter open · Esc back"
				right = "tab switch "
			} else {
				left = prefix + " / search all · Enter open/browse"
				right = "tab switch "
			}
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

// renderStatusOverlay renders the vault status overlay.
func (m AppModel) renderStatusOverlay() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	labelStyle := lipgloss.NewStyle().Foreground(colorText)
	valueStyle := lipgloss.NewStyle().Foreground(colorText).Bold(true)
	zeroStyle := lipgloss.NewStyle().Foreground(colorOverlay)
	footerStyle := lipgloss.NewStyle().Foreground(colorOverlay)

	compileAgo := formatDurationAgo(m.lastCompileTime)
	compileStyle := labelStyle
	if m.lastCompileTime == nil || time.Since(*m.lastCompileTime) > 7*24*time.Hour {
		compileStyle = lipgloss.NewStyle().Foreground(colorYellow)
	}

	renderMetric := func(label string, count int, unit string) string {
		if count == 0 {
			return zeroStyle.Render(fmt.Sprintf("  %s: 0 %s", label, unit))
		}
		return labelStyle.Render(fmt.Sprintf("  %s: ", label)) +
			valueStyle.Render(fmt.Sprintf("%d", count)) +
			labelStyle.Render(fmt.Sprintf(" %s", unit))
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render("  Vault Status"))
	lines = append(lines, "")
	lines = append(lines, compileStyle.Render(fmt.Sprintf("  Last compile: %s", compileAgo)))
	lines = append(lines, "")

	if m.vaultStatus != nil {
		lines = append(lines, renderMetric("Wiki inbox", m.vaultStatus.WikiInboxCount, "unprocessed"))
		lines = append(lines, renderMetric("Review queue", m.vaultStatus.ReviewQueueCount, "pending drafts"))
		lines = append(lines, renderMetric("Raw notes", m.vaultStatus.RawNotesSinceCompile, "since last compile"))
	}

	lines = append(lines, "")
	lines = append(lines, footerStyle.Render("  Press any key to return"))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue).
		Padding(0, 2).
		Width(42)

	box := boxStyle.Render(content)

	return lipgloss.Place(m.width, m.height-2,
		lipgloss.Center, lipgloss.Center, box)
}

// renderCompileSummary renders the compile summary overlay.
func (m AppModel) renderCompileSummary() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	labelStyle := lipgloss.NewStyle().Foreground(colorText)
	zeroStyle := lipgloss.NewStyle().Foreground(colorOverlay)
	warnStyle := lipgloss.NewStyle().Foreground(colorYellow)
	footerStyle := lipgloss.NewStyle().Foreground(colorOverlay)

	var lines []string
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render("  Compile Complete"))
	lines = append(lines, "")

	if m.compileResult == nil {
		lines = append(lines, labelStyle.Render("  No results available"))
	} else {
		renderSection := func(name string, metrics content.SectionMetrics) {
			lines = append(lines, sectionStyle.Render("  "+name))
			if metrics.Items == nil || len(metrics.Items) == 0 {
				lines = append(lines, zeroStyle.Render("    No data"))
			} else {
				for k, v := range metrics.Items {
					style := labelStyle
					if v == "0" || v == "" {
						style = zeroStyle
					}
					lines = append(lines, style.Render(fmt.Sprintf("    %s: %s", k, v)))
				}
			}
			lines = append(lines, "")
		}

		renderSection("Wiki", m.compileResult.Wiki)
		renderSection("Zettelkasten", m.compileResult.Zettelkasten)

		// Lint with warnings
		lines = append(lines, sectionStyle.Render("  Lint"))
		if m.compileResult.Lint.Items == nil || len(m.compileResult.Lint.Items) == 0 {
			lines = append(lines, zeroStyle.Render("    No data"))
		} else {
			for k, v := range m.compileResult.Lint.Items {
				style := labelStyle
				lower := strings.ToLower(strings.TrimSpace(v))
				if lower != "none" && lower != "0" && lower != "" {
					style = warnStyle
				}
				lines = append(lines, style.Render(fmt.Sprintf("    %s: %s", k, v)))
			}
		}
		lines = append(lines, "")

		// Suggestions
		if len(m.compileResult.Suggestions) > 0 {
			lines = append(lines, sectionStyle.Render("  Suggestions"))
			for _, s := range m.compileResult.Suggestions {
				lines = append(lines, labelStyle.Render("    • "+s))
			}
			lines = append(lines, "")
		}

		// Duration
		if m.compileResult.Frontmatter.DurationSeconds > 0 {
			lines = append(lines, zeroStyle.Render(fmt.Sprintf("  Duration: %ds", m.compileResult.Frontmatter.DurationSeconds)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, footerStyle.Render("  Press any key to return"))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGreen).
		Padding(0, 2).
		Width(46)

	box := boxStyle.Render(content)

	return lipgloss.Place(m.width, m.height-2,
		lipgloss.Center, lipgloss.Center, box)
}

// formatDurationAgo returns a human-readable duration since the given time.
func formatDurationAgo(t *time.Time) string {
	if t == nil {
		return "never"
	}
	d := time.Since(*t)
	switch {
	case d < time.Hour:
		return "just now"
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 4*7*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(7*24)))
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(30*24)))
	}
}

// Run starts the TUI application.
func Run(vaultPath, filePath, fileContent string) error {
	model := NewApp(vaultPath, filePath, fileContent)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}
