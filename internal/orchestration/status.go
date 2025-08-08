package orchestration

import "github.com/maniartech/orchestrator/types"

// Status is an alias to the types.Status type for backward compatibility.
// All status operations should use types.Status directly.
type Status = types.Status

// Status constants for backward compatibility
const (
	NotStarted = types.NotStarted
	Running    = types.Running
	Completed  = types.Completed
	Failed     = types.Failed
	Cancelled  = types.Cancelled
)
