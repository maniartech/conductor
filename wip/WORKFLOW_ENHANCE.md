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

## 4. Path to Production-Grade Reliability

While the enhancements above improve the core API, building "military-grade" production systems requires addressing two critical areas: fault tolerance and data consistency. The following sections outline proposals for integrating these capabilities directly into the orchestrator.

### 4.1. Proposed: Transaction Rollback Support (Saga Pattern)

**Problem:** The current execution model is purely forward-moving. If a task fails, there is no automatic mechanism to undo or compensate for the actions of previously completed tasks. This can leave the system in an inconsistent state.

**Proposed Solution:** Implement the **Saga pattern** by introducing compensation logic.

*   **`WithCompensation(compensationFunc)`:** A new builder method would allow a task to be associated with a corresponding rollback function.
    ```go
    orchestrator.Task(createOrder).
        WithCompensation(cancelOrder)
    ```
*   **Automatic Rollback:** If a sequential workflow fails, the orchestrator would automatically iterate in reverse over the successfully completed tasks and execute their compensation functions, passing the original task's output as input.

**Does this provide a complete Saga pattern?**
This proposal lays the foundation for a complete Saga implementation. A full implementation would also need to handle state management for the rollback process, ensure compensations are idempotent, and manage failures during the compensation itself. While not "complete" out-of-the-box, it is the correct architectural step.

### 4.2. Proposed: Declarative Retry Mechanism

**Problem:** Tasks can fail due to transient issues like network timeouts or temporary service unavailability. Currently, retry logic must be implemented manually inside each task, which is repetitive and error-prone.

**Proposed Solution:** Introduce a declarative retry policy.

*   **`WithRetries(policy)`:** A new builder method would allow tasks to be configured with a retry policy, including backoff strategies.
    ```go
    orchestrator.Task(fetchFromAPI).
        WithRetries(config.RetryPolicy{
            MaxAttempts: 5,
            Backoff:     config.ExponentialBackoff(1*time.Second),
            Jitter:      config.FullJitter,
        })
    ```
*   **Automatic Retries:** The orchestrator would wrap the task execution and automatically retry it according to the policy if it fails.

### Conclusion: Building "Military-Grade" Systems

Implementing both the **Saga pattern for rollbacks** and a **declarative retry mechanism** are essential steps toward building highly reliable, fault-tolerant, and resilient systems. While "military-grade" is a high bar, these features are foundational pillars that move the orchestrator from a simple workflow runner to a robust tool capable of managing complex, long-running, and mission-critical processes in a production environment. They provide the guarantees needed to maintain data consistency and recover from failure gracefully.

## 5. Achieving True Military-Grade Robustness: Advanced Features

While Saga and Retries are foundational, achieving "military-grade" robustness requires addressing several other critical dimensions. The key to incorporating them is to maintain the ergonomic API by making these advanced features **opt-in and composable**, using the same builder pattern.

### 5.1. Persistence and State Recovery

**Problem:** The orchestrator is currently in-memory. If the application crashes, the state of all running workflows is lost. A robust system must be able to resume workflows from where they left off.

**Proposed Solution:** Introduce a `StateProvider` interface that the orchestrator can use to persist and recover workflow state.
*   **API Proposal:**
    ```go
    // Define a simple interface for state management
    type StateProvider interface {
        Save(ctx context.Context, workflowID string, state []byte) error
        Load(ctx context.Context, workflowID string) ([]byte, error)
    }

    // Integrate it via a new builder method
    workflow.WithStateProvider(NewRedisProvider("redis://..."))
    ```
*   **How it works:** The orchestrator would automatically save the state of the workflow before executing each new task. Upon restart, a workflow instance with the same ID would first load its state and resume from where it left off.

### 5.2. Enhanced Observability

**Problem:** For mission-critical systems, you need deep insight into performance, errors, and workflow status. Basic logging is not enough.

**Proposed Solution:** Integrate with standard observability libraries (like OpenTelemetry).
*   **API Proposal:**
    ```go
    workflow.WithLogger(slog.Default())
    workflow.WithMetrics(otel.Meter("my-app/conductor"))
    ```
*   **How it works:** The orchestrator would emit detailed, structured logs for every significant event and record metrics like task duration, success/failure counts, and retry attempts.

### 5.3. Idempotency for Tasks

**Problem:** If a workflow resumes after a crash or a task is retried, how do you prevent an action from running twice (e.g., charging a credit card twice)?

**Proposed Solution:** Provide a declarative way to ensure task idempotency.
*   **API Proposal:**
    ```go
    orchestrator.Task(chargeCard).
        WithIdempotencyKey("charge-{{.transactionID}}")
    ```
*   **Why not use `Named()`?** `Named("my-task")` provides a **static identifier for the task definition**, while an idempotency key provides a **dynamic, unique identifier for a specific execution of that task**. Using the static name would incorrectly prevent all subsequent executions of that task for different data.
*   **How it works:** Before executing the task, the orchestrator uses the `StateProvider` to check if a task with that idempotency key has already completed successfully. If so, it skips execution and returns the saved result.

### 5.4. Dead-Letter Queue (DLQ) for Terminal Failures

**Problem:** What happens when a task fails permanently, even after all retries? The failure needs to be recorded for manual intervention.

**Proposed Solution:** Allow configuration of a DLQ handler.
*   **API Proposal:**
    ```go
    orchestrator.Task(processVideo).
        WithRetries(...).
        WithDLQ(NewKafkaDLQ("failed-tasks-topic"))
    ```
*   **How it works:** If a task fails all its retries, the orchestrator serializes the task's input and final error and sends it to the configured DLQ.

### Putting It All Together: The Ergonomic API

This approach allows a developer to start simple and progressively add layers of resilience as needed, without changing their core business logic.

**Simple Start:**
`orchestrator.Setup(myTask).Run(ctx)`

**Military-Grade:**
```go
orchestrator.Setup(myTask.
    WithRetries(policy).
    WithIdempotencyKey("...").
    WithCompensation(undoTask)).
WithStateProvider(redis).
WithLogger(slog).
WithMetrics(otel).
Run(ctx)
```
