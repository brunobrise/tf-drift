package drift

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// ApplySelectionFilters applies comma-separated include and exclude patterns to
// layer paths. Include is evaluated first, then exclude.
func ApplySelectionFilters(layers []string, includeExpr string, excludeExpr string) ([]string, error) {
	includes := splitSelectionPatterns(includeExpr)
	excludes := splitSelectionPatterns(excludeExpr)

	filtered := layers
	if len(includes) > 0 {
		var included []string
		for _, layer := range layers {
			matched, err := matchesAnyLayerPattern(layer, includes)
			if err != nil {
				return nil, err
			}
			if matched {
				included = append(included, layer)
			}
		}
		filtered = included
	}

	if len(excludes) > 0 {
		var kept []string
		for _, layer := range filtered {
			matched, err := matchesAnyLayerPattern(layer, excludes)
			if err != nil {
				return nil, err
			}
			if !matched {
				kept = append(kept, layer)
			}
		}
		filtered = kept
	}

	return filtered, nil
}

func splitSelectionPatterns(expr string) []string {
	if strings.TrimSpace(expr) == "" {
		return nil
	}

	parts := strings.Split(expr, ",")
	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		pattern := strings.TrimSpace(part)
		if pattern != "" {
			patterns = append(patterns, filepath.ToSlash(pattern))
		}
	}
	return patterns
}

func matchesAnyLayerPattern(layer string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := matchesLayerPattern(layer, pattern)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func matchesLayerPattern(layer string, pattern string) (bool, error) {
	candidates := layerPatternCandidates(layer)
	if !strings.ContainsAny(pattern, "*?[") {
		for _, candidate := range candidates {
			if candidate == pattern {
				return true, nil
			}
		}
		return false, nil
	}

	for _, candidate := range candidates {
		matched, err := path.Match(pattern, candidate)
		if err != nil {
			return false, fmt.Errorf("invalid selection pattern %q: %w", pattern, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func layerPatternCandidates(layer string) []string {
	normalized := filepath.ToSlash(filepath.Clean(layer))
	trimmed := strings.Trim(normalized, "/")
	parts := strings.Split(trimmed, "/")

	seen := make(map[string]bool)
	var candidates []string
	add := func(candidate string) {
		if candidate == "" || seen[candidate] {
			return
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
	}

	add(normalized)
	add(strings.TrimPrefix(normalized, "/"))
	add(path.Base(normalized))
	for i := range parts {
		add(strings.Join(parts[i:], "/"))
	}

	return candidates
}
