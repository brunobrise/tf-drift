package drift

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDiscoverLayers(t *testing.T) {
	// Create temp base directory
	tmpDir := t.TempDir()

	// Create structure:
	// tmpDir/
	//   terraform/
	//     aws/
	//       workload_api_dev/
	//         007_secret_dev/
	//           main.tf
	//         008_ssm_dev/
	//           (empty)
	//       .terraform/
	//         somefile.tf
	//     ignored_dir/
	//       readme.md

	paths := []string{
		filepath.Join(tmpDir, "terraform", "aws", "workload_api_dev", "007_secret_dev", "main.tf"),
		filepath.Join(tmpDir, "terraform", "aws", "workload_api_dev", "008_ssm_dev"), // Directory with no .tf files
		filepath.Join(tmpDir, "terraform", ".terraform", "somefile.tf"),              // Hidden/excluded dir
		filepath.Join(tmpDir, "terraform", "ignored_dir", "readme.md"),               // No .tf files
	}

	for _, p := range paths {
		dir := filepath.Dir(p)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if filepath.Ext(p) != "" || filepath.Base(p) == "readme.md" || filepath.Base(p) == "main.tf" {
			if err := os.WriteFile(p, []byte(""), 0644); err != nil {
				t.Fatalf("Failed to write file %s: %v", p, err)
			}
		}
	}

	// Run discovery
	discovered, err := DiscoverLayers(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverLayers failed: %v", err)
	}

	// We expect exactly:
	// <tmpDir>/terraform/aws/workload_api_dev/007_secret_dev
	expectedPath := filepath.Join(tmpDir, "terraform", "aws", "workload_api_dev", "007_secret_dev")
	// Clean the path to make sure the separators match
	expectedPath = filepath.Clean(expectedPath)

	if len(discovered) != 1 {
		t.Fatalf("Expected exactly 1 discovered layer, got %d: %v", len(discovered), discovered)
	}

	actualPath := filepath.Clean(discovered[0])
	if actualPath != expectedPath {
		t.Errorf("Expected discovered path to be %s, got %s", expectedPath, actualPath)
	}
}

func TestFilterLayers(t *testing.T) {
	layers := []string{
		"terraform/aws/workload_api_dev/007_secret_dev",
		"terraform/aws/workload_api_dev/008_ssm_dev",
		"terraform/aws/workload_api_prod/007_secret_prod",
		"terraform/vercel/vercel_shr",
	}

	// Test case 1: filter by env
	filteredByEnv := FilterLayers(layers, "workload_api_dev", "")
	sort.Strings(filteredByEnv)
	if len(filteredByEnv) != 2 || filteredByEnv[0] != "terraform/aws/workload_api_dev/007_secret_dev" {
		t.Errorf("Filter by env failed: %v", filteredByEnv)
	}

	// Test case 2: filter by layer path
	filteredByLayer := FilterLayers(layers, "", "terraform/vercel/vercel_shr")
	if len(filteredByLayer) != 1 || filteredByLayer[0] != "terraform/vercel/vercel_shr" {
		t.Errorf("Filter by layer failed: %v", filteredByLayer)
	}
}

func TestResolveDirsAndBraces(t *testing.T) {
	tmpDir := t.TempDir()

	dirs := []string{
		filepath.Join(tmpDir, "aws_dev"),
		filepath.Join(tmpDir, "aws_prod"),
		filepath.Join(tmpDir, "other_dir"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", d, err)
		}
	}

	// 1. Test ExpandBraces
	pattern := filepath.Join(tmpDir, "aws_{dev|prod}")
	expanded := ExpandBraces(pattern)
	sort.Strings(expanded)
	expectedExpanded := []string{
		filepath.Join(tmpDir, "aws_dev"),
		filepath.Join(tmpDir, "aws_prod"),
	}
	sort.Strings(expectedExpanded)

	if len(expanded) != len(expectedExpanded) {
		t.Fatalf("Expected %d expanded patterns, got %d", len(expectedExpanded), len(expanded))
	}
	for i := range expanded {
		if expanded[i] != expectedExpanded[i] {
			t.Errorf("Expected expanded[%d] = %s, got %s", i, expectedExpanded[i], expanded[i])
		}
	}

	// 2. Test ResolveDirs with Braces
	resolved, err := ResolveDirs(pattern)
	if err != nil {
		t.Fatalf("ResolveDirs failed: %v", err)
	}
	sort.Strings(resolved)
	if len(resolved) != 2 || resolved[0] != filepath.Join(tmpDir, "aws_dev") || resolved[1] != filepath.Join(tmpDir, "aws_prod") {
		t.Errorf("ResolveDirs with braces failed: %v", resolved)
	}

	// 3. Test ResolveDirs with Wildcard
	wildcardPattern := filepath.Join(tmpDir, "aws_*")
	resolvedWildcard, err := ResolveDirs(wildcardPattern)
	if err != nil {
		t.Fatalf("ResolveDirs failed: %v", err)
	}
	sort.Strings(resolvedWildcard)
	if len(resolvedWildcard) != 2 || resolvedWildcard[0] != filepath.Join(tmpDir, "aws_dev") || resolvedWildcard[1] != filepath.Join(tmpDir, "aws_prod") {
		t.Errorf("ResolveDirs with wildcard failed: %v", resolvedWildcard)
	}
}
