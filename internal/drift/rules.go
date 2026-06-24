package drift

import (
	"regexp"
	"strings"
)

type Ignores struct {
	ResourceTypes []string `json:"resource_types"`
	Attributes    []string `json:"attributes"`
}

type RulesConfig struct {
	GlobalIgnores          Ignores            `json:"global_ignores"`
	SeverityClassification map[string]string  `json:"severity_classification"`
	SeverityRules          []SeverityRule     `json:"severity_rules"`
	LayerIgnores           map[string]Ignores `json:"layer_ignores"`
}

type SeverityRule struct {
	Name            string                 `json:"name,omitempty"`
	Severity        string                 `json:"severity"`
	ResourceTypes   []string               `json:"resource_types,omitempty"`
	Attributes      []string               `json:"attributes,omitempty"`
	Actions         []string               `json:"actions,omitempty"`
	Classifications []ChangeClassification `json:"classifications,omitempty"`
	LayerPatterns   []string               `json:"layer_patterns,omitempty"`
	AddressPatterns []string               `json:"address_patterns,omitempty"`
}

// contains helper to check if string in slice
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// EvaluateChange checks if the changed attributes are ignored globally or for the specific layer.
// It returns whether the change is ignored, and its severity ("CRITICAL", "HIGH", "MEDIUM", "LOW").
func (r *RulesConfig) EvaluateChange(layerPath string, resourceType string, changedAttrs []string) (bool, string) {
	return r.evaluateChangeDetails(layerPath, DriftChange{
		Type:              resourceType,
		ChangedAttributes: changedAttrs,
	})
}

func (r *RulesConfig) EvaluateDriftChange(layerPath string, change DriftChange) (bool, string) {
	return r.evaluateChangeDetails(layerPath, change)
}

func (r *RulesConfig) evaluateChangeDetails(layerPath string, change DriftChange) (bool, string) {
	// 1. Check if resource type is ignored globally
	if contains(r.GlobalIgnores.ResourceTypes, change.Type) {
		return true, "LOW"
	}

	// 2. Check if all changed attributes are ignored
	allIgnored := true
	for _, attr := range change.ChangedAttributes {
		ignoredGlobally := contains(r.GlobalIgnores.Attributes, attr)
		ignoredInLayer := false

		if layerRules, ok := r.LayerIgnores[layerPath]; ok {
			ignoredInLayer = contains(layerRules.Attributes, attr)
		}

		if !ignoredGlobally && !ignoredInLayer {
			allIgnored = false
			break
		}
	}

	if len(change.ChangedAttributes) > 0 && allIgnored {
		return true, "LOW"
	}

	// 3. Resolve severity
	for _, rule := range r.SeverityRules {
		if rule.matches(layerPath, change) && rule.Severity != "" {
			return false, normalizeSeverity(rule.Severity)
		}
	}

	if sev, ok := r.SeverityClassification[change.Type]; ok {
		return false, normalizeSeverity(sev)
	}

	return false, "MEDIUM"
}

func (rule SeverityRule) matches(layerPath string, change DriftChange) bool {
	if len(rule.ResourceTypes) > 0 && !contains(rule.ResourceTypes, change.Type) {
		return false
	}
	if len(rule.Attributes) > 0 && !overlaps(rule.Attributes, change.ChangedAttributes) {
		return false
	}
	if len(rule.Actions) > 0 && !overlaps(rule.Actions, change.Actions) {
		return false
	}
	if len(rule.Classifications) > 0 && !matchesClassification(rule.Classifications, change.Classification) {
		return false
	}
	if len(rule.LayerPatterns) > 0 && !matchesAnyPattern(rule.LayerPatterns, layerPath) {
		return false
	}
	if len(rule.AddressPatterns) > 0 && !matchesAnyPattern(rule.AddressPatterns, change.Address) {
		return false
	}
	return true
}

func overlaps(want []string, got []string) bool {
	for _, expected := range want {
		if contains(got, expected) {
			return true
		}
	}
	return false
}

func matchesClassification(want []ChangeClassification, got ChangeClassification) bool {
	for _, expected := range want {
		if expected == got {
			return true
		}
	}
	return false
}

func matchesAnyPattern(patterns []string, value string) bool {
	normalized := strings.ReplaceAll(value, "\\", "/")
	for _, pattern := range patterns {
		if globMatches(strings.ReplaceAll(pattern, "\\", "/"), normalized) {
			return true
		}
	}
	return false
}

func globMatches(pattern string, value string) bool {
	if pattern == value {
		return true
	}
	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, `\*`, ".*")
	quoted = strings.ReplaceAll(quoted, `\?`, ".")
	return regexp.MustCompile("^" + quoted + "$").MatchString(value)
}

func normalizeSeverity(severity string) string {
	return strings.ToUpper(strings.TrimSpace(severity))
}
