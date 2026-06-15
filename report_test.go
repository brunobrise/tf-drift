package main

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
