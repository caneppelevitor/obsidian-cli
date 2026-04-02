package content

import (
	"fmt"
	"strings"
	"time"
)

// ProcessTemplate replaces {{date:YYYY-MM-DD}} placeholders with the current date.
func ProcessTemplate(template string) string {
	now := time.Now()
	dateStr := fmt.Sprintf("%d-%02d-%02d", now.Year(), int(now.Month()), now.Day())
	return dateTemplateRe.ReplaceAllString(template, dateStr)
}

// InjectMetadata adds or updates YAML frontmatter with updated_at timestamp.
func InjectMetadata(content string) string {
	lines := strings.Split(content, "\n")
	now := time.Now().UTC().Format(time.RFC3339)

	hasMetadata := false
	for _, line := range lines {
		if strings.Contains(line, "updated_at:") {
			hasMetadata = true
			break
		}
	}

	if !hasMetadata && len(lines) > 0 {
		metadata := []string{
			"---",
			fmt.Sprintf("updated_at: %s", now),
			"---",
		}

		if strings.HasPrefix(lines[0], "#") {
			// Insert after first heading
			result := make([]string, 0, len(lines)+len(metadata))
			result = append(result, lines[0])
			result = append(result, metadata...)
			result = append(result, lines[1:]...)
			return strings.Join(result, "\n")
		}
	} else if hasMetadata {
		for i, line := range lines {
			if strings.Contains(line, "updated_at:") {
				lines[i] = fmt.Sprintf("updated_at: %s", now)
			}
		}
		return strings.Join(lines, "\n")
	}

	return content
}
