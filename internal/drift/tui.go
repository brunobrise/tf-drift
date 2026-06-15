package drift

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type LayerScanFinishedMsg struct {
	Result ScanResult
}

type spinnerTickMsg struct{}

type tuiModel struct {
	layers       []string
	results      map[string]ScanResult
	processed    int
	total        int
	concurrency  int
	cursor       int
	filter       string // "ALL", "DRIFTED", "ERRORS"
	rules        RulesConfig
	detailView   bool
	lockState    bool
	quitting     bool
	spinnerTick  int
	height       int
	width        int
	detailScroll int
	baseDir      string
}

func initialModel(layers []string, rules RulesConfig, concurrency int, lockState bool, baseDir string) tuiModel {
	return tuiModel{
		layers:       layers,
		results:      make(map[string]ScanResult),
		processed:    0,
		total:        len(layers),
		concurrency:  concurrency,
		cursor:       0,
		filter:       "ALL",
		rules:        rules,
		detailView:   false,
		lockState:    lockState,
		quitting:     false,
		spinnerTick:  0,
		height:       0,
		width:        0,
		detailScroll: 0,
		baseDir:      baseDir,
	}
}

func InitialModel(layers []string, rules RulesConfig, concurrency int, lockState bool, baseDir string) tea.Model {
	return initialModel(layers, rules, concurrency, lockState, baseDir)
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// getFilteredLayers returns the subset of layers matching the active filter.
func (m tuiModel) getFilteredLayers() []string {
	var filtered []string
	for _, layer := range m.layers {
		res, scanned := m.results[layer]

		switch m.filter {
		case "DRIFTED":
			if scanned && res.Err == nil && len(res.Drifts) > 0 {
				filtered = append(filtered, layer)
			}
		case "ERRORS":
			if scanned && res.Err != nil {
				filtered = append(filtered, layer)
			}
		default:
			filtered = append(filtered, layer)
		}
	}
	return filtered
}

// getDetailLines constructs the detail representation lines for a layer.
func (m tuiModel) getDetailLines(layer string) []string {
	res, ok := m.results[layer]
	if !ok {
		return []string{"  No results yet."}
	}

	var lines []string
	if res.Err != nil {
		errLines := strings.Split(res.Err.Error(), "\n")
		for _, el := range errLines {
			lines = append(lines, "  "+el)
		}
	} else if len(res.Drifts) == 0 {
		lines = append(lines, "  No configuration drift detected in this layer.")
	} else {
		lines = append(lines, "  Detected Drifts:", "")
		for i, drift := range res.Drifts {
			// Colored Severity
			sevColor := "32" // default Green (LOW)
			switch drift.Severity {
			case "CRITICAL":
				sevColor = "1;31" // bold red
			case "HIGH":
				sevColor = "31" // red
			case "MEDIUM":
				sevColor = "33" // yellow
			}

			lines = append(lines, fmt.Sprintf("  %d. \033[1m%s\033[0m (\033[%sm%s\033[0m)",
				i+1, drift.Address, sevColor, drift.Severity))
			lines = append(lines, fmt.Sprintf("     Actions: %v", drift.Actions))
			lines = append(lines, fmt.Sprintf("     Changed attributes: %s", strings.Join(drift.ChangedAttributes, ", ")))
			lines = append(lines, "")
		}
	}
	return lines
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerTickMsg:
		m.spinnerTick++
		if m.processed < m.total {
			return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
				return spinnerTickMsg{}
			})
		}
		return m, nil

	case LayerScanFinishedMsg:
		m.results[msg.Result.Path] = msg.Result
		m.processed++
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.detailView {
				if m.detailScroll > 0 {
					m.detailScroll--
				}
			} else {
				if m.cursor > 0 {
					m.cursor--
				}
			}
			return m, nil

		case "down", "j":
			if m.detailView {
				filtered := m.getFilteredLayers()
				if len(filtered) > 0 {
					layer := filtered[m.cursor]
					lines := m.getDetailLines(layer)
					listHeight := m.height - 10
					if m.height == 0 {
						listHeight = 15
					}
					if listHeight < 3 {
						listHeight = 3
					}
					if m.detailScroll < len(lines)-listHeight {
						m.detailScroll++
					}
				}
			} else {
				filtered := m.getFilteredLayers()
				if m.cursor < len(filtered)-1 {
					m.cursor++
				}
			}
			return m, nil

		case "left", "h":
			if m.detailView {
				if m.cursor > 0 {
					m.cursor--
					m.detailScroll = 0
				}
			}
			return m, nil

		case "right", "l":
			if m.detailView {
				filtered := m.getFilteredLayers()
				if m.cursor < len(filtered)-1 {
					m.cursor++
					m.detailScroll = 0
				}
			}
			return m, nil

		case "pgup":
			if m.detailView {
				listHeight := m.height - 10
				if m.height == 0 {
					listHeight = 15
				}
				if listHeight < 3 {
					listHeight = 3
				}
				m.detailScroll -= listHeight
				if m.detailScroll < 0 {
					m.detailScroll = 0
				}
			} else {
				listHeight := m.height - 12
				if m.height == 0 {
					listHeight = 12
				}
				if listHeight < 3 {
					listHeight = 3
				}
				m.cursor -= listHeight
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			return m, nil

		case "pgdown":
			if m.detailView {
				filtered := m.getFilteredLayers()
				if len(filtered) > 0 {
					layer := filtered[m.cursor]
					lines := m.getDetailLines(layer)
					listHeight := m.height - 10
					if m.height == 0 {
						listHeight = 15
					}
					if listHeight < 3 {
						listHeight = 3
					}
					m.detailScroll += listHeight
					if m.detailScroll > len(lines)-listHeight {
						m.detailScroll = len(lines) - listHeight
					}
					if m.detailScroll < 0 {
						m.detailScroll = 0
					}
				}
			} else {
				filtered := m.getFilteredLayers()
				listHeight := m.height - 12
				if m.height == 0 {
					listHeight = 12
				}
				if listHeight < 3 {
					listHeight = 3
				}
				m.cursor += listHeight
				if m.cursor >= len(filtered) {
					m.cursor = len(filtered) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			return m, nil

		case "f":
			// Cycle filters: ALL -> DRIFTED -> ERRORS -> ALL
			switch m.filter {
			case "ALL":
				m.filter = "DRIFTED"
			case "DRIFTED":
				m.filter = "ERRORS"
			case "ERRORS":
				m.filter = "ALL"
			}
			m.cursor = 0 // Reset cursor
			m.detailScroll = 0
			return m, nil

		case "enter":
			filtered := m.getFilteredLayers()
			if len(filtered) > 0 {
				m.detailView = !m.detailView
				m.detailScroll = 0
			}
			return m, nil

		case "esc":
			if m.detailView {
				m.detailView = false
				m.detailScroll = 0
			}
			return m, nil
		}
	}

	return m, nil
}

func (m tuiModel) View() string {
	if m.quitting {
		return "Exiting tf-drift...\n"
	}

	// 1. Render Progress Header
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  \033[1;36mtf-drift\033[0m — Terraform Drift Detection\n")
	b.WriteString(fmt.Sprintf("  Concurrency: %d workers | Mode: Lock=%t\n\n", m.concurrency, m.lockState))

	// Progress bar calculation
	percent := 0.0
	if m.total > 0 {
		percent = float64(m.processed) / float64(m.total)
	}
	barWidth := 40
	filled := int(percent * float64(barWidth))

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spin := "✔"
	if m.processed < m.total {
		spin = spinnerFrames[m.spinnerTick%len(spinnerFrames)]
	}

	b.WriteString(fmt.Sprintf("  Progress: [%s] %d%% (%d/%d layers)  %s\n\n",
		bar, int(percent*100), m.processed, m.total, spin))

	filtered := m.getFilteredLayers()

	// 2. Render Detail View if open
	if m.detailView {
		if m.cursor >= len(filtered) {
			m.cursor = 0
		}
		if len(filtered) == 0 {
			m.detailView = false
			return b.String()
		}

		layer := filtered[m.cursor]
		b.WriteString(fmt.Sprintf("  \033[1;33mInspector — %s\033[0m\n", layer))
		b.WriteString("  " + strings.Repeat("─", 60) + "\n")

		lines := m.getDetailLines(layer)

		listHeight := m.height - 10
		if m.height == 0 {
			listHeight = 15
		}
		if listHeight < 3 {
			listHeight = 3
		}

		if m.detailScroll > len(lines)-listHeight {
			m.detailScroll = len(lines) - listHeight
		}
		if m.detailScroll < 0 {
			m.detailScroll = 0
		}

		endIdx := m.detailScroll + listHeight
		if endIdx > len(lines) {
			endIdx = len(lines)
		}

		for idx := m.detailScroll; idx < endIdx; idx++ {
			b.WriteString(lines[idx] + "\n")
		}

		b.WriteString("\n  [Esc/Enter] Back to List  [↑/↓] Scroll Info  [←/→] Prev/Next Layer\n")
		return b.String()
	}

	// 3. Render List View
	b.WriteString(fmt.Sprintf("  \033[1;37mActive Filter: %s\033[0m (%d layers shown)\n", m.filter, len(filtered)))
	b.WriteString("  " + strings.Repeat("─", 60) + "\n")

	if len(filtered) == 0 {
		b.WriteString("  No layers match the current filter.\n")
	} else {
		// Calculate available height for the list
		listHeight := m.height - 12
		if m.height == 0 {
			listHeight = 12
		}
		if listHeight < 3 {
			listHeight = 3
		}

		// Calculate dynamic path width for alignment
		pathWidth := 50
		if m.width > 30 {
			pathWidth = m.width - 30
		}
		if pathWidth < 20 {
			pathWidth = 20
		}

		// Render window of layers centered on cursor
		start := m.cursor - listHeight/2
		if start < 0 {
			start = 0
		}
		end := start + listHeight
		if end > len(filtered) {
			end = len(filtered)
			start = end - listHeight
			if start < 0 {
				start = 0
			}
		}

		for idx := start; idx < end; idx++ {
			layer := filtered[idx]
			res, scanned := m.results[layer]

			indicator := "  "
			if idx == m.cursor {
				indicator = "\033[1;36m> \033[0m"
			}

			// Format relative path for display
			displayPath := layer
			if rel, err := filepath.Rel(m.baseDir, layer); err == nil {
				displayPath = rel
			} else {
				displayPath = filepath.Clean(layer)
			}

			statusStr := "\033[90mPENDING\033[0m"
			if scanned {
				if res.Err != nil {
					statusStr = "\033[1;31mERROR\033[0m"
				} else if len(res.Drifts) > 0 {
					// Count severity levels
					crits := 0
					highs := 0
					meds := 0
					for _, d := range res.Drifts {
						switch d.Severity {
						case "CRITICAL":
							crits++
						case "HIGH":
							highs++
						case "MEDIUM":
							meds++
						}
					}
					if crits > 0 {
						statusStr = fmt.Sprintf("\033[1;31mDRIFTED (CRITICAL:%d)\033[0m", crits)
					} else if highs > 0 {
						statusStr = fmt.Sprintf("\033[31mDRIFTED (HIGH:%d)\033[0m", highs)
					} else if meds > 0 {
						statusStr = fmt.Sprintf("\033[33mDRIFTED (MEDIUM:%d)\033[0m", meds)
					} else {
						statusStr = fmt.Sprintf("\033[32mDRIFTED (LOW:%d)\033[0m", len(res.Drifts))
					}
				} else {
					statusStr = "\033[32mCLEAN\033[0m"
				}
			} else {
				// Spinner next to active
				statusStr = fmt.Sprintf("\033[36mSCANNING %s\033[0m", spin)
			}

			if len(displayPath) > pathWidth {
				displayPath = "..." + displayPath[len(displayPath)-(pathWidth-3):]
			}

			// Render row
			formatStr := fmt.Sprintf("%%s%%-%ds %%s\n", pathWidth)
			rowText := fmt.Sprintf(formatStr, indicator, displayPath, statusStr)
			if idx == m.cursor {
				highlightBg := "\033[48;5;238m"
				highlightedRow := highlightBg + strings.ReplaceAll(rowText, "\033[0m", "\033[0m"+highlightBg) + "\033[0m"
				b.WriteString(highlightedRow)
			} else {
				b.WriteString(rowText)
			}
		}
	}

	b.WriteString("  " + strings.Repeat("─", 60) + "\n")
	b.WriteString("  [q] Quit  [f] Filter  [Enter] View Details  [↑/↓] Scroll\n")

	return b.String()
}
