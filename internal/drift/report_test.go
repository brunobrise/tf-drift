package drift

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReportMarkdownFormat(t *testing.T) {
	results := []ScanResult{
		{
			Path: "layer1",
			Drifts: []DriftChange{
				{
					Address:        "aws_security_group.sg",
					Type:           "aws_security_group",
					Actions:        []string{"update"},
					Severity:       "CRITICAL",
					Classification: ChangeClassificationExternalDrift,
				},
			},
		},
		{
			Path: "layer2",
		},
	}

	mdOutput := formatMarkdown(results)

	if !strings.Contains(mdOutput, "## Drift Detection Summary") {
		t.Errorf("Expected markdown title in output")
	}

	if !strings.Contains(mdOutput, "layer1") || !strings.Contains(mdOutput, "DRIFTED") {
		t.Errorf("Expected layer1 and its DRIFTED status to be present in markdown")
	}

	if !strings.Contains(mdOutput, "layer2") || !strings.Contains(mdOutput, "CLEAN") {
		t.Errorf("Expected layer2 and its CLEAN status to be present in markdown")
	}
}

func TestReportTextUsesTildeForHomePath(t *testing.T) {
	results := []ScanResult{
		{
			Path: "/Users/alice/project/layer",
		},
	}

	output := formatTextWithHome(results, "/Users/alice")
	if !strings.Contains(output, "~/project/layer") {
		t.Fatalf("expected tilde path in output, got:\n%s", output)
	}
	if strings.Contains(output, "/Users/alice") {
		t.Fatalf("expected home path to be hidden, got:\n%s", output)
	}
}

func TestReportJSONIncludesClassificationAndCounts(t *testing.T) {
	results := []ScanResult{
		{
			Path: "/Users/alice/project/layer",
			Drifts: []DriftChange{
				{
					Address:        "terraform_data.pending",
					Type:           "terraform_data",
					Actions:        []string{"create"},
					Severity:       "LOW",
					Classification: ChangeClassificationPlannedChange,
				},
			},
		},
	}

	output := formatJSON(results)
	if !strings.Contains(output, `"/Users/alice/project/layer"`) {
		t.Fatalf("expected raw path in JSON output, got:\n%s", output)
	}

	var decoded []struct {
		Status             string        `json:"status"`
		ExternalDriftCount int           `json:"external_drift_count"`
		PlannedChangeCount int           `json:"planned_change_count"`
		Drifts             []DriftChange `json:"drifts"`
	}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("expected valid JSON output: %v\n%s", err, output)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected one JSON result, got %d", len(decoded))
	}
	if decoded[0].Status != "PLANNED" {
		t.Fatalf("expected PLANNED status, got %s", decoded[0].Status)
	}
	if decoded[0].ExternalDriftCount != 0 || decoded[0].PlannedChangeCount != 1 {
		t.Fatalf("expected 0 drift and 1 planned, got drift=%d planned=%d", decoded[0].ExternalDriftCount, decoded[0].PlannedChangeCount)
	}
	if len(decoded[0].Drifts) != 1 || decoded[0].Drifts[0].Classification != ChangeClassificationPlannedChange {
		t.Fatalf("expected planned classification in JSON, got %#v", decoded[0].Drifts)
	}
}

func TestReportSARIFIncludesCIAnnotations(t *testing.T) {
	results := []ScanResult{
		{
			Path: "prod/network",
			Drifts: []DriftChange{
				{
					Address:           "aws_security_group_rule.ingress",
					Type:              "aws_security_group_rule",
					Actions:           []string{"update"},
					ChangedAttributes: []string{"cidr_blocks"},
					Severity:          "CRITICAL",
					Classification:    ChangeClassificationExternalDrift,
				},
				{
					Address:           "terraform_data.pending",
					Type:              "terraform_data",
					Actions:           []string{"create"},
					ChangedAttributes: []string{"input"},
					Severity:          "LOW",
					Classification:    ChangeClassificationPlannedChange,
				},
			},
		},
		{
			Path: "prod/database",
			Err:  errString("tofu plan failed"),
		},
	}

	output := formatSARIF(results)
	var decoded struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name  string `json:"name"`
					Rules []struct {
						ID string `json:"id"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID    string                `json:"ruleId"`
				Level     string                `json:"level"`
				Message   struct{ Text string } `json:"message"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("expected valid SARIF JSON: %v\n%s", err, output)
	}

	if decoded.Version != "2.1.0" {
		t.Fatalf("expected SARIF version 2.1.0, got %q", decoded.Version)
	}
	if len(decoded.Runs) != 1 || decoded.Runs[0].Tool.Driver.Name != "tf-drift" {
		t.Fatalf("expected tf-drift SARIF driver, got %#v", decoded.Runs)
	}
	if len(decoded.Runs[0].Results) != 3 {
		t.Fatalf("expected 3 SARIF results, got %d", len(decoded.Runs[0].Results))
	}

	got := map[string]string{}
	for _, res := range decoded.Runs[0].Results {
		got[res.RuleID] = res.Level + "|" + res.Locations[0].PhysicalLocation.ArtifactLocation.URI + "|" + res.Message.Text
	}
	for _, ruleID := range []string{"tf-drift.external-drift", "tf-drift.planned-change", "tf-drift.execution-error"} {
		if _, ok := got[ruleID]; !ok {
			t.Fatalf("expected SARIF result for %s, got %#v", ruleID, got)
		}
	}
	if !strings.HasPrefix(got["tf-drift.external-drift"], "error|prod/network|") {
		t.Fatalf("expected critical drift to map to error at prod/network, got %q", got["tf-drift.external-drift"])
	}
	if !strings.HasPrefix(got["tf-drift.planned-change"], "note|prod/network|") {
		t.Fatalf("expected low planned change to map to note at prod/network, got %q", got["tf-drift.planned-change"])
	}
	if !strings.HasPrefix(got["tf-drift.execution-error"], "error|prod/database|") {
		t.Fatalf("expected execution error annotation at prod/database, got %q", got["tf-drift.execution-error"])
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}
