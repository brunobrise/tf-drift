package drift

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type BaselineFile struct {
	Version   string          `json:"version"`
	UpdatedAt string          `json:"updated_at"`
	Entries   []BaselineEntry `json:"entries"`
}

type BaselineEntry struct {
	Layer             string   `json:"layer"`
	Address           string   `json:"address"`
	Type              string   `json:"type"`
	Classification    string   `json:"classification,omitempty"`
	ChangedAttributes []string `json:"changed_attributes,omitempty"`
}

// LoadBaseline loads a baseline JSON file from disk.
func LoadBaseline(path string) (*BaselineFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read baseline file: %w", err)
	}

	var file BaselineFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse baseline file: %w", err)
	}

	return &file, nil
}

// Matches checks if a DriftChange on a layer matches this baseline entry.
func (b *BaselineFile) Matches(layerPath string, change DriftChange) bool {
	normLayer := filepath.ToSlash(layerPath)

	for _, entry := range b.Entries {
		normEntryLayer := filepath.ToSlash(entry.Layer)
		layerMatches := normLayer == normEntryLayer ||
			strings.HasSuffix(normLayer, "/"+normEntryLayer) ||
			strings.HasSuffix(normEntryLayer, "/"+normLayer)

		if !layerMatches {
			continue
		}

		if entry.Address != change.Address || entry.Type != change.Type {
			continue
		}

		if entry.Classification != "" && entry.Classification != string(change.Classification) {
			continue
		}

		if len(entry.ChangedAttributes) > 0 {
			sort.Strings(entry.ChangedAttributes)
			changeAttrs := append([]string(nil), change.ChangedAttributes...)
			sort.Strings(changeAttrs)
			if !reflect.DeepEqual(entry.ChangedAttributes, changeAttrs) {
				continue
			}
		}

		return true
	}

	return false
}

// ApplyBaseline marks detected drifts as acknowledged if they match the baseline.
func ApplyBaseline(results []ScanResult, baseline *BaselineFile) []ScanResult {
	if baseline == nil {
		return results
	}

	for i := range results {
		for j := range results[i].Drifts {
			if baseline.Matches(results[i].Path, results[i].Drifts[j]) {
				results[i].Drifts[j].Acknowledged = true
			}
		}
	}

	return results
}

// SaveBaseline writes all detected changes across results to a baseline file.
func SaveBaseline(path string, results []ScanResult) error {
	file := BaselineFile{
		Version:   "1.0",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Entries:   make([]BaselineEntry, 0),
	}

	for _, res := range results {
		if res.Err != nil {
			continue
		}
		for _, drift := range res.Drifts {
			attrs := append([]string(nil), drift.ChangedAttributes...)
			sort.Strings(attrs)
			file.Entries = append(file.Entries, BaselineEntry{
				Layer:             res.Path,
				Address:           drift.Address,
				Type:              drift.Type,
				Classification:    string(drift.Classification),
				ChangedAttributes: attrs,
			})
		}
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize baseline file: %w", err)
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create baseline directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write baseline file: %w", err)
	}

	return nil
}
