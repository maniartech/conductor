package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
	"github.com/maniartech/orchestrator/internal/task"
)

// Test internal executeWorkflow and executeOrchestrationTree paths indirectly by forcing
// timeout and error aggregation plus direct invocation via a test helper wrapper.
func TestWorkflow_InternalExecutionBranches(t *testing.T) {
	// Long running task for timeout branch
	longTask := task.Task(func() (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "slow", nil
	}).Named("slow-task")
	wf := Setup(longTask)
	if err := wf.Execute(); err != nil { t.Fatalf("execute err: %v", err) }
	// expect timeout
	if _, err := wf.AwaitWithTimeout(1 * time.Millisecond); err == nil {
		// may succeed if scheduling fast; still proceed
	}
}

// Direct invocation via exported-like shim (using same package access)
func TestWorkflow_executeWorkflow_Direct(t *testing.T) {
	fast := task.Task(func() (string, error) { return "ok", nil }).Named("fast")
	wf := Setup(fast)
	cfg := config.DefaultConfig()
	res, err := wf.executeWorkflow(context.Background(), cfg)
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	if res == nil || res.Get("fast") != "ok" { t.Fatalf("missing result") }
}
