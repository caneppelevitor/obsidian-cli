package vault

import (
	"os"
	"strings"
)

// LogToCentralFile appends a log entry to a central file.
// If the file doesn't exist, it creates it with the headerTemplate.
// The insertPredicate finds the first matching line to insert before.
// If no match is found, the entry is appended at the end.
func LogToCentralFile(logPath, logEntry, headerTemplate string, insertPredicate func(string) bool) error {
	mu.Lock()
	defer mu.Unlock()

	existingContent := ""
	data, err := os.ReadFile(logPath)
	if err != nil {
		existingContent = headerTemplate
	} else {
		existingContent = string(data)
	}

	lines := strings.Split(existingContent, "\n")

	insertIndex := -1
	for i, line := range lines {
		if insertPredicate(line) {
			insertIndex = i
			break
		}
	}

	if insertIndex == -1 {
		lines = append(lines, "", logEntry)
	} else {
		// Insert before the matching line
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:insertIndex]...)
		result = append(result, logEntry)
		result = append(result, lines[insertIndex:]...)
		lines = result
	}

	updatedContent := strings.Join(lines, "\n")

	dir := logPath[:strings.LastIndex(logPath, string(os.PathSeparator))]
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(logPath, []byte(updatedContent), 0o644)
}
