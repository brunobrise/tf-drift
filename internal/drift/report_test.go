package drift

import (
	"strings"
	"testing"
)

func TestReportMarkdownFormat(t *testing.T) {
	results := []ScanResult{
		{
			Path: "layer1",
			Drifts: []DriftChange{
				{Address: "aws_security_group.sg", Type: "aws_security_group", Actions: []string{"update"}, Severity: "CRITICAL"},
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

func TestReportJSONKeepsRawPath(t *testing.T) {
	results := []ScanResult{
		{
			Path: "/Users/alice/project/layer",
		},
	}

	output := formatJSON(results)
	if !strings.Contains(output, `"/Users/alice/project/layer"`) {
		t.Fatalf("expected raw path in JSON output, got:\n%s", output)
	}
}
