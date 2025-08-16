package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/maniartech/orchestrator"
	"github.com/maniartech/orchestrator/internal/status"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/pkg/types"
)

// BenchmarkTask_ZeroAllocation tests zero-allocation task execution
func BenchmarkTask_ZeroAllocation(b *testing.B) {
	b.ReportAllocs()

	taskFn := func() (string, error) {
		return "result", nil
	}

	ctx := context.Background()
	cfg := config.Config{
		Timeout: 5 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := task.Task(taskFn).Named("benchmark-task")
		_, _ = t.Execute(ctx, cfg)
	}
}

// BenchmarkTask_GenericTypes tests performance across different generic types
func BenchmarkTask_GenericTypes(b *testing.B) {
	benchmarks := []struct {
		name string
		fn   func() (interface{}, error)
	}{
		{"String", func() (interface{}, error) { return "test", nil }},
		{"Int", func() (interface{}, error) { return 42, nil }},
		{"Slice", func() (interface{}, error) { return []int{1, 2, 3}, nil }},
		{"Map", func() (interface{}, error) { return map[string]int{"key": 1}, nil }},
		{"Struct", func() (interface{}, error) { return struct{ Name string }{"test"}, nil }},
	}

	ctx := context.Background()
	cfg := config.Config{}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				t := task.Task(bm.fn).Named("benchmark-" + bm.name)
				_, _ = t.Execute(ctx, cfg)
			}
		})
	}
}

// BenchmarkResult_Operations tests result storage and retrieval performance
func BenchmarkResult_Operations(b *testing.B) {
	b.Run("Set", func(b *testing.B) {
		b.ReportAllocs()
		r := result.NewResult()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		}
	})

	b.Run("Get", func(b *testing.B) {
		b.ReportAllocs()
		r := result.NewResult()

		// Pre-populate
		for i := 0; i < 1000; i++ {
			r.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = r.Get(fmt.Sprintf("key-%d", i%1000))
		}
	})
}

// BenchmarkStatus_AtomicOperations tests atomic status operations performance
func BenchmarkStatus_AtomicOperations(b *testing.B) {
	manager := status.NewManager()

	b.Run("Get", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = manager.Get()
		}
	})

	b.Run("Set", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			manager.Set(status.Status(types.Status(i % 5)))
		}
	})
}

// BenchmarkWorkflow_Execution tests workflow execution performance
func BenchmarkWorkflow_Execution(b *testing.B) {
	b.Run("SingleTask", func(b *testing.B) {
		b.ReportAllocs()

		taskFn := func() (string, error) {
			return "result", nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(taskFn).Named("single-task")
			workflow := Setup(t)
			_, _ = workflow.Await()
		}
	})
}

// BenchmarkMemoryUsage tests memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("TaskCreation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = task.Task(func() (string, error) {
				return "result", nil
			}).Named(fmt.Sprintf("task-%d", i))
		}
	})

	b.Run("WorkflowSetup", func(b *testing.B) {
		b.ReportAllocs()
		t := task.Task(func() (string, error) {
			return "result", nil
		}).Named("test-task")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Setup(t)
		}
	})
}
