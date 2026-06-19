package drift

import (
	"context"
	"sync"
)

type ScanResult struct {
	Path   string
	Drifts []DriftChange
	Err    error
}

type RunnerOptions struct {
	Engine          ResolvedEngine
	LockState       bool
	ProfileOverride string
	LocalProfile    bool
	Reconfigure     bool
	MigrateState    bool
	Automation      bool
}

type RunnerFunc func(ctx context.Context, dir string, rules RulesConfig, options RunnerOptions) ([]DriftChange, error)

// ScanLayersWithRunner executes the worker pool using a custom runner (useful for testing).
func ScanLayersWithRunner(
	ctx context.Context,
	layers []string,
	rules RulesConfig,
	concurrency int,
	options RunnerOptions,
	resultsChan chan<- ScanResult,
	runner RunnerFunc,
) {
	if concurrency <= 0 {
		concurrency = 1
	}

	tasksChan := make(chan string, len(layers))
	for _, layer := range layers {
		tasksChan <- layer
	}
	close(tasksChan)

	var wg sync.WaitGroup
	// Start N worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range tasksChan {
				drifts, err := runner(ctx, path, rules, options)
				resultsChan <- ScanResult{
					Path:   path,
					Drifts: drifts,
					Err:    err,
				}
			}
		}()
	}

	// Close results channel asynchronously when all workers complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()
}

// ScanLayers executes the worker pool using the native Terraform runner.
func ScanLayers(
	ctx context.Context,
	layers []string,
	rules RulesConfig,
	concurrency int,
	options RunnerOptions,
	resultsChan chan<- ScanResult,
) {
	ScanLayersWithRunner(ctx, layers, rules, concurrency, options, resultsChan, RunPlan)
}
