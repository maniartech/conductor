// Package concurrent provides concurrent orchestration for the orchestrator library.
// It includes the ConcurrentBuilder for creating and executing concurrent workflows
// with comprehensive error handling, atomic status management, and hierarchical configuration.
package concurrent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maniartech/orchestrator/internal/orchestration"
	"github.com/maniartech/orchestrator/pkg/config"
	orchContext "github.com/maniartech/orchestrator/pkg/context"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/pkg/types"
)

// ConcurrentBuilder provides a fluent API for creating and configuring concurrent orchestrations.
// It executes child orchestrations simultaneously using goroutines with comprehensive error handling
// and configuration inheritance following Go best practices.
type ConcurrentBuilder struct {
	*orchestration.BaseContainerOrchestration
	namer         *types.HierarchicalNamer // Hierarchical naming system
	parentContext *types.NamingContext     // Parent naming context for nested orchestrations
}

// Concurrent creates a new ConcurrentBuilder with the provided orchestrations.
// The orchestrations will be executed simultaneously using goroutines.
func Concurrent(orchestrations ...types.Orchestration) *ConcurrentBuilder {
	if len(orchestrations) == 0 {
		panic("concurrent orchestration requires at least one orchestration")
	}

	// Validate all orchestrations are non-nil
	for i, orch := range orchestrations {
		if orch == nil {
			panic(fmt.Sprintf("orchestration at index %d cannot be nil", i))
		}
	}

	cb := &ConcurrentBuilder{
		BaseContainerOrchestration: orchestration.NewBaseContainerOrchestration("concurrent", orchestrations),
	}

	// Initialize hierarchical naming system
	cb.initializeNaming()

	// Initialize path resolver
	cb.GetPathResolver().SetCallbacks(
		func() string { // avoid recursive call through pathResolver
			if cb.namer != nil {
				return cb.namer.GetContext().GetPath()
			}
			return ""
		},
		func() []types.Orchestration { return cb.GetChildren() },
		func(child types.Orchestration, index int) string { return cb.getChildName(child, index) },
	)

	return cb
}

// initializeNaming initializes the hierarchical naming system for this concurrent builder.
func (cb *ConcurrentBuilder) initializeNaming() {
	// Initialize with default naming (will be updated if Named() is called)
	cb.namer = types.NewHierarchicalNamer(cb.parentContext, cb.GetName(), "concurrent", 0)
}

// SetParentContext sets the parent naming context for nested orchestrations.
func (cb *ConcurrentBuilder) SetParentContext(parentContext *types.NamingContext, index int) {
	cb.parentContext = parentContext
	cb.namer = types.NewHierarchicalNamer(parentContext, cb.GetName(), "concurrent", index)
}

// Named sets a name for the concurrent orchestration for observability and debugging.
func (cb *ConcurrentBuilder) Named(name string) types.Orchestration {
	cb.SetName(name)
	// Refresh the naming system with the new name
	cb.namer = types.NewHierarchicalNamer(cb.parentContext, name, "concurrent", 0)
	return cb
}

// GetType returns the orchestration type as "concurrent".
func (cb *ConcurrentBuilder) GetType() string {
	return "concurrent"
}

// With applies configuration to the orchestration.
func (cb *ConcurrentBuilder) With(config config.Config) types.Orchestration {
	cb.SetConfig(config)
	return cb
}

// ErrorBoundary sets the error handling strategy for this orchestration.
func (cb *ConcurrentBuilder) ErrorBoundary(strategy errors.ErrorStrategy) types.Orchestration {
	cb.SetErrorBoundary(strategy)
	return cb
}

// Execute runs the concurrent orchestration with the provided context and configuration.
func (cb *ConcurrentBuilder) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
	// Ensure concurrent can only be executed once
	if !cb.CompareAndSwapStatus(types.NotStarted, types.Running) {
		return nil, fmt.Errorf("concurrent orchestration already executed or in progress, current status: %v", cb.GetStatus())
	}

	// Apply configuration inheritance
	finalConfig := config
	if cb.GetConfig() != nil {
		finalConfig = cb.GetConfig().Inherit(config)
	}

	// Apply error boundary if specified
	if cb.GetErrorBoundary() != nil {
		finalConfig.ErrorStrategy = *cb.GetErrorBoundary()
	}

	// Create shared orchestrator context for data sharing between tasks
	if finalConfig.OrchestrationContext == nil {
		orchCtx := orchContext.NewContext(finalConfig)
		finalConfig.OrchestrationContext = orchCtx
	}

	// Execute orchestrations concurrently based on error strategy
	var executionError error
	var result *result.Result
	switch finalConfig.ErrorStrategy {
	case errors.FailFast:
		result, executionError = cb.executeFailFast(ctx, finalConfig)
	case errors.CollectAll:
		result, executionError = cb.executeCollectAll(ctx, finalConfig)
	default:
		result, executionError = cb.executeFailFast(ctx, finalConfig) // Default to FailFast
	}

	// Update status based on outcome
	if executionError != nil {
		if ctx.Err() != nil {
			cb.SetStatus(types.Cancelled)
		} else {
			cb.SetStatus(types.Completed)
		}
	} else {
		cb.SetStatus(types.Completed)
	}

	return result, executionError
}

// executeFailFast executes all orchestrations concurrently with fail-fast error handling.
func (cb *ConcurrentBuilder) executeFailFast(ctx context.Context, config config.Config) (*result.Result, error) {
	// Create execution context for cancellation
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create error collector for fail-fast strategy
	errorCollector := errors.NewErrorCollector(cb.GetChildCount(), errors.FailFast)

	// Create results storage
	results := make([]*result.Result, cb.GetChildCount())
	var resultsMu sync.RWMutex

	// Create semaphore for concurrency control
	maxConcurrency := cb.getMaxConcurrency(config)
	semaphore := make(chan struct{}, maxConcurrency)

	// Create WaitGroup for synchronization
	var wg sync.WaitGroup

	// Execute all orchestrations concurrently
	children := cb.GetChildren()
	for i, orch := range children {
		wg.Add(1)
		go func(index int, orchestration types.Orchestration) {
			defer wg.Done()

			startTime := time.Now()

			// Acquire semaphore for concurrency control
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-execCtx.Done():
				errorCollector.AddError(index, execCtx.Err(), time.Since(startTime))
				return
			}

			// Execute the orchestration
			childResult, err := orchestration.Execute(execCtx, config)
			duration := time.Since(startTime)

			if err != nil {
				errorCollector.AddError(index, err, duration)
				cancel() // Cancel all other operations on first error
				return
			}

			// Store successful result
			resultsMu.Lock()
			results[index] = childResult
			resultsMu.Unlock()

		}(i, orch)
	}

	// Wait for all goroutines to complete or be cancelled
	wg.Wait()

	// Check for any errors
	if errorCollector.HasErrors() {
		errResult := result.NewResult()
		for _, opErr := range errorCollector.GetAllErrors() {
			errResult.AddError(opErr)
		}
		return errResult, errorCollector.GetFinalError()
	}

	// Combine all successful results
	return cb.combineResults(results), nil
}

// executeCollectAll executes all orchestrations concurrently and collects all errors.
func (cb *ConcurrentBuilder) executeCollectAll(ctx context.Context, config config.Config) (*result.Result, error) {
	// Create error collector for collect-all strategy
	errorCollector := errors.NewErrorCollector(cb.GetChildCount(), errors.CollectAll)

	// Create results storage
	results := make([]*result.Result, cb.GetChildCount())
	var resultsMu sync.RWMutex

	// Create semaphore for concurrency control
	maxConcurrency := cb.getMaxConcurrency(config)
	semaphore := make(chan struct{}, maxConcurrency)

	// Create WaitGroup for synchronization
	var wg sync.WaitGroup

	// Execute all orchestrations concurrently
	children := cb.GetChildren()
	for i, orch := range children {
		wg.Add(1)
		go func(index int, orchestration types.Orchestration) {
			defer wg.Done()

			startTime := time.Now()

			// Acquire semaphore for concurrency control
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				errorCollector.AddError(index, ctx.Err(), time.Since(startTime))
				return
			}

			// Execute the orchestration
			childResult, err := orchestration.Execute(ctx, config)
			duration := time.Since(startTime)

			if err != nil {
				errorCollector.AddError(index, err, duration)
			} else {
				resultsMu.Lock()
				results[index] = childResult
				resultsMu.Unlock()
			}

		}(i, orch)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Always combine results, even if there were errors
	combinedResult := cb.combineResults(results)

	// Return error if any occurred, but still return the partial results
	if errorCollector.HasErrors() {
		// Attach collected errors to the combined result for visibility
		for _, opErr := range errorCollector.GetAllErrors() {
			combinedResult.AddError(opErr)
		}
		return combinedResult, errorCollector.GetFinalError()
	}

	return combinedResult, nil
}

// getMaxConcurrency determines the maximum number of concurrent executions allowed.
func (cb *ConcurrentBuilder) getMaxConcurrency(config config.Config) int {
	if config.MaxConcurrency > 0 {
		return config.MaxConcurrency
	}

	// Default to number of orchestrations, but cap at a reasonable limit
	const defaultMax = 100
	if defaultMax < cb.GetChildCount() {
		return cb.GetChildCount()
	}
	return defaultMax
}

// combineResults merges all child orchestration results into a single result.
func (cb *ConcurrentBuilder) combineResults(childResults []*result.Result) *result.Result {
	combined := result.NewResult()

	for _, childResult := range childResults {
		if childResult == nil {
			continue
		}

		// Merge child results directly
		combined.Merge(childResult)
	}

	return combined
}

// getChildName returns the name of a child orchestration, falling back to index.
func (cb *ConcurrentBuilder) getChildName(orch types.Orchestration, index int) string {
	if name := orch.GetName(); name != "" {
		return name
	}
	return fmt.Sprintf("concurrent_%d", index)
}

// getCurrentPath returns the current orchestration path for hierarchical lookup.
// getCurrentPath removed; path is provided directly via callbacks to BaseContainerOrchestration's resolver

// GetChildNames returns the names of all child orchestrations.
func (cb *ConcurrentBuilder) GetChildNames() []string {
	children := cb.GetChildren()
	names := make([]string, len(children))
	for i, orch := range children {
		names[i] = cb.getChildName(orch, i)
	}
	return names
}

// ListAllPaths returns all available paths in the orchestration hierarchy.
func (cb *ConcurrentBuilder) ListAllPaths() []string {
	return cb.BaseContainerOrchestration.ListAllPaths()
}

// FindByName searches for orchestrations by name in the subtree.
func (cb *ConcurrentBuilder) FindByName(name string) []types.PathMatch {
	return cb.BaseContainerOrchestration.FindByName(name)
}

// Query returns a PathQuery instance for advanced path-based queries.
func (cb *ConcurrentBuilder) Query() *types.PathQuery {
	return cb.BaseContainerOrchestration.Query()
}
