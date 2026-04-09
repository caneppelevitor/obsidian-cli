package tui

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

// VaultStatusMsg is sent after vault status metrics are fetched.
type VaultStatusMsg struct {
	Status content.VaultStatus
	Err    error
}

// CompileDoneMsg is sent when the compile process exits.
type CompileDoneMsg struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// CompileResultMsg is sent after last-compile.md is parsed.
type CompileResultMsg struct {
	Result *content.CompileResult
	Err    error
}

// LastCompileLoadedMsg is sent after startup frontmatter parse.
type LastCompileLoadedMsg struct {
	Time *time.Time
	Err  error
}

// ReviewItemsLoadedMsg is sent after _review-queue.md is parsed.
type ReviewItemsLoadedMsg struct {
	Items []content.ReviewItem
	Err   error
}

// ReviewPreviewMsg is sent with the rendered preview of a review item.
type ReviewPreviewMsg struct {
	Name    string
	Content string
}

// ReviewActionDoneMsg is sent after an approve/discard action.
type ReviewActionDoneMsg struct {
	Action string
	Name   string
	Err    error
}

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

func loadLastCompileTimeCmd(vaultPath string) tea.Cmd {
	return func() tea.Msg {
		lastCompilePath := vault.LastCompilePath(vaultPath)
		data, err := vault.ReadFile(lastCompilePath)
		if err != nil {
			// File missing is not an error — just means never compiled
			return LastCompileLoadedMsg{Time: nil, Err: nil}
		}
		yamlBlock, _ := content.ExtractFrontmatter(data)
		if yamlBlock == "" {
			return LastCompileLoadedMsg{Time: nil, Err: nil}
		}
		fm, err := content.ParseCompileFrontmatter(yamlBlock)
		if err != nil || fm.LastCompile.IsZero() {
			return LastCompileLoadedMsg{Time: nil, Err: nil}
		}
		t := fm.LastCompile
		return LastCompileLoadedMsg{Time: &t, Err: nil}
	}
}

func fetchVaultStatusCmd(vaultPath string, lastCompile *time.Time) tea.Cmd {
	return func() tea.Msg {
		var status content.VaultStatus
		status.LastCompile = lastCompile

		wikiInbox := vault.WikiInboxPath(vaultPath)
		count, _ := vault.CountUncheckedItems(wikiInbox)
		status.WikiInboxCount = count

		count, _ = vault.CountUncheckedInSection(vault.ReviewQueuePath(vaultPath), "Pending")
		status.ReviewQueueCount = count

		rawNotesDir := vault.RawNotesDir(vaultPath)
		var after time.Time
		if lastCompile != nil {
			after = *lastCompile
		}
		count, _ = vault.CountFilesModifiedAfter(rawNotesDir, after)
		status.RawNotesSinceCompile = count

		return VaultStatusMsg{Status: status, Err: nil}
	}
}

func runCompileCmd(vaultPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, "claude", "--print",
			"-p", "Read System/compile-playbook.md and execute it against the vault.",
			"--allowedTools", "Read,Write,Edit,Glob,Grep",
		)
		cmd.Dir = vaultPath

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return CompileDoneMsg{Err: err}
			}
		}

		return CompileDoneMsg{
			ExitCode: exitCode,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			Err:      nil,
		}
	}
}

func loadReviewItemsCmd(vaultRootPath string) tea.Cmd {
	return func() tea.Msg {
		data, err := vault.ReadFile(vault.ReviewQueuePath(vaultRootPath))
		if err != nil {
			return ReviewItemsLoadedMsg{Err: err}
		}
		items := content.ParseReviewItems(data)
		return ReviewItemsLoadedMsg{Items: items}
	}
}

func loadReviewPreviewCmd(vaultRootPath, itemName string) tea.Cmd {
	return func() tea.Msg {
		found := vault.FindFile(vault.ZettelkastenDir(vaultRootPath), itemName)
		if found == "" {
			return ReviewPreviewMsg{Name: itemName, Content: "File not found"}
		}
		data, err := vault.ReadFile(found)
		if err != nil {
			return ReviewPreviewMsg{Name: itemName, Content: "Error reading file"}
		}
		return ReviewPreviewMsg{Name: itemName, Content: data}
	}
}

func approveReviewItemCmd(vaultRootPath, itemName string) tea.Cmd {
	return func() tea.Msg {
		reviewPath := vault.ReviewQueuePath(vaultRootPath)
		data, err := vault.ReadFile(reviewPath)
		if err != nil {
			return ReviewActionDoneMsg{Action: "approved", Name: itemName, Err: err}
		}
		updated := content.ApproveReviewItem(data, itemName)
		err = vault.WriteFile(reviewPath, updated)
		return ReviewActionDoneMsg{Action: "approved", Name: itemName, Err: err}
	}
}

func discardReviewItemCmd(vaultRootPath, itemName string) tea.Cmd {
	return func() tea.Msg {
		reviewPath := vault.ReviewQueuePath(vaultRootPath)
		data, err := vault.ReadFile(reviewPath)
		if err != nil {
			return ReviewActionDoneMsg{Action: "discarded", Name: itemName, Err: err}
		}
		updated := content.DiscardReviewItem(data, itemName)
		err = vault.WriteFile(reviewPath, updated)
		return ReviewActionDoneMsg{Action: "discarded", Name: itemName, Err: err}
	}
}

func loadCompileResultCmd(vaultPath string) tea.Cmd {
	return func() tea.Msg {
		lastCompilePath := vault.LastCompilePath(vaultPath)
		data, err := vault.ReadFile(lastCompilePath)
		if err != nil {
			return CompileResultMsg{Err: err}
		}
		result, err := content.ParseCompileResult(data)
		return CompileResultMsg{Result: result, Err: err}
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
			"link":     "# Links Log\n\nCentralized log of all links captured across daily notes.\n\n",
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
