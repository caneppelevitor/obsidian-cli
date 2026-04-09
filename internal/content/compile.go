package content

import (
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// CompileFrontmatter holds parsed YAML frontmatter from last-compile.md.
type CompileFrontmatter struct {
	LastCompile     time.Time `yaml:"last_compile"`
	DurationSeconds int       `yaml:"duration_seconds"`
}

// SectionMetrics holds key-value pairs from a compile summary section.
type SectionMetrics struct {
	Items map[string]string
}

// LintMetrics holds lint findings from a compile summary.
type LintMetrics struct {
	Items       map[string]string
	HasWarnings bool
}

// CompileResult holds the full parsed output of a compile run.
type CompileResult struct {
	Frontmatter  CompileFrontmatter
	Wiki         SectionMetrics
	Zettelkasten SectionMetrics
	Lint         LintMetrics
	Suggestions  []string
	RawBody      string
}

// VaultStatus holds a point-in-time vault health snapshot.
type VaultStatus struct {
	LastCompile          *time.Time
	WikiInboxCount       int
	ReviewQueueCount     int
	RawNotesSinceCompile int
}

// ExtractFrontmatter splits a markdown file into its YAML frontmatter block
// and the remaining body. Returns empty yamlBlock if no frontmatter found.
func ExtractFrontmatter(content string) (yamlBlock string, body string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return "", content
	}

	// First line must be ---
	if strings.TrimSpace(lines[0]) != "---" {
		return "", content
	}

	// Find closing ---
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			yamlBlock = strings.Join(lines[1:i], "\n")
			body = strings.Join(lines[i+1:], "\n")
			return yamlBlock, body
		}
	}

	// No closing --- found
	return "", content
}

// ParseCompileFrontmatter parses YAML content into a CompileFrontmatter struct.
func ParseCompileFrontmatter(yamlContent string) (CompileFrontmatter, error) {
	var fm CompileFrontmatter
	err := yaml.Unmarshal([]byte(yamlContent), &fm)
	return fm, err
}

// ParseCompileResult parses the full content of last-compile.md into a CompileResult.
func ParseCompileResult(content string) (*CompileResult, error) {
	yamlBlock, body := ExtractFrontmatter(content)

	var result CompileResult
	result.RawBody = body

	if yamlBlock != "" {
		fm, err := ParseCompileFrontmatter(yamlBlock)
		if err != nil {
			return &result, err
		}
		result.Frontmatter = fm
	}

	// Parse body sections
	sections := splitSections(body)

	if items, ok := sections["Wiki"]; ok {
		result.Wiki = SectionMetrics{Items: parseKeyValues(items)}
	}
	if items, ok := sections["Zettelkasten"]; ok {
		result.Zettelkasten = SectionMetrics{Items: parseKeyValues(items)}
	}
	if items, ok := sections["Lint"]; ok {
		lintItems := parseKeyValues(items)
		hasWarnings := false
		for _, v := range lintItems {
			lower := strings.ToLower(strings.TrimSpace(v))
			if lower != "none" && lower != "0" && lower != "" {
				hasWarnings = true
				break
			}
		}
		result.Lint = LintMetrics{Items: lintItems, HasWarnings: hasWarnings}
	}
	if items, ok := sections["Suggestions"]; ok {
		result.Suggestions = parseBullets(items)
	}

	return &result, nil
}

// splitSections splits markdown body by ## headings into named sections.
func splitSections(body string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(body, "\n")

	var currentSection string
	var currentLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if currentSection != "" {
				sections[currentSection] = strings.Join(currentLines, "\n")
			}
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentLines = nil
		} else if currentSection != "" {
			currentLines = append(currentLines, line)
		}
	}
	if currentSection != "" {
		sections[currentSection] = strings.Join(currentLines, "\n")
	}

	return sections
}

// parseKeyValues extracts "Label: value" pairs from section content.
func parseKeyValues(content string) map[string]string {
	items := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		// Strip leading bullet/dash
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			if key != "" {
				items[key] = val
			}
		}
	}
	return items
}

// parseBullets extracts bullet items from section content.
func parseBullets(content string) []string {
	var items []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			item := strings.TrimPrefix(line, "- ")
			item = strings.TrimPrefix(item, "* ")
			if item != "" {
				items = append(items, strings.TrimSpace(item))
			}
		}
	}
	return items
}
