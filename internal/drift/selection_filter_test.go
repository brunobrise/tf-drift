package drift

import (
	"reflect"
	"testing"
)

func TestApplySelectionFiltersIncludeExclude(t *testing.T) {
	layers := []string{
		"/repo/examples/clean-empty",
		"/repo/examples/drift-new-resource",
		"/repo/examples/error-invalid-config",
		"/repo/terraform/prod/network",
	}

	filtered, err := ApplySelectionFilters(layers, "clean-empty,drift-*", "")
	if err != nil {
		t.Fatalf("ApplySelectionFilters include failed: %v", err)
	}
	expected := []string{
		"/repo/examples/clean-empty",
		"/repo/examples/drift-new-resource",
	}
	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("expected include result %v, got %v", expected, filtered)
	}

	filtered, err = ApplySelectionFilters(layers, "", "error-*,terraform/prod/*")
	if err != nil {
		t.Fatalf("ApplySelectionFilters exclude failed: %v", err)
	}
	expected = []string{
		"/repo/examples/clean-empty",
		"/repo/examples/drift-new-resource",
	}
	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("expected exclude result %v, got %v", expected, filtered)
	}
}

func TestApplySelectionFiltersIncludeThenExcludePreservesOrder(t *testing.T) {
	layers := []string{
		"/repo/examples/clean-empty",
		"/repo/examples/drift-new-resource",
		"/repo/examples/error-invalid-config",
	}

	filtered, err := ApplySelectionFilters(layers, "examples/*", "error-*")
	if err != nil {
		t.Fatalf("ApplySelectionFilters failed: %v", err)
	}
	expected := []string{
		"/repo/examples/clean-empty",
		"/repo/examples/drift-new-resource",
	}
	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("expected ordered result %v, got %v", expected, filtered)
	}
}

func TestApplySelectionFiltersRejectsInvalidGlob(t *testing.T) {
	_, err := ApplySelectionFilters([]string{"/repo/examples/clean-empty"}, "[", "")
	if err == nil {
		t.Fatal("expected invalid include glob to return an error")
	}
}
