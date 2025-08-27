package task

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/orchestrator"
	. "github.com/maniartech/orchestrator/pkg/builders/task"
	"github.com/maniartech/orchestrator/pkg/config"
)

func TestTaskBuilderStressConcurrentExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	const numTasks = 200
	const concurrency = 50
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	for i := 0; i < numTasks; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			tk := Task(func(ctx orchestrator.Context) (int, error) {
				time.Sleep(time.Microsecond * time.Duration(i%5))
				return i, nil
			}).Named(fmt.Sprintf("stress-task-%d", i))
			res, err := tk.Execute(context.Background(), config.DefaultConfig())
			if err != nil {
				t.Errorf("task %d failed: %v", i, err)
				return
			}
			if res == nil {
				t.Errorf("task %d nil result", i)
				return
			}
			if v, ok := res.Get(fmt.Sprintf("stress-task-%d", i)).(int); !ok || v != i {
				t.Errorf("task %d wrong value %v", i, v)
			}
		}()
	}
	wg.Wait()
}

func TestTaskBuilderStressErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	const numTasks = 150
	var wg sync.WaitGroup
	for i := 0; i < numTasks; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			var tk *TaskBuilder[string]
			if i%3 == 0 {
				tk = Task(func(ctx orchestrator.Context) (string, error) {
					panic(fmt.Sprintf("panic-%d", i))
				})
			} else if i%3 == 1 {
				tk = Task(func(ctx orchestrator.Context) (string, error) {
					return "", fmt.Errorf("error-%d", i)
				})
			} else {
				tk = Task(func(ctx orchestrator.Context) (string, error) {
					return fmt.Sprintf("success-%d", i), nil
				})
			}
			tk.Named(fmt.Sprintf("stress-error-task-%d", i))
			res, err := tk.Execute(context.Background(), config.DefaultConfig())
			if res == nil {
				t.Errorf("task %d nil result", i)
				return
			}
			if i%3 == 2 {
				if err != nil || res.HasErrors() {
					t.Errorf("success task %d unexpected error", i)
				}
			} else {
				if err == nil || !res.HasErrors() {
					t.Errorf("error/panic task %d expected error", i)
				}
			}
		}()
	}
	wg.Wait()
}
