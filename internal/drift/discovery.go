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
