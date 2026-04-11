package tui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/config"
	"github.com/caneppelevitor/obsidian-cli/internal/content"
	"github.com/caneppelevitor/obsidian-cli/internal/logging"
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

// CompileProgressMsg is sent for each line of streamed compile output.
type CompileProgressMsg struct {
	Line          string
	IsPhaseMarker bool
	PhaseNumber   string
	PhaseName     string
}

// CompileTickMsg drives per-second re-renders during compile (for elapsed time).
type CompileTickMsg struct{}

// CompileTokensMsg carries token usage info from a stream-json event.
type CompileTokensMsg struct {
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	CostUSD             float64 // only set by the final result event
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

// streamJSONParseResult holds everything extracted from a single stream-json event.
type streamJSONParseResult struct {
	Lines []string          // display lines
	Usage *CompileTokensMsg // token counts (nil if event had no usage data)
}

// parseStreamJSONLine extracts display lines and token usage from a single line of
// claude's --output-format stream-json output. Each input line is a JSON event.
//
// Event shapes:
//   - {"type":"system","subtype":"init",...}
//   - {"type":"assistant","message":{"content":[...], "usage":{"input_tokens":N,...}}}
//   - {"type":"user","message":{"content":[{"type":"tool_result",...}]}}
//   - {"type":"result","subtype":"success","result":"...","total_cost_usd":0.05,"usage":{...}}
func parseStreamJSONLine(line string) streamJSONParseResult {
	var res streamJSONParseResult

	line = strings.TrimSpace(line)
	if line == "" || !strings.HasPrefix(line, "{") {
		if line != "" {
			res.Lines = []string{line}
		}
		return res
	}

	var evt map[string]any
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		res.Lines = []string{line} // fallback: show raw
		return res
	}

	eventType, _ := evt["type"].(string)
	switch eventType {
	case "system":
		subtype, _ := evt["subtype"].(string)
		if subtype == "init" {
			res.Lines = []string{"▸ Claude Code initialized"}
		}
	case "assistant":
		res.Lines = extractAssistantDisplay(evt)
		res.Usage = extractAssistantUsage(evt)
	case "user":
		res.Lines = extractToolResultDisplay(evt)
	case "result":
		if resultText, ok := evt["result"].(string); ok && resultText != "" {
			res.Lines = strings.Split(strings.TrimSpace(resultText), "\n")
		}
		res.Usage = extractResultUsage(evt)
	}
	return res
}

// extractAssistantUsage pulls token counts from an assistant message's usage field.
// Returns nil if no usage data is present.
func extractAssistantUsage(evt map[string]any) *CompileTokensMsg {
	msg, ok := evt["message"].(map[string]any)
	if !ok {
		return nil
	}
	usage, ok := msg["usage"].(map[string]any)
	if !ok {
		return nil
	}
	return &CompileTokensMsg{
		InputTokens:         intFromJSON(usage["input_tokens"]),
		OutputTokens:        intFromJSON(usage["output_tokens"]),
		CacheReadTokens:     intFromJSON(usage["cache_read_input_tokens"]),
		CacheCreationTokens: intFromJSON(usage["cache_creation_input_tokens"]),
	}
}

// extractResultUsage pulls token counts and cost from the final result event.
func extractResultUsage(evt map[string]any) *CompileTokensMsg {
	result := &CompileTokensMsg{}
	hasData := false

	if cost, ok := evt["total_cost_usd"].(float64); ok {
		result.CostUSD = cost
		hasData = true
	}
	if usage, ok := evt["usage"].(map[string]any); ok {
		result.InputTokens = intFromJSON(usage["input_tokens"])
		result.OutputTokens = intFromJSON(usage["output_tokens"])
		result.CacheReadTokens = intFromJSON(usage["cache_read_input_tokens"])
		result.CacheCreationTokens = intFromJSON(usage["cache_creation_input_tokens"])
		hasData = true
	}

	if !hasData {
		return nil
	}
	return result
}

// intFromJSON safely converts a JSON number (decoded as float64) to int.
func intFromJSON(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

// extractAssistantDisplay pulls text content and tool_use calls from an
// assistant message event for display.
func extractAssistantDisplay(evt map[string]any) []string {
	msg, ok := evt["message"].(map[string]any)
	if !ok {
		return nil
	}
	contentArr, ok := msg["content"].([]any)
	if !ok {
		return nil
	}

	var out []string
	for _, c := range contentArr {
		cMap, ok := c.(map[string]any)
		if !ok {
			continue
		}
		cType, _ := cMap["type"].(string)
		switch cType {
		case "text":
			if text, ok := cMap["text"].(string); ok {
				for _, line := range strings.Split(text, "\n") {
					if strings.TrimSpace(line) != "" {
						out = append(out, line)
					}
				}
			}
		case "tool_use":
			name, _ := cMap["name"].(string)
			input, _ := cMap["input"].(map[string]any)
			out = append(out, formatToolUse(name, input))
		}
	}
	return out
}

// extractToolResultDisplay shows a short line for tool results (user messages
// in the stream contain tool_result blocks).
func extractToolResultDisplay(evt map[string]any) []string {
	// Keep this quiet — tool results are usually verbose file contents.
	// Just return nothing; the tool_use line already gave us context.
	return nil
}

// formatToolUse returns a single-line summary of a tool invocation.
func formatToolUse(name string, input map[string]any) string {
	switch name {
	case "Read":
		if p, ok := input["file_path"].(string); ok {
			return "  → Read " + shortenPath(p)
		}
	case "Write":
		if p, ok := input["file_path"].(string); ok {
			return "  → Write " + shortenPath(p)
		}
	case "Edit":
		if p, ok := input["file_path"].(string); ok {
			return "  → Edit " + shortenPath(p)
		}
	case "Glob":
		if p, ok := input["pattern"].(string); ok {
			return "  → Glob " + p
		}
	case "Grep":
		if p, ok := input["pattern"].(string); ok {
			return "  → Grep " + p
		}
	case "Bash":
		if p, ok := input["command"].(string); ok {
			if len(p) > 60 {
				p = p[:57] + "..."
			}
			return "  → Bash " + p
		}
	}
	return "  → " + name
}

// shortenPath returns a compact path representation (last 2 segments).
func shortenPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 2 {
		return p
	}
	return ".../" + strings.Join(parts[len(parts)-2:], "/")
}

// runCompileCmd starts the compile subprocess and streams its stdout line-by-line.
// Returns a tea.Cmd that starts the process and a CancelFunc to stop it.
// The goroutine sends CompileProgressMsg for each line and CompileDoneMsg on exit.
func runCompileCmd(vaultPath string) (tea.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	// Use --verbose --output-format stream-json to get real-time events
	// (tool calls, assistant messages). Without this, claude --print buffers
	// all output until the subprocess finishes.
	//
	// --permission-mode bypassPermissions is used because the compile skill
	// needs to edit .claude/rules/vault-map.md which is treated as a sensitive
	// path by Claude Code. acceptEdits is not sufficient for .claude/ files.
	// This is safe because: the user explicitly triggers the compile against
	// their own vault, the skill is user-authored, and all changes are tracked
	// in git so nothing is destructive.
	cmd := exec.CommandContext(ctx, "claude", "--print",
		"--verbose",
		"--output-format", "stream-json",
		"--permission-mode", "bypassPermissions",
		"/compile",
	)
	cmd.Dir = vaultPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return func() tea.Msg {
			return CompileDoneMsg{Err: err}
		}, func() {}
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	teaCmd := func() tea.Msg {
		logging.Info("compile subprocess starting", "dir", vaultPath)
		if err := cmd.Start(); err != nil {
			logging.Error("compile subprocess failed to start", "err", err)
			cancel()
			return CompileDoneMsg{Err: err}
		}
		logging.Info("compile subprocess started", "pid", cmd.Process.Pid)

		// Stream stdout line-by-line in a goroutine.
		// Each line is a JSON event from claude --output-format stream-json.
		// We extract human-readable messages and forward them as CompileProgressMsg.
		go func() {
			defer cancel()

			scanner := bufio.NewScanner(stdout)
			scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024)
			lineCount := 0
			for scanner.Scan() {
				line := scanner.Text()
				lineCount++

				parsed := parseStreamJSONLine(line)
				for _, displayLine := range parsed.Lines {
					num, name, ok := content.ParsePhaseMarker(displayLine)
					if teaProgram != nil {
						teaProgram.Send(CompileProgressMsg{
							Line:          displayLine,
							IsPhaseMarker: ok,
							PhaseNumber:   num,
							PhaseName:     name,
						})
					}
				}
				if parsed.Usage != nil && teaProgram != nil {
					teaProgram.Send(*parsed.Usage)
				}
			}

			if scanErr := scanner.Err(); scanErr != nil {
				logging.Warn("compile scanner error", "err", scanErr)
			}

			waitErr := cmd.Wait()
			exitCode := 0
			if waitErr != nil {
				if exitErr, ok := waitErr.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}
			logging.Info("compile subprocess exited",
				"exitCode", exitCode,
				"lineCount", lineCount,
				"waitErr", fmt.Sprintf("%v", waitErr),
			)

			if teaProgram != nil {
				teaProgram.Send(CompileDoneMsg{
					ExitCode: exitCode,
					Stderr:   stderrBuf.String(),
					Err:      waitErr,
				})
			}
		}()

		// Return nil — the goroutine sends messages asynchronously.
		return nil
	}

	return teaCmd, cancel
}

// compileTickCmd returns a tea.Cmd that fires CompileTickMsg after 1 second.
// The Update() handler re-dispatches this while compile is running, driving
// per-second re-renders of the elapsed time counter.
func compileTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return CompileTickMsg{}
	})
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
