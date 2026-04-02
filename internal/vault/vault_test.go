package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListMarkdownFiles(t *testing.T) {
	tmp := t.TempDir()

	// Create test structure
	os.MkdirAll(filepath.Join(tmp, "subdir"), 0o755)
	os.MkdirAll(filepath.Join(tmp, ".hidden"), 0o755)

	os.WriteFile(filepath.Join(tmp, "note1.md"), []byte("# Note 1"), 0o644)
	os.WriteFile(filepath.Join(tmp, "subdir", "note2.md"), []byte("# Note 2"), 0o644)
	os.WriteFile(filepath.Join(tmp, ".hidden", "secret.md"), []byte("# Secret"), 0o644)
	os.WriteFile(filepath.Join(tmp, "readme.txt"), []byte("Not markdown"), 0o644)

	files, err := ListMarkdownFiles(tmp)
	if err != nil {
		t.Fatalf("ListMarkdownFiles error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(files), files)
	}

	// Check that .hidden directory was skipped
	for _, f := range files {
		if strings.Contains(f, ".hidden") {
			t.Errorf("Should not include files from hidden directories: %s", f)
		}
	}
}

func TestReadWriteFile(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "test.md")

	content := "# Test\n\nSome content"
	if err := WriteFile(filePath, content); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	read, err := ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	if read != content {
		t.Errorf("ReadFile = %q, want %q", read, content)
	}
}

func TestLogToCentralFile(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "tasks-log.md")

	header := "# Task Log\n\n"
	entry := "- [ ] New task *[[2024-01-01]]*"
	predicate := func(line string) bool {
		trimmed := strings.TrimSpace(line)
		return strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]")
	}

	// First entry - creates file
	if err := LogToCentralFile(logPath, entry, header, predicate); err != nil {
		t.Fatalf("LogToCentralFile error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	if !strings.Contains(string(data), entry) {
		t.Error("Log file should contain the entry")
	}
	if !strings.Contains(string(data), "# Task Log") {
		t.Error("Log file should contain the header")
	}

	// Second entry - inserts before first
	entry2 := "- [ ] Second task *[[2024-01-02]]*"
	if err := LogToCentralFile(logPath, entry2, header, predicate); err != nil {
		t.Fatalf("LogToCentralFile error: %v", err)
	}

	data, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	content := string(data)
	idx1 := strings.Index(content, entry2)
	idx2 := strings.Index(content, entry)
	if idx1 >= idx2 {
		t.Error("Second entry should appear before first entry")
	}
}

func TestEnsureDailyNote(t *testing.T) {
	tmp := t.TempDir()

	// Create new note
	path, content, created, err := EnsureDailyNote(tmp)
	if err != nil {
		t.Fatalf("EnsureDailyNote error: %v", err)
	}

	if !created {
		t.Error("Expected created to be true for new note")
	}
	if path == "" {
		t.Error("Expected non-empty path")
	}
	if !strings.Contains(content, "## Tasks") {
		t.Error("Content should contain Tasks section")
	}
	if strings.Contains(content, "{{date:YYYY-MM-DD}}") {
		t.Error("Template placeholder should be replaced")
	}

	// Open existing note
	_, _, created2, err := EnsureDailyNote(tmp)
	if err != nil {
		t.Fatalf("EnsureDailyNote (existing) error: %v", err)
	}
	if created2 {
		t.Error("Expected created to be false for existing note")
	}
}
