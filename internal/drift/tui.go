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
	styleName    tuiStyleName
	styles       tuiStyles
}

func initialModel(layers []string, rules RulesConfig, concurrency int, lockState bool, baseDir string) tuiModel {
	return initialModelWithStyle(layers, rules, concurrency, lockState, baseDir, tuiStyleModern)
}

func initialModelWithStyle(layers []string, rules RulesConfig, concurrency int, lockState bool, baseDir string, styleName tuiStyleName) tuiModel {
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
		styleName:    styleName,
		styles:       newTUIStyles(styleName),
	}
}

func InitialModel(layers []string, rules RulesConfig, concurrency int, lockState bool, baseDir string) tea.Model {
	return initialModel(layers, rules, concurrency, lockState, baseDir)
}

func InitialModelWithStyle(layers []string, rules RulesConfig, concurrency int, lockState bool, baseDir string, styleName tuiStyleName) tea.Model {
	return initialModelWithStyle(layers, rules, concurrency, lockState, baseDir, styleName)
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
	styles := m.styles
	if styles.name == "" {
		styles = newTUIStyles(tuiStyleModern)
	}

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
			severity := styles.clean(drift.Severity)
			switch drift.Severity {
			case "CRITICAL":
				severity = styles.err(drift.Severity)
			case "HIGH":
				severity = styles.err(drift.Severity)
			case "MEDIUM":
				severity = styles.drifted(drift.Severity)
			}

			lines = append(lines, fmt.Sprintf("  %d. %s (%s)",
				i+1, styles.accent.Render(drift.Address), severity))
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

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
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

		case tea.MouseButtonWheelDown:
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
		}
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

		case "pgup", "pageup":
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

		case "pgdown", "pagedown":
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
			if len(m.getFilteredLayers()) == 0 {
				m.detailView = false
			}
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

	styles := m.styles
	if styles.name == "" {
		styles = newTUIStyles(tuiStyleModern)
	}

	// 1. Render Progress Header
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + styles.title.Render("tf-drift") + " - Terraform/OpenTofu Drift Detection\n")
	_, _ = fmt.Fprintf(&b, "  Concurrency: %d workers | Mode: Lock=%t\n\n", m.concurrency, m.lockState)

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
	spin := "done"
	if m.processed < m.total {
		spin = spinnerFrames[m.spinnerTick%len(spinnerFrames)]
	}

	_, _ = fmt.Fprintf(&b, "  Progress: [%s] %d%% (%d/%d layers)  %s\n\n",
		bar, int(percent*100), m.processed, m.total, spin)

	filtered := m.getFilteredLayers()

	// 2. Render Detail View if open
	if m.detailView && len(filtered) > 0 {
		if m.cursor >= len(filtered) {
			m.cursor = 0
		}

		layer := filtered[m.cursor]
		_, _ = fmt.Fprintf(&b, "  %s\n", styles.warning.Render("Inspector - "+layer))
		b.WriteString("  " + styles.line(60) + "\n")

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
	_, _ = fmt.Fprintf(&b, "  %s (%d layers shown)\n", styles.accent.Render("Active Filter: "+m.filter), len(filtered))
	b.WriteString("  " + styles.line(60) + "\n")

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
				indicator = "> "
			}

			// Format relative path for display
			displayPath := layer
			if rel, err := filepath.Rel(m.baseDir, layer); err == nil {
				if rel == "." {
					displayPath = filepath.Base(layer)
				} else {
					displayPath = rel
				}
			} else {
				displayPath = filepath.Clean(layer)
			}

			var statusText string
			var styleStatus func(string) string
			if scanned {
				if res.Err != nil {
					statusText = "ERROR"
					styleStatus = styles.err
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
						statusText = fmt.Sprintf("DRIFTED (CRITICAL:%d)", crits)
						styleStatus = styles.err
					} else if highs > 0 {
						statusText = fmt.Sprintf("DRIFTED (HIGH:%d)", highs)
						styleStatus = styles.err
					} else if meds > 0 {
						statusText = fmt.Sprintf("DRIFTED (MEDIUM:%d)", meds)
						styleStatus = styles.drifted
					} else {
						statusText = fmt.Sprintf("DRIFTED (LOW:%d)", len(res.Drifts))
						styleStatus = styles.drifted
					}
				} else {
					statusText = "CLEAN"
					styleStatus = styles.clean
				}
			} else {
				unscannedBefore := 0
				for i := 0; i < idx; i++ {
					if _, alreadyScanned := m.results[filtered[i]]; !alreadyScanned {
						unscannedBefore++
					}
				}
				if unscannedBefore < m.concurrency {
					statusText = fmt.Sprintf("SCANNING %s", spin)
					styleStatus = styles.scanning
				} else {
					statusText = "PENDING"
					styleStatus = styles.muted
				}
			}

			if len(displayPath) > pathWidth {
				displayPath = "..." + displayPath[len(displayPath)-(pathWidth-3):]
			}

			// Render row
			statusStr := statusText
			if idx != m.cursor && styleStatus != nil {
				statusStr = styleStatus(statusText)
			}
			formatStr := fmt.Sprintf("%%s%%-%ds %%s", pathWidth)
			rowText := fmt.Sprintf(formatStr, indicator, displayPath, statusStr)
			if idx == m.cursor {
				b.WriteString(styles.focus(rowText, m.width) + "\n")
			} else {
				b.WriteString(rowText + "\n")
			}
		}
	}

	b.WriteString("  " + styles.line(60) + "\n")
	b.WriteString("  [q] Quit  [f] Filter  [Enter] View Details  [↑/↓] Scroll\n")

	return b.String()
}
