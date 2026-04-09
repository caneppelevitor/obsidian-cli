package vault

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CountUncheckedItems reads a file and counts lines containing "- [ ]".
// Returns 0 (not error) if the file doesn't exist.
func CountUncheckedItems(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "- [ ]") {
			count++
		}
	}
	return count, scanner.Err()
}

// CountUncheckedInSection counts "- [ ]" lines under a specific ## heading
// until the next heading of equal or higher level. Returns 0 if file or
// section doesn't exist.
func CountUncheckedInSection(filePath, sectionHeading string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	count := 0
	inSection := false
	target := "## " + sectionHeading

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == target {
			inSection = true
			continue
		}

		if inSection {
			// Stop at next heading of equal or higher level
			if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "# ") {
				break
			}
			if strings.Contains(line, "- [ ]") {
				count++
			}
		}
	}

	return count, scanner.Err()
}

// CountFilesModifiedAfter counts .md files in dirPath with modification time
// after the given time. Returns 0 (not error) if directory doesn't exist.
// If after is zero value, counts all .md files.
func CountFilesModifiedAfter(dirPath string, after time.Time) (int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if after.IsZero() {
			count++
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(after) {
			count++
		}
	}

	// Also check subdirectories
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subCount, err := CountFilesModifiedAfter(filepath.Join(dirPath, entry.Name()), after)
		if err != nil {
			continue
		}
		count += subCount
	}

	return count, nil
}
