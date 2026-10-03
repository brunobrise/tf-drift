package drift

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBaselineMatching(t *testing.T) {
	baseline := &BaselineFile{
		Version:   "1.0",
		UpdatedAt: "2026-10-03T00:00:00Z",
		Entries: []BaselineEntry{
			{
				Layer:             "terraform/aws/vpc",
				Address:           "aws_security_group.default",
				Type:              "aws_security_group",
				Classification:    "EXTERNAL_DRIFT",
				ChangedAttributes: []string{"ingress"},
			},
		},
	}

	matchingChange := DriftChange{
		Address:           "aws_security_group.default",
		Type:              "aws_security_group",
		Classification:    ChangeClassificationExternalDrift,
		ChangedAttributes: []string{"ingress"},
	}

	nonMatchingAddress := DriftChange{
		Address:           "aws_security_group.custom",
		Type:              "aws_security_group",
		Classification:    ChangeClassificationExternalDrift,
		ChangedAttributes: []string{"ingress"},
	}

	nonMatchingAttr := DriftChange{
		Address:           "aws_security_group.default",
		Type:              "aws_security_group",
		Classification:    ChangeClassificationExternalDrift,
		ChangedAttributes: []string{"egress"},
	}

	if !baseline.Matches("terraform/aws/vpc", matchingChange) {
		t.Errorf("expected baseline to match matchingChange")
	}

	if baseline.Matches("terraform/aws/vpc", nonMatchingAddress) {
		t.Errorf("expected baseline NOT to match nonMatchingAddress")
	}

	if baseline.Matches("terraform/aws/vpc", nonMatchingAttr) {
		t.Errorf("expected baseline NOT to match nonMatchingAttr")
	}

	if baseline.Matches("terraform/aws/other", matchingChange) {
		t.Errorf("expected baseline NOT to match different layer")
	}
}

func TestSaveAndLoadBaseline(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, "baseline.json")

	results := []ScanResult{
		{
			Path: "infra/prod/network",
			Drifts: []DriftChange{
				{
					Address:           "aws_vpc.main",
					Type:              "aws_vpc",
					Classification:    ChangeClassificationExternalDrift,
					ChangedAttributes: []string{"tags"},
				},
			},
		},
	}

	if err := SaveBaseline(baselinePath, results); err != nil {
		t.Fatalf("SaveBaseline failed: %v", err)
	}

	data, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("failed to read written baseline: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("baseline file is empty")
	}

	loaded, err := LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("LoadBaseline failed: %v", err)
	}

	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 baseline entry, got %d", len(loaded.Entries))
	}

	entry := loaded.Entries[0]
	if entry.Address != "aws_vpc.main" || entry.Type != "aws_vpc" {
		t.Errorf("unexpected entry data: %+v", entry)
	}
}

func TestApplyBaseline(t *testing.T) {
	baseline := &BaselineFile{
		Version: "1.0",
		Entries: []BaselineEntry{
			{
				Layer:   "infra/prod",
				Address: "aws_s3_bucket.data",
				Type:    "aws_s3_bucket",
			},
		},
	}

	results := []ScanResult{
		{
			Path: "infra/prod",
			Drifts: []DriftChange{
				{
					Address: "aws_s3_bucket.data",
					Type:    "aws_s3_bucket",
				},
				{
					Address: "aws_s3_bucket.logs",
					Type:    "aws_s3_bucket",
				},
			},
		},
	}

	applied := ApplyBaseline(results, baseline)
	if !applied[0].Drifts[0].Acknowledged {
		t.Errorf("expected aws_s3_bucket.data to be acknowledged")
	}
	if applied[0].Drifts[1].Acknowledged {
		t.Errorf("expected aws_s3_bucket.logs NOT to be acknowledged")
	}
}
