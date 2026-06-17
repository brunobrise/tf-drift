package drift

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoverLayers recursively finds directories containing at least one .tf file,
// skipping hidden directories like .terraform or .git.
func DiscoverLayers(baseDir string) ([]string, error) {
	var layers []string
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			// Skip hidden directories (starts with dot, except the root dir ".")
			if name != "." && name != ".." && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// If it's a .tf file, add its directory to layers
		if filepath.Ext(path) == ".tf" {
			dir := filepath.Dir(path)
			// Avoid duplicates
			found := false
			for _, l := range layers {
				if l == dir {
					found = true
					break
				}
			}
			if !found {
				layers = append(layers, dir)
			}
		}
		return nil
	})

	return layers, err
}

// FilterLayers filters the discovered layers based on environment and layer name flags.
func FilterLayers(layers []string, envFilter string, layerFilter string) []string {
	var filtered []string
	for _, layer := range layers {
		normalized := filepath.ToSlash(layer)

		if layerFilter != "" {
			normFilter := filepath.ToSlash(layerFilter)
			if strings.HasSuffix(normalized, normFilter) {
				filtered = append(filtered, layer)
			}
			continue
		}

		if envFilter != "" {
			parts := strings.Split(normalized, "/")
			match := false
			for _, part := range parts {
				if part == envFilter {
					match = true
					break
				}
			}
			if match {
				filtered = append(filtered, layer)
			}
			continue
		}

		filtered = append(filtered, layer)
	}
	return filtered
}

// ExpandBraces recursively expands brace patterns separated by '|' (e.g. "path/{a|b}")
// into multiple choices. If no braces are found, it returns the pattern itself.
func ExpandBraces(pattern string) []string {
	start := strings.Index(pattern, "{")
	if start == -1 {
		return []string{pattern}
	}
	end := strings.Index(pattern[start:], "}")
	if end == -1 {
		return []string{pattern}
	}
	end = start + end

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	choicesStr := pattern[start+1 : end]
	choices := strings.Split(choicesStr, "|")

	var results []string
	for _, choice := range choices {
		expanded := prefix + choice + suffix
		results = append(results, ExpandBraces(expanded)...)
	}
	return results
}

// ResolveDirs resolves a directory pattern (supporting glob wildcards and braces)
// to a list of existing absolute directory paths.
func ResolveDirs(dirPattern string) ([]string, error) {
	patterns := ExpandBraces(dirPattern)

	var resolvedDirs []string
	seen := make(map[string]bool)

	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.IsDir() {
				abs, err := filepath.Abs(match)
				if err != nil {
					abs = match
				}
				if !seen[abs] {
					seen[abs] = true
					resolvedDirs = append(resolvedDirs, abs)
				}
			}
		}
	}

	return resolvedDirs, nil
}

// DeduplicateStrings removes duplicate strings from a slice.
func DeduplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, val := range slice {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

// StaticPrefix returns the static parent directory path before any wildcard
// or brace expansion characters (*, ?, [, {). If no wildcards exist, it
// returns the pattern itself.
func StaticPrefix(pattern string) string {
	idx := strings.IndexAny(pattern, "*?[{")
	if idx == -1 {
		return pattern
	}
	prefix := pattern[:idx]
	lastSep := strings.LastIndexAny(prefix, "/\\")
	if lastSep == -1 {
		return "."
	}
	return prefix[:lastSep]
}
