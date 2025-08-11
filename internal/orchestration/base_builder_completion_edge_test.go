package orchestration

import (
	"context"
	"testing"

	"github.com/maniartech/orchestrator/types"
)

func TestBaseOrchestrationBuilder_CompleteExecution_CancelledNoError(t *testing.T){
	b := NewBaseOrchestrationBuilder("sequential")
	b.SetStatus(0) // NotStarted
	// transition to running via validation
	if err := b.ValidateExecutionPreconditions(); err != nil { t.Fatalf("precondition failed %v", err) }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b.CompleteExecution(ctx, nil)
	// Implementation currently sets Completed even if ctx cancelled but no error
	if st := b.GetStatus(); st != types.Completed { t.Errorf("expected Completed got %v", st) }
}

func TestBaseOrchestrationBuilder_CompleteExecution_CancelledWithError(t *testing.T){
	b := NewBaseOrchestrationBuilder("sequential")
	if err := b.ValidateExecutionPreconditions(); err != nil { t.Fatalf("precondition failed %v", err) }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b.CompleteExecution(ctx, context.Canceled)
	if st := b.GetStatus(); st != types.Cancelled { t.Errorf("expected Cancelled got %v", st) }
}

func TestBaseOrchestrationBuilder_CompleteExecution_Idempotent(t *testing.T){
	b := NewBaseOrchestrationBuilder("task")
	if err := b.ValidateExecutionPreconditions(); err != nil { t.Fatalf("precondition failed %v", err) }
	b.CompleteExecution(context.Background(), nil)
	first := b.GetStatus()
	b.CompleteExecution(context.Background(), nil)
	second := b.GetStatus()
	if first != second { t.Errorf("expected idempotent status, got %v then %v", first, second) }
}

func TestBaseOrchestrationBuilder_ValidateExecutionPreconditions_AfterCompletion(t *testing.T){
	b := NewBaseOrchestrationBuilder("task")
	if err := b.ValidateExecutionPreconditions(); err != nil { t.Fatalf("first validation failed %v", err) }
	b.CompleteExecution(context.Background(), nil)
	if err := b.ValidateExecutionPreconditions(); err == nil { t.Error("expected second validation to fail after completion") }
}
