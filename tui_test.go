package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIModelUpdateKeyboard(t *testing.T) {
	layers := []string{"layer1", "layer2", "layer3"}
	m := initialModel(layers, RulesConfig{}, 2, false)

	// Verify initial state
	if m.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", m.cursor)
	}

	// Press down arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(tuiModel)

	if m.cursor != 1 {
		t.Errorf("Expected cursor to move to 1, got %d", m.cursor)
	}

	// Press down arrow again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(tuiModel)

	if m.cursor != 2 {
		t.Errorf("Expected cursor to move to 2, got %d", m.cursor)
	}

	// Press down arrow at boundary - should wrap around or stay at boundary
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(tuiModel)

	// Staying at boundary (2) is standard practice
	if m.cursor != 2 {
		t.Errorf("Expected cursor to stay at 2, got %d", m.cursor)
	}
}

func TestTUIModelUpdateScanResult(t *testing.T) {
	layers := []string{"layer1", "layer2"}
	m := initialModel(layers, RulesConfig{}, 2, false)

	if m.processed != 0 {
		t.Errorf("Expected processed 0, got %d", m.processed)
	}

	// Send result
	res := ScanResult{
		Path: "layer1",
		Drifts: []DriftChange{
			{Address: "aws_iam_policy.admin", Type: "aws_iam_policy", Actions: []string{"update"}, Severity: "CRITICAL"},
		},
		Err: nil,
	}

	newModel, _ := m.Update(LayerScanFinishedMsg{Result: res})
	m = newModel.(tuiModel)

	if m.processed != 1 {
		t.Errorf("Expected processed 1, got %d", m.processed)
	}

	storedRes, ok := m.results["layer1"]
	if !ok {
		t.Fatalf("Expected results to contain layer1")
	}

	if len(storedRes.Drifts) != 1 || storedRes.Drifts[0].Severity != "CRITICAL" {
		t.Errorf("Expected stored drift to match input")
	}
}
