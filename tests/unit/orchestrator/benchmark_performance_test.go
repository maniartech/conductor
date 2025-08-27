package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/maniartech/orchestrator"
	. "github.com/maniartech/orchestrator"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
)

// BenchmarkTask_BasicExecution tests basic task execution performance
func BenchmarkTask_BasicExecution(b *testing.B) {
	b.ReportAllocs()

	taskFn := func(ctx orchestrator.Context) (string, error) {
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

// BenchmarkWorkflow_BasicExecution tests basic workflow execution performance
func BenchmarkWorkflow_BasicExecution(b *testing.B) {
	b.ReportAllocs()

	taskFn := func(ctx orchestrator.Context) (string, error) {
		return "result", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := task.Task(taskFn).Named("workflow-task")
		workflow := Setup(t)
		_, _ = workflow.Await()
	}
}

// BenchmarkTask_ErrorHandling tests error handling performance
func BenchmarkTask_ErrorHandling(b *testing.B) {
	b.Run("NoErrors", func(b *testing.B) {
		b.ReportAllocs()

		taskFn := func(ctx orchestrator.Context) (string, error) {
			return "success", nil
		}

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(taskFn).Named("no-error-task")
			_, _ = t.Execute(ctx, cfg)
		}
	})

	b.Run("WithErrors", func(b *testing.B) {
		b.ReportAllocs()

		taskFn := func(ctx orchestrator.Context) (string, error) {
			return "", fmt.Errorf("test error")
		}

		ctx := context.Background()
		cfg := config.Config{}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			t := task.Task(taskFn).Named("error-task")
			_, _ = t.Execute(ctx, cfg)
		}
	})
}

// BenchmarkTask_TypeVariations tests performance across different types
func BenchmarkTask_TypeVariations(b *testing.B) {
	ctx := context.Background()
	cfg := config.Config{}

	b.Run("StringType", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			t := task.Task(func(ctx orchestrator.Context) (string, error) {
				return "result", nil
			}).Named("string-task")
			_, _ = t.Execute(ctx, cfg)
		}
	})

	b.Run("IntType", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			t := task.Task(func(ctx orchestrator.Context) (int, error) {
				return 42, nil
			}).Named("int-task")
			_, _ = t.Execute(ctx, cfg)
		}
	})

	b.Run("StructType", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		type TestStruct struct {
			ID   int
			Name string
		}

		for i := 0; i < b.N; i++ {
			t := task.Task(func(ctx orchestrator.Context) (TestStruct, error) {
				return TestStruct{ID: 1, Name: "test"}, nil
			}).Named("struct-task")
			_, _ = t.Execute(ctx, cfg)
		}
	})
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("TaskCreation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = task.Task(func(ctx orchestrator.Context) (string, error) {
				return "result", nil
			}).Named("memory-task")
		}
	})

	b.Run("WorkflowSetup", func(b *testing.B) {
		b.ReportAllocs()

		t := task.Task(func(ctx orchestrator.Context) (string, error) {
			return "result", nil
		}).Named("test-task")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Setup(t)
		}
	})
}

// BenchmarkScalability tests performance scaling
func BenchmarkScalability(b *testing.B) {
	sizes := []int{1, 5, 10}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Tasks_%d", size), func(b *testing.B) {
			b.ReportAllocs()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create and execute multiple tasks
				for j := 0; j < size; j++ {
					t := task.Task(func(ctx orchestrator.Context) (string, error) {
						return "result", nil
					}).Named(fmt.Sprintf("scale-task-%d", j))

					workflow := Setup(t)
					_, _ = workflow.Await()
				}
			}
		})
	}
}

// BenchmarkConcurrentWorkflows tests concurrent workflow execution
func BenchmarkConcurrentWorkflows(b *testing.B) {
	b.ReportAllocs()

	const numWorkflows = 5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create multiple workflows and execute them sequentially
		// (since we don't have concurrent orchestration implemented yet)
		for j := 0; j < numWorkflows; j++ {
			t := task.Task(func(ctx orchestrator.Context) (string, error) {
				return "result", nil
			}).Named(fmt.Sprintf("concurrent-workflow-%d", j))

			workflow := Setup(t)
			_, _ = workflow.Await()
		}
	}
}

// BenchmarkLongRunningTasks tests performance with longer tasks
func BenchmarkLongRunningTasks(b *testing.B) {
	durations := []time.Duration{
		time.Microsecond,
		10 * time.Microsecond,
		100 * time.Microsecond,
	}

	for _, duration := range durations {
		b.Run(fmt.Sprintf("Duration_%s", duration), func(b *testing.B) {
			b.ReportAllocs()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				t := task.Task(func(ctx orchestrator.Context) (string, error) {
					time.Sleep(duration)
					return "completed", nil
				}).Named(fmt.Sprintf("long-task-%s", duration))

				workflow := Setup(t)
				_, _ = workflow.Await()
			}
		})
	}
}

// BenchmarkContextCancellation tests cancellation performance
func BenchmarkContextCancellation(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		t := task.Task(func(ctx orchestrator.Context) (string, error) {
			time.Sleep(time.Millisecond) // Long enough to be cancelled
			return "result", nil
		}).Named("cancellable-task")

		workflow := Setup(t)

		// Cancel immediately
		cancel()

		_, _ = workflow.AwaitWithContext(ctx)
	}
}
