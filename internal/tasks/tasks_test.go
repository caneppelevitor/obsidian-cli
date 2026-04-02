package tasks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleTaskLog = `# Task Log

Centralized log of all tasks created across daily notes.

- [ ] Buy groceries *[[2024-01-15]]*
- [x] Write tests *[[2024-01-14]]*
- [ ] Review PR #do *[[2024-01-15]]*
`

func TestReadTaskLog(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "tasks-log.md")
	os.WriteFile(logPath, []byte(sampleTaskLog), 0o644)

	tasks, err := ReadTaskLog(logPath)
	if err != nil {
		t.Fatalf("ReadTaskLog error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// First task
	if tasks[0].Content != "Buy groceries" {
		t.Errorf("tasks[0].Content = %q, want %q", tasks[0].Content, "Buy groceries")
	}
	if tasks[0].Completed {
		t.Error("tasks[0] should not be completed")
	}
	if tasks[0].SourceFile != "2024-01-15" {
		t.Errorf("tasks[0].SourceFile = %q, want %q", tasks[0].SourceFile, "2024-01-15")
	}

	// Second task (completed)
	if !tasks[1].Completed {
		t.Error("tasks[1] should be completed")
	}

	// Third task
	if tasks[2].Content != "Review PR #do" {
		t.Errorf("tasks[2].Content = %q, want %q", tasks[2].Content, "Review PR #do")
	}
}

func TestReadTaskLogNotFound(t *testing.T) {
	tasks, err := ReadTaskLog("/nonexistent/path")
	if err != nil {
		t.Fatalf("Expected no error for missing file, got %v", err)
	}
	if tasks != nil {
		t.Errorf("Expected nil tasks, got %v", tasks)
	}
}

func TestCompleteTask(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "tasks-log.md")
	os.WriteFile(logPath, []byte(sampleTaskLog), 0o644)

	tasks, _ := ReadTaskLog(logPath)

	// Complete first task
	if err := CompleteTask(logPath, 0, tasks); err != nil {
		t.Fatalf("CompleteTask error: %v", err)
	}

	// Verify
	data, _ := os.ReadFile(logPath)
	content := string(data)
	if strings.Contains(content, "- [ ] Buy groceries") {
		t.Error("Task should be marked as completed")
	}
	if !strings.Contains(content, "- [x] Buy groceries") {
		t.Error("Task should have [x] marker")
	}
}

func TestCompleteTaskAlreadyCompleted(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "tasks-log.md")
	os.WriteFile(logPath, []byte(sampleTaskLog), 0o644)

	tasks, _ := ReadTaskLog(logPath)

	// Try to complete already completed task
	err := CompleteTask(logPath, 1, tasks)
	if err == nil {
		t.Error("Expected error for already completed task")
	}
}

func TestCompleteTaskInvalidIndex(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "tasks-log.md")
	os.WriteFile(logPath, []byte(sampleTaskLog), 0o644)

	tasks, _ := ReadTaskLog(logPath)

	err := CompleteTask(logPath, -1, tasks)
	if err == nil {
		t.Error("Expected error for negative index")
	}

	err = CompleteTask(logPath, 99, tasks)
	if err == nil {
		t.Error("Expected error for out of range index")
	}
}

func TestFormatTaskDisplay(t *testing.T) {
	task := Task{
		Content:    "Buy groceries",
		Completed:  false,
		SourceFile: "2024-01-15",
	}

	display := FormatTaskDisplay(task, 0)
	if !strings.Contains(display, "Buy groceries") {
		t.Error("Display should contain task content")
	}
	if !strings.Contains(display, "○") {
		t.Error("Pending task should show ○")
	}

	task.Completed = true
	display = FormatTaskDisplay(task, 0)
	if !strings.Contains(display, "✓") {
		t.Error("Completed task should show ✓")
	}
}
