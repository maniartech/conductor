package concurrent

import (
	"context"
	"fmt"
	"testing"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
	"github.com/maniartech/orchestrator/types"

	. "github.com/maniartech/orchestrator/internal/concurrent"
)

// BenchmarkConcurrentExecution benchmarks basic concurrent execution
func BenchmarkConcurrentExecution(b *testing.B) {
	const taskCount = 10

	ctx := context.Background()
	cfg := config.DefaultConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create fresh tasks for each iteration
		var orchestrations []types.Orchestration
		for j := 0; j < taskCount; j++ {
			value := fmt.Sprintf("task-%d", j)
			orchestrations = append(orchestrations, task.Task(func() (string, error) {
				return value, nil
			}))
		}

		concurrent := Concurrent(orchestrations...)
		_, err := concurrent.Execute(ctx, cfg)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
