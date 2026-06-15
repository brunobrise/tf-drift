package main

type Ignores struct {
	ResourceTypes []string `json:"resource_types"`
	Attributes    []string `json:"attributes"`
}

type RulesConfig struct {
	GlobalIgnores          Ignores            `json:"global_ignores"`
	SeverityClassification map[string]string  `json:"severity_classification"`
	LayerIgnores           map[string]Ignores `json:"layer_ignores"`
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
	// 1. Check if resource type is ignored globally
	if contains(r.GlobalIgnores.ResourceTypes, resourceType) {
		return true, "LOW"
	}

	// 2. Check if all changed attributes are ignored
	allIgnored := true
	for _, attr := range changedAttrs {
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

	if allIgnored {
		return true, "LOW"
	}

	// 3. Resolve severity
	if sev, ok := r.SeverityClassification[resourceType]; ok {
		return false, sev
	}

	return false, "MEDIUM"
}
