package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/config"
	"github.com/caneppelevitor/obsidian-cli/internal/content"
	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

// Messages

// SavedMsg is sent after a file save completes.
type SavedMsg struct{ Err error }

// FileLoadedMsg is sent after a file is loaded.
type FileLoadedMsg struct {
	Content string
	Path    string
	Err     error
}

// TasksLoadedMsg is sent after tasks are loaded.
type TasksLoadedMsg struct {
	Tasks []tasks.Task
	Err   error
}

// TaskCompletedMsg is sent after a task is completed.
type TaskCompletedMsg struct{ Err error }

// LoggedMsg is sent after a log entry is written.
type LoggedMsg struct{ Err error }

// FileViewLoadedMsg is sent when a file is loaded for viewing in the Files tab.
type FileViewLoadedMsg struct {
	Content string
	Path    string
	Err     error
}

// StatusMsg displays a message in the status bar temporarily.
type StatusMsg struct{ Text string }

// FileListMsg is sent with a list of entries in the current directory.
type FileListMsg struct {
	Entries []vault.DirEntry
	Dir     string // relative directory path
}

// FilePreviewMsg is sent with the preview content for a file or directory.
type FilePreviewMsg struct {
	Name      string
	Content   string
	WordCount int
	LineCount int
	ModTime   string
	Size      string
	Sections  []string
	Tags      []string
	IsDir     bool
	DirStats  dirStats
}

// Commands

func saveFileCmd(filePath, fileContent string) tea.Cmd {
	return func() tea.Msg {
		// Inject metadata before saving
		updatedContent := content.InjectMetadata(fileContent)
		err := vault.WriteFile(filePath, updatedContent)
		return SavedMsg{Err: err}
	}
}

func loadFileCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		data, err := vault.ReadFile(filePath)
		return FileLoadedMsg{Content: data, Path: filePath, Err: err}
	}
}

func loadTasksCmd(vaultPath string) tea.Cmd {
	return func() tea.Msg {
		taskLogFile, err := config.GetTaskLogFile()
		if err != nil {
			return TasksLoadedMsg{Err: err}
		}
		taskLogPath := filepath.Join(vaultPath, taskLogFile)
		taskList, err := tasks.ReadTaskLog(taskLogPath)
		return TasksLoadedMsg{Tasks: taskList, Err: err}
	}
}

func completeTaskCmd(vaultPath string, taskIndex int, allTasks []tasks.Task) tea.Cmd {
	return func() tea.Msg {
		taskLogFile, err := config.GetTaskLogFile()
		if err != nil {
			return TaskCompletedMsg{Err: err}
		}
		taskLogPath := filepath.Join(vaultPath, taskLogFile)
		err = tasks.CompleteTask(taskLogPath, taskIndex, allTasks)
		return TaskCompletedMsg{Err: err}
	}
}

func logEntryCmd(vaultPath, currentFile, rawContent, logType string) tea.Cmd {
	return func() tea.Msg {
		logFile, err := config.GetLogFile(logType)
		if err != nil {
			return LoggedMsg{Err: err}
		}

		sourceFile := "unknown"
		if currentFile != "" {
			sourceFile = strings.TrimSuffix(filepath.Base(currentFile), ".md")
		}

		// Log type metadata
		headers := map[string]string{
			"task":     "# Task Log\n\nCentralized log of all tasks created across daily notes.\n\n",
			"idea":     "# Ideas Log\n\nCentralized log of all ideas captured across daily notes.\n\n",
			"question": "# Questions Log\n\nCentralized log of all questions captured across daily notes.\n\n",
			"insight":  "# Insights Log\n\nCentralized log of all insights captured across daily notes.\n\n",
		}
		header, ok := headers[logType]
		if !ok {
			return LoggedMsg{Err: fmt.Errorf("unknown log type: %s", logType)}
		}

		// Tasks use checkbox predicate, everything else uses bullet predicate
		isTaskLine := func(line string) bool {
			trimmed := strings.TrimSpace(line)
			return strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]")
		}
		isBulletLine := func(line string) bool {
			trimmed := strings.TrimSpace(line)
			return strings.HasPrefix(trimmed, "- ") && !strings.Contains(line, "[ ]") && !strings.Contains(line, "[x]")
		}

		var entry string
		var predicate func(string) bool
		if logType == "task" {
			entry = fmt.Sprintf("- [ ] %s *[[%s]]*", rawContent, sourceFile)
			predicate = isTaskLine
		} else {
			entry = fmt.Sprintf("- %s *[[%s]]*", rawContent, sourceFile)
			predicate = isBulletLine
		}

		logPath := filepath.Join(vaultPath, logFile)
		err = vault.LogToCentralFile(logPath, entry, header, predicate)
		return LoggedMsg{Err: err}
	}
}
