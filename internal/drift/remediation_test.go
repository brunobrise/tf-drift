package drift

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRemediationRecipes(t *testing.T) {
	results := []ScanResult{
		{
			Path: "terraform/aws/network",
			Drifts: []DriftChange{
				{
					Address:        "aws_security_group.web",
					Type:           "aws_security_group",
					Classification: ChangeClassificationExternalDrift,
				},
			},
		},
	}

	recipes := GenerateRemediationRecipes(results, "tofu")
	if len(recipes) != 1 {
		t.Fatalf("expected 1 remediation recipe, got %d", len(recipes))
	}

	r := recipes[0]
	if r.Layer != "terraform/aws/network" {
		t.Errorf("unexpected layer: %s", r.Layer)
	}
	if r.Address != "aws_security_group.web" {
		t.Errorf("unexpected address: %s", r.Address)
	}
	if r.RevertCommand != "tofu apply -target='aws_security_group.web'" {
		t.Errorf("unexpected revert command: %s", r.RevertCommand)
	}
	if r.AcceptCommand != "tofu apply -refresh-only -target='aws_security_group.web'" {
		t.Errorf("unexpected accept command: %s", r.AcceptCommand)
	}
	if !strings.Contains(r.AuditCommand, "cloudtrail") {
		t.Errorf("expected audit command to include cloudtrail for aws resource, got: %s", r.AuditCommand)
	}
}

func TestExportRemediationScript(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "remediate.sh")

	results := []ScanResult{
		{
			Path: "infra/prod/network",
			Drifts: []DriftChange{
				{
					Address:        "aws_vpc.main",
					Type:           "aws_vpc",
					Classification: ChangeClassificationExternalDrift,
				},
			},
		},
	}

	if err := ExportRemediationScript(scriptPath, results, "terraform"); err != nil {
		t.Fatalf("ExportRemediationScript failed: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read exported script: %v", err)
	}

	str := string(content)
	if !strings.HasPrefix(str, "#!/usr/bin/env bash") {
		t.Errorf("expected bash shebang at start of script")
	}
	if !strings.Contains(str, "terraform apply -target='aws_vpc.main'") {
		t.Errorf("expected script to contain target apply command")
	}
	if !strings.Contains(str, "terraform apply -refresh-only -target='aws_vpc.main'") {
		t.Errorf("expected script to contain refresh-only apply command")
	}
}
