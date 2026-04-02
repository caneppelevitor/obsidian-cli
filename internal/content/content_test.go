package content

import (
	"strings"
	"testing"
)

func TestParseContentInput(t *testing.T) {
	tests := []struct {
		input    string
		section  string
		logType  string
		raw      string
		isNil    bool
	}{
		{"[] Buy groceries", "Tasks", "task", "Buy groceries", false},
		{"[]Buy groceries", "Tasks", "task", "Buy groceries", false},
		{"- Learn Go", "Ideas", "idea", "Learn Go", false},
		{"-Learn Go", "Ideas", "idea", "Learn Go", false},
		{"? Why is the sky blue", "Questions", "question", "Why is the sky blue", false},
		{"! Great insight", "Insights", "insight", "Great insight", false},
		{"plain text", "", "", "", true},
		{"  [] Trimmed input  ", "Tasks", "task", "Trimmed input", false},
	}

	for _, tt := range tests {
		result := ParseContentInput(tt.input)
		if tt.isNil {
			if result != nil {
				t.Errorf("ParseContentInput(%q) = %+v, want nil", tt.input, result)
			}
			continue
		}
		if result == nil {
			t.Errorf("ParseContentInput(%q) = nil, want non-nil", tt.input)
			continue
		}
		if result.Section != tt.section {
			t.Errorf("ParseContentInput(%q).Section = %q, want %q", tt.input, result.Section, tt.section)
		}
		if result.LogType != tt.logType {
			t.Errorf("ParseContentInput(%q).LogType = %q, want %q", tt.input, result.LogType, tt.logType)
		}
		if result.RawContent != tt.raw {
			t.Errorf("ParseContentInput(%q).RawContent = %q, want %q", tt.input, result.RawContent, tt.raw)
		}
	}
}

func TestParseContentInputFormatted(t *testing.T) {
	result := ParseContentInput("[] Buy groceries")
	if result.FormattedContent != "- [ ] Buy groceries" {
		t.Errorf("FormattedContent = %q, want %q", result.FormattedContent, "- [ ] Buy groceries")
	}

	result = ParseContentInput("- Learn Go")
	if result.FormattedContent != "- Learn Go" {
		t.Errorf("FormattedContent = %q, want %q", result.FormattedContent, "- Learn Go")
	}
}

func TestFindSectionIndex(t *testing.T) {
	lines := []string{
		"# Title",
		"",
		"## Tasks",
		"- [ ] Something",
		"",
		"## Ideas",
		"- An idea",
	}

	if idx := FindSectionIndex(lines, "Tasks"); idx != 2 {
		t.Errorf("FindSectionIndex(Tasks) = %d, want 2", idx)
	}
	if idx := FindSectionIndex(lines, "Ideas"); idx != 5 {
		t.Errorf("FindSectionIndex(Ideas) = %d, want 5", idx)
	}
	if idx := FindSectionIndex(lines, "Missing"); idx != -1 {
		t.Errorf("FindSectionIndex(Missing) = %d, want -1", idx)
	}
}

func TestAddToSection(t *testing.T) {
	content := "# Title\n\n## Tasks\n\n## Ideas\n"

	result := AddToSection(content, "Tasks", "- [ ] New task")
	if result == nil {
		t.Fatal("AddToSection returned nil")
	}
	if !strings.Contains(result.NewContent, "- [ ] New task") {
		t.Error("NewContent should contain the new task")
	}

	// Section not found
	result = AddToSection(content, "Missing", "content")
	if result != nil {
		t.Error("AddToSection should return nil for missing section")
	}
}

func TestAddToSectionWithExistingContent(t *testing.T) {
	content := "# Title\n\n## Tasks\n- [ ] Existing\n\n## Ideas\n"

	result := AddToSection(content, "Tasks", "- [ ] New task")
	if result == nil {
		t.Fatal("AddToSection returned nil")
	}

	lines := strings.Split(result.NewContent, "\n")
	// The new task should be after "- [ ] Existing"
	foundExisting := false
	foundNew := false
	for _, line := range lines {
		if line == "- [ ] Existing" {
			foundExisting = true
		}
		if line == "- [ ] New task" {
			if !foundExisting {
				t.Error("New task should appear after existing task")
			}
			foundNew = true
		}
	}
	if !foundNew {
		t.Error("New task not found in output")
	}
}

func TestAddContent(t *testing.T) {
	original := "Line 1\nLine 2"

	// Append
	result := AddContent(original, "Line 3", "append")
	if !strings.HasSuffix(result.NewContent, "Line 3") {
		t.Errorf("Append: got %q", result.NewContent)
	}

	// Prepend
	result = AddContent(original, "Line 0", "prepend")
	if !strings.HasPrefix(result.NewContent, "Line 0") {
		t.Errorf("Prepend: got %q", result.NewContent)
	}
	if result.InsertedLine != 1 {
		t.Errorf("Prepend InsertedLine = %d, want 1", result.InsertedLine)
	}

	// Replace
	result = AddContent(original, "Replaced", "replace")
	if result.NewContent != "Replaced" {
		t.Errorf("Replace: got %q", result.NewContent)
	}
}

func TestInsertContentAtLine(t *testing.T) {
	content := "Line 0\nLine 1\nLine 2"

	result := InsertContentAtLine(content, "Inserted", 1)
	if result == nil {
		t.Fatal("InsertContentAtLine returned nil")
	}
	lines := strings.Split(result.NewContent, "\n")
	if lines[1] != "Inserted" {
		t.Errorf("lines[1] = %q, want %q", lines[1], "Inserted")
	}
	if result.InsertedLine != 2 {
		t.Errorf("InsertedLine = %d, want 2", result.InsertedLine)
	}

	// Out of bounds
	result = InsertContentAtLine(content, "Bad", -1)
	if result != nil {
		t.Error("Expected nil for negative line number")
	}
}

func TestReplaceContentAtLine(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"

	result := ReplaceContentAtLine(content, "Replaced", 2)
	if result == nil {
		t.Fatal("ReplaceContentAtLine returned nil")
	}
	lines := strings.Split(result.NewContent, "\n")
	if lines[1] != "Replaced" {
		t.Errorf("lines[1] = %q, want %q", lines[1], "Replaced")
	}

	// Out of bounds
	result = ReplaceContentAtLine(content, "Bad", 0)
	if result != nil {
		t.Error("Expected nil for line 0")
	}
	result = ReplaceContentAtLine(content, "Bad", 4)
	if result != nil {
		t.Error("Expected nil for line beyond end")
	}
}

func TestProcessTemplate(t *testing.T) {
	template := "# {{date:YYYY-MM-DD}}\n\nSome content"
	result := ProcessTemplate(template)

	if strings.Contains(result, "{{date:YYYY-MM-DD}}") {
		t.Error("Template placeholder was not replaced")
	}
	if !strings.HasPrefix(result, "# 20") {
		t.Errorf("Expected date to start with '20', got %q", result[:10])
	}
}

func TestInjectMetadata(t *testing.T) {
	// No existing metadata, starts with heading
	content := "# My Note\nSome content"
	result := InjectMetadata(content)
	if !strings.Contains(result, "updated_at:") {
		t.Error("Should inject updated_at")
	}
	if !strings.Contains(result, "---") {
		t.Error("Should inject frontmatter delimiters")
	}

	// Existing metadata
	content = "# Title\n---\nupdated_at: 2024-01-01T00:00:00Z\n---\nContent"
	result = InjectMetadata(content)
	if strings.Contains(result, "2024-01-01") {
		t.Error("Should update the timestamp, not keep the old one")
	}

	// No heading, no metadata
	content = "Just plain content"
	result = InjectMetadata(content)
	if result != content {
		t.Error("Should not modify content without heading")
	}
}
