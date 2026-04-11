package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/caneppelevitor/obsidian-cli/internal/content"
)

// renderCompileProgress renders the streaming compile loading screen with
// current phase, elapsed time, and recent output lines.
func (m AppModel) renderCompileProgress() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	phaseStyle := lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	labelStyle := lipgloss.NewStyle().Foreground(colorText)
	dimLogStyle := lipgloss.NewStyle().Foreground(colorOverlay)
	footerStyle := lipgloss.NewStyle().Foreground(colorOverlay)

	elapsed := time.Since(m.compileStartTime)
	elapsedStr := formatElapsed(elapsed)

	var phaseLine string
	if m.compileProgress != nil && m.compileProgress.CurrentPhase != "" {
		phaseLine = phaseStyle.Render(fmt.Sprintf("  Phase %s/6: %s",
			m.compileProgress.PhaseNumber, m.compileProgress.CurrentPhase))
	} else {
		phaseLine = dimLogStyle.Render("  Starting compile...")
	}

	spinnerLine := m.spinner.View() + " " + titleStyle.Render("Compiling vault...")

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+spinnerLine)
	lines = append(lines, "")
	lines = append(lines, phaseLine)
	lines = append(lines, labelStyle.Render(fmt.Sprintf("  Elapsed: %s", elapsedStr)))

	// Token usage (live)
	if m.compileProgress != nil && (m.compileProgress.InputTokens > 0 || m.compileProgress.OutputTokens > 0) {
		in := m.compileProgress.InputTokens
		out := m.compileProgress.OutputTokens
		cacheRead := m.compileProgress.CacheReadTokens
		tokenLine := fmt.Sprintf("  Tokens: %s in · %s out",
			formatTokens(in), formatTokens(out))
		if cacheRead > 0 {
			tokenLine += fmt.Sprintf(" · %s cached", formatTokens(cacheRead))
		}
		lines = append(lines, dimLogStyle.Render(tokenLine))
	}

	lines = append(lines, "")

	// Recent activity log
	if m.compileProgress != nil && len(m.compileProgress.RecentLines) > 0 {
		lines = append(lines, dimLogStyle.Render("  Recent activity:"))
		maxLineWidth := m.width - 8
		if maxLineWidth < 20 {
			maxLineWidth = 20
		}
		for _, line := range m.compileProgress.RecentLines {
			clean := sanitizeLine(line)
			lines = append(lines, dimLogStyle.Render("    "+truncateRunes(clean, maxLineWidth)))
		}
		lines = append(lines, "")
	}

	lines = append(lines, footerStyle.Render("  Esc cancel · Tab background"))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	viewportHeight := m.height - 5
	if viewportHeight < 10 {
		viewportHeight = 10
	}

	return activeBorderStyle.
		Width(m.width - 2).
		Height(viewportHeight).
		Render(content)
}

// ansiRe matches ANSI escape sequences (color codes, cursor movement, etc.).
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// sanitizeLine strips ANSI escapes, control characters, and normalizes whitespace
// for safe display in the activity log.
func sanitizeLine(s string) string {
	s = ansiRe.ReplaceAllString(s, "")
	// Replace tabs and carriage returns with spaces, strip other control chars
	var b strings.Builder
	for _, r := range s {
		if r == '\t' || r == '\r' {
			b.WriteByte(' ')
			continue
		}
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// truncateRunes safely truncates a string to at most maxWidth runes,
// appending "..." if truncated. Handles multi-byte UTF-8 correctly.
func truncateRunes(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}

// formatTokens returns a compact number like "1.2k" or "350" or "2.4M".
func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// formatElapsed returns a short human-readable duration (e.g., "2m 15s").
func formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", mins, secs)
}

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
		} else if m.lastCompileElapsed > 0 {
			lines = append(lines, zeroStyle.Render(fmt.Sprintf("  Duration: %s", formatElapsed(m.lastCompileElapsed))))
		}

		// Token usage + cost (from the stream-json result event)
		tok := m.lastCompileTokens
		if tok.InputTokens > 0 || tok.OutputTokens > 0 {
			lines = append(lines, "")
			lines = append(lines, sectionStyle.Render("  Token Usage"))
			lines = append(lines, labelStyle.Render(fmt.Sprintf("    Input:        %s", formatTokens(tok.InputTokens))))
			lines = append(lines, labelStyle.Render(fmt.Sprintf("    Output:       %s", formatTokens(tok.OutputTokens))))
			if tok.CacheReadTokens > 0 {
				lines = append(lines, labelStyle.Render(fmt.Sprintf("    Cache read:   %s", formatTokens(tok.CacheReadTokens))))
			}
			if tok.CacheCreationTokens > 0 {
				lines = append(lines, labelStyle.Render(fmt.Sprintf("    Cache create: %s", formatTokens(tok.CacheCreationTokens))))
			}
			if tok.CostUSD > 0 {
				lines = append(lines, labelStyle.Render(fmt.Sprintf("    Cost:         $%.4f", tok.CostUSD)))
			}
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
