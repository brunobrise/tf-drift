package drift

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestScanLayersWorkerPool(t *testing.T) {
	layers := []string{
		"layer1", "layer2", "layer3", "layer4", "layer5",
	}

	concurrency := 2
	var activeRuns int32
	var maxActiveRuns int32
	var totalRuns int32

	// Mock runner function that simulates work and tracks concurrency
	expectedOptions := RunnerOptions{
		Engine:          ResolvedEngine{Name: "opentofu", Binary: "tofu"},
		LockState:       true,
		ProfileOverride: "dev",
		LocalProfile:    true,
		Reconfigure:     true,
		MigrateState:    true,
	}

	mockRunner := func(ctx context.Context, dir string, rules RulesConfig, options RunnerOptions) ([]DriftChange, error) {
		if options != expectedOptions {
			t.Fatalf("expected runner options %#v, got %#v", expectedOptions, options)
		}

		// Increment active runs
		currentActive := atomic.AddInt32(&activeRuns, 1)

		// Track maximum active runs seen
		for {
			max := atomic.LoadInt32(&maxActiveRuns)
			if currentActive > max {
				if atomic.CompareAndSwapInt32(&maxActiveRuns, max, currentActive) {
					break
				}
			} else {
				break
			}
		}

		// Simulate execution time
		time.Sleep(10 * time.Millisecond)

		atomic.AddInt32(&totalRuns, 1)
		atomic.AddInt32(&activeRuns, -1)

		return nil, nil
	}

	rules := RulesConfig{}
	resultsChan := make(chan ScanResult, len(layers))

	// Run worker pool
	ScanLayersWithRunner(context.Background(), layers, rules, concurrency, expectedOptions, resultsChan, mockRunner)

	// Collect results
	resultsCount := 0
	for range resultsChan {
		resultsCount++
	}

	if resultsCount != len(layers) {
		t.Errorf("Expected %d results, got %d", len(layers), resultsCount)
	}

	if totalRuns != int32(len(layers)) {
		t.Errorf("Expected %d total runs, got %d", len(layers), totalRuns)
	}

	if maxActiveRuns > int32(concurrency) {
		t.Errorf("Concurrency limit exceeded. Max active runs was %d, expected limit %d", maxActiveRuns, concurrency)
	}
}
