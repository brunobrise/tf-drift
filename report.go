package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatMarkdown returns a GFM markdown table of the scan results.
func formatMarkdown(results []ScanResult) string {
	var sb strings.Builder
	sb.WriteString("## Drift Detection Summary\n\n")
	sb.WriteString("| Layer Path | Status | Severity / Details |\n")
	sb.WriteString("| :--- | :--- | :--- |\n")

	driftedLayersCount := 0
	errorLayersCount := 0

	var detailSB strings.Builder

	for _, res := range results {
		if res.Err != nil {
			errorLayersCount++
			sb.WriteString(fmt.Sprintf("| `%s` | ❌ ERROR | %v |\n", res.Path, res.Err))
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
			sb.WriteString(fmt.Sprintf("| `%s` | 🔴 DRIFTED | %d changes (Max: %s) |\n", res.Path, len(res.Drifts), maxSev))

			// Add detailed list
			detailSB.WriteString(fmt.Sprintf("### Layer `%s` Drift Details\n\n", res.Path))
			for _, d := range res.Drifts {
				detailSB.WriteString(fmt.Sprintf("* **%s** (%s) — Actions: %v\n", d.Address, d.Severity, d.Actions))
				detailSB.WriteString(fmt.Sprintf("  * Changed attributes: `%s`\n", strings.Join(d.ChangedAttributes, "`, `")))
			}
			detailSB.WriteString("\n")
		} else {
			sb.WriteString(fmt.Sprintf("| `%s` | 🟢 CLEAN | — |\n", res.Path))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("**Summary:** %d layers scanned, %d drifted, %d errors.\n\n",
		len(results), driftedLayersCount, errorLayersCount))

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
	var sb strings.Builder
	sb.WriteString("*Terraform Drift Detection Results:*\n")

	drifted := 0
	errors := 0

	for _, res := range results {
		if res.Err != nil {
			errors++
			sb.WriteString(fmt.Sprintf("• `%s`: :x: *ERROR* - %v\n", res.Path, res.Err))
		} else if len(res.Drifts) > 0 {
			drifted++
			sb.WriteString(fmt.Sprintf("• `%s`: :red_circle: *DRIFTED* (%d changes)\n", res.Path, len(res.Drifts)))
		}
	}

	sb.WriteString(fmt.Sprintf("\n*Scan summary:* Scanned: %d | Drifted: %d | Errors: %d",
		len(results), drifted, errors))

	return sb.String()
}

// formatText returns normal text.
func formatText(results []ScanResult) string {
	var sb strings.Builder
	sb.WriteString("tf-drift Scan Results:\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	drifted := 0
	errors := 0

	for _, res := range results {
		if res.Err != nil {
			errors++
			sb.WriteString(fmt.Sprintf("[ERROR]   %s: %v\n", res.Path, res.Err))
		} else if len(res.Drifts) > 0 {
			drifted++
			sb.WriteString(fmt.Sprintf("[DRIFTED] %s: %d changes detected\n", res.Path, len(res.Drifts)))
			for _, d := range res.Drifts {
				sb.WriteString(fmt.Sprintf("  - %s (%s)\n", d.Address, d.Severity))
			}
		} else {
			sb.WriteString(fmt.Sprintf("[CLEAN]   %s\n", res.Path))
		}
	}

	sb.WriteString(strings.Repeat("-", 60) + "\n")
	sb.WriteString(fmt.Sprintf("Total layers scanned: %d | Drifted: %d | Errors: %d\n",
		len(results), drifted, errors))

	return sb.String()
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
