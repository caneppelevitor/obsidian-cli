package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// mu serializes file writes to prevent concurrent access issues.
var mu sync.Mutex

// ReadFile reads the content of a file.
func ReadFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes content to a file, serialized with a mutex.
func WriteFile(filePath, content string) error {
	mu.Lock()
	defer mu.Unlock()

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	return os.WriteFile(filePath, []byte(content), 0o644)
}

// ListMarkdownFiles recursively finds all .md files in a directory,
// skipping hidden directories (starting with .).
// Returns paths relative to baseDir.
func ListMarkdownFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != dir {
			return filepath.SkipDir
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}

		return nil
	})

	return files, err
}

// FileInfo holds metadata about a markdown file.
type FileInfo struct {
	RelPath  string
	ModTime  string
}

// ListMarkdownFilesWithInfo returns markdown files with modification dates.
func ListMarkdownFilesWithInfo(dir string) ([]FileInfo, error) {
	files, err := ListMarkdownFiles(dir)
	if err != nil {
		return nil, err
	}

	var infos []FileInfo
	for _, f := range files {
		fullPath := filepath.Join(dir, f)
		stat, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		infos = append(infos, FileInfo{
			RelPath: f,
			ModTime: stat.ModTime().Format("2006-01-02"),
		})
	}

	return infos, nil
}
