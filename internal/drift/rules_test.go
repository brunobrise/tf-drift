package drift

import (
	"encoding/json"
	"testing"
)

func TestParseRules(t *testing.T) {
	rulesJSON := `{
		"global_ignores": {
			"resource_types": ["aws_autoscaling_group"],
			"attributes": ["tags", "desired_capacity"]
		},
		"severity_classification": {
			"aws_iam_policy": "CRITICAL",
			"aws_security_group_rule": "CRITICAL",
			"aws_rds_cluster": "HIGH"
		},
		"layer_ignores": {
			"aws/workload_api_dev/500_rds_dev": {
				"attributes": ["database_name"]
			}
		}
	}`

	var rules RulesConfig
	err := json.Unmarshal([]byte(rulesJSON), &rules)
	if err != nil {
		t.Fatalf("Failed to parse rules JSON: %v", err)
	}

	if len(rules.GlobalIgnores.ResourceTypes) != 1 || rules.GlobalIgnores.ResourceTypes[0] != "aws_autoscaling_group" {
		t.Errorf("Expected global ignores resource types to contain aws_autoscaling_group")
	}

	if rules.SeverityClassification["aws_iam_policy"] != "CRITICAL" {
		t.Errorf("Expected aws_iam_policy severity to be CRITICAL")
	}
}

func TestFilterDrift(t *testing.T) {
	rules := RulesConfig{
		GlobalIgnores: Ignores{
			ResourceTypes: []string{"aws_autoscaling_group"},
			Attributes:    []string{"tags", "desired_capacity"},
		},
		SeverityClassification: map[string]string{
			"aws_iam_policy":          "CRITICAL",
			"aws_security_group_rule": "CRITICAL",
		},
		LayerIgnores: map[string]Ignores{
			"aws/workload_api_dev/500_rds_dev": {
				Attributes: []string{"database_name"},
			},
		},
	}

	// Case 1: Resource type ignored globally
	ignored, _ := rules.EvaluateChange("aws/workload_api_dev/008_ssm_dev", "aws_autoscaling_group", []string{"tags"})
	if !ignored {
		t.Errorf("Expected resource type aws_autoscaling_group to be ignored")
	}

	// Case 2: Attribute ignored globally
	ignored, sev := rules.EvaluateChange("aws/workload_api_dev/008_ssm_dev", "aws_instance", []string{"tags"})
	if !ignored {
		t.Errorf("Expected tags-only change on aws_instance to be ignored")
	}
	if sev != "LOW" {
		t.Errorf("Expected default low severity for ignored changes, got %s", sev)
	}

	// Case 3: Significant change, critical severity
	ignored, sev = rules.EvaluateChange("aws/workload_api_dev/008_ssm_dev", "aws_iam_policy", []string{"policy"})
	if ignored {
		t.Errorf("Expected policy change on aws_iam_policy NOT to be ignored")
	}
	if sev != "CRITICAL" {
		t.Errorf("Expected critical severity for aws_iam_policy, got %s", sev)
	}

	// Case 4: Attribute ignored for specific layer
	ignored, _ = rules.EvaluateChange("aws/workload_api_dev/500_rds_dev", "aws_rds_cluster", []string{"database_name"})
	if !ignored {
		t.Errorf("Expected database_name change on 500_rds_dev to be ignored")
	}
}
