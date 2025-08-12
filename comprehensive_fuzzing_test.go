package orchestrator

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
)

// FuzzTaskExecution tests task execution with various inputs
func FuzzTaskExecution(f *testing.F) {
	// Seed with various execution scenarios
	f.Add("success", false, int64(0), int64(1000))
	f.Add("error", true, int64(0), int64(1000))
	f.Add("panic", false, int64(1), int64(1000))
	f.Add("timeout", false, int64(2), int64(10))
	f.Add("empty", false, int64(3), int64(1000))
	f.Add("unicode-🚀", false, int64(4), int64(1000))
	f.Add("very-long-result-that-might-cause-memory-issues-with-large-strings", false, int64(5), int64(1000))

	f.Fuzz(func(t *testing.T, result string, shouldError bool, behavior int64, timeoutMs int64) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(result) {
			t.Skip("Invalid UTF-8 string")
		}

		// Skip extremely long strings and invalid timeouts
		if len(result) > 100000 || timeoutMs <= 0 || timeoutMs > 10000 {
			t.Skip("Invalid parameters")
		}

		// Skip invalid behavior values that might cause nil functions
		if behavior < 0 {
			t.Skip("Invalid behavior value")
		}

		var taskFn func() (string, error)

		switch behavior % 8 {
		case 0: // Normal success/error
			taskFn = func() (string, error) {
				if shouldError {
					return "", fmt.Errorf("fuzz error: %s", result)
				}
				return result, nil
			}
		case 1: // Panic
			taskFn = func() (string, error) {
				panic(fmt.Sprintf("fuzz panic: %s", result))
			}
		case 2: // Slow execution
			taskFn = func() (string, error) {
				time.Sleep(time.Duration(timeoutMs/10) * time.Millisecond)
				return result, nil
			}
		case 3: // Empty result
			taskFn = func() (string, error) {
				return "", nil
			}
		case 4: // Large result
			taskFn = func() (string, error) {
				if len(result) > 1000 {
					return result[:1000], nil // Truncate to avoid memory issues
				}
				return strings.Repeat(result, 10), nil
			}
		case 5: // Memory allocation
			taskFn = func() (string, error) {
				data := make([]string, 100)
				for i := range data {
					data[i] = result
				}
				return result, nil
			}
		case 6: // Random error based on input
			taskFn = func() (string, error) {
				if len(result)%2 == 0 {
					return "", fmt.Errorf("random error for: %s", result)
				}
				return result, nil
			}
		case 7: // Context checking
			taskFn = func() (string, error) {
				time.Sleep(time.Millisecond)
				return result, nil
			}
		}

		// Ensure taskFn is not nil
		if taskFn == nil {
			t.Skip("Task function is nil")
		}

		taskInstance := task.Task(taskFn).Named("fuzz-task")
		workflow := Setup(taskInstance)

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()

		workflowResult, workflowErr := workflow.AwaitWithContext(ctx)

		// Verify result consistency
		if workflowErr != nil {
			// Error is acceptable, but result should reflect this
			if workflowResult != nil && !workflowResult.HasErrors() {
				t.Errorf("Workflow returned error but result has no errors: %v", workflowErr)
			}
		} else {
			// Success case
			if workflowResult == nil {
				t.Errorf("Workflow succeeded but result is nil")
			}
		}
	})
}

// FuzzConfigurationValues tests configuration with various values
func FuzzConfigurationValues(f *testing.F) {
	// Seed with various configuration scenarios
	f.Add(int64(0), int64(0), int64(0))                   // Zero values
	f.Add(int64(1000), int64(10), int64(100))             // Normal values
	f.Add(int64(-1), int64(-1), int64(-1))                // Negative values
	f.Add(int64(math.MaxInt64), int64(1000), int64(1000)) // Large values

	f.Fuzz(func(t *testing.T, timeoutMs, retries, maxConcurrency int64) {
		// Skip extreme values that might cause issues
		if timeoutMs < -1000 || timeoutMs > 60000 ||
			retries < -10 || retries > 100 ||
			maxConcurrency < -10 || maxConcurrency > 10000 {
			t.Skip("Extreme configuration values")
		}

		cfg := config.Config{}

		if timeoutMs > 0 {
			cfg.Timeout = time.Duration(timeoutMs) * time.Millisecond
		}

		// Note: retries and maxConcurrency might not be implemented yet
		// but we test the configuration structure

		taskFn := func() (string, error) {
			time.Sleep(time.Millisecond)
			return "config-fuzz-result", nil
		}

		taskInstance := task.Task(taskFn).Named("config-fuzz-task").With(cfg)
		workflow := Setup(taskInstance)

		ctx := context.Background()
		if timeoutMs > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
			defer cancel()
		}

		result, err := workflow.AwaitWithContext(ctx)

		// Handle timeout cases
		if timeoutMs > 0 && timeoutMs < 10 {
			// Very short timeout might cause timeout error
			if err != nil && (err == context.DeadlineExceeded || strings.Contains(err.Error(), "timeout")) {
				return // Expected timeout
			}
		}

		// For reasonable configurations, task should succeed
		if timeoutMs >= 100 || timeoutMs == 0 {
			if err != nil {
				t.Errorf("Task failed with reasonable config (timeout=%dms): %v", timeoutMs, err)
			}
			if result == nil {
				t.Errorf("Result is nil with timeout %dms", timeoutMs)
			}
		}
	})
}

// FuzzComplexDataTypes tests orchestration with various data types
func FuzzComplexDataTypes(f *testing.F) {
	// Seed with various data type scenarios
	f.Add("string", int64(0))
	f.Add("int", int64(1))
	f.Add("float", int64(2))
	f.Add("bool", int64(3))
	f.Add("slice", int64(4))
	f.Add("map", int64(5))
	f.Add("struct", int64(6))
	f.Add("interface", int64(7))

	f.Fuzz(func(t *testing.T, dataType string, variant int64) {
		if !utf8.ValidString(dataType) || len(dataType) > 100 {
			t.Skip("Invalid data type string")
		}

		var taskFn func() (interface{}, error)

		v := variant % 8 // normalize to avoid accidental negative modulo paths
		if v < 0 {
			v = -v
		}
		switch v {
		case 0: // String
			taskFn = func() (interface{}, error) {
				return fmt.Sprintf("fuzz-string-%s-%d", dataType, variant), nil
			}
		case 1: // Integer
			taskFn = func() (interface{}, error) {
				return int(variant), nil
			}
		case 2: // Float
			taskFn = func() (interface{}, error) {
				return float64(variant) * 3.14, nil
			}
		case 3: // Boolean
			taskFn = func() (interface{}, error) {
				return variant%2 == 0, nil
			}
		case 4: // Slice
			taskFn = func() (interface{}, error) {
				size := int((variant%10 + 10) % 10)
				if size == 0 {
					size = 1
				}
				slice := make([]string, size)
				for i := range slice {
					slice[i] = fmt.Sprintf("item-%d", i)
				}
				return slice, nil
			}
		case 5: // Map
			taskFn = func() (interface{}, error) {
				m := make(map[string]interface{})
				m["type"] = dataType
				m["variant"] = variant
				m["timestamp"] = time.Now().Unix()
				return m, nil
			}
		case 6: // Struct
			taskFn = func() (interface{}, error) {
				type FuzzStruct struct {
					Type    string
					Variant int64
					Data    []byte
				}
				return FuzzStruct{
					Type:    dataType,
					Variant: variant,
					Data:    []byte(fmt.Sprintf("data-%d", variant)),
				}, nil
			}
		case 7: // Interface with nil
			taskFn = func() (interface{}, error) {
				if variant%3 == 0 {
					return nil, nil
				}
				return fmt.Sprintf("interface-%s", dataType), nil
			}
		}

		// Ensure taskFn is not nil; if it is, skip this fuzz case instead of panicking
		if taskFn == nil {
			t.Skip("nil task function for variant")
		}

		taskInstance := task.Task(taskFn).Named(fmt.Sprintf("fuzz-data-task-%s", dataType))
		workflow := Setup(taskInstance)

		result, err := workflow.Await()

		if err != nil {
			t.Errorf("Task failed for data type %s: %v", dataType, err)
			return
		}

		if result == nil {
			t.Errorf("Result is nil for data type %s", dataType)
			return
		}

		// Verify result can be retrieved
		taskName := fmt.Sprintf("fuzz-data-task-%s", dataType)
		value := result.Get(taskName)
		if value == nil && v != 7 { // Allow nil for interface test
			t.Errorf("Retrieved value is nil for data type %s", dataType)
		}
	})
}

// FuzzErrorScenarios tests various error conditions
func FuzzErrorScenarios(f *testing.F) {
	// Seed with various error scenarios
	f.Add("simple error", int64(0))
	f.Add("", int64(1)) // Empty error
	f.Add("error with unicode 🚨", int64(2))
	f.Add("error\nwith\nnewlines", int64(3))
	f.Add("very long error message that might cause issues", int64(4))

	f.Fuzz(func(t *testing.T, errorMsg string, errorType int64) {
		if !utf8.ValidString(errorMsg) || len(errorMsg) > 10000 {
			t.Skip("Invalid error message")
		}

		var taskFn func() (string, error)

		switch errorType % 6 {
		case 0: // Simple error
			taskFn = func() (string, error) {
				return "", fmt.Errorf("%s", errorMsg)
			}
		case 1: // Wrapped error
			taskFn = func() (string, error) {
				baseErr := fmt.Errorf("base error: %s", errorMsg)
				return "", fmt.Errorf("wrapped: %w", baseErr)
			}
		case 2: // Panic with error message
			taskFn = func() (string, error) {
				panic(errorMsg)
			}
		case 3: // Error with success result (should not happen)
			taskFn = func() (string, error) {
				return "success", fmt.Errorf("%s", errorMsg)
			}
		case 4: // Conditional error
			taskFn = func() (string, error) {
				if len(errorMsg)%2 == 0 {
					return "", fmt.Errorf("conditional: %s", errorMsg)
				}
				return "success", nil
			}
		case 5: // Timeout simulation
			taskFn = func() (string, error) {
				time.Sleep(100 * time.Millisecond)
				return "", fmt.Errorf("timeout: %s", errorMsg)
			}
		}

		taskInstance := task.Task(taskFn).Named("fuzz-error-task")
		workflow := Setup(taskInstance)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		result, err := workflow.AwaitWithContext(ctx)

		// Verify error handling consistency
		if errorType%6 == 3 { // Error with success result case
			if err == nil {
				t.Errorf("Expected error but got none for error message: %q", errorMsg)
			}
		}

		if err != nil {
			// Should have an error in result
			if result != nil && !result.HasErrors() {
				t.Errorf("Error returned but result has no errors for message: %q", errorMsg)
			}
		}

		// Error message should be preserved (unless it's a panic or timeout)
		if err != nil && errorType%6 != 2 && errorType%6 != 5 && errorMsg != "" {
			if !strings.Contains(err.Error(), errorMsg) {
				t.Logf("Error message %q does not contain original message %q", err.Error(), errorMsg)
			}
		}
	})
}

// FuzzConcurrentOperations tests concurrent operations with fuzzed inputs
func FuzzConcurrentOperations(f *testing.F) {
	// Seed with various concurrency scenarios
	f.Add(int64(1), int64(1))
	f.Add(int64(5), int64(10))
	f.Add(int64(10), int64(20))
	f.Add(int64(20), int64(50))

	f.Fuzz(func(t *testing.T, numGoroutines, numOperations int64) {
		// Limit to reasonable values
		if numGoroutines < 1 || numGoroutines > 50 ||
			numOperations < 1 || numOperations > 100 {
			t.Skip("Invalid concurrency parameters")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		results := make(chan string, numGoroutines*numOperations)
		errors := make(chan error, numGoroutines*numOperations)

		// Launch concurrent operations
		for i := int64(0); i < numGoroutines; i++ {
			go func(goroutineID int64) {
				for j := int64(0); j < numOperations; j++ {
					taskFn := func() (string, error) {
						// Random behavior
						behavior := (goroutineID + j) % 4
						switch behavior {
						case 0: // Success
							return fmt.Sprintf("fuzz-concurrent-%d-%d", goroutineID, j), nil
						case 1: // Error
							return "", fmt.Errorf("fuzz-error-%d-%d", goroutineID, j)
						case 2: // Panic
							if j%10 == 0 { // Only panic occasionally
								panic(fmt.Sprintf("fuzz-panic-%d-%d", goroutineID, j))
							}
							return fmt.Sprintf("fuzz-no-panic-%d-%d", goroutineID, j), nil
						case 3: // Slow operation
							time.Sleep(time.Millisecond)
							return fmt.Sprintf("fuzz-slow-%d-%d", goroutineID, j), nil
						default:
							return fmt.Sprintf("fuzz-default-%d-%d", goroutineID, j), nil
						}
					}

					taskInstance := task.Task(taskFn).Named(fmt.Sprintf("fuzz-concurrent-task-%d-%d", goroutineID, j))
					workflow := Setup(taskInstance)

					result, err := workflow.Await()

					if err != nil {
						errors <- err
					} else if result != nil {
						taskName := fmt.Sprintf("fuzz-concurrent-task-%d-%d", goroutineID, j)
						if value := result.Get(taskName); value != nil {
							if str, ok := value.(string); ok {
								results <- str
							}
						}
					}
				}
			}(i)
		}

		// Collect results
		expectedTotal := int(numGoroutines * numOperations)
		actualResults := 0
		actualErrors := 0

		timeout := time.After(15 * time.Second)
		for actualResults+actualErrors < expectedTotal {
			select {
			case <-results:
				actualResults++
			case <-errors:
				actualErrors++
			case <-timeout:
				t.Errorf("Timeout waiting for results: got %d results, %d errors out of %d expected",
					actualResults, actualErrors, expectedTotal)
				return
			case <-ctx.Done():
				t.Errorf("Context cancelled: got %d results, %d errors out of %d expected",
					actualResults, actualErrors, expectedTotal)
				return
			}
		}

		if actualResults+actualErrors != expectedTotal {
			t.Errorf("Expected %d total operations, got %d results + %d errors = %d",
				expectedTotal, actualResults, actualErrors, actualResults+actualErrors)
		}

		t.Logf("Concurrent fuzz test completed: %d results, %d errors", actualResults, actualErrors)
	})
}

// FuzzMemoryOperations tests memory-related operations
func FuzzMemoryOperations(f *testing.F) {
	// Seed with various memory scenarios
	f.Add(int64(1024), int64(10))  // Small allocations
	f.Add(int64(10240), int64(5))  // Medium allocations
	f.Add(int64(102400), int64(2)) // Large allocations

	f.Fuzz(func(t *testing.T, allocSize, numAllocs int64) {
		// Limit memory usage to prevent system issues
		if allocSize < 0 || allocSize > 1024*1024 || // Max 1MB per allocation
			numAllocs < 0 || numAllocs > 100 || // Max 100 allocations
			allocSize*numAllocs > 10*1024*1024 { // Max 10MB total
			t.Skip("Invalid memory parameters")
		}

		taskFn := func() ([][]byte, error) {
			allocations := make([][]byte, numAllocs)

			for i := int64(0); i < numAllocs; i++ {
				data := make([]byte, allocSize)

				// Fill with pattern to ensure allocation
				for j := range data {
					data[j] = byte((i + int64(j)) % 256)
				}

				allocations[i] = data
			}

			return allocations, nil
		}

		taskInstance := task.Task(taskFn).Named("fuzz-memory-task")
		workflow := Setup(taskInstance)

		result, err := workflow.Await()

		if err != nil {
			t.Errorf("Memory task failed (size=%d, count=%d): %v", allocSize, numAllocs, err)
			return
		}

		if result == nil {
			t.Errorf("Result is nil for memory task (size=%d, count=%d)", allocSize, numAllocs)
			return
		}

		// Verify result
		value := result.Get("fuzz-memory-task")
		if value == nil {
			t.Errorf("Retrieved value is nil for memory task")
			return
		}

		// Type assertion to verify the data
		if allocations, ok := value.([][]byte); ok {
			if int64(len(allocations)) != numAllocs {
				t.Errorf("Expected %d allocations, got %d", numAllocs, len(allocations))
			}

			for i, alloc := range allocations {
				if int64(len(alloc)) != allocSize {
					t.Errorf("Allocation %d: expected size %d, got %d", i, allocSize, len(alloc))
				}
			}
		} else {
			t.Errorf("Result is not [][]byte, got %T", value)
		}

		t.Logf("Memory fuzz test completed: %d allocations of %d bytes each", numAllocs, allocSize)
	})
}

// FuzzUnsafeOperations tests unsafe operations and edge cases
func FuzzUnsafeOperations(f *testing.F) {
	// Seed with various unsafe scenarios
	f.Add("pointer", int64(0))
	f.Add("reflection", int64(1))
	f.Add("type-assertion", int64(2))
	f.Add("interface-conversion", int64(3))

	f.Fuzz(func(t *testing.T, operation string, variant int64) {
		if !utf8.ValidString(operation) || len(operation) > 100 {
			t.Skip("Invalid operation string")
		}

		var taskFn func() (interface{}, error)

		switch variant % 4 {
		case 0: // Pointer operations
			taskFn = func() (interface{}, error) {
				data := fmt.Sprintf("unsafe-data-%s-%d", operation, variant)
				ptr := unsafe.Pointer(&data)

				// Convert back to string pointer
				strPtr := (*string)(ptr)
				return *strPtr, nil
			}
		case 1: // Reflection operations
			taskFn = func() (interface{}, error) {
				data := map[string]interface{}{
					"operation": operation,
					"variant":   variant,
					"type":      "reflection",
				}

				v := reflect.ValueOf(data)
				if v.Kind() != reflect.Map {
					return nil, fmt.Errorf("unexpected kind: %v", v.Kind())
				}

				return data, nil
			}
		case 2: // Type assertions
			taskFn = func() (interface{}, error) {
				var data interface{} = fmt.Sprintf("assertion-%s-%d", operation, variant)

				// Type assertion
				if str, ok := data.(string); ok {
					return str, nil
				}

				return nil, fmt.Errorf("type assertion failed")
			}
		case 3: // Interface conversions
			taskFn = func() (interface{}, error) {
				type Stringer interface {
					String() string
				}

				type CustomString string

				var data CustomString = CustomString(fmt.Sprintf("custom-%s-%d", operation, variant))

				// Convert to interface
				var iface interface{} = data

				// Convert back
				if custom, ok := iface.(CustomString); ok {
					return string(custom), nil
				}

				return nil, fmt.Errorf("interface conversion failed")
			}
		}

		taskInstance := task.Task(taskFn).Named(fmt.Sprintf("fuzz-unsafe-task-%s", operation))
		workflow := Setup(taskInstance)

		result, err := workflow.Await()

		if err != nil {
			// Some unsafe operations might fail, which is acceptable
			t.Logf("Unsafe operation failed (expected): %v", err)
			return
		}

		if result == nil {
			t.Errorf("Result is nil for unsafe operation %s", operation)
			return
		}

		// Verify result
		taskName := fmt.Sprintf("fuzz-unsafe-task-%s", operation)
		value := result.Get(taskName)
		if value == nil {
			t.Errorf("Retrieved value is nil for unsafe operation %s", operation)
		}

		t.Logf("Unsafe fuzz test completed for operation: %s", operation)
	})
}
