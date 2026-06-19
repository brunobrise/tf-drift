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

// RunPlan executes the selected engine plan command on the target layer directory.
// It returns a list of detected drift changes, or an error.
func RunPlan(ctx context.Context, layerDir string, rules RulesConfig, options RunnerOptions) ([]DriftChange, error) {
	engine := options.Engine
	if engine.Binary == "" {
		engine = ResolvedEngine{Name: "terraform", Binary: "terraform"}
	}

	// Profile override & local profile support
	if options.ProfileOverride != "" || options.LocalProfile {
		files, err := os.ReadDir(layerDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".tf" {
					filePath := filepath.Join(layerDir, file.Name())
					origContent, err := os.ReadFile(filePath)
					if err == nil {
						contentStr := string(origContent)
						if strings.Contains(contentStr, "assume_role") || strings.Contains(contentStr, "profile") {
							modifiedContent, hasProfile := modifyProviderText(contentStr, options.ProfileOverride)
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
	if os.IsNotExist(statErr) || options.Reconfigure || options.MigrateState {
		args := []string{"init", "-input=false"}
		if options.Reconfigure {
			args = append(args, "-reconfigure")
		}
		if options.MigrateState {
			args = append(args, "-migrate-state")
		}
		cmd := exec.CommandContext(ctx, engine.Binary, args...)
		cmd.Dir = layerDir
		if options.Automation {
			cmd.Env = append(os.Environ(), "TF_IN_AUTOMATION=1")
		}
		// Capture output to prevent corrupting interactive TUI
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("%s init failed: %s%s: %w", engine.Binary, string(out), engineFailureHint(engine), err)
		}
	}

	// 2. Run terraform plan
	planFile := "tfplan"
	planPath := filepath.Join(layerDir, planFile)
	defer func() { _ = os.Remove(planPath) }()

	args := []string{"plan", "-detailed-exitcode", "-out=" + planFile}
	if !options.LockState {
		args = append(args, "-lock=false")
	}

	cmdPlan := exec.CommandContext(ctx, engine.Binary, args...)
	cmdPlan.Dir = layerDir
	if options.Automation {
		cmdPlan.Env = append(os.Environ(), "TF_IN_AUTOMATION=1")
	}
	planOutput, err := cmdPlan.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return nil, fmt.Errorf("%s plan execution failed: %w", engine.Binary, err)
		}
	}

	switch exitCode {
	case 0:
		// No changes/drift
		return nil, nil
	case 2:
		// Drift detected. Run show -json to extract diff.
		cmdShow := exec.CommandContext(ctx, engine.Binary, "show", "-json", planFile)
		cmdShow.Dir = layerDir
		if options.Automation {
			cmdShow.Env = append(os.Environ(), "TF_IN_AUTOMATION=1")
		}
		var stdoutBuf, stderrBuf bytes.Buffer
		cmdShow.Stdout = &stdoutBuf
		cmdShow.Stderr = &stderrBuf
		err := cmdShow.Run()
		if err != nil {
			return nil, fmt.Errorf("%s show failed: %s%s: %w", engine.Binary, stderrBuf.String(), engineFailureHint(engine), err)
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
		return nil, fmt.Errorf("%s plan failed (exit code %d): %s%s", engine.Binary, exitCode, string(planOutput), engineFailureHint(engine))
	}
}

func engineFailureHint(engine ResolvedEngine) string {
	if engine.Name != "opentofu" {
		return ""
	}
	return "\nOpenTofu hint: check explicit provider source addresses, registry resolution, provider version constraints, state encryption keys, and saved plan handling."
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
