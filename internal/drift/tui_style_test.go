package drift

import (
	"regexp"
	"strings"
	"testing"
)

func TestResolveTUIStyle(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		env     string
		noColor bool
		want    tuiStyleName
	}{
		{name: "default", want: tuiStyleModern},
		{name: "flag wins", flag: "accessible", env: "classic", want: tuiStyleAccessible},
		{name: "env fallback", env: "classic", want: tuiStyleClassic},
		{name: "unknown flag falls back", flag: "vaporwave", want: tuiStyleModern},
		{name: "no color forces minimal", flag: "modern", env: "accessible", noColor: true, want: tuiStyleMinimal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveTUIStyle(tt.flag, tt.env, tt.noColor)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestMinimalStyleScanViewHasReadableStatusesWithoutANSI(t *testing.T) {
	m := initialModelWithStyle([]string{"layer1", "layer2"}, RulesConfig{}, 2, false, ".", tuiStyleMinimal)
	m.results["layer1"] = ScanResult{Path: "layer1"}
	m.width = 80
	m.height = 24

	view := m.View()
	for _, expected := range []string{"tf-drift", "CLEAN", "SCANNING", "[q] Quit", "[Enter] View Details"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected minimal scan view to contain %q, got:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "\x1b[") {
		t.Fatalf("expected minimal scan view to avoid ANSI escapes, got:\n%s", view)
	}
}

func TestSelectionViewUsesSharedMinimalStyle(t *testing.T) {
	m := initialSelectionModelWithStyle([]string{"/repo/examples/clean-empty"}, "/repo/examples", tuiStyleMinimal)
	m.width = 80
	m.height = 24

	view := m.View()
	for _, expected := range []string{"Select Terraform/OpenTofu configs", "[x] clean-empty", "[Space] Tick", "[Enter] Scan"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected minimal selection view to contain %q, got:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "\x1b[") {
		t.Fatalf("expected minimal selection view to avoid ANSI escapes, got:\n%s", view)
	}
}

func TestSelectionModernStyleKeepsRowsOnSeparateLines(t *testing.T) {
	m := initialSelectionModelWithStyle([]string{
		"/repo/examples/clean-empty",
		"/repo/examples/drift-new-resource",
	}, "/repo", tuiStyleModern)
	m.width = 80
	m.height = 24

	view := stripANSI(m.View())
	lines := strings.Split(view, "\n")
	first := findLineContaining(t, lines, "> [x] examples/clean-empty")
	second := findLineContaining(t, lines, "  [x] examples/drift-new-resource")
	if !strings.HasPrefix(first, "> [x] examples/clean-empty") || !strings.HasPrefix(second, "  [x] examples/drift-new-resource") {
		t.Fatalf("expected selected row to preserve newline before next row, got:\n%s", view)
	}
}

func TestSelectionModernStylePadsFocusedRowToFullWidth(t *testing.T) {
	m := initialSelectionModelWithStyle([]string{"/repo/examples/clean-empty"}, "/repo", tuiStyleModern)
	m.width = 80
	m.height = 24

	lines := strings.Split(stripANSI(m.View()), "\n")
	focused := findLineContaining(t, lines, "> [x] examples/clean-empty")
	if len(focused) != 80 {
		t.Fatalf("expected focused selection row width 80, got %d: %q", len(focused), focused)
	}
}

func TestSelectionModernFocusedRowDoesNotResetBeforeText(t *testing.T) {
	m := initialSelectionModelWithStyle([]string{"/repo/examples/clean-empty"}, "/repo", tuiStyleModern)
	m.width = 80
	m.height = 24

	focused := findLineContaining(t, strings.Split(m.View(), "\n"), "examples/clean-empty")
	prefix := focused[:strings.Index(focused, "examples/clean-empty")]
	if strings.Contains(prefix, "\x1b[0m") {
		t.Fatalf("expected focused row background not to reset before row text, got %q", focused)
	}
}

func TestScanModernStylePadsFocusedRowToFullWidth(t *testing.T) {
	m := initialModelWithStyle([]string{"layer1"}, RulesConfig{}, 2, false, ".", tuiStyleModern)
	m.results["layer1"] = ScanResult{Path: "layer1"}
	m.width = 80
	m.height = 24

	lines := strings.Split(stripANSI(m.View()), "\n")
	focused := findLineContaining(t, lines, "> layer1")
	if len(focused) != 80 {
		t.Fatalf("expected focused scan row width 80, got %d: %q", len(focused), focused)
	}
}

func TestScanModernFocusedRowDoesNotResetBeforeText(t *testing.T) {
	m := initialModelWithStyle([]string{"layer1"}, RulesConfig{}, 2, false, ".", tuiStyleModern)
	m.results["layer1"] = ScanResult{Path: "layer1"}
	m.width = 80
	m.height = 24

	focused := findLineContaining(t, strings.Split(m.View(), "\n"), "layer1")
	prefix := focused[:strings.Index(focused, "layer1")]
	if strings.Contains(prefix, "\x1b[0m") {
		t.Fatalf("expected focused row background not to reset before row text, got %q", focused)
	}
}

func findLineContaining(t *testing.T, lines []string, needle string) string {
	t.Helper()
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return line
		}
	}
	t.Fatalf("expected line containing %q in %#v", needle, lines)
	return ""
}

func stripANSI(value string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(value, "")
}
