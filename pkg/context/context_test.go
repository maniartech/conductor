package context

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/config"
)

// TestBasicContextOperations tests the core context functionality
func TestBasicContextOperations(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)

	t.Run("Get and Set operations", func(t *testing.T) {
		// Test setting and getting values
		ctx.Set("key1", "value1")
		ctx.Set("key2", 42)
		ctx.Set("key3", true)

		if got := ctx.Get("key1"); got != "value1" {
			t.Errorf("Expected 'value1', got %v", got)
		}

		if got := ctx.Get("key2"); got != 42 {
			t.Errorf("Expected 42, got %v", got)
		}

		if got := ctx.Get("key3"); got != true {
			t.Errorf("Expected true, got %v", got)
		}

		// Test getting non-existent key
		if got := ctx.Get("nonexistent"); got != nil {
			t.Errorf("Expected nil for non-existent key, got %v", got)
		}
	})

	t.Run("Config access", func(t *testing.T) {
		config := ctx.Config()
		if config.Timeout != cfg.Timeout {
			t.Errorf("Expected timeout %v, got %v", cfg.Timeout, config.Timeout)
		}
	})

	t.Run("Cancellation", func(t *testing.T) {
		// Test that Done() channel is not closed initially
		select {
		case <-ctx.Done():
			t.Error("Context should not be cancelled initially")
		default:
			// Expected
		}

		// Test cancellation
		ctx.Cancel()

		// Test that Done() channel is closed after cancellation
		select {
		case <-ctx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Context should be cancelled")
		}

		// Test error after cancellation
		if err := ctx.Err(); err == nil {
			t.Error("Expected error after cancellation")
		}
	})
}

// TestEnhancedTimeoutManagement tests the enhanced timeout functionality
func TestEnhancedTimeoutManagement(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)

	t.Run("WithTimeout creates context with timeout", func(t *testing.T) {
		ctxWithTimeout := ctx.WithTimeout(100 * time.Millisecond)

		// Should not be expired initially
		if ctxWithTimeout.IsExpired() {
			t.Error("Context should not be expired initially")
		}

		// Should have remaining time
		remaining := ctxWithTimeout.GetRemainingTime()
		if remaining <= 0 || remaining > 100*time.Millisecond {
			t.Errorf("Expected remaining time between 0 and 100ms, got: %v", remaining)
		}

		// Wait for timeout
		time.Sleep(150 * time.Millisecond)

		// Should be expired now
		if !ctxWithTimeout.IsExpired() {
			t.Error("Context should be expired after timeout")
		}

		// Remaining time should be 0
		if ctxWithTimeout.GetRemainingTime() != 0 {
			t.Error("Remaining time should be 0 for expired context")
		}
	})

	t.Run("WithDeadline creates context with deadline", func(t *testing.T) {
		deadline := time.Now().Add(100 * time.Millisecond)
		ctxWithDeadline := ctx.WithDeadline(deadline)

		// Should not be expired initially
		if ctxWithDeadline.IsExpired() {
			t.Error("Context should not be expired initially")
		}

		// Wait for deadline
		time.Sleep(150 * time.Millisecond)

		// Should be expired now
		if !ctxWithDeadline.IsExpired() {
			t.Error("Context should be expired after deadline")
		}
	})

	t.Run("Context without timeout", func(t *testing.T) {
		// Context without timeout should never expire
		if ctx.IsExpired() {
			t.Error("Context without timeout should not be expired")
		}

		if ctx.GetRemainingTime() != 0 {
			t.Error("Context without timeout should have 0 remaining time")
		}
	})

	t.Run("Timeout cancellation integration", func(t *testing.T) {
		ctxWithTimeout := ctx.WithTimeout(50 * time.Millisecond)

		// Wait for timeout to trigger cancellation
		select {
		case <-ctxWithTimeout.Done():
			// Expected - timeout should trigger cancellation
		case <-time.After(100 * time.Millisecond):
			t.Error("Context should be cancelled by timeout")
		}

		// Should be expired
		if !ctxWithTimeout.IsExpired() {
			t.Error("Context should be expired after timeout cancellation")
		}
	})
}

// TestNestedOrchestrationLifecycle tests the nested context functionality
func TestNestedOrchestrationLifecycle(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)

	t.Run("CreateChild creates child context", func(t *testing.T) {
		childCtx := ctx.CreateChild("test-child")

		// Child should have correct path
		expectedPath := "/test-child"
		if childCtx.GetPath() != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, childCtx.GetPath())
		}

		// Child should have parent
		parent := childCtx.GetParent()
		if parent == nil {
			t.Error("Child should have parent")
		}

		// Root context should not have parent
		if ctx.GetParent() != nil {
			t.Error("Root context should not have parent")
		}
	})

	t.Run("Nested hierarchy", func(t *testing.T) {
		// Create nested hierarchy
		childCtx := ctx.CreateChild("level1")
		grandchildCtx := childCtx.CreateChild("level2")
		greatGrandchildCtx := grandchildCtx.CreateChild("level3")

		// Test paths
		if childCtx.GetPath() != "/level1" {
			t.Errorf("Expected path /level1, got %s", childCtx.GetPath())
		}

		if grandchildCtx.GetPath() != "/level1/level2" {
			t.Errorf("Expected path /level1/level2, got %s", grandchildCtx.GetPath())
		}

		if greatGrandchildCtx.GetPath() != "/level1/level2/level3" {
			t.Errorf("Expected path /level1/level2/level3, got %s", greatGrandchildCtx.GetPath())
		}

		// Test parent relationships
		if grandchildCtx.GetParent() != childCtx {
			t.Error("Grandchild parent should be child")
		}

		if greatGrandchildCtx.GetParent() != grandchildCtx {
			t.Error("Great-grandchild parent should be grandchild")
		}
	})

	t.Run("Child inherits parent configuration", func(t *testing.T) {
		// Set value in parent
		ctx.Set("parent_value", "test")

		// Create child
		childCtx := ctx.CreateChild("test-child")

		// Child should inherit parent's configuration
		parentConfig := ctx.Config()
		childConfig := childCtx.Config()

		if childConfig.Timeout != parentConfig.Timeout {
			t.Error("Child should inherit parent timeout configuration")
		}

		// Child should have its own value storage
		childCtx.Set("child_value", "child_test")

		// Parent should not have child's value
		if ctx.Get("child_value") != nil {
			t.Error("Parent should not have child's value")
		}

		// Child should not have parent's value (separate storage)
		if childCtx.Get("parent_value") != nil {
			t.Error("Child should have separate value storage from parent")
		}
	})

	t.Run("Cancellation propagation", func(t *testing.T) {
		childCtx := ctx.CreateChild("test-child")
		grandchildCtx := childCtx.CreateChild("test-grandchild")

		// Cancel parent
		ctx.Cancel()

		// Children should be cancelled too
		select {
		case <-childCtx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Child should be cancelled when parent is cancelled")
		}

		select {
		case <-grandchildCtx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Grandchild should be cancelled when parent is cancelled")
		}
	})
}

// TestGracefulShutdownCoordination tests the shutdown functionality
func TestGracefulShutdownCoordination(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)

	t.Run("IsShuttingDown returns false initially", func(t *testing.T) {
		if ctx.IsShuttingDown() {
			t.Error("Context should not be shutting down initially")
		}
	})

	t.Run("Shutdown channel is not closed initially", func(t *testing.T) {
		select {
		case <-ctx.Shutdown():
			t.Error("Shutdown channel should not be closed initially")
		default:
			// Expected
		}
	})
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)

	t.Run("Concurrent Get/Set operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100
		numOperations := 100

		// Start multiple goroutines doing concurrent operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("key_%d_%d", id, j)
					value := fmt.Sprintf("value_%d_%d", id, j)

					ctx.Set(key, value)
					got := ctx.Get(key)

					if got != value {
						t.Errorf("Expected %s, got %v", value, got)
					}
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("Concurrent timeout operations", func(t *testing.T) {
		ctxWithTimeout := ctx.WithTimeout(100 * time.Millisecond)
		var wg sync.WaitGroup
		numGoroutines := 50

		// Start multiple goroutines checking timeout concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = ctxWithTimeout.IsExpired()
					_ = ctxWithTimeout.GetRemainingTime()
					time.Sleep(1 * time.Millisecond)
				}
			}()
		}

		wg.Wait()

		// Should eventually be expired
		time.Sleep(150 * time.Millisecond)
		if !ctxWithTimeout.IsExpired() {
			t.Error("Context should be expired after concurrent access")
		}
	})

	t.Run("Concurrent child creation", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 50
		children := make([]Context, numGoroutines)

		// Create children concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				children[id] = ctx.CreateChild(fmt.Sprintf("child_%d", id))
			}(i)
		}

		wg.Wait()

		// Verify all children were created correctly
		for i, child := range children {
			expectedPath := fmt.Sprintf("/child_%d", i)
			if child.GetPath() != expectedPath {
				t.Errorf("Expected path %s, got %s", expectedPath, child.GetPath())
			}
		}
	})
}

// TestPerformanceCharacteristics tests zero-allocation operations
func TestPerformanceCharacteristics(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)
	ctxWithTimeout := ctx.WithTimeout(1 * time.Hour) // Long timeout

	t.Run("IsExpired is zero-allocation", func(t *testing.T) {
		allocs := testing.AllocsPerRun(1000, func() {
			_ = ctxWithTimeout.IsExpired()
		})

		if allocs > 0 {
			t.Errorf("IsExpired should be zero-allocation, got %.2f allocs/op", allocs)
		}
	})

	t.Run("GetRemainingTime is zero-allocation", func(t *testing.T) {
		allocs := testing.AllocsPerRun(1000, func() {
			_ = ctxWithTimeout.GetRemainingTime()
		})

		if allocs > 0 {
			t.Errorf("GetRemainingTime should be zero-allocation, got %.2f allocs/op", allocs)
		}
	})

	t.Run("IsShuttingDown is zero-allocation", func(t *testing.T) {
		allocs := testing.AllocsPerRun(1000, func() {
			_ = ctx.IsShuttingDown()
		})

		if allocs > 0 {
			t.Errorf("IsShuttingDown should be zero-allocation, got %.2f allocs/op", allocs)
		}
	})

	t.Run("GetPath is zero-allocation", func(t *testing.T) {
		childCtx := ctx.CreateChild("test-child")

		allocs := testing.AllocsPerRun(1000, func() {
			_ = childCtx.GetPath()
		})

		if allocs > 0 {
			t.Errorf("GetPath should be zero-allocation, got %.2f allocs/op", allocs)
		}
	})
}

// BenchmarkContextOperations benchmarks context operations
func BenchmarkContextOperations(b *testing.B) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)
	ctxWithTimeout := ctx.WithTimeout(1 * time.Hour)

	b.Run("Get", func(b *testing.B) {
		ctx.Set("benchmark_key", "benchmark_value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ctx.Get("benchmark_key")
		}
	})

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx.Set("benchmark_key", i)
		}
	})

	b.Run("IsExpired", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ctxWithTimeout.IsExpired()
		}
	})

	b.Run("GetRemainingTime", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ctxWithTimeout.GetRemainingTime()
		}
	})

	b.Run("IsShuttingDown", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ctx.IsShuttingDown()
		}
	})

	b.Run("GetPath", func(b *testing.B) {
		childCtx := ctx.CreateChild("benchmark-child")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = childCtx.GetPath()
		}
	})

	b.Run("CreateChild", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ctx.CreateChild(fmt.Sprintf("child_%d", i))
		}
	})
}

// BenchmarkConcurrentAccess benchmarks concurrent context access
func BenchmarkConcurrentAccess(b *testing.B) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)

	b.Run("ConcurrentGetSet", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ctx.Set("key", "value")
				_ = ctx.Get("key")
			}
		})
	})

	b.Run("ConcurrentTimeoutCheck", func(b *testing.B) {
		ctxWithTimeout := ctx.WithTimeout(1 * time.Hour)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = ctxWithTimeout.IsExpired()
				_ = ctxWithTimeout.GetRemainingTime()
			}
		})
	})
}
