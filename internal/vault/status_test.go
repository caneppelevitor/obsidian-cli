package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCountUncheckedItems(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.md")

	content := "# Inbox\n\n- [ ] Item 1\n- [x] Done item\n- [ ] Item 2\n- Regular line\n"
	os.WriteFile(path, []byte(content), 0o644)

	count, err := CountUncheckedItems(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestCountUncheckedItemsMissingFile(t *testing.T) {
	count, err := CountUncheckedItems("/nonexistent/file.md")
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestCountUncheckedInSection(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "review.md")

	content := "# Queue\n\n## Pending\n\n- [ ] Draft A\n- [ ] Draft B\n- [x] Done\n\n## Approved\n\n- [x] Old\n"
	os.WriteFile(path, []byte(content), 0o644)

	count, err := CountUncheckedInSection(path, "Pending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestCountUncheckedInSectionMissing(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.md")
	os.WriteFile(path, []byte("# No pending section\n"), 0o644)

	count, err := CountUncheckedInSection(path, "Pending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestCountFilesModifiedAfter(t *testing.T) {
	tmp := t.TempDir()

	// Create files with known mod times
	past := time.Now().Add(-24 * time.Hour)
	recent := time.Now()

	oldFile := filepath.Join(tmp, "old.md")
	os.WriteFile(oldFile, []byte("old"), 0o644)
	os.Chtimes(oldFile, past, past)

	newFile := filepath.Join(tmp, "new.md")
	os.WriteFile(newFile, []byte("new"), 0o644)
	os.Chtimes(newFile, recent, recent)

	// Also a non-md file
	os.WriteFile(filepath.Join(tmp, "skip.txt"), []byte("skip"), 0o644)

	count, err := CountFilesModifiedAfter(tmp, past.Add(time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (only new.md)", count)
	}
}

func TestCountFilesModifiedAfterZeroTime(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a.md"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(tmp, "b.md"), []byte("b"), 0o644)

	count, err := CountFilesModifiedAfter(tmp, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (zero time = count all)", count)
	}
}

func TestCountFilesModifiedAfterMissingDir(t *testing.T) {
	count, err := CountFilesModifiedAfter("/nonexistent/dir", time.Now())
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}
