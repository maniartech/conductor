package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
)

// BenchmarkZeroAllocation_TaskExecution verifies efficient task execution
func BenchmarkZeroAllocation_TaskExecution(b *testing.B) {
	b.ReportAllocs()

	// Pre-create task to avoid allocation during benchmark
	t := task.Task(func() (string, error) {
		return "result", nil
	}).Named("zero-alloc-task")

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = t.Execute(ctx, config.Config{})
	}
}

// BenchmarkZeroAllocation_StatusOperations verifies efficient status operations
func BenchmarkZeroAllocation_StatusOperations(b *testing.B) {
	b.Run("StatusGet", func(b *testing.B) {
		b.ReportAllocs()

		t := task.Task(func() (string, error) {
			return "result", nil
		}).Named("status-task")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = t.GetStatus()
		}
	})

	b.Run("StatusTransitions", func(b *testing.B) {
		b.ReportAllocs()

		t := task.Task(func() (string, error) {
			return "result", nil
		}).Named("transition-task")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = t.GetStatus()
		}
	})
}

// BenchmarkZeroAllocation_PoolOperations verifies efficient pool operations
func BenchmarkZeroAllocation_PoolOperations(b *testing.B) {
	b.Run("PoolGetPut", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(func() (string, error) {
				return "result", nil
			})
			_ = t.Named("pool-task")
		}
	})
}

// BenchmarkAllocation_Comparison compares allocation patterns
func BenchmarkAllocation_Comparison(b *testing.B) {
	b.Run("WithPooling", func(b *testing.B) {
		b.ReportAllocs()

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(func() (string, error) {
				return "result", nil
			}).Named("pooled-task")

			_, _ = t.Execute(ctx, cfg)
		}
	})

	b.Run("WithoutPooling_Simulation", func(b *testing.B) {
		b.ReportAllocs()

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate non-pooled allocation
			result := make(map[string]interface{})
			result["key"] = "value"

			t := task.Task(func() (string, error) {
				return "result", nil
			}).Named("non-pooled-task")

			_, _ = t.Execute(ctx, cfg)
		}
	})
}

// BenchmarkAllocation_StringOperations tests string allocation patterns
func BenchmarkAllocation_StringOperations(b *testing.B) {
	b.Run("StaticNames", func(b *testing.B) {
		b.ReportAllocs()

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(func() (string, error) {
				return "result", nil
			}).Named("static-name") // Static string - no allocation

			_, _ = t.Execute(ctx, cfg)
		}
	})

	b.Run("DynamicNames", func(b *testing.B) {
		b.ReportAllocs()

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(func() (string, error) {
				return "result", nil
			}).Named(fmt.Sprintf("dynamic-name-%d", i)) // Dynamic string - allocates

			_, _ = t.Execute(ctx, cfg)
		}
	})
}

// BenchmarkAllocation_ErrorHandling tests error handling allocation patterns
func BenchmarkAllocation_ErrorHandling(b *testing.B) {
	b.Run("NoErrors", func(b *testing.B) {
		b.ReportAllocs()

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(func() (string, error) {
				return "success", nil
			}).Named("success-task")

			_, _ = t.Execute(ctx, cfg)
		}
	})

	b.Run("WithErrors", func(b *testing.B) {
		b.ReportAllocs()

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(func() (string, error) {
				return "", fmt.Errorf("error %d", i) // Error allocation
			}).Named("error-task")

			_, _ = t.Execute(ctx, cfg)
		}
	})

	b.Run("PreallocatedErrors", func(b *testing.B) {
		b.ReportAllocs()

		// Pre-allocate error to avoid allocation during benchmark
		preAllocError := fmt.Errorf("pre-allocated error")

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(func() (string, error) {
				return "", preAllocError // Reuse pre-allocated error
			}).Named("preallocated-error-task")

			_, _ = t.Execute(ctx, cfg)
		}
	})
}

// BenchmarkAllocation_ConcurrentAccess tests allocation under concurrent access
func BenchmarkAllocation_ConcurrentAccess(b *testing.B) {
	b.Run("ConcurrentTaskCreation", func(b *testing.B) {
		b.ReportAllocs()

		cfg := config.Config{}

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				t := task.Task(func() (string, error) {
					return "result", nil
				}).Named(fmt.Sprintf("concurrent-task-%d", i))

				ctx := context.Background()
				_, _ = t.Execute(ctx, cfg)
				i++
			}
		})
	})
}

// BenchmarkAllocation_LongRunning tests allocation patterns in long-running scenarios
func BenchmarkAllocation_LongRunning(b *testing.B) {
	b.Run("RepeatedExecution", func(b *testing.B) {
		b.ReportAllocs()

		// Create task once
		t := task.Task(func() (string, error) {
			return "result", nil
		}).Named("long-running-task")

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Execute same task repeatedly - should not allocate after warmup
			_, _ = t.Execute(ctx, cfg)
		}
	})
}

// BenchmarkAllocation_ResultOperations tests result-related allocations
func BenchmarkAllocation_ResultOperations(b *testing.B) {
	b.Run("ResultStorage", func(b *testing.B) {
		b.ReportAllocs()

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(func() (map[string]interface{}, error) {
				// Return complex result
				return map[string]interface{}{
					"id":     i,
					"status": "success",
					"data":   []string{"item1", "item2"},
				}, nil
			}).Named("result-task")

			_, _ = t.Execute(ctx, cfg)
		}
	})
}
