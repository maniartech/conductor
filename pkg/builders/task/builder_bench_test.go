package task

import (
	"context"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/types"
)

func BenchmarkTaskCreation(b *testing.B) {
	fn := func() (string, error) { return "test", nil }
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Task(fn)
	}
}

func BenchmarkTaskBuilderFluentAPI(b *testing.B) {
	fn := func() (string, error) { return "test", nil }
	cfg := config.Config{ErrorStrategy: errors.CollectAll, Timeout: 30 * time.Second, MaxConcurrency: 100}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Task(fn).Named("benchmark").With(cfg).ErrorBoundary(errors.FailFast)
	}
}

func BenchmarkTaskExecution(b *testing.B) {
	tk := Task(func() (string, error) { return "benchmark", nil }).Named("benchmark-task")
	cfg := config.DefaultConfig()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tk.Execute(ctx, cfg)
	}
}

func BenchmarkTaskBuilderSafeExecute(b *testing.B) {
	tk := Task(func() (string, error) { return "benchmark", nil })
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tk.safeExecute(ctx)
	}
}

func BenchmarkTaskBuilderStatusOperations(b *testing.B) {
	tk := Task(func() (string, error) { return "test", nil })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tk.GetStatus()
		tk.SetStatus(types.Running)
		tk.CompareAndSwapStatus(types.Running, types.Completed)
	}
}

func BenchmarkTaskBuilderStackCapture(b *testing.B) {
	tk := Task(func() (string, error) { return "test", nil })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tk.captureStack()
	}
}

func BenchmarkTaskBuilderConcurrentExecution(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Task(func() (string, error) { return "benchmark", nil }).Execute(context.Background(), config.DefaultConfig())
		}
	})
}
