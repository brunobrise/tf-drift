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
	plannedLayersCount := 0
	errorLayersCount := 0

	var detailSB strings.Builder

	for _, res := range results {
		path := homePathForDisplay(res.Path, home)
		externalDrifts, plannedChanges := changeCounts(res)
		if res.Err != nil {
			errorLayersCount++
			_, _ = fmt.Fprintf(&sb, "| `%s` | ❌ ERROR | %v |\n", path, res.Err)
		} else if len(res.Drifts) > 0 {
			if externalDrifts > 0 {
				driftedLayersCount++
			}
			if plannedChanges > 0 {
				plannedLayersCount++
			}
			maxSev := maxSeverity(res.Drifts)
			_, _ = fmt.Fprintf(&sb, "| `%s` | %s | %s (Max: %s) |\n",
				path, markdownStatus(res), changeSummary(externalDrifts, plannedChanges), maxSev)

			// Add detailed list
			_, _ = fmt.Fprintf(&detailSB, "### Layer `%s` Change Details\n\n", path)
			for _, d := range res.Drifts {
				_, _ = fmt.Fprintf(&detailSB, "* **%s** [%s] (%s) — Actions: %v\n",
					d.Address, d.Classification, d.Severity, d.Actions)
				_, _ = fmt.Fprintf(&detailSB, "  * Changed attributes: `%s`\n", strings.Join(d.ChangedAttributes, "`, `"))
			}
			detailSB.WriteString("\n")
		} else {
			_, _ = fmt.Fprintf(&sb, "| `%s` | 🟢 CLEAN | — |\n", path)
		}
	}

	sb.WriteString("\n")
	_, _ = fmt.Fprintf(&sb, "**Summary:** %d layers scanned, %d with external drift, %d with planned changes, %d errors.\n\n",
		len(results), driftedLayersCount, plannedLayersCount, errorLayersCount)

	if detailSB.Len() > 0 {
		sb.WriteString("## Change Details\n\n")
		sb.WriteString(detailSB.String())
	}

	return sb.String()
}

// formatJSON returns structured JSON of all results.
func formatJSON(results []ScanResult) string {
	type JSONResult struct {
		Path               string        `json:"path"`
		Status             string        `json:"status"`
		ExternalDriftCount int           `json:"external_drift_count,omitempty"`
		PlannedChangeCount int           `json:"planned_change_count,omitempty"`
		Drifts             []DriftChange `json:"drifts,omitempty"`
		Error              string        `json:"error,omitempty"`
	}

	var jsonResults []JSONResult
	for _, res := range results {
		status := resultStatus(res)
		var errMsg string
		if res.Err != nil {
			errMsg = res.Err.Error()
		}
		externalDrifts, plannedChanges := changeCounts(res)
		jsonResults = append(jsonResults, JSONResult{
			Path:               res.Path,
			Status:             status,
			ExternalDriftCount: externalDrifts,
			PlannedChangeCount: plannedChanges,
			Drifts:             res.Drifts,
			Error:              errMsg,
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
	planned := 0
	errors := 0

	for _, res := range results {
		path := homePathForDisplay(res.Path, home)
		externalDrifts, plannedChanges := changeCounts(res)
		if res.Err != nil {
			errors++
			_, _ = fmt.Fprintf(&sb, "• `%s`: :x: *ERROR* - %v\n", path, res.Err)
		} else if len(res.Drifts) > 0 {
			if externalDrifts > 0 {
				drifted++
			}
			if plannedChanges > 0 {
				planned++
			}
			_, _ = fmt.Fprintf(&sb, "• `%s`: %s *%s* (%s)\n",
				path, slackStatusIcon(res), resultStatus(res), changeSummary(externalDrifts, plannedChanges))
		}
	}

	_, _ = fmt.Fprintf(&sb, "\n*Scan summary:* Scanned: %d | External drift: %d | Planned: %d | Errors: %d",
		len(results), drifted, planned, errors)

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
	planned := 0
	errors := 0

	for _, res := range results {
		path := homePathForDisplay(res.Path, home)
		externalDrifts, plannedChanges := changeCounts(res)
		if res.Err != nil {
			errors++
			_, _ = fmt.Fprintf(&sb, "[ERROR]   %s: %v\n", path, res.Err)
		} else if len(res.Drifts) > 0 {
			if externalDrifts > 0 {
				drifted++
			}
			if plannedChanges > 0 {
				planned++
			}
			_, _ = fmt.Fprintf(&sb, "[%s] %s: %s detected\n", resultStatus(res), path, changeSummary(externalDrifts, plannedChanges))
			for _, d := range res.Drifts {
				_, _ = fmt.Fprintf(&sb, "  - [%s] %s (%s)\n", d.Classification, d.Address, d.Severity)
			}
		} else {
			_, _ = fmt.Fprintf(&sb, "[CLEAN]   %s\n", path)
		}
	}

	sb.WriteString(strings.Repeat("-", 60) + "\n")
	_, _ = fmt.Fprintf(&sb, "Total layers scanned: %d | External drift: %d | Planned changes: %d | Errors: %d\n",
		len(results), drifted, planned, errors)

	return sb.String()
}

func resultStatus(res ScanResult) string {
	if res.Err != nil {
		return "ERROR"
	}
	externalDrifts, plannedChanges := changeCounts(res)
	switch {
	case externalDrifts > 0 && plannedChanges > 0:
		return "DRIFTED_AND_PLANNED"
	case externalDrifts > 0:
		return "DRIFTED"
	case plannedChanges > 0:
		return "PLANNED"
	default:
		return "CLEAN"
	}
}

func changeCounts(res ScanResult) (int, int) {
	externalDrifts := 0
	plannedChanges := 0
	for _, change := range res.Drifts {
		switch change.Classification {
		case ChangeClassificationPlannedChange:
			plannedChanges++
		default:
			externalDrifts++
		}
	}
	return externalDrifts, plannedChanges
}

func changeSummary(externalDrifts, plannedChanges int) string {
	parts := make([]string, 0, 2)
	if externalDrifts > 0 {
		parts = append(parts, changeLabel(externalDrifts, "external drift", "external drifts"))
	}
	if plannedChanges > 0 {
		parts = append(parts, changeLabel(plannedChanges, "planned change", "planned changes"))
	}
	if len(parts) == 0 {
		return "0 changes"
	}
	return strings.Join(parts, ", ")
}

func changeLabel(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func maxSeverity(changes []DriftChange) string {
	maxSev := "LOW"
	for _, d := range changes {
		if d.Severity == "CRITICAL" {
			maxSev = "CRITICAL"
		} else if d.Severity == "HIGH" && maxSev != "CRITICAL" {
			maxSev = "HIGH"
		} else if d.Severity == "MEDIUM" && maxSev != "CRITICAL" && maxSev != "HIGH" {
			maxSev = "MEDIUM"
		}
	}
	return maxSev
}

func markdownStatus(res ScanResult) string {
	switch resultStatus(res) {
	case "DRIFTED_AND_PLANNED":
		return "🔴 DRIFTED + 🟡 PLANNED"
	case "DRIFTED":
		return "🔴 DRIFTED"
	case "PLANNED":
		return "🟡 PLANNED"
	default:
		return "🟢 CLEAN"
	}
}

func slackStatusIcon(res ScanResult) string {
	switch resultStatus(res) {
	case "DRIFTED", "DRIFTED_AND_PLANNED":
		return ":red_circle:"
	case "PLANNED":
		return ":large_yellow_circle:"
	default:
		return ":white_check_mark:"
	}
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
