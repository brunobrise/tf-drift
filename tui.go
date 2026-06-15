package main

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
	layers      []string
	results     map[string]ScanResult
	processed   int
	total       int
	concurrency int
	cursor      int
	filter      string // "ALL", "DRIFTED", "ERRORS"
	rules       RulesConfig
	detailView  bool
	lockState   bool
	quitting    bool
	spinnerTick int
}

func initialModel(layers []string, rules RulesConfig, concurrency int, lockState bool) tuiModel {
	return tuiModel{
		layers:      layers,
		results:     make(map[string]ScanResult),
		processed:   0,
		total:       len(layers),
		concurrency: concurrency,
		cursor:      0,
		filter:      "ALL",
		rules:       rules,
		detailView:  false,
		lockState:   lockState,
		quitting:    false,
		spinnerTick: 0,
	}
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

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			filtered := m.getFilteredLayers()
			if m.cursor < len(filtered)-1 {
				m.cursor++
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
			return m, nil

		case "enter":
			filtered := m.getFilteredLayers()
			if len(filtered) > 0 {
				m.detailView = !m.detailView
			}
			return m, nil

		case "esc":
			if m.detailView {
				m.detailView = false
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
		res := m.results[layer]

		b.WriteString(fmt.Sprintf("  \033[1;33mInspector — %s\033[0m\n", layer))
		b.WriteString("  " + strings.Repeat("─", 60) + "\n")

		if res.Err != nil {
			b.WriteString(fmt.Sprintf("  \033[1;31mError Executing Plan:\033[0m\n  %v\n", res.Err))
		} else if len(res.Drifts) == 0 {
			b.WriteString("  No configuration drift detected in this layer.\n")
		} else {
			b.WriteString("  Detected Drifts:\n\n")
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

				b.WriteString(fmt.Sprintf("  %d. \033[1m%s\033[0m (\033[%sm%s\033[0m)\n",
					i+1, drift.Address, sevColor, drift.Severity))
				b.WriteString(fmt.Sprintf("     Actions: %v\n", drift.Actions))
				b.WriteString(fmt.Sprintf("     Changed attributes: %s\n\n", strings.Join(drift.ChangedAttributes, ", ")))
			}
		}

		b.WriteString("\n  [Esc/Enter] Back to List\n")
		return b.String()
	}

	// 3. Render List View
	b.WriteString(fmt.Sprintf("  \033[1;37mActive Filter: %s\033[0m (%d layers shown)\n", m.filter, len(filtered)))
	b.WriteString("  " + strings.Repeat("─", 60) + "\n")

	if len(filtered) == 0 {
		b.WriteString("  No layers match the current filter.\n")
	} else {
		// Render window of layers
		start := m.cursor - 5
		if start < 0 {
			start = 0
		}
		end := start + 12
		if end > len(filtered) {
			end = len(filtered)
		}

		for idx := start; idx < end; idx++ {
			layer := filtered[idx]
			res, scanned := m.results[layer]

			indicator := "  "
			if idx == m.cursor {
				indicator = "\033[1;36m> \033[0m"
			}

			// Format relative path for display
			displayPath := filepath.Clean(layer)

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

			// Render row
			b.WriteString(fmt.Sprintf("%s%-50s %s\n", indicator, displayPath, statusStr))
		}
	}

	b.WriteString("  " + strings.Repeat("─", 60) + "\n")
	b.WriteString("  [q] Quit  [f] Filter  [Enter] View Details  [↑/↓] Scroll\n")

	return b.String()
}
