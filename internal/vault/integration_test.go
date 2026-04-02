package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/caneppelevitor/obsidian-cli/internal/content"
	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
)

// TestFullWorkflow tests the complete flow:
// init vault → create daily note → add task → verify task log → complete task
func TestFullWorkflow(t *testing.T) {
	vaultDir := t.TempDir()

	// 1. Create daily note
	notePath, noteContent, created, err := EnsureDailyNote(vaultDir)
	if err != nil {
		t.Fatalf("EnsureDailyNote: %v", err)
	}
	if !created {
		t.Fatal("Expected new note to be created")
	}
	if !strings.Contains(noteContent, "## Tasks") {
		t.Fatal("Note should contain Tasks section")
	}

	// 2. Parse and add a task
	parsed := content.ParseContentInput("[] Buy groceries #do")
	if parsed == nil {
		t.Fatal("ParseContentInput returned nil for task input")
	}
	if parsed.Section != "Tasks" {
		t.Errorf("Section = %q, want Tasks", parsed.Section)
	}
	if parsed.LogType != "task" {
		t.Errorf("LogType = %q, want task", parsed.LogType)
	}

	// 3. Add to section
	result := content.AddToSection(noteContent, parsed.Section, parsed.FormattedContent)
	if result == nil {
		t.Fatal("AddToSection returned nil")
	}
	if !strings.Contains(result.NewContent, "- [ ] Buy groceries #do") {
		t.Fatal("Content should contain the task")
	}

	// 4. Save the note
	if err := WriteFile(notePath, result.NewContent); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// 5. Log to central task file
	taskLogPath := filepath.Join(vaultDir, "tasks-log.md")
	sourceFile := strings.TrimSuffix(filepath.Base(notePath), ".md")
	entry := "- [ ] Buy groceries #do *[[" + sourceFile + "]]*"
	header := "# Task Log\n\nCentralized log of all tasks.\n\n"
	predicate := func(line string) bool {
		trimmed := strings.TrimSpace(line)
		return strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]")
	}

	if err := LogToCentralFile(taskLogPath, entry, header, predicate); err != nil {
		t.Fatalf("LogToCentralFile: %v", err)
	}

	// 6. Read task log and verify
	taskList, err := tasks.ReadTaskLog(taskLogPath)
	if err != nil {
		t.Fatalf("ReadTaskLog: %v", err)
	}
	if len(taskList) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(taskList))
	}
	if taskList[0].Content != "Buy groceries #do" {
		t.Errorf("Task content = %q, want %q", taskList[0].Content, "Buy groceries #do")
	}
	if taskList[0].Completed {
		t.Error("Task should not be completed")
	}

	// 7. Complete the task
	if err := tasks.CompleteTask(taskLogPath, 0, taskList); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	// 8. Verify completion
	taskList2, err := tasks.ReadTaskLog(taskLogPath)
	if err != nil {
		t.Fatalf("ReadTaskLog after complete: %v", err)
	}
	if !taskList2[0].Completed {
		t.Error("Task should be completed after CompleteTask")
	}

	// 9. Add an idea
	ideaParsed := content.ParseContentInput("- Learn Bubble Tea")
	if ideaParsed == nil || ideaParsed.Section != "Ideas" {
		t.Fatal("Failed to parse idea input")
	}

	ideaResult := content.AddToSection(result.NewContent, ideaParsed.Section, ideaParsed.FormattedContent)
	if ideaResult == nil {
		t.Fatal("AddToSection for idea returned nil")
	}
	if !strings.Contains(ideaResult.NewContent, "- Learn Bubble Tea") {
		t.Fatal("Content should contain the idea")
	}

	// 10. Verify metadata injection
	withMeta := content.InjectMetadata(ideaResult.NewContent)
	if !strings.Contains(withMeta, "updated_at:") {
		t.Fatal("Metadata should be injected")
	}

	// 11. Verify file listing
	files, err := ListMarkdownFiles(vaultDir)
	if err != nil {
		t.Fatalf("ListMarkdownFiles: %v", err)
	}
	// Should find the daily note and task log
	found := false
	for _, f := range files {
		if strings.HasSuffix(f, ".md") {
			found = true
		}
	}
	if !found {
		t.Fatal("Should find at least one markdown file")
	}

	// 12. Verify existing note is not recreated
	_, _, created2, err := EnsureDailyNote(vaultDir)
	if err != nil {
		t.Fatalf("EnsureDailyNote (existing): %v", err)
	}
	if created2 {
		t.Fatal("Should not recreate existing note")
	}
}

// TestEdgeCases tests error handling and edge conditions.
func TestEdgeCases(t *testing.T) {
	// Empty vault
	emptyDir := t.TempDir()
	files, err := ListMarkdownFiles(emptyDir)
	if err != nil {
		t.Fatalf("ListMarkdownFiles on empty dir: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}

	// Read non-existent file
	_, err = ReadFile(filepath.Join(emptyDir, "nonexistent.md"))
	if err == nil {
		t.Error("Expected error reading non-existent file")
	}

	// Write to nested non-existent directory
	nestedPath := filepath.Join(emptyDir, "a", "b", "c", "test.md")
	if err := WriteFile(nestedPath, "test content"); err != nil {
		t.Fatalf("WriteFile should create nested dirs: %v", err)
	}
	data, _ := os.ReadFile(nestedPath)
	if string(data) != "test content" {
		t.Error("File content mismatch")
	}

	// Parse invalid content input
	result := content.ParseContentInput("plain text")
	if result != nil {
		t.Error("Plain text should not match any prefix")
	}

	// Empty content operations
	emptyResult := content.AddToSection("", "Tasks", "- [ ] task")
	if emptyResult != nil {
		t.Error("AddToSection on empty content with no section should return nil")
	}
}
