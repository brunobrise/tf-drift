package drift

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIModelUpdateKeyboard(t *testing.T) {
	layers := []string{"layer1", "layer2", "layer3"}
	m := initialModel(layers, RulesConfig{}, 2, false, ".")

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
	m := initialModel(layers, RulesConfig{}, 2, false, ".")

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

func TestTUIWindowSize(t *testing.T) {
	layers := []string{"layer1"}
	m := initialModel(layers, RulesConfig{}, 2, false, ".")

	if m.width != 0 || m.height != 0 {
		t.Errorf("Expected initial dimensions to be 0, got %dx%d", m.width, m.height)
	}

	// Send WindowSizeMsg
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(tuiModel)

	if m.width != 80 || m.height != 24 {
		t.Errorf("Expected dimensions to be 80x24, got %dx%d", m.width, m.height)
	}
}

func TestTUIDetailViewScroll(t *testing.T) {
	layers := []string{"layer1"}
	m := initialModel(layers, RulesConfig{}, 2, false, ".")
	m.results["layer1"] = ScanResult{
		Path: "layer1",
		Drifts: []DriftChange{
			{Address: "drift1", Severity: "CRITICAL"},
			{Address: "drift2", Severity: "CRITICAL"},
			{Address: "drift3", Severity: "CRITICAL"},
			{Address: "drift4", Severity: "CRITICAL"},
			{Address: "drift5", Severity: "CRITICAL"},
		},
	}
	m.detailView = true
	m.height = 10 // small height to force scrollability

	// Verify initial scroll is 0
	if m.detailScroll != 0 {
		t.Errorf("Expected initial scroll offset 0, got %d", m.detailScroll)
	}

	// Press down arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(tuiModel)

	if m.detailScroll != 1 {
		t.Errorf("Expected scroll offset to be 1 after Down press, got %d", m.detailScroll)
	}

	// Press up arrow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(tuiModel)

	if m.detailScroll != 0 {
		t.Errorf("Expected scroll offset to be 0 after Up press, got %d", m.detailScroll)
	}
}

func TestTUIPaging(t *testing.T) {
	// 1. Test List View Paging
	layers := []string{"l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8", "l9", "l10"}
	m := initialModel(layers, RulesConfig{}, 2, false, ".")
	m.height = 16 // listHeight will be 16 - 12 = 4

	// Cursor at 0, press pgdown
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = newModel.(tuiModel)

	if m.cursor != 4 {
		t.Errorf("Expected cursor to jump to 4 on PgDown, got %d", m.cursor)
	}

	// Press pgup
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = newModel.(tuiModel)

	if m.cursor != 0 {
		t.Errorf("Expected cursor to jump back to 0 on PgUp, got %d", m.cursor)
	}

	// 2. Test Detail View Paging
	m.detailView = true
	m.height = 13 // listHeight will be 13 - 10 = 3
	m.results["l1"] = ScanResult{
		Path: "l1",
		Drifts: []DriftChange{
			{Address: "d1", Severity: "CRITICAL"},
			{Address: "d2", Severity: "CRITICAL"},
			{Address: "d3", Severity: "CRITICAL"},
			{Address: "d4", Severity: "CRITICAL"},
			{Address: "d5", Severity: "CRITICAL"},
			{Address: "d6", Severity: "CRITICAL"},
		},
	}

	// Pgdown in detail view
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = newModel.(tuiModel)

	if m.detailScroll != 3 {
		t.Errorf("Expected detailScroll to jump to 3 on PgDown, got %d", m.detailScroll)
	}

	// Pgup in detail view
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = newModel.(tuiModel)

	if m.detailScroll != 0 {
		t.Errorf("Expected detailScroll to return to 0 on PgUp, got %d", m.detailScroll)
	}
}

func TestTUIDetailViewLayerNavigation(t *testing.T) {
	layers := []string{"layer1", "layer2", "layer3"}
	m := initialModel(layers, RulesConfig{}, 2, false, ".")
	m.results["layer1"] = ScanResult{Path: "layer1"}
	m.results["layer2"] = ScanResult{Path: "layer2"}
	m.results["layer3"] = ScanResult{Path: "layer3"}

	// 1. When detailView is false, h/l/left/right should NOT change cursor or detailScroll
	m.detailView = false
	m.cursor = 1
	m.detailScroll = 5

	// Left/h
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m2 := newModel.(tuiModel)
	if m2.cursor != 1 {
		t.Errorf("Expected cursor to remain 1 when detailView is false, got %d", m2.cursor)
	}

	// Right/l
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m2 = newModel.(tuiModel)
	if m2.cursor != 1 {
		t.Errorf("Expected cursor to remain 1 when detailView is false, got %d", m2.cursor)
	}

	// 2. When detailView is true, left/h should move cursor back and reset scroll
	m.detailView = true
	m.cursor = 1
	m.detailScroll = 3

	// Press h
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(tuiModel)
	if m.cursor != 0 {
		t.Errorf("Expected cursor to decrement to 0 on 'h', got %d", m.cursor)
	}
	if m.detailScroll != 0 {
		t.Errorf("Expected detailScroll to reset to 0, got %d", m.detailScroll)
	}

	// Press left arrow at boundary (0)
	m.cursor = 0
	m.detailScroll = 4
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = newModel.(tuiModel)
	if m.cursor != 0 {
		t.Errorf("Expected cursor to stay at 0 on boundary left press, got %d", m.cursor)
	}
	if m.detailScroll != 4 {
		t.Errorf("Expected detailScroll to remain 4 on boundary left press, got %d", m.detailScroll)
	}

	// 3. When detailView is true, right/l should move cursor forward and reset scroll
	m.cursor = 1
	m.detailScroll = 3

	// Press l
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = newModel.(tuiModel)
	if m.cursor != 2 {
		t.Errorf("Expected cursor to increment to 2 on 'l', got %d", m.cursor)
	}
	if m.detailScroll != 0 {
		t.Errorf("Expected detailScroll to reset to 0, got %d", m.detailScroll)
	}

	// Press right arrow at boundary (2)
	m.cursor = 2
	m.detailScroll = 4
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = newModel.(tuiModel)
	if m.cursor != 2 {
		t.Errorf("Expected cursor to stay at 2 on boundary right press, got %d", m.cursor)
	}
	if m.detailScroll != 4 {
		t.Errorf("Expected detailScroll to remain 4 on boundary right press, got %d", m.detailScroll)
	}
}
