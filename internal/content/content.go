package content

import (
	"fmt"
	"regexp"
	"strings"
)

// ContentResult holds the result of a content manipulation operation.
type ContentResult struct {
	NewContent   string
	InsertedLine int // 1-based line number
}

// ParsedInput holds the result of parsing a prefixed user input.
type ParsedInput struct {
	Section          string // "Tasks", "Ideas", "Questions", "Insights"
	FormattedContent string // "- [ ] Buy groceries"
	LogType          string // "task", "idea", "question", "insight"
	RawContent       string // "Buy groceries"
}

// FindSectionIndex returns the 0-based index of the "## sectionName" header, or -1.
func FindSectionIndex(lines []string, sectionName string) int {
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") && strings.Contains(line, sectionName) {
			return i
		}
	}
	return -1
}

// AddToSection inserts content after the last non-empty line in the named section.
// Returns nil if the section is not found.
func AddToSection(currentContent, sectionName, content string) *ContentResult {
	lines := strings.Split(currentContent, "\n")
	sectionIndex := FindSectionIndex(lines, sectionName)

	if sectionIndex == -1 {
		return nil
	}

	insertIndex := sectionIndex + 1
	lastContentLine := sectionIndex
	hasContent := false

	for insertIndex < len(lines) && !strings.HasPrefix(lines[insertIndex], "## ") {
		if strings.TrimSpace(lines[insertIndex]) != "" {
			lastContentLine = insertIndex
			hasContent = true
		}
		insertIndex++
	}

	var actualInsertLine int
	if !hasContent {
		// Insert right after section header
		lines = insertAt(lines, sectionIndex+1, content)
		actualInsertLine = sectionIndex + 1
	} else {
		// Insert after last content line
		lines = insertAt(lines, lastContentLine+1, content)
		actualInsertLine = lastContentLine + 1
	}

	return &ContentResult{
		NewContent:   strings.Join(lines, "\n"),
		InsertedLine: actualInsertLine + 1, // 1-based
	}
}

// AddContent appends, prepends, or replaces content.
func AddContent(currentContent, newContent, mode string) *ContentResult {
	lines := strings.Split(currentContent, "\n")

	switch mode {
	case "prepend":
		return &ContentResult{
			NewContent:   newContent + "\n" + currentContent,
			InsertedLine: 1,
		}
	case "replace":
		return &ContentResult{
			NewContent:   newContent,
			InsertedLine: 1,
		}
	default: // "append"
		return &ContentResult{
			NewContent:   currentContent + "\n" + newContent,
			InsertedLine: len(lines) + 1,
		}
	}
}

// InsertContentAtLine inserts content at the given 0-based line number.
// Returns nil if lineNumber is out of bounds.
func InsertContentAtLine(currentContent, content string, lineNumber int) *ContentResult {
	lines := strings.Split(currentContent, "\n")

	if lineNumber < 0 || lineNumber > len(lines) {
		return nil
	}

	lines = insertAt(lines, lineNumber, content)
	return &ContentResult{
		NewContent:   strings.Join(lines, "\n"),
		InsertedLine: lineNumber + 1,
	}
}

// ReplaceContentAtLine replaces the line at the given 1-based line number.
// Returns nil if lineNumber is out of bounds.
func ReplaceContentAtLine(currentContent, content string, lineNumber int) *ContentResult {
	lines := strings.Split(currentContent, "\n")

	if lineNumber < 1 || lineNumber > len(lines) {
		return nil
	}

	lines[lineNumber-1] = content
	return &ContentResult{
		NewContent: strings.Join(lines, "\n"),
	}
}

// ParseContentInput parses prefixed input into a structured result.
// Returns nil if the input doesn't match any known prefix.
func ParseContentInput(input string) *ParsedInput {
	trimmed := strings.TrimSpace(input)

	if strings.HasPrefix(trimmed, "[]") {
		raw := strings.TrimSpace(trimmed[2:])
		return &ParsedInput{
			Section:          "Tasks",
			FormattedContent: fmt.Sprintf("- [ ] %s", raw),
			LogType:          "task",
			RawContent:       raw,
		}
	}

	if strings.HasPrefix(trimmed, "-") {
		raw := strings.TrimSpace(trimmed[1:])
		return &ParsedInput{
			Section:          "Ideas",
			FormattedContent: fmt.Sprintf("- %s", raw),
			LogType:          "idea",
			RawContent:       raw,
		}
	}

	if strings.HasPrefix(trimmed, "?") {
		raw := strings.TrimSpace(trimmed[1:])
		return &ParsedInput{
			Section:          "Questions",
			FormattedContent: fmt.Sprintf("- %s", raw),
			LogType:          "question",
			RawContent:       raw,
		}
	}

	if strings.HasPrefix(trimmed, "!") {
		raw := strings.TrimSpace(trimmed[1:])
		return &ParsedInput{
			Section:          "Insights",
			FormattedContent: fmt.Sprintf("- %s", raw),
			LogType:          "insight",
			RawContent:       raw,
		}
	}

	return nil
}

func insertAt(lines []string, index int, value string) []string {
	result := make([]string, len(lines)+1)
	copy(result, lines[:index])
	result[index] = value
	copy(result[index+1:], lines[index:])
	return result
}

var dateTemplateRe = regexp.MustCompile(`\{\{date:YYYY-MM-DD\}\}`)
