package drift

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectionModelTogglesAndStarts(t *testing.T) {
	layers := []string{"layer1", "layer2", "layer3"}
	m := initialSelectionModel(layers, ".")

	if m.selectedCount() != 3 {
		t.Fatalf("expected all layers selected by default, got %d", m.selectedCount())
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = newModel.(selectionModel)
	if m.selected["layer1"] {
		t.Fatal("expected space to untick current layer")
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(selectionModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(selectionModel)
	if m.selectedCount() != 0 {
		t.Fatalf("expected none selected after n, got %d", m.selectedCount())
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(selectionModel)
	if m.done {
		t.Fatal("expected enter with no selected layers to stay on picker")
	}
	if m.message != "Select at least one config" {
		t.Fatalf("expected empty selection message, got %q", m.message)
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newModel.(selectionModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(selectionModel)

	if !m.done || m.cancelled {
		t.Fatalf("expected enter with selected layers to finish, done=%t cancelled=%t", m.done, m.cancelled)
	}
	if !reflect.DeepEqual(m.selectedLayers, layers) {
		t.Fatalf("expected selected layers %v, got %v", layers, m.selectedLayers)
	}
}

func TestSelectionModelQuitCancels(t *testing.T) {
	m := initialSelectionModel([]string{"layer1"}, ".")

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(selectionModel)

	if !m.cancelled {
		t.Fatal("expected q to cancel selection")
	}
}

func TestSelectionModelViewShowsCheckboxesAndControls(t *testing.T) {
	m := initialSelectionModel([]string{"/repo/examples/clean-empty"}, "/repo/examples")
	m.width = 80
	m.height = 24

	view := m.View()
	for _, expected := range []string{
		"Select Terraform configs",
		"[x] clean-empty",
		"[Space] Tick",
		"[Enter] Scan",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected picker view to contain %q, got:\n%s", expected, view)
		}
	}
}
