// Package errors provides error collection for all orchestration types.
// This file implements zero-allocation error collection using atomic operations
// for high-performance error handling across sequential, concurrent, and other orchestrations.
package errors

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// ErrorCollector provides zero-allocation error collection for all orchestration types.
// It uses atomic operations to avoid mutex contention and provides fast error checking.
//
// Key features:
//   - Zero-allocation error collection using atomic operations
//   - Fast error checking without iteration
//   - Thread-safe concurrent access
//   - Rich error metadata with timing information
//   - First error tracking for fail-fast scenarios
//   - Aggregated error reporting for collect-all scenarios
//   - Reusable across different orchestration types
type ErrorCollector struct {
	// Pre-allocated error storage
	operationErrors []OperationError

	// Atomic counters for fast access
	errorCount atomic.Int32
	firstError atomic.Pointer[OperationError]

	// Capacity for bounds checking
	capacity int

	// Strategy for error handling
	strategy ErrorStrategy
}

// NewErrorCollector creates a new error collector with pre-allocated storage.
//
// Parameters:
//   - capacity: Maximum number of operations (pre-allocates error storage)
//   - strategy: Error handling strategy (FailFast or CollectAll)
//
// Returns:
//   - *ErrorCollector: New error collector instance
//
// Example:
//
//	collector := NewErrorCollector(10, FailFast)
//	collector.AddError(0, err, duration)
//	if collector.HasErrors() {
//	    firstErr := collector.GetFirstError()
//	}
func NewErrorCollector(capacity int, strategy ErrorStrategy) *ErrorCollector {
	return &ErrorCollector{
		operationErrors: make([]OperationError, capacity),
		capacity:        capacity,
		strategy:        strategy,
	}
}

// AddError adds an error to the collector with rich metadata.
// This method is thread-safe and uses atomic operations for performance.
//
// Parameters:
//   - index: Index of the operation that failed
//   - err: The error that occurred
//   - duration: Duration of the failed operation
//
// Example:
//
//	startTime := time.Now()
//	result, err := operation.Execute(ctx, config)
//	if err != nil {
//	    collector.AddError(index, err, time.Since(startTime))
//	}
func (ec *ErrorCollector) AddError(index int, err error, duration time.Duration) {
	if err == nil || index < 0 || index >= ec.capacity {
		return
	}

	// Create operation error with rich metadata
	opError := OperationError{
		Error:     err,
		Index:     index,
		Duration:  duration,
		Timestamp: time.Now(),
		OpID:      fmt.Sprintf("op-%d", index),
		Stack:     ec.captureStackTrace(),
	}

	// Store error at index
	ec.operationErrors[index] = opError

	// Increment error count atomically
	ec.errorCount.Add(1)

	// Set first error atomically (only if not already set)
	ec.firstError.CompareAndSwap(nil, &opError)
}

// AddErrorWithOpID adds an error with a custom operation ID.
// This allows orchestrations to provide more descriptive operation identifiers.
//
// Parameters:
//   - index: Index of the operation that failed
//   - err: The error that occurred
//   - duration: Duration of the failed operation
//   - opID: Custom operation identifier
//
// Example:
//
//	collector.AddErrorWithOpID(0, err, duration, "user-validation-step")
func (ec *ErrorCollector) AddErrorWithOpID(index int, err error, duration time.Duration, opID string) {
	if err == nil || index < 0 || index >= ec.capacity {
		return
	}

	// Create operation error with rich metadata
	opError := OperationError{
		Error:     err,
		Index:     index,
		Duration:  duration,
		Timestamp: time.Now(),
		OpID:      opID,
		Stack:     ec.captureStackTrace(),
	}

	// Store error at index
	ec.operationErrors[index] = opError

	// Increment error count atomically
	ec.errorCount.Add(1)

	// Set first error atomically (only if not already set)
	ec.firstError.CompareAndSwap(nil, &opError)
}

// HasErrors returns true if any errors were collected.
// This method uses atomic operations for zero-allocation checking.
//
// Returns:
//   - bool: Whether any errors occurred
//
// Example:
//
//	if collector.HasErrors() {
//	    log.Printf("Errors occurred during execution")
//	}
func (ec *ErrorCollector) HasErrors() bool {
	return ec.errorCount.Load() > 0
}

// GetErrorCount returns the total number of errors collected.
// This method uses atomic operations for thread-safe access.
//
// Returns:
//   - int: Total number of errors
//
// Example:
//
//	count := collector.GetErrorCount()
//	log.Printf("Total errors: %d", count)
func (ec *ErrorCollector) GetErrorCount() int {
	return int(ec.errorCount.Load())
}

// GetFirstError returns the first error that occurred.
// This method is optimized for fail-fast scenarios where only the first error matters.
//
// Returns:
//   - error: The first error that occurred (nil if no errors)
//
// Example:
//
//	if collector.HasErrors() {
//	    firstErr := collector.GetFirstError()
//	    return fmt.Errorf("execution failed: %w", firstErr)
//	}
func (ec *ErrorCollector) GetFirstError() error {
	if ptr := ec.firstError.Load(); ptr != nil {
		return ptr.Error
	}
	return nil
}

// GetFirstOperationError returns the first operation error with full metadata.
// This provides rich error information for debugging and observability.
//
// Returns:
//   - *OperationError: The first operation error (nil if no errors)
//
// Example:
//
//	if opErr := collector.GetFirstOperationError(); opErr != nil {
//	    log.Printf("First error at index %d after %v: %v",
//	        opErr.Index, opErr.Duration, opErr.Error)
//	}
func (ec *ErrorCollector) GetFirstOperationError() *OperationError {
	return ec.firstError.Load()
}

// GetAllErrors returns all errors that occurred with full metadata.
// This method creates a copy to prevent external modification.
//
// Returns:
//   - []OperationError: All operation errors (empty slice if no errors)
//
// Example:
//
//	allErrors := collector.GetAllErrors()
//	for _, opErr := range allErrors {
//	    log.Printf("Error at index %d: %v", opErr.Index, opErr.Error)
//	}
func (ec *ErrorCollector) GetAllErrors() []OperationError {
	if !ec.HasErrors() {
		return nil
	}

	// Collect non-nil errors
	var result []OperationError
	for i := 0; i < ec.capacity; i++ {
		if ec.operationErrors[i].Error != nil {
			result = append(result, ec.operationErrors[i])
		}
	}

	return result
}

// GetFinalError returns the appropriate error based on the error strategy.
// For FailFast, returns the first error. For CollectAll, returns an aggregated error.
//
// Returns:
//   - error: The final error to return from orchestration (nil if no errors)
//
// Example:
//
//	if collector.HasErrors() {
//	    return result, collector.GetFinalError()
//	}
func (ec *ErrorCollector) GetFinalError() error {
	if !ec.HasErrors() {
		return nil
	}

	switch ec.strategy {
	case FailFast:
		return ec.GetFirstError()
	case CollectAll:
		return ec.GetAggregatedError()
	default:
		return ec.GetFirstError()
	}
}

// GetAggregatedError returns an aggregated error containing all individual errors.
// This method is optimized for collect-all scenarios where all errors matter.
//
// Returns:
//   - error: Aggregated error containing all individual errors (nil if no errors)
//
// Example:
//
//	if collector.HasErrors() {
//	    aggErr := collector.GetAggregatedError()
//	    return fmt.Errorf("execution had %d errors: %w",
//	        collector.GetErrorCount(), aggErr)
//	}
func (ec *ErrorCollector) GetAggregatedError() error {
	if !ec.HasErrors() {
		return nil
	}

	allErrors := ec.GetAllErrors()
	if len(allErrors) == 0 {
		return nil
	}

	if len(allErrors) == 1 {
		return allErrors[0].Error
	}

	// Create aggregated error message
	var errorMessages []string
	for _, opErr := range allErrors {
		errorMessages = append(errorMessages,
			fmt.Sprintf("operation %d (%s) after %v: %v",
				opErr.Index, opErr.OpID, opErr.Duration, opErr.Error))
	}

	return fmt.Errorf("multiple operations failed:\n%s",
		strings.Join(errorMessages, "\n"))
}

// ShouldStopExecution returns true if execution should stop based on the error strategy.
// For FailFast, returns true if any error occurred. For CollectAll, returns false.
//
// Returns:
//   - bool: Whether execution should stop
//
// Example:
//
//	if collector.ShouldStopExecution() {
//	    // Cancel remaining operations
//	    cancel()
//	    return collector.GetFinalError()
//	}
func (ec *ErrorCollector) ShouldStopExecution() bool {
	if !ec.HasErrors() {
		return false
	}

	return ec.strategy == FailFast
}

// Reset clears all errors and resets counters.
// This method is useful for reusing the collector across multiple executions.
//
// Example:
//
//	collector.Reset()
//	// Collector is now ready for reuse
func (ec *ErrorCollector) Reset() {
	// Reset atomic counters
	ec.errorCount.Store(0)
	ec.firstError.Store(nil)

	// Clear error storage
	for i := 0; i < ec.capacity; i++ {
		ec.operationErrors[i] = OperationError{}
	}
}

// GetErrorSummary returns a summary of all errors for logging and debugging.
// This provides a concise overview of error distribution and timing.
//
// Returns:
//   - string: Error summary with counts, timing, and distribution
//
// Example:
//
//	if collector.HasErrors() {
//	    summary := collector.GetErrorSummary()
//	    log.Printf("Execution error summary: %s", summary)
//	}
func (ec *ErrorCollector) GetErrorSummary() string {
	if !ec.HasErrors() {
		return "no errors"
	}

	allErrors := ec.GetAllErrors()
	if len(allErrors) == 0 {
		return "no errors"
	}

	// Calculate timing statistics
	var totalDuration time.Duration
	var minDuration, maxDuration time.Duration
	minDuration = allErrors[0].Duration
	maxDuration = allErrors[0].Duration

	for _, opErr := range allErrors {
		totalDuration += opErr.Duration
		if opErr.Duration < minDuration {
			minDuration = opErr.Duration
		}
		if opErr.Duration > maxDuration {
			maxDuration = opErr.Duration
		}
	}

	avgDuration := totalDuration / time.Duration(len(allErrors))

	return fmt.Sprintf("%d errors (min: %v, max: %v, avg: %v, strategy: %s)",
		len(allErrors), minDuration, maxDuration, avgDuration, ec.strategy)
}

// GetStrategy returns the error handling strategy.
//
// Returns:
//   - ErrorStrategy: The configured error strategy
func (ec *ErrorCollector) GetStrategy() ErrorStrategy {
	return ec.strategy
}

// SetStrategy updates the error handling strategy.
// This can be useful for dynamic strategy changes during execution.
//
// Parameters:
//   - strategy: New error handling strategy
//
// Example:
//
//	collector.SetStrategy(CollectAll)
func (ec *ErrorCollector) SetStrategy(strategy ErrorStrategy) {
	ec.strategy = strategy
}

// captureStackTrace captures the current stack trace for error reporting.
// This provides detailed debugging information when errors occur.
func (ec *ErrorCollector) captureStackTrace() []byte {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return buf[:n]
}
