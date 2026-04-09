package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/content"
)

// renderStatusOverlay renders the vault status overlay.
func (m AppModel) renderStatusOverlay() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	labelStyle := lipgloss.NewStyle().Foreground(colorText)
	valueStyle := lipgloss.NewStyle().Foreground(colorText).Bold(true)
	zeroStyle := lipgloss.NewStyle().Foreground(colorOverlay)
	footerStyle := lipgloss.NewStyle().Foreground(colorOverlay)

	compileAgo := formatDurationAgo(m.lastCompileTime)
	compileStyle := labelStyle
	if m.lastCompileTime == nil || time.Since(*m.lastCompileTime) > 7*24*time.Hour {
		compileStyle = lipgloss.NewStyle().Foreground(colorYellow)
	}

	renderMetric := func(label string, count int, unit string) string {
		if count == 0 {
			return zeroStyle.Render(fmt.Sprintf("  %s: 0 %s", label, unit))
		}
		return labelStyle.Render(fmt.Sprintf("  %s: ", label)) +
			valueStyle.Render(fmt.Sprintf("%d", count)) +
			labelStyle.Render(fmt.Sprintf(" %s", unit))
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render("  Vault Status"))
	lines = append(lines, "")
	lines = append(lines, compileStyle.Render(fmt.Sprintf("  Last compile: %s", compileAgo)))
	lines = append(lines, "")

	if m.vaultStatus != nil {
		lines = append(lines, renderMetric("Wiki inbox", m.vaultStatus.WikiInboxCount, "unprocessed"))
		lines = append(lines, renderMetric("Review queue", m.vaultStatus.ReviewQueueCount, "pending drafts"))
		lines = append(lines, renderMetric("Raw notes", m.vaultStatus.RawNotesSinceCompile, "since last compile"))
	}

	lines = append(lines, "")
	lines = append(lines, footerStyle.Render("  Press any key to return"))
	lines = append(lines, "")

	boxContent := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue).
		Padding(0, 2).
		Width(42)

	box := boxStyle.Render(boxContent)

	return lipgloss.Place(m.width, m.height-2,
		lipgloss.Center, lipgloss.Center, box)
}

// renderCompileSummary renders the compile summary as scrollable content.
func (m AppModel) renderCompileSummary() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	labelStyle := lipgloss.NewStyle().Foreground(colorText)
	zeroStyle := lipgloss.NewStyle().Foreground(colorOverlay)
	warnStyle := lipgloss.NewStyle().Foreground(colorYellow)

	var lines []string
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render("  Compile Complete"))
	lines = append(lines, "")

	if m.compileResult == nil {
		lines = append(lines, labelStyle.Render("  No results available"))
	} else {
		renderSection := func(name string, metrics content.SectionMetrics) {
			lines = append(lines, sectionStyle.Render("  "+name))
			if metrics.Items == nil || len(metrics.Items) == 0 {
				lines = append(lines, zeroStyle.Render("    No data"))
			} else {
				for k, v := range metrics.Items {
					style := labelStyle
					if v == "0" || v == "" {
						style = zeroStyle
					}
					lines = append(lines, style.Render(fmt.Sprintf("    %s: %s", k, v)))
				}
			}
			lines = append(lines, "")
		}

		renderSection("Wiki", m.compileResult.Wiki)
		renderSection("Zettelkasten", m.compileResult.Zettelkasten)

		lines = append(lines, sectionStyle.Render("  Lint"))
		if m.compileResult.Lint.Items == nil || len(m.compileResult.Lint.Items) == 0 {
			lines = append(lines, zeroStyle.Render("    No data"))
		} else {
			for k, v := range m.compileResult.Lint.Items {
				style := labelStyle
				lower := strings.ToLower(strings.TrimSpace(v))
				if lower != "none" && lower != "0" && lower != "" {
					style = warnStyle
				}
				lines = append(lines, style.Render(fmt.Sprintf("    %s: %s", k, v)))
			}
		}
		lines = append(lines, "")

		if len(m.compileResult.Suggestions) > 0 {
			lines = append(lines, sectionStyle.Render("  Suggestions"))
			for _, s := range m.compileResult.Suggestions {
				lines = append(lines, labelStyle.Render("    • "+s))
			}
			lines = append(lines, "")
		}

		if m.compileResult.Frontmatter.DurationSeconds > 0 {
			lines = append(lines, zeroStyle.Render(fmt.Sprintf("  Duration: %ds", m.compileResult.Frontmatter.DurationSeconds)))
		}
	}

	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// renderCompileSummaryContent returns the compile summary for the viewport.
func (m AppModel) renderCompileSummaryContent() string {
	return m.renderCompileSummary()
}

// formatDurationAgo returns a human-readable duration since the given time.
func formatDurationAgo(t *time.Time) string {
	if t == nil {
		return "never"
	}
	d := time.Since(*t)
	switch {
	case d < time.Hour:
		return "just now"
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 4*7*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(7*24)))
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(30*24)))
	}
}
