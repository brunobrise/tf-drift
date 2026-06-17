package drift

import (
	"path/filepath"
	"sort"
	"testing"
)

func TestExamplesExposeMultipleStatuses(t *testing.T) {
	examplesDir := filepath.Clean(filepath.Join("..", "..", "examples"))

	layers, err := DiscoverLayers(examplesDir)
	if err != nil {
		t.Fatalf("DiscoverLayers examples failed: %v", err)
	}

	var names []string
	for _, layer := range layers {
		names = append(names, filepath.Base(layer))
	}
	sort.Strings(names)

	expected := []string{
		"clean-empty",
		"drift-new-resource",
		"error-invalid-config",
	}

	if len(names) != len(expected) {
		t.Fatalf("expected %d examples, got %d: %v", len(expected), len(names), names)
	}
	for i := range expected {
		if names[i] != expected[i] {
			t.Fatalf("expected example %q at index %d, got %q from %v", expected[i], i, names[i], names)
		}
	}
}
