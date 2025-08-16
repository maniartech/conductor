package errors

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewErrorCollector tests error collector creation
func TestNewErrorCollector(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		strategy ErrorStrategy
	}{
		{
			name:     "FailFast collector",
			capacity: 10,
			strategy: FailFast,
		},
		{
			name:     "CollectAll collector",
			capacity: 5,
			strategy: CollectAll,
		},
		{
			name:     "Large capacity",
			capacity: 1000,
			strategy: FailFast,
		},
		{
			name:     "Zero capacity",
			capacity: 0,
			strategy: CollectAll,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewErrorCollector(tt.capacity, tt.strategy)

			if collector == nil {
				t.Fatal("Expected non-nil collector")
			}
			if collector.capacity != tt.capacity {
				t.Errorf("Expected capacity %d, got %d", tt.capacity, collector.capacity)
			}
			if collector.strategy != tt.strategy {
				t.Errorf("Expected strategy %v, got %v", tt.strategy, collector.strategy)
			}
			if len(collector.operationErrors) != tt.capacity {
				t.Errorf("Expected operation errors slice length %d, got %d", tt.capacity, len(collector.operationErrors))
			}
			if collector.GetErrorCount() != 0 {
				t.Errorf("Expected initial error count 0, got %d", collector.GetErrorCount())
			}
			if collector.HasErrors() {
				t.Error("Expected no errors initially")
			}
		})
	}
}

// TestErrorCollector_AddError tests basic error addition
func TestErrorCollector_AddError(t *testing.T) {
	collector := NewErrorCollector(5, FailFast)

	// Test adding valid error
	err1 := errors.New("test error 1")
	collector.AddError(0, err1, time.Millisecond*100)

	if !collector.HasErrors() {
		t.Error("Expected collector to have errors")
	}
	if collector.GetErrorCount() != 1 {
		t.Errorf("Expected error count 1, got %d", collector.GetErrorCount())
	}

	firstErr := collector.GetFirstError()
	if firstErr != err1 {
		t.Errorf("Expected first error %v, got %v", err1, firstErr)
	}

	// Test adding another error
	err2 := errors.New("test error 2")
	collector.AddError(1, err2, time.Millisecond*200)

	if collector.GetErrorCount() != 2 {
		t.Errorf("Expected error count 2, got %d", collector.GetErrorCount())
	}

	// First error should remain the same
	if collector.GetFirstError() != err1 {
		t.Errorf("Expected first error to remain %v, got %v", err1, collector.GetFirstError())
	}
}

// TestErrorCollector_AddError_EdgeCases tests edge cases for error addition
func TestErrorCollector_AddError_EdgeCases(t *testing.T) {
	collector := NewErrorCollector(3, FailFast)

	// Test nil error (should be ignored)
	collector.AddError(0, nil, time.Millisecond)
	if collector.HasErrors() {
		t.Error("Expected no errors when adding nil error")
	}

	// Test negative index (should be ignored)
	collector.AddError(-1, errors.New("negative index"), time.Millisecond)
	if collector.HasErrors() {
		t.Error("Expected no errors when adding error with negative index")
	}

	// Test index out of bounds (should be ignored)
	collector.AddError(5, errors.New("out of bounds"), time.Millisecond)
	if collector.HasErrors() {
		t.Error("Expected no errors when adding error with out of bounds index")
	}

	// Test valid error
	err := errors.New("valid error")
	collector.AddError(1, err, time.Millisecond)
	if !collector.HasErrors() {
		t.Error("Expected collector to have errors after adding valid error")
	}
}

// TestErrorCollector_AddErrorWithOpID tests custom operation ID
func TestErrorCollector_AddErrorWithOpID(t *testing.T) {
	collector := NewErrorCollector(5, CollectAll)

	err := errors.New("test error")
	opID := "custom-operation-id"
	collector.AddErrorWithOpID(0, err, time.Millisecond*150, opID)

	if !collector.HasErrors() {
		t.Error("Expected collector to have errors")
	}

	opErr := collector.GetFirstOperationError()
	if opErr == nil {
		t.Fatal("Expected non-nil operation error")
	}
	if opErr.OpID != opID {
		t.Errorf("Expected OpID %s, got %s", opID, opErr.OpID)
	}
	if opErr.Error != err {
		t.Errorf("Expected error %v, got %v", err, opErr.Error)
	}
	if opErr.Duration != time.Millisecond*150 {
		t.Errorf("Expected duration 150ms, got %v", opErr.Duration)
	}
}

// TestErrorCollector_GetAllErrors tests retrieving all errors
func TestErrorCollector_GetAllErrors(t *testing.T) {
	collector := NewErrorCollector(5, CollectAll)

	// Test empty collector
	allErrors := collector.GetAllErrors()
	if allErrors != nil {
		t.Errorf("Expected nil for empty collector, got %v", allErrors)
	}

	// Add some errors
	testErrors := []error{
		errors.New("error 1"),
		errors.New("error 2"),
		errors.New("error 3"),
	}

	for i, err := range testErrors {
		collector.AddError(i, err, time.Millisecond*time.Duration(i+1))
	}

	allErrors = collector.GetAllErrors()
	if len(allErrors) != len(testErrors) {
		t.Errorf("Expected %d errors, got %d", len(testErrors), len(allErrors))
	}

	for i, opErr := range allErrors {
		if opErr.Error != testErrors[i] {
			t.Errorf("Error %d: expected %v, got %v", i, testErrors[i], opErr.Error)
		}
		if opErr.Index != i {
			t.Errorf("Error %d: expected index %d, got %d", i, i, opErr.Index)
		}
		expectedDuration := time.Millisecond * time.Duration(i+1)
		if opErr.Duration != expectedDuration {
			t.Errorf("Error %d: expected duration %v, got %v", i, expectedDuration, opErr.Duration)
		}
	}
}

// TestErrorCollector_GetFinalError tests final error based on strategy
func TestErrorCollector_GetFinalError(t *testing.T) {
	tests := []struct {
		name     string
		strategy ErrorStrategy
		errors   []error
		wantNil  bool
	}{
		{
			name:     "No errors returns nil",
			strategy: FailFast,
			errors:   []error{},
			wantNil:  true,
		},
		{
			name:     "FailFast returns first error",
			strategy: FailFast,
			errors:   []error{errors.New("first"), errors.New("second")},
			wantNil:  false,
		},
		{
			name:     "CollectAll returns aggregated error",
			strategy: CollectAll,
			errors:   []error{errors.New("first"), errors.New("second")},
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewErrorCollector(10, tt.strategy)

			for i, err := range tt.errors {
				collector.AddError(i, err, time.Millisecond)
			}

			finalErr := collector.GetFinalError()
			if tt.wantNil && finalErr != nil {
				t.Errorf("Expected nil error, got %v", finalErr)
			}
			if !tt.wantNil && finalErr == nil {
				t.Error("Expected non-nil error, got nil")
			}

			if tt.strategy == FailFast && len(tt.errors) > 0 {
				if finalErr != tt.errors[0] {
					t.Errorf("FailFast: expected first error %v, got %v", tt.errors[0], finalErr)
				}
			}
		})
	}
}

// TestErrorCollector_GetAggregatedError tests error aggregation
func TestErrorCollector_GetAggregatedError(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		wantNil  bool
		contains []string
	}{
		{
			name:    "No errors returns nil",
			errors:  []error{},
			wantNil: true,
		},
		{
			name:     "Single error returns that error",
			errors:   []error{errors.New("single error")},
			wantNil:  false,
			contains: []string{"single error"},
		},
		{
			name: "Multiple errors returns aggregated",
			errors: []error{
				errors.New("first error"),
				errors.New("second error"),
			},
			wantNil: false,
			contains: []string{
				"multiple operations failed",
				"first error",
				"second error",
				"operation 0",
				"operation 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewErrorCollector(10, CollectAll)

			for i, err := range tt.errors {
				collector.AddError(i, err, time.Millisecond*time.Duration(i+1))
			}

			aggErr := collector.GetAggregatedError()
			if tt.wantNil && aggErr != nil {
				t.Errorf("Expected nil error, got %v", aggErr)
			}
			if !tt.wantNil && aggErr == nil {
				t.Error("Expected non-nil error, got nil")
			}

			if aggErr != nil {
				errMsg := aggErr.Error()
				for _, expected := range tt.contains {
					if !strings.Contains(errMsg, expected) {
						t.Errorf("Expected error message to contain '%s', got: %s", expected, errMsg)
					}
				}
			}
		})
	}
}

// TestErrorCollector_ShouldStopExecution tests execution stopping logic
func TestErrorCollector_ShouldStopExecution(t *testing.T) {
	tests := []struct {
		name       string
		strategy   ErrorStrategy
		hasErrors  bool
		shouldStop bool
	}{
		{
			name:       "FailFast with no errors should not stop",
			strategy:   FailFast,
			hasErrors:  false,
			shouldStop: false,
		},
		{
			name:       "FailFast with errors should stop",
			strategy:   FailFast,
			hasErrors:  true,
			shouldStop: true,
		},
		{
			name:       "CollectAll with no errors should not stop",
			strategy:   CollectAll,
			hasErrors:  false,
			shouldStop: false,
		},
		{
			name:       "CollectAll with errors should not stop",
			strategy:   CollectAll,
			hasErrors:  true,
			shouldStop: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewErrorCollector(5, tt.strategy)

			if tt.hasErrors {
				collector.AddError(0, errors.New("test error"), time.Millisecond)
			}

			shouldStop := collector.ShouldStopExecution()
			if shouldStop != tt.shouldStop {
				t.Errorf("Expected shouldStop %v, got %v", tt.shouldStop, shouldStop)
			}
		})
	}
}

// TestErrorCollector_Reset tests collector reset functionality
func TestErrorCollector_Reset(t *testing.T) {
	collector := NewErrorCollector(5, CollectAll)

	// Add some errors
	collector.AddError(0, errors.New("error 1"), time.Millisecond)
	collector.AddError(1, errors.New("error 2"), time.Millisecond)

	if !collector.HasErrors() {
		t.Error("Expected collector to have errors before reset")
	}

	// Reset collector
	collector.Reset()

	if collector.HasErrors() {
		t.Error("Expected no errors after reset")
	}
	if collector.GetErrorCount() != 0 {
		t.Errorf("Expected error count 0 after reset, got %d", collector.GetErrorCount())
	}
	if collector.GetFirstError() != nil {
		t.Errorf("Expected nil first error after reset, got %v", collector.GetFirstError())
	}

	allErrors := collector.GetAllErrors()
	if allErrors != nil {
		t.Errorf("Expected nil all errors after reset, got %v", allErrors)
	}
}

// TestErrorCollector_GetErrorSummary tests error summary generation
func TestErrorCollector_GetErrorSummary(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		expected string
		contains []string
	}{
		{
			name:     "No errors",
			errors:   []error{},
			expected: "no errors",
		},
		{
			name: "Multiple errors with timing",
			errors: []error{
				errors.New("error 1"),
				errors.New("error 2"),
				errors.New("error 3"),
			},
			contains: []string{
				"3 errors",
				"min:",
				"max:",
				"avg:",
				"strategy:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewErrorCollector(10, FailFast)

			for i, err := range tt.errors {
				// Add different durations for timing tests
				duration := time.Millisecond * time.Duration((i+1)*10)
				collector.AddError(i, err, duration)
			}

			summary := collector.GetErrorSummary()

			if tt.expected != "" {
				if summary != tt.expected {
					t.Errorf("Expected summary '%s', got '%s'", tt.expected, summary)
				}
			}

			for _, expected := range tt.contains {
				if !strings.Contains(summary, expected) {
					t.Errorf("Expected summary to contain '%s', got: %s", expected, summary)
				}
			}
		})
	}
}

// TestErrorCollector_GetSetStrategy tests strategy getter and setter
func TestErrorCollector_GetSetStrategy(t *testing.T) {
	collector := NewErrorCollector(5, FailFast)

	// Test initial strategy
	if collector.GetStrategy() != FailFast {
		t.Errorf("Expected initial strategy FailFast, got %v", collector.GetStrategy())
	}

	// Test setting new strategy
	collector.SetStrategy(CollectAll)
	if collector.GetStrategy() != CollectAll {
		t.Errorf("Expected strategy CollectAll after setting, got %v", collector.GetStrategy())
	}

	// Test that strategy change affects behavior
	collector.AddError(0, errors.New("test"), time.Millisecond)
	if collector.ShouldStopExecution() {
		t.Error("Expected CollectAll strategy to not stop execution")
	}

	collector.SetStrategy(FailFast)
	if !collector.ShouldStopExecution() {
		t.Error("Expected FailFast strategy to stop execution when errors exist")
	}
}

// TestErrorCollector_ConcurrentAccess tests thread safety
func TestErrorCollector_ConcurrentAccess(t *testing.T) {
	collector := NewErrorCollector(1000, CollectAll)

	var wg sync.WaitGroup
	numGoroutines := 10
	errorsPerGoroutine := 10

	// Add errors concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < errorsPerGoroutine; j++ {
				index := goroutineID*errorsPerGoroutine + j
				err := errors.New("concurrent error")
				collector.AddError(index, err, time.Microsecond*time.Duration(index))
			}
		}(i)
	}

	wg.Wait()

	expectedCount := numGoroutines * errorsPerGoroutine
	if collector.GetErrorCount() != expectedCount {
		t.Errorf("Expected %d errors, got %d", expectedCount, collector.GetErrorCount())
	}

	allErrors := collector.GetAllErrors()
	if len(allErrors) != expectedCount {
		t.Errorf("Expected %d errors in slice, got %d", expectedCount, len(allErrors))
	}
}

// TestErrorCollector_ConcurrentReadWrite tests concurrent read/write operations
func TestErrorCollector_ConcurrentReadWrite(t *testing.T) {
	collector := NewErrorCollector(100, CollectAll)

	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			collector.AddError(i, errors.New("write error"), time.Microsecond)
		}
	}()

	// Reader goroutines - simplified to avoid hanging
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				// Perform read operations
				collector.HasErrors()
				collector.GetErrorCount()
				collector.GetFirstError()
			}
		}()
	}

	wg.Wait()

	// Verify final state
	if !collector.HasErrors() {
		t.Error("Expected collector to have errors after concurrent operations")
	}
}

// TestErrorCollector_MemoryUsage tests memory efficiency
func TestErrorCollector_MemoryUsage(t *testing.T) {
	capacity := 1000
	collector := NewErrorCollector(capacity, CollectAll)

	// Fill collector to capacity
	for i := 0; i < capacity; i++ {
		collector.AddError(i, errors.New("memory test error"), time.Microsecond)
	}

	// Verify all errors are stored
	if collector.GetErrorCount() != capacity {
		t.Errorf("Expected %d errors, got %d", capacity, collector.GetErrorCount())
	}

	allErrors := collector.GetAllErrors()
	if len(allErrors) != capacity {
		t.Errorf("Expected %d errors in slice, got %d", capacity, len(allErrors))
	}

	// Test reset clears memory efficiently
	collector.Reset()
	if collector.HasErrors() {
		t.Error("Expected no errors after reset")
	}

	// Verify we can reuse the collector
	collector.AddError(0, errors.New("reuse test"), time.Microsecond)
	if !collector.HasErrors() {
		t.Error("Expected collector to work after reset")
	}
}

// BenchmarkErrorCollector_AddError benchmarks error addition performance
func BenchmarkErrorCollector_AddError(b *testing.B) {
	collector := NewErrorCollector(b.N, CollectAll)
	err := errors.New("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.AddError(i%1000, err, time.Microsecond)
	}
}

// BenchmarkErrorCollector_HasErrors benchmarks error checking performance
func BenchmarkErrorCollector_HasErrors(b *testing.B) {
	collector := NewErrorCollector(1000, CollectAll)
	collector.AddError(0, errors.New("test error"), time.Microsecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.HasErrors()
	}
}

// BenchmarkErrorCollector_GetErrorCount benchmarks error count retrieval
func BenchmarkErrorCollector_GetErrorCount(b *testing.B) {
	collector := NewErrorCollector(1000, CollectAll)
	collector.AddError(0, errors.New("test error"), time.Microsecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetErrorCount()
	}
}

// BenchmarkErrorCollector_GetAllErrors benchmarks getting all errors
func BenchmarkErrorCollector_GetAllErrors(b *testing.B) {
	collector := NewErrorCollector(100, CollectAll)

	// Add some errors
	for i := 0; i < 50; i++ {
		collector.AddError(i, errors.New("benchmark error"), time.Microsecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetAllErrors()
	}
}

// BenchmarkErrorCollector_ConcurrentAccess benchmarks concurrent access
func BenchmarkErrorCollector_ConcurrentAccess(b *testing.B) {
	collector := NewErrorCollector(b.N, CollectAll)
	err := errors.New("concurrent benchmark error")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			collector.AddError(i%1000, err, time.Microsecond)
			collector.HasErrors()
			i++
		}
	})
}
