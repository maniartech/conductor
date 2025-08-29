# Workflow API Enhancement and Refactoring

This document outlines the significant enhancements and refactoring applied to the `Workflow` API to improve its usability, clarity, and robustness.

## 1. Simplified and Unified Execution API

The previous `Workflow` API had a confusing proliferation of execution methods (`Execute`, `ExecuteBlocking`, `Await`, `AwaitWithContext`, `AwaitWithTimeout`). This has been streamlined into a more intuitive and consistent set of methods.

### New API

* **`Run(ctx context.Context) (*result.Result, error)`**
  * **Purpose:** Synchronous (blocking) execution.
  * **Behavior:** Starts the workflow, waits for it to complete, and returns the final result and error. The provided `context.Context` controls the entire lifecycle.
  * **Replaces:** `ExecuteBlocking`, `Await`, `AwaitWithContext`.

* **`RunAsync(ctx context.Context) error`**
  * **Purpose:** Asynchronous (non-blocking) execution.
  * **Behavior:** Starts the workflow in a background goroutine and returns immediately. The provided `context.Context` controls the workflow's lifecycle. An error is returned only if the workflow fails to start (e.g., if already running).
  * **Replaces:** `Execute`.

* **`Result() (*result.Result, error)`**
  * **Purpose:** Retrieve the outcome of an asynchronous workflow.
  * **Behavior:** Blocks until the asynchronously running workflow completes, then returns its final result and error. It can be called multiple times safely.

### Old (Deprecated) API

* `Execute()`
* `ExecuteBlocking()`
* `Await()`
* `AwaitWithContext()`
* `AwaitWithTimeout()`

### Example Usage

**Synchronous Execution:**
```go
// Old way
// result, err := workflow.Await()
// or
// result, err := workflow.ExecuteBlocking()

// New way
result, err := workflow.Run(context.Background())
```

**Asynchronous Execution:**
```go
// Old way
// if err := workflow.Execute(); err != nil { ... }
// result, err := workflow.Await()

// New way
if err := workflow.RunAsync(context.Background()); err != nil {
    // handle start error
}
// ... do other work ...
result, err := workflow.Result()
```

## 2. Enhanced Context and Data Handling

Passing initial data into a workflow is now more straightforward.

* **`WithData(data map[string]any) *Workflow`**
  * **Purpose:** A new fluent method to set initial data for the orchestration context.
  * **Behavior:** This method populates the `config.Data` map, making the data available to all tasks within the workflow via `orchContext.Context`.

### Example Usage

```go
data := map[string]any{"userID": 123, "traceID": "xyz-789"}

workflow := orchestrator.Setup(myOrchestration).WithData(data)

result, err := workflow.Run(context.Background())
```

## 3. Bug Fix: `AwaitWithTimeout` Resource Leak

A critical bug was fixed where `AwaitWithTimeout` would return an error on timeout but would not stop the underlying workflow, leading to a resource leak.

* **The Fix:** The method now calls `w.CancelWithReason()` upon timeout. This ensures the workflow's context is canceled, signaling all running tasks to terminate and preventing the goroutine from leaking.

### Corrected Behavior
```go
func (w *Workflow) AwaitWithTimeout(timeout time.Duration) (*result.Result, error) {
	// ...
	select {
	case <-w.executionDone:
		return w.getResult()
	case <-time.After(timeout):
		// This now correctly cancels the workflow
		reason := fmt.Sprintf("workflow execution timed out after %v", timeout)
		w.CancelWithReason(reason)
		return nil, fmt.Errorf(reason)
	}
}
```
