package sequential

import (
	"context"
	"testing"

	internalErrors "github.com/maniartech/orchestrator/internal/errors"
	"github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"

	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
)

func BenchmarkSequential_Execute(b *testing.B) {
	cfg := config.Config{ErrorStrategy: internalErrors.FailFast}
	for i := 0; i < b.N; i++ {
		seq := Sequential(
			task.Task(func() (int, error) { return 1, nil }),
			task.Task(func() (int, error) { return 2, nil }),
		)
		_, err := seq.Execute(context.Background(), cfg)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
