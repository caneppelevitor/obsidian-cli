package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/caneppelevitor/obsidian-cli/internal/content"
)

// TodayDate returns today's date in YYYY-MM-DD format.
func TodayDate() string {
	now := time.Now()
	return fmt.Sprintf("%d-%02d-%02d", now.Year(), int(now.Month()), now.Day())
}

// MonthFolder returns the current month folder in YYYY-MM format.
func MonthFolder() string {
	now := time.Now()
	return fmt.Sprintf("%d-%02d", now.Year(), int(now.Month()))
}

// DailyNoteFilename returns today's daily note filename.
func DailyNoteFilename() string {
	return TodayDate() + ".md"
}

// DailyNotePath returns the full path to today's daily note.
func DailyNotePath(vaultPath string) string {
	return filepath.Join(vaultPath, MonthFolder(), DailyNoteFilename())
}

// DailyNoteTemplate is the default template for new daily notes.
const DailyNoteTemplate = `# {{date:YYYY-MM-DD}}

##  Insights

## Tasks

## Ideas

## Questions

## Links to Expand

## Tags
#daily #inbox
`

// EnsureDailyNote creates the daily note if it doesn't exist.
// Returns the file path and content.
func EnsureDailyNote(vaultPath string) (filePath string, fileContent string, created bool, err error) {
	notePath := DailyNotePath(vaultPath)
	monthDir := filepath.Join(vaultPath, MonthFolder())

	// Ensure vault directory exists
	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		return "", "", false, fmt.Errorf("creating vault directory: %w", err)
	}

	// Ensure month folder exists
	if err := os.MkdirAll(monthDir, 0o755); err != nil {
		return "", "", false, fmt.Errorf("creating month folder: %w", err)
	}

	// Check if daily note exists
	if _, statErr := os.Stat(notePath); statErr == nil {
		// File exists, read it
		data, err := os.ReadFile(notePath)
		if err != nil {
			return "", "", false, fmt.Errorf("reading daily note: %w", err)
		}
		return notePath, string(data), false, nil
	}

	// Create new daily note from template
	processedTemplate := content.ProcessTemplate(DailyNoteTemplate)
	if err := os.WriteFile(notePath, []byte(processedTemplate), 0o644); err != nil {
		return "", "", false, fmt.Errorf("creating daily note: %w", err)
	}

	return notePath, processedTemplate, true, nil
}
