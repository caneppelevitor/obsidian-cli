package vault

import (
	"io/fs"
	"path/filepath"
)

// Vault structure path helpers. Centralizes all Knowledge/System path
// construction so callers don't hardcode vault directory layout.

func LastCompilePath(vaultRoot string) string {
	return filepath.Join(vaultRoot, "System", "last-compile.md")
}

func WikiInboxPath(vaultRoot string) string {
	return filepath.Join(vaultRoot, "Knowledge", "wiki", "_inbox.md")
}

func ZettelkastenDir(vaultRoot string) string {
	return filepath.Join(vaultRoot, "Knowledge", "zettelkasten")
}

func ReviewQueuePath(vaultRoot string) string {
	return filepath.Join(vaultRoot, "Knowledge", "zettelkasten", "_review-queue.md")
}

func RawNotesDir(vaultRoot string) string {
	return filepath.Join(vaultRoot, "Knowledge", "zettelkasten", "1-raw-notes")
}

// FindFile searches recursively in dir for a file matching name+".md".
// Returns the full path or empty string if not found.
func FindFile(dir, name string) string {
	target := name + ".md"
	var found string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == target {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
