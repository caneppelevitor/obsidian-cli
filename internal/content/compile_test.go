package content

import (
	"testing"
)

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantYAML string
		wantBody string
	}{
		{
			"with frontmatter",
			"---\nlast_compile: 2026-04-08\nduration_seconds: 45\n---\n\n## Wiki\nContent here",
			"last_compile: 2026-04-08\nduration_seconds: 45",
			"\n## Wiki\nContent here",
		},
		{
			"no frontmatter",
			"# Just a title\n\nSome content",
			"",
			"# Just a title\n\nSome content",
		},
		{
			"empty content",
			"",
			"",
			"",
		},
		{
			"unclosed frontmatter",
			"---\nkey: value\nno closing",
			"",
			"---\nkey: value\nno closing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, body := ExtractFrontmatter(tt.input)
			if yaml != tt.wantYAML {
				t.Errorf("yaml = %q, want %q", yaml, tt.wantYAML)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestParseCompileFrontmatter(t *testing.T) {
	yaml := "last_compile: 2026-04-08T10:00:00Z\nduration_seconds: 45"
	fm, err := ParseCompileFrontmatter(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.LastCompile.IsZero() {
		t.Error("LastCompile should not be zero")
	}
	if fm.DurationSeconds != 45 {
		t.Errorf("DurationSeconds = %d, want 45", fm.DurationSeconds)
	}
}

func TestParseCompileFrontmatterEmpty(t *testing.T) {
	fm, err := ParseCompileFrontmatter("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fm.LastCompile.IsZero() {
		t.Error("LastCompile should be zero for empty input")
	}
}

func TestParseCompileResult(t *testing.T) {
	input := "---\nlast_compile: 2026-04-08T10:00:00Z\nduration_seconds: 30\n---\n\n## Wiki\n\nRaw items processed: 3\nArticles created: 1\n\n## Zettelkasten\n\nRaw notes scanned: 5\n\n## Lint\n\nOrphan notes: none\nBroken links: 2\n\n## Suggestions\n\n- Connect A to B\n- Review C\n"

	result, err := ParseCompileResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Frontmatter.DurationSeconds != 30 {
		t.Errorf("DurationSeconds = %d, want 30", result.Frontmatter.DurationSeconds)
	}

	if len(result.Wiki.Items) == 0 {
		t.Error("Wiki.Items should not be empty")
	}

	if result.Lint.Items["Broken links"] != "2" {
		t.Errorf("Lint broken links = %q, want %q", result.Lint.Items["Broken links"], "2")
	}

	if !result.Lint.HasWarnings {
		t.Error("Lint.HasWarnings should be true (broken links = 2)")
	}

	if len(result.Suggestions) != 2 {
		t.Errorf("Suggestions count = %d, want 2", len(result.Suggestions))
	}
}

func TestParseCompileResultMissingSections(t *testing.T) {
	input := "---\nlast_compile: 2026-04-08T10:00:00Z\n---\n\nJust some text, no sections"

	result, err := ParseCompileResult(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Wiki.Items) != 0 {
		t.Error("Wiki.Items should be empty for missing section")
	}
	if result.RawBody == "" {
		t.Error("RawBody should contain the body text")
	}
}
