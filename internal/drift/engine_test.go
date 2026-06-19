package drift

import (
	"errors"
	"testing"
)

func TestResolveEngineBinaryAutoPrefersOpenTofu(t *testing.T) {
	calls := []string{}
	lookup := func(name string) (string, error) {
		calls = append(calls, name)
		switch name {
		case "tofu":
			return "/bin/tofu", nil
		case "terraform":
			return "/bin/terraform", nil
		default:
			return "", errors.New("not found")
		}
	}

	resolved, err := ResolveEngineBinary("auto", lookup)
	if err != nil {
		t.Fatalf("ResolveEngineBinary auto failed: %v", err)
	}
	if resolved.Name != "opentofu" || resolved.Binary != "tofu" {
		t.Fatalf("expected opentofu/tofu, got %#v", resolved)
	}
	if len(calls) != 1 || calls[0] != "tofu" {
		t.Fatalf("expected auto to try tofu first only, got %v", calls)
	}
}

func TestResolveEngineBinaryAutoFallsBackToTerraform(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "terraform" {
			return "/bin/terraform", nil
		}
		return "", errors.New("not found")
	}

	resolved, err := ResolveEngineBinary("", lookup)
	if err != nil {
		t.Fatalf("ResolveEngineBinary empty auto failed: %v", err)
	}
	if resolved.Name != "terraform" || resolved.Binary != "terraform" {
		t.Fatalf("expected terraform fallback, got %#v", resolved)
	}
}

func TestResolveEngineBinaryRequiresSelectedEngine(t *testing.T) {
	lookup := func(name string) (string, error) {
		return "", errors.New("not found")
	}

	if _, err := ResolveEngineBinary("opentofu", lookup); err == nil {
		t.Fatal("expected missing opentofu executable to fail")
	}
	if _, err := ResolveEngineBinary("terraform", lookup); err == nil {
		t.Fatal("expected missing terraform executable to fail")
	}
}

func TestResolveEngineBinaryRejectsUnknownEngine(t *testing.T) {
	lookup := func(name string) (string, error) {
		return "/bin/" + name, nil
	}

	_, err := ResolveEngineBinary("pulumi", lookup)
	if err == nil {
		t.Fatal("expected unknown engine to fail")
	}
}
