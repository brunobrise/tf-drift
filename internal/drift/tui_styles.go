package drift

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type tuiStyleName string

const (
	tuiStyleModern     tuiStyleName = "modern"
	tuiStyleClassic    tuiStyleName = "classic"
	tuiStyleMinimal    tuiStyleName = "minimal"
	tuiStyleAccessible tuiStyleName = "accessible"
)

type tuiStyles struct {
	name      tuiStyleName
	title     lipgloss.Style
	subtle    lipgloss.Style
	accent    lipgloss.Style
	warning   lipgloss.Style
	success   lipgloss.Style
	danger    lipgloss.Style
	pending   lipgloss.Style
	selected  lipgloss.Style
	separator string
}

func ResolveTUIStyle(flagValue string) tuiStyleName {
	return resolveTUIStyle(flagValue, os.Getenv("TF_DRIFT_TUI_STYLE"), os.Getenv("NO_COLOR") != "")
}

func resolveTUIStyle(flagValue, envValue string, noColor bool) tuiStyleName {
	if noColor {
		return tuiStyleMinimal
	}
	if parsed, ok := parseTUIStyle(flagValue); ok {
		return parsed
	}
	if parsed, ok := parseTUIStyle(envValue); ok {
		return parsed
	}
	return tuiStyleModern
}

func parseTUIStyle(value string) (tuiStyleName, bool) {
	switch tuiStyleName(strings.ToLower(strings.TrimSpace(value))) {
	case tuiStyleModern:
		return tuiStyleModern, true
	case tuiStyleClassic:
		return tuiStyleClassic, true
	case tuiStyleMinimal:
		return tuiStyleMinimal, true
	case tuiStyleAccessible:
		return tuiStyleAccessible, true
	default:
		return "", false
	}
}

func newTUIStyles(name tuiStyleName) tuiStyles {
	if _, ok := parseTUIStyle(string(name)); !ok {
		name = tuiStyleModern
	}

	plain := lipgloss.NewStyle()
	styles := tuiStyles{
		name:      name,
		title:     plain,
		subtle:    plain,
		accent:    plain,
		warning:   plain,
		success:   plain,
		danger:    plain,
		pending:   plain,
		selected:  plain,
		separator: "-",
	}

	switch name {
	case tuiStyleMinimal:
		return styles
	case tuiStyleClassic:
		styles.title = plain.Bold(true).Foreground(lipgloss.Color("14"))
		styles.accent = plain.Bold(true).Foreground(lipgloss.Color("15"))
		styles.warning = plain.Foreground(lipgloss.Color("11"))
		styles.success = plain.Foreground(lipgloss.Color("10"))
		styles.danger = plain.Bold(true).Foreground(lipgloss.Color("9"))
		styles.pending = plain.Foreground(lipgloss.Color("8"))
		styles.selected = plain.Background(lipgloss.Color("238"))
		styles.separator = "─"
	case tuiStyleAccessible:
		styles.title = plain.Bold(true).Foreground(lipgloss.Color("15"))
		styles.accent = plain.Bold(true).Underline(true)
		styles.warning = plain.Bold(true)
		styles.success = plain.Bold(true)
		styles.danger = plain.Bold(true).Underline(true)
		styles.pending = plain.Faint(true)
		styles.selected = plain.Reverse(true)
		styles.separator = "═"
	default:
		styles.title = plain.Bold(true).Foreground(lipgloss.Color("39"))
		styles.subtle = plain.Foreground(lipgloss.Color("245"))
		styles.accent = plain.Bold(true).Foreground(lipgloss.Color("15"))
		styles.warning = plain.Foreground(lipgloss.Color("214"))
		styles.success = plain.Foreground(lipgloss.Color("42"))
		styles.danger = plain.Bold(true).Foreground(lipgloss.Color("203"))
		styles.pending = plain.Foreground(lipgloss.Color("245"))
		styles.selected = plain.Background(lipgloss.Color("238"))
		styles.separator = "─"
	}

	return styles
}

func (s tuiStyles) scanning(value string) string {
	return s.accent.Render(value)
}

func (s tuiStyles) clean(value string) string {
	return s.success.Render(value)
}

func (s tuiStyles) drifted(value string) string {
	return s.warning.Render(value)
}

func (s tuiStyles) err(value string) string {
	return s.danger.Render(value)
}

func (s tuiStyles) muted(value string) string {
	return s.pending.Render(value)
}

func (s tuiStyles) focus(row string, width int) string {
	if width > 0 {
		return s.selected.Width(width).Render(row)
	}
	return s.selected.Render(row)
}

func (s tuiStyles) line(width int) string {
	if width < 1 {
		width = 60
	}
	return strings.Repeat(s.separator, width)
}
