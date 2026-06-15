package main

import (
	"context"
	"sync"
)

type ScanResult struct {
	Path   string
	Drifts []DriftChange
	Err    error
}

type RunnerFunc func(ctx context.Context, dir string, rules RulesConfig, lockState bool, profileOverride string, localProfile bool) ([]DriftChange, error)

// ScanLayersWithRunner executes the worker pool using a custom runner (useful for testing).
func ScanLayersWithRunner(
	ctx context.Context,
	layers []string,
	rules RulesConfig,
	concurrency int,
	lockState bool,
	profileOverride string,
	localProfile bool,
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
				drifts, err := runner(ctx, path, rules, lockState, profileOverride, localProfile)
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
	lockState bool,
	profileOverride string,
	localProfile bool,
	resultsChan chan<- ScanResult,
) {
	ScanLayersWithRunner(ctx, layers, rules, concurrency, lockState, profileOverride, localProfile, resultsChan, RunPlan)
}
