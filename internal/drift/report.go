package drift

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// formatMarkdown returns a GFM markdown table of the scan results.
func formatMarkdown(results []ScanResult) string {
	return formatMarkdownWithHome(results, userHomeForDisplay())
}

func formatMarkdownWithHome(results []ScanResult, home string) string {
	var sb strings.Builder
	sb.WriteString("## Drift Detection Summary\n\n")
	sb.WriteString("| Layer Path | Status | Severity / Details |\n")
	sb.WriteString("| :--- | :--- | :--- |\n")

	driftedLayersCount := 0
	errorLayersCount := 0

	var detailSB strings.Builder

	for _, res := range results {
		path := homePathForDisplay(res.Path, home)
		if res.Err != nil {
			errorLayersCount++
			_, _ = fmt.Fprintf(&sb, "| `%s` | ❌ ERROR | %v |\n", path, res.Err)
		} else if len(res.Drifts) > 0 {
			driftedLayersCount++
			maxSev := "LOW"
			for _, d := range res.Drifts {
				if d.Severity == "CRITICAL" {
					maxSev = "CRITICAL"
				} else if d.Severity == "HIGH" && maxSev != "CRITICAL" {
					maxSev = "HIGH"
				} else if d.Severity == "MEDIUM" && maxSev != "CRITICAL" && maxSev != "HIGH" {
					maxSev = "MEDIUM"
				}
			}
			_, _ = fmt.Fprintf(&sb, "| `%s` | 🔴 DRIFTED | %d changes (Max: %s) |\n", path, len(res.Drifts), maxSev)

			// Add detailed list
			_, _ = fmt.Fprintf(&detailSB, "### Layer `%s` Drift Details\n\n", path)
			for _, d := range res.Drifts {
				_, _ = fmt.Fprintf(&detailSB, "* **%s** (%s) — Actions: %v\n", d.Address, d.Severity, d.Actions)
				_, _ = fmt.Fprintf(&detailSB, "  * Changed attributes: `%s`\n", strings.Join(d.ChangedAttributes, "`, `"))
			}
			detailSB.WriteString("\n")
		} else {
			_, _ = fmt.Fprintf(&sb, "| `%s` | 🟢 CLEAN | — |\n", path)
		}
	}

	sb.WriteString("\n")
	_, _ = fmt.Fprintf(&sb, "**Summary:** %d layers scanned, %d drifted, %d errors.\n\n",
		len(results), driftedLayersCount, errorLayersCount)

	if detailSB.Len() > 0 {
		sb.WriteString("## Drift Details\n\n")
		sb.WriteString(detailSB.String())
	}

	return sb.String()
}

// formatJSON returns structured JSON of all results.
func formatJSON(results []ScanResult) string {
	type JSONResult struct {
		Path   string        `json:"path"`
		Status string        `json:"status"`
		Drifts []DriftChange `json:"drifts,omitempty"`
		Error  string        `json:"error,omitempty"`
	}

	var jsonResults []JSONResult
	for _, res := range results {
		status := "CLEAN"
		var errMsg string
		if res.Err != nil {
			status = "ERROR"
			errMsg = res.Err.Error()
		} else if len(res.Drifts) > 0 {
			status = "DRIFTED"
		}
		jsonResults = append(jsonResults, JSONResult{
			Path:   res.Path,
			Status: status,
			Drifts: res.Drifts,
			Error:  errMsg,
		})
	}

	data, err := json.MarshalIndent(jsonResults, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to serialize: %v"}`, err)
	}
	return string(data)
}

// formatSlack returns a simple Slack block-kit compatible text representation.
func formatSlack(results []ScanResult) string {
	return formatSlackWithHome(results, userHomeForDisplay())
}

func formatSlackWithHome(results []ScanResult, home string) string {
	var sb strings.Builder
	sb.WriteString("*Terraform/OpenTofu Drift Detection Results:*\n")

	drifted := 0
	errors := 0

	for _, res := range results {
		path := homePathForDisplay(res.Path, home)
		if res.Err != nil {
			errors++
			_, _ = fmt.Fprintf(&sb, "• `%s`: :x: *ERROR* - %v\n", path, res.Err)
		} else if len(res.Drifts) > 0 {
			drifted++
			_, _ = fmt.Fprintf(&sb, "• `%s`: :red_circle: *DRIFTED* (%d changes)\n", path, len(res.Drifts))
		}
	}

	_, _ = fmt.Fprintf(&sb, "\n*Scan summary:* Scanned: %d | Drifted: %d | Errors: %d",
		len(results), drifted, errors)

	return sb.String()
}

// formatText returns normal text.
func formatText(results []ScanResult) string {
	return formatTextWithHome(results, userHomeForDisplay())
}

func formatTextWithHome(results []ScanResult, home string) string {
	var sb strings.Builder
	sb.WriteString("tf-drift Scan Results:\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	drifted := 0
	errors := 0

	for _, res := range results {
		path := homePathForDisplay(res.Path, home)
		if res.Err != nil {
			errors++
			_, _ = fmt.Fprintf(&sb, "[ERROR]   %s: %v\n", path, res.Err)
		} else if len(res.Drifts) > 0 {
			drifted++
			_, _ = fmt.Fprintf(&sb, "[DRIFTED] %s: %d changes detected\n", path, len(res.Drifts))
			for _, d := range res.Drifts {
				_, _ = fmt.Fprintf(&sb, "  - %s (%s)\n", d.Address, d.Severity)
			}
		} else {
			_, _ = fmt.Fprintf(&sb, "[CLEAN]   %s\n", path)
		}
	}

	sb.WriteString(strings.Repeat("-", 60) + "\n")
	_, _ = fmt.Fprintf(&sb, "Total layers scanned: %d | Drifted: %d | Errors: %d\n",
		len(results), drifted, errors)

	return sb.String()
}

func userHomeForDisplay() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// PrintNonInteractiveReport prints the results to stdout in the selected format.
func PrintNonInteractiveReport(results []ScanResult, format string) {
	switch strings.ToLower(format) {
	case "json":
		fmt.Println(formatJSON(results))
	case "markdown":
		fmt.Println(formatMarkdown(results))
	case "slack":
		fmt.Println(formatSlack(results))
	default:
		fmt.Println(formatText(results))
	}
}
