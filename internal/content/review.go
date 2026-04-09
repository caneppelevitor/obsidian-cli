package content

import (
	"fmt"
	"regexp"
	"strings"
)

// ReviewItem represents a pending draft from _review-queue.md.
type ReviewItem struct {
	Name    string
	Line    string
	Checked bool
}

// ParseReviewItems extracts unchecked items from the ## Pending section of
// _review-queue.md content. Items must match `- [ ] [[name]]` pattern.
func ParseReviewItems(fileContent string) []ReviewItem {
	lines := strings.Split(fileContent, "\n")
	inPending := false
	wikiLinkRe := regexp.MustCompile(`\[\[([^\]]+)\]\]`)

	var items []ReviewItem
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "## Pending" {
			inPending = true
			continue
		}
		if inPending && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if !inPending {
			continue
		}

		if strings.Contains(line, "- [ ]") {
			matches := wikiLinkRe.FindStringSubmatch(line)
			if len(matches) >= 2 {
				items = append(items, ReviewItem{
					Name:    matches[1],
					Line:    line,
					Checked: false,
				})
			}
		}
	}

	return items
}

// ApproveReviewItem moves an item from ## Pending to ## Approved (checked)
// in _review-queue.md content. Creates ## Approved section if missing.
func ApproveReviewItem(fileContent, itemName string) string {
	return moveReviewItem(fileContent, itemName, "Approved")
}

// DiscardReviewItem moves an item from ## Pending to ## Discarded
// in _review-queue.md content. Creates ## Discarded section if missing.
func DiscardReviewItem(fileContent, itemName string) string {
	return moveReviewItem(fileContent, itemName, "Discarded")
}

func moveReviewItem(fileContent, itemName, targetSection string) string {
	lines := strings.Split(fileContent, "\n")
	target := fmt.Sprintf("[[%s]]", itemName)

	// Remove the item from ## Pending
	var result []string
	removed := false
	for _, line := range lines {
		if strings.Contains(line, target) && strings.Contains(line, "- [ ]") && !removed {
			removed = true
			continue
		}
		result = append(result, line)
	}

	if !removed {
		return fileContent
	}

	// Find or create the target section and append the checked item
	content := strings.Join(result, "\n")
	checkedLine := fmt.Sprintf("- [x] [[%s]]", itemName)
	sectionHeader := fmt.Sprintf("## %s", targetSection)

	if idx := strings.Index(content, sectionHeader); idx >= 0 {
		// Find the end of the section header line
		afterHeader := idx + len(sectionHeader)
		rest := content[afterHeader:]
		// Find the next line after the header
		nlIdx := strings.Index(rest, "\n")
		if nlIdx >= 0 {
			insertAt := afterHeader + nlIdx + 1
			content = content[:insertAt] + checkedLine + "\n" + content[insertAt:]
		} else {
			content += "\n" + checkedLine + "\n"
		}
	} else {
		// Append the section at the end
		content = strings.TrimRight(content, "\n") + "\n\n" + sectionHeader + "\n\n" + checkedLine + "\n"
	}

	return content
}
