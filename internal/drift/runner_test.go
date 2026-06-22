package drift

import (
	"reflect"
	"strings"
	"testing"
)

func TestGetChangedAttributes(t *testing.T) {
	before := map[string]interface{}{
		"ami":           "ami-123456",
		"instance_type": "t3.micro",
		"tags": map[string]interface{}{
			"Env": "dev",
		},
	}

	after := map[string]interface{}{
		"ami":           "ami-123456",
		"instance_type": "t3.small", // changed
		"tags": map[string]interface{}{
			"Env": "prod", // changed
		},
		"new_field": "hello", // added
	}

	expected := []string{"instance_type", "new_field", "tags"}

	changed := getChangedAttributes(before, after)
	if !reflect.DeepEqual(changed, expected) {
		t.Errorf("Expected changed attributes %v, got %v", expected, changed)
	}
}

func TestParsePlanJSON(t *testing.T) {
	planJSON := `{
		"resource_changes": [
			{
				"address": "aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"change": {
					"actions": ["update"],
					"before": {
						"instance_type": "t2.micro",
						"tags": {"Env": "dev"}
					},
					"after": {
						"instance_type": "t3.micro",
						"tags": {"Env": "dev"}
					}
				}
			},
			{
				"address": "aws_security_group.allow_tls",
				"type": "aws_security_group",
				"name": "allow_tls",
				"change": {
					"actions": ["no-op"]
				}
			}
		]
	}`

	changes, err := parsePlanJSON([]byte(planJSON))
	if err != nil {
		t.Fatalf("Failed to parse plan JSON: %v", err)
	}

	// We only expect drift/changes for actions that are not no-op (i.e. update, create, delete)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 resource change with action, got %d", len(changes))
	}

	change := changes[0]
	if change.Address != "aws_instance.web" {
		t.Errorf("Expected address aws_instance.web, got %s", change.Address)
	}

	if change.Classification != ChangeClassificationPlannedChange {
		t.Errorf("Expected planned change classification, got %s", change.Classification)
	}

	if len(change.ChangedAttributes) != 1 || change.ChangedAttributes[0] != "instance_type" {
		t.Errorf("Expected only instance_type to be changed, got %v", change.ChangedAttributes)
	}
}

func TestParsePlanJSONResourceDrift(t *testing.T) {
	planJSON := `{
		"resource_drift": [
			{
				"address": "aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"action_reason": "delete_because_no_resource_config",
				"change": {
					"actions": ["delete"],
					"before": {"instance_type": "t3.micro"},
					"after": null
				}
			},
			{
				"address": "data.aws_ami.latest",
				"type": "aws_ami",
				"name": "latest",
				"change": {
					"actions": ["read"]
				}
			}
		]
	}`

	changes, err := parsePlanJSON([]byte(planJSON))
	if err != nil {
		t.Fatalf("Failed to parse plan JSON: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("Expected 1 drift change, got %d", len(changes))
	}

	change := changes[0]
	if change.Classification != ChangeClassificationExternalDrift {
		t.Errorf("Expected external drift classification, got %s", change.Classification)
	}
	if change.ActionReason != "delete_because_no_resource_config" {
		t.Errorf("Expected action reason to be preserved, got %q", change.ActionReason)
	}
	if len(change.ChangedAttributes) != 1 || change.ChangedAttributes[0] != "instance_type" {
		t.Errorf("Expected instance_type to be changed, got %v", change.ChangedAttributes)
	}
}

func TestParsePlanJSONClassifiesBothAndFiltersModes(t *testing.T) {
	planJSON := `{
		"resource_drift": [
			{
				"address": "aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"change": {
					"actions": ["update"],
					"before": {"instance_type": "t2.micro"},
					"after": {"instance_type": "t3.micro"}
				}
			}
		],
		"resource_changes": [
			{
				"address": "aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"change": {
					"actions": ["update"],
					"before": {"instance_type": "t3.micro"},
					"after": {"instance_type": "t2.micro"}
				}
			},
			{
				"address": "terraform_data.pending",
				"type": "terraform_data",
				"name": "pending",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {"input": "expected"}
				}
			}
		]
	}`

	tests := []struct {
		name string
		mode ScanMode
		want []ChangeClassification
	}{
		{
			name: "both",
			mode: ScanModeBoth,
			want: []ChangeClassification{ChangeClassificationExternalDrift, ChangeClassificationPlannedChange},
		},
		{
			name: "drift",
			mode: ScanModeDrift,
			want: []ChangeClassification{ChangeClassificationExternalDrift},
		},
		{
			name: "plan",
			mode: ScanModePlan,
			want: []ChangeClassification{ChangeClassificationPlannedChange},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := parsePlanJSONForMode([]byte(planJSON), tt.mode)
			if err != nil {
				t.Fatalf("Failed to parse plan JSON: %v", err)
			}
			if len(changes) != len(tt.want) {
				t.Fatalf("Expected %d changes, got %d: %#v", len(tt.want), len(changes), changes)
			}
			for i, want := range tt.want {
				if changes[i].Classification != want {
					t.Fatalf("Change %d classification: want %s, got %s", i, want, changes[i].Classification)
				}
			}
		})
	}
}

func TestParseScanMode(t *testing.T) {
	tests := []struct {
		input string
		want  ScanMode
	}{
		{input: "", want: ScanModeBoth},
		{input: "both", want: ScanModeBoth},
		{input: "DRIFT", want: ScanModeDrift},
		{input: " plan ", want: ScanModePlan},
	}

	for _, tt := range tests {
		got, err := ParseScanMode(tt.input)
		if err != nil {
			t.Fatalf("ParseScanMode(%q) returned error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("ParseScanMode(%q): want %s, got %s", tt.input, tt.want, got)
		}
	}

	if _, err := ParseScanMode("bad"); err == nil {
		t.Fatalf("Expected invalid scan mode error")
	}
}

func TestModifyProviders(t *testing.T) {
	input1 := `provider "aws" {
  region = "eu-west-3"
  assume_role {
    role_arn = "arn:aws:iam::123:role/role"
  }
}`

	output1, hasProfile1 := modifyProviderText(input1, "local-profile")
	if !hasProfile1 {
		t.Errorf("Expected hasProfile1 to be true when profile is injected")
	}
	if !strings.Contains(output1, `profile = "local-profile"`) {
		t.Errorf("Expected output to contain profile definition")
	}
	if !strings.Contains(output1, "# assume_role {") {
		t.Errorf("Expected assume_role block to be commented out")
	}

	input2 := `provider "aws" {
  region = "eu-west-3"
  # profile = "dev-api"
  assume_role {
    role_arn = "arn:aws:iam::123:role/role"
  }
}`

	output2, hasProfile2 := modifyProviderText(input2, "local-profile")
	if !hasProfile2 {
		t.Errorf("Expected hasProfile2 to be true")
	}
	if !strings.Contains(output2, `profile = "local-profile"`) {
		t.Errorf("Expected output2 to contain profile override, got:\n%s", output2)
	}
	if strings.Contains(output2, `profile = "dev-api"`) {
		t.Errorf("Expected old profile dev-api to be overridden")
	}

	// Test Case 3: uncomment only (no override)
	input3 := `provider "aws" {
  region = "eu-west-3"
  # profile = "dev-api"
  assume_role {
    role_arn = "arn"
  }
}`
	output3, hasProfile3 := modifyProviderText(input3, "")
	if !hasProfile3 {
		t.Errorf("Expected hasProfile3 to be true (uncommented profile)")
	}
	if !strings.Contains(output3, `profile = "dev-api"`) {
		t.Errorf("Expected output3 to contain uncommented profile, got:\n%s", output3)
	}
	if strings.Contains(output3, `# profile = "dev-api"`) {
		t.Errorf("Expected profile not to remain commented")
	}

	// Test Case 4: warning case (no profile mentioned at all, no override)
	input4 := `provider "aws" {
  region = "eu-west-3"
  assume_role {
    role_arn = "arn"
  }
}`
	output4, hasProfile4 := modifyProviderText(input4, "")
	if hasProfile4 {
		t.Errorf("Expected hasProfile4 to be false when no profile is present and no override provided")
	}
	if !strings.Contains(output4, "# assume_role {") {
		t.Errorf("Expected assume_role block to be commented out even if no profile is found")
	}
}
