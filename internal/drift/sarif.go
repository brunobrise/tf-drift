package drift

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	ShortDescription sarifTextMessage `json:"shortDescription"`
}

type sarifResult struct {
	RuleID     string                 `json:"ruleId"`
	Level      string                 `json:"level"`
	Message    sarifTextMessage       `json:"message"`
	Locations  []sarifLocation        `json:"locations"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type sarifTextMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

func formatSARIF(results []ScanResult) string {
	log := sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "tf-drift",
						InformationURI: "https://github.com/brunobrise/tf-drift",
						Rules: []sarifRule{
							sarifRuleDefinition("tf-drift.external-drift", "External drift", "Terraform/OpenTofu plan JSON reported external infrastructure drift."),
							sarifRuleDefinition("tf-drift.planned-change", "Planned change", "Terraform/OpenTofu plan JSON reported a pending configuration change."),
							sarifRuleDefinition("tf-drift.execution-error", "Execution error", "Terraform/OpenTofu execution failed while scanning a layer."),
						},
					},
				},
			},
		},
	}

	for _, res := range results {
		if res.Err != nil {
			log.Runs[0].Results = append(log.Runs[0].Results, sarifResult{
				RuleID:  "tf-drift.execution-error",
				Level:   "error",
				Message: sarifTextMessage{Text: fmt.Sprintf("tf-drift failed for layer %s: %v", res.Path, res.Err)},
				Locations: []sarifLocation{
					sarifLayerLocation(res.Path),
				},
			})
			continue
		}

		for _, change := range res.Drifts {
			log.Runs[0].Results = append(log.Runs[0].Results, sarifResult{
				RuleID:  sarifRuleID(change.Classification),
				Level:   sarifLevel(change.Severity),
				Message: sarifTextMessage{Text: sarifChangeMessage(change)},
				Locations: []sarifLocation{
					sarifLayerLocation(res.Path),
				},
				Properties: map[string]interface{}{
					"address":            change.Address,
					"type":               change.Type,
					"actions":            change.Actions,
					"changed_attributes": change.ChangedAttributes,
					"severity":           change.Severity,
					"classification":     change.Classification,
					"action_reason":      change.ActionReason,
				},
			})
		}
	}

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to serialize SARIF: %v"}`, err)
	}
	return string(data)
}

func sarifRuleDefinition(id string, name string, description string) sarifRule {
	return sarifRule{
		ID:               id,
		Name:             name,
		ShortDescription: sarifTextMessage{Text: description},
	}
}

func sarifRuleID(classification ChangeClassification) string {
	if classification == ChangeClassificationPlannedChange {
		return "tf-drift.planned-change"
	}
	return "tf-drift.external-drift"
}

func sarifLevel(severity string) string {
	switch strings.ToUpper(severity) {
	case "CRITICAL", "HIGH":
		return "error"
	case "MEDIUM":
		return "warning"
	default:
		return "note"
	}
}

func sarifChangeMessage(change DriftChange) string {
	classification := strings.ToLower(strings.ReplaceAll(string(change.Classification), "_", " "))
	return fmt.Sprintf("%s detected for %s with %s severity", classification, change.Address, defaultSeverity(change.Severity))
}

func defaultSeverity(severity string) string {
	if strings.TrimSpace(severity) == "" {
		return "UNKNOWN"
	}
	return severity
}

func sarifLayerLocation(path string) sarifLocation {
	return sarifLocation{
		PhysicalLocation: sarifPhysicalLocation{
			ArtifactLocation: sarifArtifactLocation{
				URI: filepath.ToSlash(path),
			},
		},
	}
}
