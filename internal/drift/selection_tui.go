package drift

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type selectionModel struct {
	layers         []string
	selected       map[string]bool
	selectedLayers []string
	cursor         int
	height         int
	width          int
	baseDir        string
	message        string
	done           bool
	cancelled      bool
	styleName      tuiStyleName
	styles         tuiStyles
}

func initialSelectionModel(layers []string, baseDir string) selectionModel {
	return initialSelectionModelWithStyle(layers, baseDir, tuiStyleModern)
}

func initialSelectionModelWithStyle(layers []string, baseDir string, styleName tuiStyleName) selectionModel {
	selected := make(map[string]bool, len(layers))
	for _, layer := range layers {
		selected[layer] = true
	}

	return selectionModel{
		layers:    layers,
		selected:  selected,
		baseDir:   baseDir,
		styleName: styleName,
		styles:    newTUIStyles(styleName),
	}
}

// RunLayerSelection displays a checkbox picker and returns the selected layers.
func RunLayerSelection(layers []string, baseDir string) ([]string, bool, error) {
	return RunLayerSelectionWithStyle(layers, baseDir, tuiStyleModern)
}

func RunLayerSelectionWithStyle(layers []string, baseDir string, styleName tuiStyleName) ([]string, bool, error) {
	model := initialSelectionModelWithStyle(layers, baseDir, styleName)
	program := tea.NewProgram(model, tea.WithMouseCellMotion())

	finalModel, err := program.Run()
	if err != nil {
		return nil, false, err
	}

	model, ok := finalModel.(selectionModel)
	if !ok {
		return nil, false, fmt.Errorf("unexpected selection model type %T", finalModel)
	}
	if model.cancelled || !model.done {
		return nil, false, nil
	}
	return model.selectedLayers, true, nil
}

func (m selectionModel) Init() tea.Cmd {
	return nil
}

func (m selectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.MouseButtonWheelDown:
			if m.cursor < len(m.layers)-1 {
				m.cursor++
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.layers)-1 {
				m.cursor++
			}
			return m, nil

		case "pgup", "pageup":
			m.cursor -= m.listHeight()
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil

		case "pgdown", "pagedown":
			m.cursor += m.listHeight()
			if m.cursor >= len(m.layers) {
				m.cursor = len(m.layers) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil

		case " ":
			if len(m.layers) > 0 {
				layer := m.layers[m.cursor]
				m.selected[layer] = !m.selected[layer]
				m.message = ""
			}
			return m, nil

		case "a":
			for _, layer := range m.layers {
				m.selected[layer] = true
			}
			m.message = ""
			return m, nil

		case "n":
			for _, layer := range m.layers {
				m.selected[layer] = false
			}
			m.message = ""
			return m, nil

		case "enter":
			selectedLayers := m.selectedLayerList()
			if len(selectedLayers) == 0 {
				m.message = "Select at least one config"
				return m, nil
			}
			m.selectedLayers = selectedLayers
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m selectionModel) View() string {
	if m.cancelled {
		return "Selection cancelled.\n"
	}

	var b strings.Builder
	styles := m.styles
	if styles.name == "" {
		styles = newTUIStyles(tuiStyleModern)
	}

	b.WriteString("\n")
	b.WriteString("  " + styles.title.Render("tf-drift") + " - Select Terraform configs\n")
	_, _ = fmt.Fprintf(&b, "  Selected: %d/%d\n", m.selectedCount(), len(m.layers))
	if m.message != "" {
		_, _ = fmt.Fprintf(&b, "  %s\n", styles.warning.Render(m.message))
	}
	b.WriteString("\n")
	b.WriteString("  " + styles.line(60) + "\n")

	if len(m.layers) == 0 {
		b.WriteString("  No Terraform configs discovered.\n")
	} else {
		listHeight := m.listHeight()
		start := m.cursor - listHeight/2
		if start < 0 {
			start = 0
		}
		end := start + listHeight
		if end > len(m.layers) {
			end = len(m.layers)
			start = end - listHeight
			if start < 0 {
				start = 0
			}
		}

		pathWidth := 54
		if m.width > 20 {
			pathWidth = m.width - 12
		}
		if pathWidth < 20 {
			pathWidth = 20
		}

		for idx := start; idx < end; idx++ {
			layer := m.layers[idx]
			marker := "[ ]"
			if m.selected[layer] {
				marker = "[x]"
			}

			indicator := "  "
			if idx == m.cursor {
				indicator = styles.title.Render(">") + " "
			}

			displayPath := selectionDisplayPath(m.baseDir, layer)
			if len(displayPath) > pathWidth {
				displayPath = "..." + displayPath[len(displayPath)-(pathWidth-3):]
			}

			rowText := fmt.Sprintf("%s%s %s", indicator, marker, displayPath)
			if idx == m.cursor {
				rowText = styles.focus(rowText)
			}
			b.WriteString(rowText + "\n")
		}
	}

	b.WriteString("  " + styles.line(60) + "\n")
	b.WriteString("  [Space] Tick  [a] All  [n] None  [Enter] Scan  [q] Quit\n")
	return b.String()
}

func (m selectionModel) listHeight() int {
	listHeight := m.height - 10
	if m.height == 0 {
		listHeight = 12
	}
	if listHeight < 3 {
		listHeight = 3
	}
	return listHeight
}

func (m selectionModel) selectedCount() int {
	count := 0
	for _, layer := range m.layers {
		if m.selected[layer] {
			count++
		}
	}
	return count
}

func (m selectionModel) selectedLayerList() []string {
	selectedLayers := make([]string, 0, len(m.layers))
	for _, layer := range m.layers {
		if m.selected[layer] {
			selectedLayers = append(selectedLayers, layer)
		}
	}
	return selectedLayers
}

func selectionDisplayPath(baseDir string, layer string) string {
	if rel, err := filepath.Rel(baseDir, layer); err == nil {
		if rel == "." {
			return filepath.Base(layer)
		}
		return rel
	}
	return filepath.Clean(layer)
}
