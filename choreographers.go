package orchestrator

// Orchestrator provides a way to
// orchestrate the execution of multiple functions in a synchronous or
// asynchronous manner. It also provides a way to handle errors and
// handle the result of the function execution. It
// manages the root context and provides a way to cancel the execution.
// It also provides read-only access to the result of the functions.
// It stores the result of the function execution in a thread-safe way.
// These results can be accessed by the functions which are executed
// after the function which produced the result. It also provides a way
// to cancel the entire execution or a specific function execution. During
// execution, each function receives the orchestrator context which provides
// access to the result of the functions which are executed before it.
type Orchestrator struct {
	orchestration *Orchestration

	Results map[string]interface{}
}

// New creates a new orchestrator instance.
func New(orchestration *Orchestration) *Orchestrator {
	return &Orchestrator{
		orchestration: orchestration,
	}
}
