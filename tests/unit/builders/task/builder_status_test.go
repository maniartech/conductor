package task

import (
	"context"
	"sync"
	"testing"
	"time"

	. "github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/types"
)

func TestTaskBuilderAtomicStatusManagement(t *testing.T) {
	tk := Task(func() (string, error) { return "test", nil })
	if tk.GetStatus() != types.NotStarted {
		t.Errorf("expected NotStarted got %v", tk.GetStatus())
	}
	if !tk.CompareAndSwapStatus(types.NotStarted, types.Running) {
		t.Error("CAS should succeed")
	}
	if tk.GetStatus() != types.Running {
		t.Errorf("expected Running")
	}
	if tk.CompareAndSwapStatus(types.NotStarted, types.Running) {
		t.Error("CAS should fail second time")
	}
	tk.SetStatus(types.Completed)
	if tk.GetStatus() != types.Completed {
		t.Errorf("expected Completed")
	}
}

func TestTaskBuilderConcurrentStatusAccess(t *testing.T) {
	tk := Task(func() (string, error) { time.Sleep(10 * time.Millisecond); return "test", nil })
	var wg sync.WaitGroup
	const n = 50
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = tk.GetStatus()
		}()
	}
	go func() {
		tk.Execute(context.Background(), config.DefaultConfig())
	}()
	wg.Wait()
}

func TestTaskBuilderSingleExecution(t *testing.T) {
	count := 0
	tk := Task(func() (string, error) { count++; return "test", nil })
	cfg := config.DefaultConfig()
	if _, err := tk.Execute(context.Background(), cfg); err != nil {
		t.Fatalf("first execute err %v", err)
	}
	if _, err := tk.Execute(context.Background(), cfg); err == nil {
		t.Error("second execution should fail")
	}
	if count != 1 {
		t.Errorf("expected single invocation got %d", count)
	}
}
