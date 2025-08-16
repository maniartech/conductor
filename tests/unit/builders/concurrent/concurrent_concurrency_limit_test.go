package concurrent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
	"github.com/maniartech/orchestrator/types"

	. "github.com/maniartech/orchestrator/internal/concurrent"
)

func TestConcurrent_ConcurrencyLimit(t *testing.T) {
	const maxConc = 2
	var current int32
	var maxObserved int32

	var orch []types.Orchestration
	for i := 0; i < 6; i++ {
		orch = append(orch, task.Task(func() (int, error) {
			c := atomic.AddInt32(&current, 1)
			for {
				m := atomic.LoadInt32(&maxObserved)
				if c <= m || atomic.CompareAndSwapInt32(&maxObserved, m, c) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			atomic.AddInt32(&current, -1)
			return int(c), nil
		}))
	}
	c := Concurrent(orch...).With(config.Config{MaxConcurrency: maxConc})
	_, err := c.Execute(context.Background(), config.DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if atomic.LoadInt32(&maxObserved) > maxConc {
		t.Errorf("observed %d > %d", maxObserved, maxConc)
	}
}
