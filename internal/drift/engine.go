package drift

import (
	"fmt"
	"os/exec"
	"strings"
)

type ResolvedEngine struct {
	Name   string
	Binary string
}

type executableLookup func(string) (string, error)

func ResolveEngine(engine string) (ResolvedEngine, error) {
	return ResolveEngineBinary(engine, exec.LookPath)
}

func ResolveEngineBinary(engine string, lookup executableLookup) (ResolvedEngine, error) {
	selected := strings.ToLower(strings.TrimSpace(engine))
	if selected == "" {
		selected = "auto"
	}

	switch selected {
	case "auto":
		if _, err := lookup("tofu"); err == nil {
			return ResolvedEngine{Name: "opentofu", Binary: "tofu"}, nil
		}
		if _, err := lookup("terraform"); err == nil {
			return ResolvedEngine{Name: "terraform", Binary: "terraform"}, nil
		}
		return ResolvedEngine{}, fmt.Errorf("no supported IaC engine found: install OpenTofu (tofu) or Terraform, or set -engine")
	case "terraform":
		return requireEngine("terraform", "terraform", lookup)
	case "opentofu", "tofu":
		return requireEngine("opentofu", "tofu", lookup)
	default:
		return ResolvedEngine{}, fmt.Errorf("unsupported engine %q: use auto, terraform, opentofu, or tofu", engine)
	}
}

func requireEngine(name string, binary string, lookup executableLookup) (ResolvedEngine, error) {
	if _, err := lookup(binary); err != nil {
		return ResolvedEngine{}, fmt.Errorf("%s executable %q not found in PATH: %w", name, binary, err)
	}
	return ResolvedEngine{Name: name, Binary: binary}, nil
}
