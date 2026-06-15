package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
)

type PlanJSON struct {
	ResourceChanges []ResourceChange `json:"resource_changes"`
}

type ResourceChange struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Change  Change `json:"change"`
}

type Change struct {
	Actions []string               `json:"actions"`
	Before  map[string]interface{} `json:"before"`
	After   map[string]interface{} `json:"after"`
}

type DriftChange struct {
	Address           string   `json:"address"`
	Type              string   `json:"type"`
	Actions           []string `json:"actions"`
	ChangedAttributes []string `json:"changed_attributes"`
	Severity          string   `json:"severity"`
}

// getChangedAttributes compares two maps and returns sorted list of changed keys
func getChangedAttributes(before, after map[string]interface{}) []string {
	keysMap := make(map[string]bool)
	for k := range before {
		keysMap[k] = true
	}
	for k := range after {
		keysMap[k] = true
	}

	var keys []string
	for k := range keysMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var changed []string
	for _, k := range keys {
		vBefore := before[k]
		vAfter := after[k]
		if !reflect.DeepEqual(vBefore, vAfter) {
			changed = append(changed, k)
		}
	}
	return changed
}

// parsePlanJSON parses the JSON representation from terraform show
func parsePlanJSON(data []byte) ([]DriftChange, error) {
	var plan PlanJSON
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}

	var changes []DriftChange
	for _, rc := range plan.ResourceChanges {
		// If actions contains only "no-op" or "read", it's not a drift change
		hasAction := false
		for _, action := range rc.Change.Actions {
			if action != "no-op" && action != "read" {
				hasAction = true
				break
			}
		}
		if !hasAction {
			continue
		}

		changedAttrs := getChangedAttributes(rc.Change.Before, rc.Change.After)

		changes = append(changes, DriftChange{
			Address:           rc.Address,
			Type:              rc.Type,
			Actions:           rc.Change.Actions,
			ChangedAttributes: changedAttrs,
		})
	}
	return changes, nil
}

// RunPlan executes the terraform plan command on the target layer directory.
// It returns a list of detected drift changes, or an error.
func RunPlan(layerDir string, rules RulesConfig, lockState bool) ([]DriftChange, error) {
	// Set plugin cache dir if not set
	if os.Getenv("TF_PLUGIN_CACHE_DIR") == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			cacheDir := filepath.Join(homeDir, ".terraform.d", "plugin-cache")
			_ = os.MkdirAll(cacheDir, 0755)
			os.Setenv("TF_PLUGIN_CACHE_DIR", cacheDir)
		}
	}

	// 1. Initialize if needed
	tfDir := filepath.Join(layerDir, ".terraform")
	if _, err := os.Stat(tfDir); os.IsNotExist(err) {
		cmd := exec.Command("terraform", "init", "-input=false")
		cmd.Dir = layerDir
		// Capture output to prevent corrupting interactive TUI
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("terraform init failed: %s: %w", string(out), err)
		}
	}

	// 2. Run terraform plan
	planFile := "tfplan"
	planPath := filepath.Join(layerDir, planFile)
	defer os.Remove(planPath)

	args := []string{"plan", "-detailed-exitcode", "-out=" + planFile}
	if !lockState {
		args = append(args, "-lock=false")
	}

	cmdPlan := exec.Command("terraform", args...)
	cmdPlan.Dir = layerDir
	planOutput, err := cmdPlan.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return nil, fmt.Errorf("terraform plan execution failed: %w", err)
		}
	}

	switch exitCode {
	case 0:
		// No changes/drift
		return nil, nil
	case 2:
		// Drift detected. Run terraform show -json to extract diff
		cmdShow := exec.Command("terraform", "show", "-json", planFile)
		cmdShow.Dir = layerDir
		showOutput, err := cmdShow.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("terraform show failed: %s: %w", string(showOutput), err)
		}

		rawChanges, err := parsePlanJSON(showOutput)
		if err != nil {
			return nil, err
		}

		// Filter changes using rules.json
		var filteredChanges []DriftChange
		for _, change := range rawChanges {
			ignored, severity := rules.EvaluateChange(layerDir, change.Type, change.ChangedAttributes)
			if !ignored {
				change.Severity = severity
				filteredChanges = append(filteredChanges, change)
			}
		}

		return filteredChanges, nil
	default:
		// Any other exit code is a genuine error
		return nil, fmt.Errorf("terraform plan failed (exit code %d): %s", exitCode, string(planOutput))
	}
}
