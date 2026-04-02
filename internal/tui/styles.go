package tui

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

// Color definitions matching the Node.js version's 256-color palette.
var (
	colorCyan    = lipgloss.Color("6")
	colorMagenta = lipgloss.Color("5")
	colorGreen   = lipgloss.Color("2")
	colorYellow  = lipgloss.Color("3")
	colorBlue    = lipgloss.Color("4")
	colorGray    = lipgloss.Color("8")
	colorWhite   = lipgloss.Color("7")
	colorRed     = lipgloss.Color("1")
)

// Styles used throughout the TUI.
var (
	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(colorGray)

	tabBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("4")).
			Foreground(lipgloss.Color("15"))

	statusBarStyle = lipgloss.NewStyle().
			Reverse(true)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan)

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(colorCyan)

	cheatSheetStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	// Markdown line styles
	headingH1Style = lipgloss.NewStyle().
			Foreground(colorMagenta).
			Bold(true)

	headingH2Style = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	checkboxStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	bulletStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	hashtagStyle = lipgloss.NewStyle().
			Foreground(colorBlue)
)

// Regex patterns for markdown styling.
var (
	reH2       = regexp.MustCompile(`^##\s+`)
	reH1       = regexp.MustCompile(`^#\s+`)
	reCheckbox = regexp.MustCompile(`^\s*-\s+\[[ x]\]\s+`)
	reBullet   = regexp.MustCompile(`^\s*-\s+`)
	reHashtag  = regexp.MustCompile(`^#\w+`)
)

// StyleMarkdownLine applies syntax highlighting to a single line of markdown.
func StyleMarkdownLine(line string, eisenhowerTags map[string]string) string {
	switch {
	case reH2.MatchString(line):
		styled := applyEisenhowerTags(line, eisenhowerTags)
		return headingH2Style.Render(styled)

	case reH1.MatchString(line):
		styled := applyEisenhowerTags(line, eisenhowerTags)
		return headingH1Style.Render(styled)

	case reCheckbox.MatchString(line):
		styled := applyEisenhowerTags(line, eisenhowerTags)
		return checkboxStyle.Render(styled)

	case reBullet.MatchString(line):
		styled := applyEisenhowerTags(line, eisenhowerTags)
		return bulletStyle.Render(styled)

	case reHashtag.MatchString(line):
		return hashtagStyle.Render(line)

	default:
		return applyEisenhowerTags(line, eisenhowerTags)
	}
}

// applyEisenhowerTags replaces Eisenhower tag text with colored versions.
func applyEisenhowerTags(line string, tags map[string]string) string {
	if tags == nil {
		return line
	}

	for tag, colorCode := range tags {
		if strings.Contains(line, tag) {
			tagStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorCode)).
				Bold(true)
			coloredTag := tagStyle.Render(tag)
			line = strings.ReplaceAll(line, tag, coloredTag)
		}
	}

	return line
}

// RenderTabBar renders the tab bar with the active tab highlighted.
func RenderTabBar(tabs []string, activeTab int, width int) string {
	var parts []string
	for i, tab := range tabs {
		if i == activeTab {
			parts = append(parts, tabActiveStyle.Render(" ● "+tab+" "))
		} else {
			parts = append(parts, tabInactiveStyle.Render(" "+tab+" "))
		}
	}
	content := strings.Join(parts, tabInactiveStyle.Render("|"))
	return tabBarStyle.Width(width).Render(content)
}
