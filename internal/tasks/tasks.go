package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Task represents a parsed task from the task log.
type Task struct {
	Index        int
	Content      string
	Completed    bool
	SourceFile   string
	LineNumber   int
	OriginalLine string
}

var taskRegex = regexp.MustCompile(`^- \[(.)\] (.+?)( \*\[\[(.+?)\]\]\*)?$`)

// ReadTaskLog parses the task log file and returns all tasks.
func ReadTaskLog(taskLogPath string) ([]Task, error) {
	data, err := os.ReadFile(taskLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var tasks []Task

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- [ ]") && !strings.HasPrefix(trimmed, "- [x]") {
			continue
		}

		matches := taskRegex.FindStringSubmatch(trimmed)
		if matches == nil {
			continue
		}

		completed := matches[1] == "x"
		sourceFile := "unknown"
		if matches[4] != "" {
			sourceFile = matches[4]
		}

		tasks = append(tasks, Task{
			Index:        len(tasks),
			Content:      matches[2],
			Completed:    completed,
			SourceFile:   sourceFile,
			LineNumber:   i,
			OriginalLine: trimmed,
		})
	}

	return tasks, nil
}

// CompleteTask marks a task as completed by index.
func CompleteTask(taskLogPath string, taskIndex int, tasks []Task) error {
	if taskIndex < 0 || taskIndex >= len(tasks) {
		return fmt.Errorf("invalid task index: must be between 1 and %d", len(tasks))
	}

	task := tasks[taskIndex]
	if task.Completed {
		return fmt.Errorf("task is already completed")
	}

	data, err := os.ReadFile(taskLogPath)
	if err != nil {
		return fmt.Errorf("reading task log: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if task.LineNumber >= len(lines) {
		return fmt.Errorf("line number out of range")
	}

	lines[task.LineNumber] = strings.Replace(lines[task.LineNumber], "- [ ]", "- [x]", 1)

	return os.WriteFile(taskLogPath, []byte(strings.Join(lines, "\n")), 0o644)
}

// FilterRecent filters tasks by source file modification date.
func FilterRecent(tasks []Task, days int, vaultPath string) []Task {
	cutoff := time.Now().AddDate(0, 0, -days)
	var recent []Task

	for _, task := range tasks {
		sourcePath := filepath.Join(vaultPath, task.SourceFile+".md")
		stat, err := os.Stat(sourcePath)
		if err != nil {
			// Include tasks whose source can't be found
			recent = append(recent, task)
			continue
		}
		if stat.ModTime().After(cutoff) || stat.ModTime().Equal(cutoff) {
			recent = append(recent, task)
		}
	}

	return recent
}

// FormatTaskDisplay formats a task for console display.
func FormatTaskDisplay(task Task, displayIndex int) string {
	status := "○"
	if task.Completed {
		status = "✓"
	}
	return fmt.Sprintf("[%d] %s %s (%s)", displayIndex+1, status, task.Content, task.SourceFile)
}
