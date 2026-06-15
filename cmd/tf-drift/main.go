package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/brunobrise/tf-drift/internal/drift"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

var version = "dev"

func main() {
	dirFlag := flag.String("dir", ".", "Path to the target directory to scan")
	envFlag := flag.String("env", "", "Filter layers by environment name")
	layerFlag := flag.String("layer", "", "Filter layers by specific layer path")
	concurrencyFlag := flag.Int("concurrency", 5, "Max parallel workers")
	formatFlag := flag.String("format", "text", "Non-interactive output format (text|json|markdown|slack)")
	lockFlag := flag.Bool("lock", false, "Enable state locking")
	rulesFlag := flag.String("rules", "rules.json", "Path to the rules configuration file")
	nonInteractiveFlag := flag.Bool("non-interactive", false, "Force disable TUI")
	profileOverrideFlag := flag.String("profile-override", "", "Override AWS provider profile and comment out assume_role blocks")
	localProfileFlag := flag.Bool("local-profile", false, "Comment out assume_role blocks and uncomment existing profiles in provider configs")
	versionFlag := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("tf-drift %s\n", version)
		os.Exit(0)
	}

	// 1. Load Rules Config
	var rules drift.RulesConfig
	rulesPath := *rulesFlag
	if rulesData, err := os.ReadFile(rulesPath); err == nil {
		if err := json.Unmarshal(rulesData, &rules); err != nil {
			fmt.Printf("Warning: Failed to parse rules config: %v. Using defaults.\n", err)
		}
	}

	// 2. Discover and Filter Layers
	baseDir := *dirFlag
	allLayers, err := drift.DiscoverLayers(baseDir)
	if err != nil {
		fmt.Printf("Error discovering layers: %v\n", err)
		os.Exit(1)
	}

	layers := drift.FilterLayers(allLayers, *envFlag, *layerFlag)
	if len(layers) == 0 {
		fmt.Println("No Terraform configuration layers discovered.")
		os.Exit(0)
	}

	// 3. Signal handling context for clean termination
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Determine execution mode (TUI vs Non-Interactive)
	useTUI := !*nonInteractiveFlag && isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stdin.Fd())

	if useTUI {
		logFile, err := os.OpenFile("tf-drift.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			log.SetOutput(logFile)
			log.SetFlags(log.LstdFlags | log.Lmicroseconds)
			defer func() { _ = logFile.Close() }()
		} else {
			log.SetOutput(io.Discard)
		}
	} else {
		log.SetOutput(os.Stderr)
		log.SetFlags(0)
	}

	resultsChan := make(chan drift.ScanResult, len(layers))
	drift.ScanLayers(ctx, layers, rules, *concurrencyFlag, *lockFlag, *profileOverrideFlag, *localProfileFlag, resultsChan)

	if !useTUI {
		// Non-interactive Mode (standard stdout report, good for CI/CD)
		var results []drift.ScanResult
		for res := range resultsChan {
			results = append(results, res)
		}

		drift.PrintNonInteractiveReport(results, *formatFlag)

		// Exit code logic for CI
		hasErrors := false
		hasDrifts := false
		for _, res := range results {
			if res.Err != nil {
				hasErrors = true
			} else if len(res.Drifts) > 0 {
				hasDrifts = true
			}
		}

		if hasErrors {
			os.Exit(1)
		}
		if hasDrifts {
			os.Exit(2)
		}
		os.Exit(0)
	}

	// TUI Mode
	m := drift.InitialModel(layers, rules, *concurrencyFlag, *lockFlag, baseDir)
	p := tea.NewProgram(m)

	// Goroutine to forward progress from workers channel to Bubble Tea program loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res, ok := <-resultsChan:
				if !ok {
					return
				}
				p.Send(drift.LayerScanFinishedMsg{Result: res})
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
