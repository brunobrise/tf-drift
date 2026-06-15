package main

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

	if len(change.ChangedAttributes) != 1 || change.ChangedAttributes[0] != "instance_type" {
		t.Errorf("Expected only instance_type to be changed, got %v", change.ChangedAttributes)
	}
}

func TestModifyProviders(t *testing.T) {
	input1 := `provider "aws" {
  region = "eu-west-3"
  assume_role {
    role_arn = "arn:aws:iam::123:role/role"
  }
}`

	output1 := modifyProviderText(input1, "local-profile")
	// Clean formatting differences if any, or test exact content
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

	output2 := modifyProviderText(input2, "local-profile")
	if !strings.Contains(output2, `profile = "local-profile"`) {
		t.Errorf("Expected output2 to contain profile override, got:\n%s", output2)
	}
	if strings.Contains(output2, `profile = "dev-api"`) {
		t.Errorf("Expected old profile dev-api to be overridden")
	}
}
