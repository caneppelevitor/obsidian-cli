package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
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
	"github.com/caneppelevitor/obsidian-cli/internal/logging"
	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

const (
	tabNotes   = 0
	tabTasks   = 1
	tabFiles   = 2
	tabCompile = 3 // Only present when m.compiling || m.compileResult != nil
)

// compileTabVisible returns true if the Compile tab should be shown in the tab bar.
// The tab appears when a compile is running, when there's a cached result to view,
// or when the user is currently on the compile tab (prevents the tab from vanishing
// mid-transition between CompileDoneMsg and CompileResultMsg).
func (m AppModel) compileTabVisible() bool {
	return m.compiling || m.compileResult != nil || m.activeTab == tabCompile
}

// numTabs returns the count of visible tabs (3 normally, 4 when compile is active).
func (m AppModel) numTabs() int {
	if m.compileTabVisible() {
		return 4
	}
	return 3
}

// quitCmd cancels any running compile and returns tea.Quit.
// Used by all Quit key handlers to avoid leaking the compile subprocess.
func (m *AppModel) quitCmd() tea.Cmd {
	if m.compileCancel != nil {
		m.compileCancel()
	}
	return tea.Quit
}

// teaProgram is the package-level reference to the running Bubble Tea program.
// Set by Run() at startup. Used by streaming goroutines to send messages back
// into the Update() loop. Only streaming goroutines should touch this.
var teaProgram *tea.Program

// CompileProgress tracks live state of a running compile for UI display.
type CompileProgress struct {
	CurrentPhase string   // e.g., "Wiki Compilation"
	PhaseNumber  string   // e.g., "1" or "2.5"
	RecentLines  []string // ring buffer, max 10 lines

	// Token usage (accumulated from assistant events)
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	CostUSD             float64
}

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

	// Review mode
	reviewMode           bool
	reviewItems          []content.ReviewItem
	reviewCursor         int
	reviewDraftPath      string // path of draft being viewed (empty = list mode)
	reviewPreviewContent string // rendered preview of selected item
	reviewLastPreviewed  string // name of last previewed item (avoids re-loading)

	// Compile & Status
	showingStatus        bool
	showingCompileSummary bool
	compiling            bool
	compileResult        *content.CompileResult
	vaultStatus          *content.VaultStatus
	lastCompileTime      *time.Time

	// Streaming compile state
	compileProgress    *CompileProgress   // nil when no compile running
	compileCancel      context.CancelFunc // nil when no compile running
	compileStartTime   time.Time
	lastCompileTokens  CompileTokensMsg // persists after compile finishes (for summary display)
	lastCompileElapsed time.Duration    // persists after compile finishes

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

func (m AppModel) Update(msg tea.Msg) (model tea.Model, cmd tea.Cmd) {
	// Panic recovery: log the crash before dying so we have a forensic trail.
	defer func() {
		if r := recover(); r != nil {
			logging.Error("panic in Update()",
				"recover", fmt.Sprintf("%v", r),
				"msgType", fmt.Sprintf("%T", msg),
				"stack", string(debug.Stack()),
			)
			panic(r) // re-panic so Bubble Tea exits cleanly
		}
	}()

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)

		viewportHeight := m.height - 8 // tab(2) + border(2) + sep(1) + input(1) + gap(1) + status(1)
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
			m.updateViewportContent()
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
		// Compile tab key handling
		if m.activeTab == tabCompile {
			if key.Matches(msg, m.keys.Quit) {
				return m, m.quitCmd()
			}
			if msg.String() == "esc" || msg.String() == "escape" {
				if m.compiling {
					// Cancel the running compile
					if m.compileCancel != nil {
						m.compileCancel()
					}
					m.statusText = "Compile cancelled"
					m.activeTab = tabNotes
					m.input.Focus()
					return m, nil
				}
				// Not compiling: dismiss the compile tab (clear cached result)
				m.compileResult = nil
				m.activeTab = tabNotes
				m.input.Focus()
				return m, nil
			}
			if key.Matches(msg, m.keys.Tab) {
				// Cycle to next tab (wrap around all visible tabs)
				m.activeTab = (m.activeTab + 1) % m.numTabs()
				if m.activeTab == tabNotes {
					m.input.Focus()
				}
				m.updateViewportContent()
				return m, nil
			}
			// Forward scroll keys to viewport when viewing summary
			if !m.compiling && m.compileResult != nil {
				var vpCmd tea.Cmd
				m.viewport, vpCmd = m.viewport.Update(msg)
				if vpCmd != nil {
					cmds = append(cmds, vpCmd)
				}
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}

		// Dismiss overlays on any keypress
		if m.showingStatus {
			m.showingStatus = false
			return m, nil
		}
		if m.showingCompileSummary {
			if msg.String() == "esc" || msg.String() == "escape" || msg.String() == "q" {
				m.showingCompileSummary = false
				m.viewport.SetContent(m.renderContent())
				return m, nil
			}
			// Forward scroll keys to viewport
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			if vpCmd != nil {
				return m, vpCmd
			}
			return m, nil
		}

		// Review mode: list navigation + actions
		// (skip if editing — let fileEditMode handler below take over)
		if m.reviewMode && !m.fileEditMode {
			if m.reviewDraftPath != "" {
				// Viewing a draft
				switch {
				case key.Matches(msg, m.keys.Quit):
					return m, m.quitCmd()
				case msg.String() == "esc" || msg.String() == "escape":
					m.reviewDraftPath = ""
					m.fileViewContent = ""
					return m, nil
				case msg.String() == "e":
					// Edit the draft file
					m.fileEditMode = true
					m.fileViewPath = m.reviewDraftPath
					m.editor.SetValue(m.fileViewContent)
					viewportHeight := m.height - 5
					if viewportHeight < 5 {
						viewportHeight = 5
					}
					m.editor.SetWidth(m.width - 4)
					m.editor.SetHeight(viewportHeight)
					return m, m.editor.Focus()
				case msg.String() == "a":
					if len(m.reviewItems) > 0 && m.reviewCursor < len(m.reviewItems) {
						name := m.reviewItems[m.reviewCursor].Name
						m.loading = true
						return m, tea.Batch(approveReviewItemCmd(m.vaultRootPath, name), m.spinner.Tick)
					}
					return m, nil
				case msg.String() == "d":
					if len(m.reviewItems) > 0 && m.reviewCursor < len(m.reviewItems) {
						name := m.reviewItems[m.reviewCursor].Name
						m.loading = true
						return m, tea.Batch(discardReviewItemCmd(m.vaultRootPath, name), m.spinner.Tick)
					}
					return m, nil
				default:
					var vpCmd tea.Cmd
					m.viewport, vpCmd = m.viewport.Update(msg)
					if vpCmd != nil {
						cmds = append(cmds, vpCmd)
					}
					return m, tea.Batch(cmds...)
				}
			}
			// Review list mode
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, m.quitCmd()
			case msg.String() == "esc" || msg.String() == "escape":
				m.reviewMode = false
				m.reviewItems = nil
				m.reviewDraftPath = ""
				m.fileViewContent = ""
				m.reviewPreviewContent = ""
				m.reviewLastPreviewed = ""
				m.viewport.SetContent(m.renderContent())
				m.input.Focus()
				return m, nil
			case msg.String() == "j" || msg.String() == "down":
				if m.reviewCursor < len(m.reviewItems)-1 {
					m.reviewCursor++
				}
				return m, m.loadReviewPreviewIfNeeded()
			case msg.String() == "k" || msg.String() == "up":
				if m.reviewCursor > 0 {
					m.reviewCursor--
				}
				return m, m.loadReviewPreviewIfNeeded()
			case msg.String() == "enter":
				if len(m.reviewItems) > 0 && m.reviewCursor < len(m.reviewItems) {
					item := m.reviewItems[m.reviewCursor]
					m.loading = true
					return m, tea.Batch(func() tea.Msg {
						found := vault.FindFile(vault.ZettelkastenDir(m.vaultRootPath), item.Name)
						if found == "" {
							return StatusMsg{Text: fmt.Sprintf("Draft not found: %s", item.Name)}
						}
						data, err := vault.ReadFile(found)
						if err != nil {
							return StatusMsg{Text: fmt.Sprintf("Error reading draft: %v", err)}
						}
						return FileViewLoadedMsg{Content: data, Path: found, Err: nil}
					}, m.spinner.Tick)
				}
				return m, nil
			case msg.String() == "a":
				if len(m.reviewItems) > 0 && m.reviewCursor < len(m.reviewItems) {
					name := m.reviewItems[m.reviewCursor].Name
					m.loading = true
					return m, tea.Batch(approveReviewItemCmd(m.vaultRootPath, name), m.spinner.Tick)
				}
				return m, nil
			case msg.String() == "d":
				if len(m.reviewItems) > 0 && m.reviewCursor < len(m.reviewItems) {
					name := m.reviewItems[m.reviewCursor].Name
					m.loading = true
					return m, tea.Batch(discardReviewItemCmd(m.vaultRootPath, name), m.spinner.Tick)
				}
				return m, nil
			}
			return m, nil
		}

		// Edit mode: forward everything to textarea except Esc and Ctrl+S
		if m.editMode {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, m.quitCmd()
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

		// File edit mode in Files tab (or review draft edit)
		if m.fileEditMode {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, m.quitCmd()
			case msg.String() == "esc" || msg.String() == "escape":
				if m.reviewMode {
					// Save and return to review draft view
					m.fileEditMode = false
					m.fileViewContent = m.editor.Value()
					m.editor.Blur()
					m.loading = true
					cmds = append(cmds, saveFileCmd(m.reviewDraftPath, m.fileViewContent), m.spinner.Tick)
					m.viewport.SetContent(m.renderFileViewContent())
					return m, tea.Batch(cmds...)
				}
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
				return m, m.quitCmd()
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
				m.activeTab = (m.activeTab + 1) % m.numTabs()
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
				return m, m.quitCmd()
			case key.Matches(msg, m.keys.Tab):
				m.activeTab = (m.activeTab + 1) % m.numTabs()
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
				return m, m.quitCmd()
			case key.Matches(msg, m.keys.Tab):
				m.activeTab = (m.activeTab + 1) % m.numTabs()
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
			return m, m.quitCmd()

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			m.activeTab = (m.activeTab + 1) % m.numTabs()
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
			logging.Error("save failed", "err", msg.Err, "file", m.currentFile)
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
			m.updateViewportContent()
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
			if m.reviewMode {
				// In review mode, render draft in viewport without entering file view mode
				m.reviewDraftPath = msg.Path
				m.fileViewContent = msg.Content
				m.viewport.SetContent(m.renderFileViewContent())
			} else {
				m.fileViewMode = true
				m.fileViewPath = msg.Path
				m.fileViewContent = msg.Content
				m.fileEditMode = false
				m.viewport.SetContent(m.renderFileViewContent())
			}
		} else {
			if m.reviewMode {
				m.reviewDraftPath = ""
				m.statusText = fmt.Sprintf("Draft not found: %s", filepath.Base(msg.Path))
			} else {
				m.statusText = fmt.Sprintf("Error: %v", msg.Err)
			}
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

	case CompileProgressMsg:
		if m.compileProgress == nil {
			// Compile was cancelled or already done — ignore stragglers.
			break
		}
		if msg.IsPhaseMarker {
			logging.Info("compile phase transition",
				"number", msg.PhaseNumber,
				"name", msg.PhaseName,
			)
			m.compileProgress.CurrentPhase = sanitizeLine(msg.PhaseName)
			m.compileProgress.PhaseNumber = msg.PhaseNumber
		}
		logging.Debug("compile output", "line", msg.Line)
		clean := sanitizeLine(msg.Line)
		if strings.TrimSpace(clean) != "" {
			m.compileProgress.RecentLines = append(m.compileProgress.RecentLines, clean)
			if len(m.compileProgress.RecentLines) > 10 {
				m.compileProgress.RecentLines = m.compileProgress.RecentLines[len(m.compileProgress.RecentLines)-10:]
			}
		}

	case CompileTickMsg:
		// Drive per-second re-renders while compile is running.
		if m.compiling {
			cmds = append(cmds, compileTickCmd())
		}

	case CompileTokensMsg:
		if m.compileProgress == nil {
			break
		}
		// Track cumulative maximums — assistant events report per-message usage,
		// final result event reports the total. We take the max so we show the
		// highest count seen so far (the result event's total will dominate).
		if msg.InputTokens > m.compileProgress.InputTokens {
			m.compileProgress.InputTokens = msg.InputTokens
		}
		if msg.OutputTokens > m.compileProgress.OutputTokens {
			m.compileProgress.OutputTokens = msg.OutputTokens
		}
		if msg.CacheReadTokens > m.compileProgress.CacheReadTokens {
			m.compileProgress.CacheReadTokens = msg.CacheReadTokens
		}
		if msg.CacheCreationTokens > m.compileProgress.CacheCreationTokens {
			m.compileProgress.CacheCreationTokens = msg.CacheCreationTokens
		}
		if msg.CostUSD > m.compileProgress.CostUSD {
			m.compileProgress.CostUSD = msg.CostUSD
		}

	case CompileDoneMsg:
		logging.Info("compile done",
			"exitCode", msg.ExitCode,
			"err", fmt.Sprintf("%v", msg.Err),
			"stderrLen", len(msg.Stderr),
			"elapsed", time.Since(m.compileStartTime).String(),
		)
		wasOnCompileTab := m.activeTab == tabCompile
		wasCancelled := msg.Err != nil && (msg.Err == context.Canceled ||
			(msg.ExitCode == -1) ||
			strings.Contains(msg.Stderr, "signal: killed"))

		// Preserve token usage and elapsed time for the summary view
		if m.compileProgress != nil {
			m.lastCompileTokens = CompileTokensMsg{
				InputTokens:         m.compileProgress.InputTokens,
				OutputTokens:        m.compileProgress.OutputTokens,
				CacheReadTokens:     m.compileProgress.CacheReadTokens,
				CacheCreationTokens: m.compileProgress.CacheCreationTokens,
				CostUSD:             m.compileProgress.CostUSD,
			}
		}
		m.lastCompileElapsed = time.Since(m.compileStartTime)

		m.compiling = false
		m.compileProgress = nil
		m.compileCancel = nil
		m.loading = false

		if wasCancelled {
			// Cleanup only — status text already set by Esc handler.
			if m.statusText == "" {
				m.statusText = "Compile cancelled"
			}
		} else if msg.Err != nil {
			m.statusText = fmt.Sprintf("Compile error: %v", msg.Err)
			// Compile tab goes away on error
			if m.activeTab == tabCompile {
				m.activeTab = tabNotes
				m.input.Focus()
			}
		} else if msg.ExitCode != 0 {
			errMsg := msg.Stderr
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
			m.statusText = fmt.Sprintf("Compile failed (exit %d): %s", msg.ExitCode, errMsg)
			if m.activeTab == tabCompile {
				m.activeTab = tabNotes
				m.input.Focus()
			}
		} else {
			// Success: load the result and refresh the last-compile time.
			// The compile tab stays visible showing the summary.
			cmds = append(cmds,
				loadCompileResultCmd(m.vaultRootPath),
				loadLastCompileTimeCmd(m.vaultRootPath),
			)
			if !wasOnCompileTab {
				m.statusText = "✓ Compile complete — type /compile to view summary"
			}
		}

	case CompileResultMsg:
		if msg.Err == nil && msg.Result != nil {
			m.compileResult = msg.Result
			// Render summary into viewport so it's ready when user views the Compile tab
			m.viewport.SetContent(m.renderCompileSummaryContent())
			m.viewport.SetYOffset(0)
		} else if msg.Err != nil {
			m.statusText = fmt.Sprintf("Error parsing compile results: %v", msg.Err)
		}

	case ReviewItemsLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Error loading review queue: %v", msg.Err)
			m.reviewMode = false
		} else if len(msg.Items) == 0 {
			m.statusText = "No pending drafts"
			m.reviewMode = false
		} else {
			m.reviewItems = msg.Items
			if m.reviewCursor >= len(msg.Items) {
				m.reviewCursor = max(0, len(msg.Items)-1)
			}
			m.reviewPreviewContent = ""
			m.reviewLastPreviewed = ""
			cmds = append(cmds, m.loadReviewPreviewIfNeeded())
		}

	case ReviewPreviewMsg:
		m.reviewPreviewContent = msg.Content
		m.reviewLastPreviewed = msg.Name

	case ReviewActionDoneMsg:
		m.loading = false
		if msg.Err != nil {
			logging.Error("review action failed", "action", msg.Action, "name", msg.Name, "err", msg.Err)
			m.statusText = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			logging.Info("review action completed", "action", msg.Action, "name", msg.Name)
			action := strings.ToUpper(msg.Action[:1]) + msg.Action[1:]
			m.statusText = fmt.Sprintf("%s: %s", action, msg.Name)
			m.reviewDraftPath = ""
			cmds = append(cmds, loadReviewItemsCmd(m.vaultRootPath))
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

	// Update text input (only when not on files tab or review mode)
	if m.activeTab != tabFiles && !m.reviewMode {
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)
		if inputCmd != nil {
			cmds = append(cmds, inputCmd)
		}
	}

	// Update viewport (only when not on files tab or review mode)
	if m.activeTab != tabFiles && !m.reviewMode {
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

func (m AppModel) View() (view tea.View) {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("panic in View()",
				"recover", fmt.Sprintf("%v", r),
				"activeTab", m.activeTab,
				"compiling", m.compiling,
				"stack", string(debug.Stack()),
			)
			panic(r)
		}
	}()

	if !m.ready {
		return tea.NewView("Initializing...")
	}

	tabs := []string{"Daily Note", "Tasks", "Files"}
	if m.compileTabVisible() {
		label := "Compile"
		if m.compiling {
			label = "⚙ Compiling"
		} else if m.compileResult != nil {
			label = "✓ Compile"
		}
		tabs = append(tabs, label)
	}
	tabBar := RenderTabBar(tabs, m.activeTab, m.width)

	// Full-screen overlays
	if m.showingStatus {
		v := tea.NewView(m.renderStatusOverlay())
		v.AltScreen = true
		return v
	}
	if m.showingCompileSummary {
		border := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Width(m.width - 2)
		mainContent := border.Render(m.viewport.View())
		statusBar := statusBarStyle.Width(m.width).Render(
			" Compile Summary  j/k scroll · Esc back")
		result := lipgloss.JoinVertical(lipgloss.Left, tabBar, mainContent, statusBar)
		v := tea.NewView(result)
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	// Review mode rendering
	if m.reviewMode {
		var mainContent string
		if m.fileEditMode {
			mainContent = m.renderEditBox(filepath.Base(m.reviewDraftPath))
		} else if m.reviewDraftPath != "" {
			// Viewing a draft file
			border := activeBorderStyle
			vpContent := m.viewport.View()
			mainContent = border.Width(m.width - 2).Render(vpContent)
		} else {
			mainContent = m.renderReviewList()
		}

		var statusBar string
		if m.fileEditMode {
			statusBar = statusBarStyle.Width(m.width).Render(
				" " + renderEditModeIndicator() + "  esc save & exit · ctrl+s save")
		} else if m.reviewDraftPath != "" {
			statusBar = statusBarStyle.Width(m.width).Render(
				" " + filepath.Base(m.reviewDraftPath) + "  e edit · a approve · d discard · Esc back to list")
		} else {
			statusBar = m.buildStatusBar()
		}

		result := lipgloss.JoinVertical(lipgloss.Left, tabBar, mainContent, statusBar)
		v := tea.NewView(result)
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		v.ReportFocus = true
		v.WindowTitle = "Obsidian: Review Queue"
		return v
	}

	// Main content area with active/inactive borders
	var mainContent string
	switch {
	case m.activeTab == tabCompile && m.compiling:
		mainContent = m.renderCompileProgress()
	case m.activeTab == tabCompile && m.compileResult != nil:
		// Summary view: render the viewport with green border (like before)
		border := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Width(m.width - 2)
		mainContent = border.Render(m.viewport.View())
	case m.activeTab == tabNotes && m.editMode:
		mainContent = m.renderEditBox(filepath.Base(m.currentFile))
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
		mainContent = m.renderEditBox(filepath.Base(m.fileViewPath))
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
		// Notes view mode: input line framed by separators above and below
		sep := separatorStyle.Render(strings.Repeat("─", m.width))
		modePill := DetectInputMode(m.input.Value())
		inputLine := " " + modePill + " " + inputPromptStyle.Render("> ") + m.input.View()
		result = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			mainContent,
			sep,
			inputLine,
			sep,
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

// updateViewportContent sets the viewport content based on current mode.
// Centralizes the conditional viewport update that was duplicated 5+ times.
func (m *AppModel) updateViewportContent() {
	if m.activeTab == tabCompile && m.compileResult != nil && !m.compiling {
		m.viewport.SetContent(m.renderCompileSummaryContent())
	} else if m.showingCompileSummary {
		m.viewport.SetContent(m.renderCompileSummaryContent())
	} else if m.reviewMode && m.reviewDraftPath != "" {
		m.viewport.SetContent(m.renderFileViewContent())
	} else if m.fileViewMode {
		m.viewport.SetContent(m.renderFileViewContent())
	} else {
		m.viewport.SetContent(m.renderContent())
	}
}

// renderEditBox renders the standard editor box with title border.
func (m AppModel) renderEditBox(title string) string {
	titleBar := BorderWithTitle("EDITING: "+title, m.width-2, true)
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(colorBlue).
		Width(m.width - 2).
		Render(m.editor.View())
	return titleBar + "\n" + body
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
		// Compile indicator: when compile is running AND user is on a different tab,
		// show live phase. Otherwise show static "Last compile: Xd ago".
		var compileIndicator string
		if m.compiling && m.activeTab != tabCompile {
			phaseLabel := "Starting"
			if m.compileProgress != nil && m.compileProgress.PhaseNumber != "" {
				phaseLabel = fmt.Sprintf("Phase %s/6", m.compileProgress.PhaseNumber)
			}
			elapsed := formatElapsed(time.Since(m.compileStartTime))
			compileIndicator = lipgloss.NewStyle().Foreground(colorGreen).
				Background(colorSurface0).Bold(true).
				Render(fmt.Sprintf("%s Compile: %s · %s", m.spinner.View(), phaseLabel, elapsed))
		} else {
			compileAgo := formatDurationAgo(m.lastCompileTime)
			compileIndicator = fmt.Sprintf("Last compile: %s", compileAgo)
			if m.lastCompileTime == nil || time.Since(*m.lastCompileTime) > 7*24*time.Hour {
				compileIndicator = lipgloss.NewStyle().Foreground(colorYellow).
					Background(colorSurface0).Render(compileIndicator)
			}
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
		case tabCompile:
			if m.compiling {
				left = prefix + " Esc cancel · Tab to switch away"
			} else {
				left = prefix + " j/k scroll · Esc dismiss · Tab switch"
			}
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

// Review list rendering → review_tab.go
// Compile summary + status overlay rendering → compile_tab.go

// Review list rendering -> review_tab.go
// Compile summary + status overlay rendering -> compile_tab.go

// Run starts the TUI application.
func Run(vaultPath, filePath, fileContent string) error {
	// Initialize debug logger from config (no-op if disabled)
	if cfg, err := config.Load(); err == nil && cfg.Debug.Enabled {
		if logErr := logging.Init(
			cfg.Debug.Enabled,
			cfg.Debug.LogFile,
			cfg.Debug.Level,
			cfg.Debug.TruncateOnStart,
		); logErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize debug log: %v\n", logErr)
		}
		defer logging.Close()
	}

	logging.Info("obsidian-cli started",
		"vaultPath", vaultPath,
		"currentFile", filePath,
	)

	model := NewApp(vaultPath, filePath, fileContent)
	p := tea.NewProgram(model)
	teaProgram = p
	_, err := p.Run()

	if err != nil {
		logging.Error("tea program exited with error", "err", err)
	} else {
		logging.Info("obsidian-cli stopped cleanly")
	}

	return err
}
