package main

import (
	"reflect"
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
