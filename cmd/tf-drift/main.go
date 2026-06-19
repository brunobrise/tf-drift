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
	"path/filepath"
	"syscall"

	"github.com/brunobrise/tf-drift/internal/drift"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

var version = "dev"

func main() {
	dirFlag := flag.String("dir", ".", "Path to the target directory to scan (supports glob and {a|b} choices)")
	envFlag := flag.String("env", "", "Filter layers by environment name")
	layerFlag := flag.String("layer", "", "Filter layers by specific layer path")
	includeFlag := flag.String("include", "", "Comma-separated config suffix/glob patterns to include")
	excludeFlag := flag.String("exclude", "", "Comma-separated config suffix/glob patterns to exclude")
	concurrencyFlag := flag.Int("concurrency", 5, "Max parallel workers")
	formatFlag := flag.String("format", "text", "Non-interactive output format (text|json|markdown|slack)")
	lockFlag := flag.Bool("lock", false, "Enable state locking")
	rulesFlag := flag.String("rules", "rules.json", "Path to the rules configuration file")
	nonInteractiveFlag := flag.Bool("non-interactive", false, "Force disable TUI")
	tuiStyleFlag := flag.String("tui-style", "", "TUI style (modern|classic|minimal|accessible); defaults to modern or TF_DRIFT_TUI_STYLE")
	profileOverrideFlag := flag.String("profile-override", "", "Override AWS provider profile and comment out assume_role blocks")
	localProfileFlag := flag.Bool("local-profile", false, "Comment out assume_role blocks and uncomment existing profiles in provider configs")
	engineFlag := flag.String("engine", "auto", "IaC engine to run (auto|terraform|opentofu|tofu)")
	reconfigureFlag := flag.Bool("reconfigure", false, "Run engine init with -reconfigure flag")
	migrateStateFlag := flag.Bool("migrate-state", false, "Run engine init with -migrate-state flag")
	versionFlag := false
	registerVersionFlags(flag.CommandLine, &versionFlag)

	flag.Parse()

	if versionFlag {
		fmt.Printf("tf-drift %s\n", resolvedVersion())
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
	resolvedDirs, err := drift.ResolveDirs(baseDir)
	if err != nil {
		fmt.Printf("Error resolving directory pattern: %v\n", err)
		os.Exit(1)
	}
	if len(resolvedDirs) == 0 {
		fmt.Printf("No matching directories found for pattern: %s\n", baseDir)
		os.Exit(0)
	}

	var allLayers []string
	for _, d := range resolvedDirs {
		layers, err := drift.DiscoverLayers(d)
		if err != nil {
			fmt.Printf("Error discovering layers in %s: %v\n", d, err)
			os.Exit(1)
		}
		allLayers = append(allLayers, layers...)
	}

	allLayers = drift.DeduplicateStrings(allLayers)

	layers := drift.FilterLayers(allLayers, *envFlag, *layerFlag)
	layers, err = drift.ApplySelectionFilters(layers, *includeFlag, *excludeFlag)
	if err != nil {
		fmt.Printf("Error applying selection filters: %v\n", err)
		os.Exit(1)
	}
	if len(layers) == 0 {
		fmt.Println("No Terraform/OpenTofu configuration layers selected.")
		os.Exit(0)
	}

	// 3. Signal handling context for clean termination
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	displayBaseDir := drift.StaticPrefix(baseDir)
	if absBaseDir, err := filepath.Abs(displayBaseDir); err == nil {
		displayBaseDir = absBaseDir
	}

	// 4. Determine execution mode (TUI vs Non-Interactive)
	useTUI := !*nonInteractiveFlag && isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stdin.Fd())
	engine, err := drift.ResolveEngine(*engineFlag)
	if err != nil {
		fmt.Printf("Error resolving engine: %v\n", err)
		os.Exit(1)
	}

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

	if useTUI {
		tuiStyle := drift.ResolveTUIStyle(*tuiStyleFlag)
		selectedLayers, proceed, err := drift.RunLayerSelectionWithStyle(layers, displayBaseDir, tuiStyle)
		if err != nil {
			fmt.Printf("Error running selection TUI: %v\n", err)
			os.Exit(1)
		}
		if !proceed {
			os.Exit(0)
		}
		layers = selectedLayers
	}

	resultsChan := make(chan drift.ScanResult, len(layers))
	options := drift.RunnerOptions{
		Engine:          engine,
		LockState:       *lockFlag,
		ProfileOverride: *profileOverrideFlag,
		LocalProfile:    *localProfileFlag,
		Reconfigure:     *reconfigureFlag,
		MigrateState:    *migrateStateFlag,
		Automation:      !useTUI,
	}

	drift.ScanLayers(ctx, layers, rules, *concurrencyFlag, options, resultsChan)

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
	tuiStyle := drift.ResolveTUIStyle(*tuiStyleFlag)
	m := drift.InitialModelWithStyle(layers, rules, *concurrencyFlag, *lockFlag, displayBaseDir, tuiStyle)
	p := tea.NewProgram(m, tea.WithMouseCellMotion())

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
