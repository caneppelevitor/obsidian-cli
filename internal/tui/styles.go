package tui

import (
	"fmt"
	"image/color"
	"os"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

// ─── Adaptive Color Palette ─────────────────────────────────────────────────
// Uses Catppuccin Mocha (dark) / Latte (light) with automatic detection.
var ld = lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))

var (
	colorBase     = ld(lipgloss.Color("#eff1f5"), lipgloss.Color("#1e1e2e"))
	colorSurface0 = ld(lipgloss.Color("#ccd0da"), lipgloss.Color("#313244"))
	colorSurface1 = ld(lipgloss.Color("#bcc0cc"), lipgloss.Color("#45475a"))
	colorOverlay  = ld(lipgloss.Color("#7c7f93"), lipgloss.Color("#6c7086"))
	colorSubtext  = ld(lipgloss.Color("#5c5f77"), lipgloss.Color("#a6adc8"))
	colorText     = ld(lipgloss.Color("#4c4f69"), lipgloss.Color("#cdd6f4"))

	colorBlue     = ld(lipgloss.Color("#1e66f5"), lipgloss.Color("#89b4fa"))
	colorGreen    = ld(lipgloss.Color("#40a02b"), lipgloss.Color("#a6e3a1"))
	colorPeach    = ld(lipgloss.Color("#fe640b"), lipgloss.Color("#fab387"))
	colorMauve    = ld(lipgloss.Color("#8839ef"), lipgloss.Color("#cba6f7"))
	colorRed      = ld(lipgloss.Color("#d20f39"), lipgloss.Color("#f38ba8"))
	colorLavender = ld(lipgloss.Color("#7287fd"), lipgloss.Color("#b4befe"))
	colorYellow   = ld(lipgloss.Color("#df8e1d"), lipgloss.Color("#f9e2af"))
	colorTeal     = ld(lipgloss.Color("#179299"), lipgloss.Color("#94e2d5"))
)

// ─── Eisenhower Tag Colors (harmonized with palette) ────────────────────────
// These override the config values for display; config still stores 256-color indices
var eisenhowerDisplayColors = map[string]color.Color{
	"#do":        lipgloss.Color("#f38ba8"), // urgent + important (red)
	"#delegate":  lipgloss.Color("#fab387"), // urgent + not important (peach)
	"#schedule":  lipgloss.Color("#89b4fa"), // not urgent + important (blue)
	"#eliminate": lipgloss.Color("#6c7086"), // not urgent + not important (dim)
}

// ─── Border Styles ──────────────────────────────────────────────────────────
var (
	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBlue)

	inactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSurface1)
)

// ─── Tab Bar ────────────────────────────────────────────────────────────────
var (
	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(colorOverlay)

	tabUnderlineStyle = lipgloss.NewStyle().
				Foreground(colorBlue)
)

// ─── Status Bar ─────────────────────────────────────────────────────────────
var (
	statusBarStyle = lipgloss.NewStyle().
			Background(colorSurface0).
			Foreground(colorSubtext)

	statusBarDimStyle = lipgloss.NewStyle().
				Background(colorSurface0).
				Foreground(colorOverlay)
)

// ─── Input ──────────────────────────────────────────────────────────────────
var (
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(colorBlue).
				Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(colorSurface1)
)

// ─── Content ────────────────────────────────────────────────────────────────
var (
	lineNumberStyle = lipgloss.NewStyle().
			Foreground(colorOverlay)

	cheatSheetStyle = lipgloss.NewStyle().
			Foreground(colorOverlay)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorOverlay)
)

// ─── Markdown Styles ────────────────────────────────────────────────────────
var (
	headingH1Style = lipgloss.NewStyle().
			Foreground(colorMauve).
			Bold(true)

	headingH2Style = lipgloss.NewStyle().
			Foreground(colorTeal).
			Bold(true)

	checkboxStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	checkboxDoneStyle = lipgloss.NewStyle().
				Foreground(colorOverlay).
				Strikethrough(true)

	bulletStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	hashtagStyle = lipgloss.NewStyle().
			Foreground(colorLavender)

	quoteStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Italic(true)
)

// ─── Input Mode Pills ──────────────────────────────────────────────────────
func inputModePill(label string, bg color.Color) string {
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(colorBase).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

// DetectInputMode returns a styled pill based on the current input prefix.
func DetectInputMode(input string) string {
	trimmed := strings.TrimSpace(input)
	switch {
	case strings.HasPrefix(trimmed, "[]"):
		return inputModePill("TASK", colorGreen)
	case strings.HasPrefix(trimmed, "-"):
		return inputModePill("IDEA", colorYellow)
	case strings.HasPrefix(trimmed, "?"):
		return inputModePill("ASK", colorBlue)
	case strings.HasPrefix(trimmed, "!"):
		return inputModePill("NOTE", colorMauve)
	case strings.HasPrefix(trimmed, "/"):
		return inputModePill("CMD", colorPeach)
	default:
		return inputModePill("INPUT", colorSurface1)
	}
}

// ─── Regex Patterns ─────────────────────────────────────────────────────────
var (
	reH2          = regexp.MustCompile(`^##\s+`)
	reH1          = regexp.MustCompile(`^#\s+`)
	reCheckbox    = regexp.MustCompile(`^\s*-\s+\[ \]\s+`)
	reCheckboxDone = regexp.MustCompile(`^\s*-\s+\[x\]\s+`)
	reBullet      = regexp.MustCompile(`^\s*-\s+`)
	reHashtag     = regexp.MustCompile(`^#\w+`)
	reQuote       = regexp.MustCompile(`^>\s+`)
	reHRule       = regexp.MustCompile(`^---+$`)
)

// StyleMarkdownLine applies syntax highlighting to a single line of markdown.
func StyleMarkdownLine(line string, eisenhowerTags map[string]string) string {
	switch {
	case reHRule.MatchString(strings.TrimSpace(line)):
		return dimStyle.Render(line)

	case reH2.MatchString(line):
		styled := applyEisenhowerTags(line, eisenhowerTags)
		return headingH2Style.Render(styled)

	case reH1.MatchString(line):
		styled := applyEisenhowerTags(line, eisenhowerTags)
		return headingH1Style.Render(styled)

	case reCheckboxDone.MatchString(line):
		return checkboxDoneStyle.Render(line)

	case reCheckbox.MatchString(line):
		styled := applyEisenhowerTags(line, eisenhowerTags)
		return checkboxStyle.Render(styled)

	case reQuote.MatchString(line):
		return quoteStyle.Render(line)

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

	for tag := range tags {
		if strings.Contains(line, tag) {
			dc, ok := eisenhowerDisplayColors[tag]
			if !ok {
				dc = colorOverlay
			}
			tagStyle := lipgloss.NewStyle().
				Foreground(dc).
				Bold(true)
			coloredTag := tagStyle.Render(tag)
			line = strings.ReplaceAll(line, tag, coloredTag)
		}
	}

	return line
}

// ─── Tab Bar ────────────────────────────────────────────────────────────────

// RenderTabBar renders tabs with an underline indicator on the active tab.
func RenderTabBar(tabs []string, activeTab int, width int) string {
	var tabLine strings.Builder
	var positions []struct{ start, end int }
	pos := 2

	for i, tab := range tabs {
		padding := "  "
		label := padding + tab + padding
		if i == activeTab {
			tabLine.WriteString(tabActiveStyle.Render(label))
		} else {
			tabLine.WriteString(tabInactiveStyle.Render(label))
		}
		positions = append(positions, struct{ start, end int }{pos, pos + len(label)})
		pos += len(label)
	}

	// Build underline with accent color under active tab
	underline := make([]byte, width)
	for i := range underline {
		underline[i] = ' '
	}

	activePos := positions[activeTab]
	tabText := tabLine.String()

	underlineStr := strings.Repeat(" ", activePos.start) +
		tabUnderlineStyle.Render(strings.Repeat("─", activePos.end-activePos.start))

	return tabText + "\n" + underlineStr
}

// ─── Empty States ───────────────────────────────────────────────────────────

// RenderEmptyState creates a centered empty state message.
func RenderEmptyState(title, hint string, width, height int) string {
	dot := dimStyle.Render("○")
	titleLine := lipgloss.NewStyle().Foreground(colorSubtext).Bold(true).Render(title)
	hintLine := dimStyle.Render(hint)

	content := lipgloss.JoinVertical(lipgloss.Center,
		dot, "", titleLine, "", hintLine, "", dot,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// ─── Title in Border ────────────────────────────────────────────────────────

// BorderWithTitle renders a top border line with an embedded title.
func BorderWithTitle(title string, width int, active bool) string {
	borderColor := colorSurface1
	if active {
		borderColor = colorBlue
	}

	titleRendered := lipgloss.NewStyle().
		Foreground(colorText).Bold(true).Render(title)

	left := lipgloss.NewStyle().Foreground(borderColor).Render("╭─ ")
	right := lipgloss.NewStyle().Foreground(borderColor).
		Render(" " + strings.Repeat("─", max(0, width-len(title)-6)) + "╮")

	return fmt.Sprintf("%s%s%s", left, titleRendered, right)
}
