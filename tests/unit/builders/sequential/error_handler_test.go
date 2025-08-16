package sequential

import (
	"testing"

	. "github.com/maniartech/orchestrator/pkg/builders/sequential"
	"github.com/maniartech/orchestrator/pkg/errors"
)

// TestCreateSequentialErrorContext ensures the helper delegates correctly to errors.CreateErrorContext
func TestCreateSequentialErrorContext(t *testing.T) {
	ctx := CreateSequentialErrorContext("seq-name", "seq-id", 5, errors.FailFast, nil)
	if ctx == nil {
		t.Fatalf("expected non-nil context")
	}
	if ctx.OrchestrationName != "seq-name" || ctx.OrchestrationID != "seq-id" || ctx.TotalSteps != 5 {
		t.Fatalf("unexpected context values: %+v", ctx)
	}
	if ctx.ErrorStrategy != errors.FailFast {
		t.Fatalf("expected FailFast strategy")
	}
}
