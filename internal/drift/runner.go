package drift

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
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

var (
	initCacheOnce sync.Once
)

func setupPluginCache() {
	if os.Getenv("TF_PLUGIN_CACHE_DIR") == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			cacheDir := filepath.Join(homeDir, ".terraform.d", "plugin-cache")
			_ = os.MkdirAll(cacheDir, 0755)
			_ = os.Setenv("TF_PLUGIN_CACHE_DIR", cacheDir)
		}
	}
}

// RunPlan executes the terraform plan command on the target layer directory.
// It returns a list of detected drift changes, or an error.
func RunPlan(ctx context.Context, layerDir string, rules RulesConfig, lockState bool, profileOverride string, localProfile bool, reconfigure bool, migrateState bool) ([]DriftChange, error) {
	// Profile override & local profile support
	if profileOverride != "" || localProfile {
		files, err := os.ReadDir(layerDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".tf" {
					filePath := filepath.Join(layerDir, file.Name())
					origContent, err := os.ReadFile(filePath)
					if err == nil {
						contentStr := string(origContent)
						if strings.Contains(contentStr, "assume_role") || strings.Contains(contentStr, "profile") {
							modifiedContent, hasProfile := modifyProviderText(contentStr, profileOverride)
							if !hasProfile && strings.Contains(contentStr, "assume_role") {
								log.Printf("Warning: File %s has assume_role commented but no AWS profile was mentioned.", filePath)
							}
							err = os.WriteFile(filePath, []byte(modifiedContent), 0644)
							if err == nil {
								defer func(path string, orig []byte) {
									_ = os.WriteFile(path, orig, 0644)
								}(filePath, origContent)
							}
						}
					}
				}
			}
		}
	}

	// Set plugin cache dir once safely
	initCacheOnce.Do(setupPluginCache)

	// 1. Initialize if needed, or if reconfigure/migrateState is requested
	tfDir := filepath.Join(layerDir, ".terraform")
	_, statErr := os.Stat(tfDir)
	if os.IsNotExist(statErr) || reconfigure || migrateState {
		args := []string{"init", "-input=false"}
		if reconfigure {
			args = append(args, "-reconfigure")
		}
		if migrateState {
			args = append(args, "-migrate-state")
		}
		cmd := exec.CommandContext(ctx, "terraform", args...)
		cmd.Dir = layerDir
		// Capture output to prevent corrupting interactive TUI
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("terraform init failed: %s: %w", string(out), err)
		}
	}

	// 2. Run terraform plan
	planFile := "tfplan"
	planPath := filepath.Join(layerDir, planFile)
	defer func() { _ = os.Remove(planPath) }()

	args := []string{"plan", "-detailed-exitcode", "-out=" + planFile}
	if !lockState {
		args = append(args, "-lock=false")
	}

	cmdPlan := exec.CommandContext(ctx, "terraform", args...)
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
		cmdShow := exec.CommandContext(ctx, "terraform", "show", "-json", planFile)
		cmdShow.Dir = layerDir
		var stdoutBuf, stderrBuf bytes.Buffer
		cmdShow.Stdout = &stdoutBuf
		cmdShow.Stderr = &stderrBuf
		err := cmdShow.Run()
		if err != nil {
			return nil, fmt.Errorf("terraform show failed: %s: %w", stderrBuf.String(), err)
		}
		showOutput := stdoutBuf.Bytes()

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

// modifyProviderText comments out any assume_role block and injects/uncomments the specified profile.
// It returns the modified content and a boolean indicating if a profile was successfully configured/found.
func modifyProviderText(content string, profileOverride string) (string, bool) {
	// 1. Comment out assume_role { ... } block
	reAssume := regexp.MustCompile(`(?s)assume_role\s*\{[^\}]*\}`)
	content = reAssume.ReplaceAllStringFunc(content, func(m string) string {
		lines := strings.Split(m, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "//") && line != "" {
				lead := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[i] = lead + "# " + strings.TrimLeft(line, " \t")
			}
		}
		return strings.Join(lines, "\n")
	})

	hasProfile := false

	// 2. Handle profile uncommenting/overriding
	reProfile := regexp.MustCompile(`(?m)^\s*(#\s*|//\s*)?profile\s*=\s*"(.*)"`)
	if profileOverride != "" {
		if reProfile.MatchString(content) {
			content = reProfile.ReplaceAllString(content, fmt.Sprintf(`  profile = "%s"`, profileOverride))
		} else {
			// Inject after provider "aws" {
			reProvider := regexp.MustCompile(`(?s)(provider\s*"aws"\s*\{)`)
			content = reProvider.ReplaceAllString(content, fmt.Sprintf("$1\n  profile = \"%s\"", profileOverride))
		}
		hasProfile = true
	} else {
		// Uncomment existing commented profile
		reCommented := regexp.MustCompile(`(?m)^\s*(#\s*|//\s*)profile\s*=\s*"(.*)"`)
		if reCommented.MatchString(content) {
			content = reCommented.ReplaceAllString(content, `  profile = "$2"`)
			hasProfile = true
		} else {
			// Check if there is already an uncommented profile
			reUncommented := regexp.MustCompile(`(?m)^\s*profile\s*=\s*".*"`)
			if reUncommented.MatchString(content) {
				hasProfile = true
			}
		}
	}

	return content, hasProfile
}
