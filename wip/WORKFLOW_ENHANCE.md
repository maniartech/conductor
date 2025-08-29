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

## 4. The Ultimate Ergonomic API: A Unified Configuration Model

The most significant architectural enhancement is the move to a **unified, hierarchical configuration model**. Instead of separate config objects, all configuration is now applied directly to orchestrations using a fluent builder pattern. This creates a single, intuitive, and powerful way to define behavior.

### 4.1. Hierarchical and Scoped Configuration

Configuration is applied directly to any orchestration (`Task`, `Sequential`, `Concurrent`, etc.) and is **hierarchically inherited** by its children. This allows for both global and fine-grained control.

*   **Orchestration-Scoped (Inherited):** These settings apply to the orchestration they are attached to and all of its children.
    *   `WithRetries(policy)`
    *   `WithLogger(logger)`
    *   `WithMetrics(meter)`
    *   `WithErrorStrategy(strategy)`
    *   `WithStateProvider(provider)`
    *   `WithID(id)`

*   **Task-Specific (Not Inherited):** These settings only make sense on an individual `Task`.
    *   `Named(name)`
    *   `WithCompensation(compensationFunc)`
    *   `WithIdempotencyKey(keyTemplate)`
    *   `WithDLQ(dlqHandler)`

### 4.2. Advanced Flow Control: The `Fallback` Orchestrator

To handle situations where a task might fail permanently, a `Fallback` orchestrator provides a chain of alternative execution paths.

*   **API Proposal:**
    ```go
    orchestrator.Fallback(
        orchestrator.Task(tryFastPaymentProvider),      // Attempt 1
        orchestrator.Task(tryReliablePaymentProvider),  // Attempt 2 (if 1 fails)
        orchestrator.Task(notifyAdminAndQueueJob),      // Attempt 3 (if 2 fails)
    )
    ```
*   **How it works:** The `Fallback` orchestrator executes the first task. If it succeeds, the orchestrator returns its result. If it fails (after any retries), it proceeds to the next orchestration in the chain. This continues until one succeeds or all have failed.

## 5. Military-Grade Reliability Patterns

The unified API makes implementing robust reliability patterns clean and declarative.

### 5.1. Saga Pattern and Compensation Failures

*   **`WithCompensation(compensationFunc)`:** Enables the Saga pattern for rollbacks.
*   **Strategy for Compensation Failures:** A truly robust system must plan for failures during a rollback. The strategy is a multi-layered defense:
    1.  **Retry:** The compensation task itself should have a retry policy.
    2.  **Fallback:** If the primary compensation fails, a fallback compensation can be attempted (e.g., issue store credit instead of a refund).
    3.  **Alert and DLQ:** If all automated compensations fail, the workflow must enter a `ROLLBACK_FAILED` state, trigger a high-priority alert for manual intervention, and send the failure context to a Dead-Letter Queue.

### 5.2. Idempotency

*   **`WithIdempotencyKey(keyTemplate)`:** Ensures a task with a specific input is only executed once, even across retries or workflow restarts. This requires a `WithStateProvider` to be configured on a parent orchestration.

### 5.3. Persistence and State Recovery

*   **`WithStateProvider(provider)`:** When applied to the root orchestration, this enables the entire workflow to be persistent. The orchestrator will automatically save state between steps and can resume from the point of failure after a crash.

## 6. Putting It All Together: The Final API

This design leads to an incredibly expressive and powerful API where the entire workflow, including all its complex reliability logic, is defined in a single, readable block.

```go
// Define the entire workflow and its configuration in one go.
workflow := orchestrator.Setup(
    orchestrator.Sequential(
        orchestrator.Task(createOrder).
            Named("CreateOrder").
            WithCompensation(cancelOrder),

        // This task has its own specific retry and idempotency logic.
        orchestrator.Task(chargeCard).
            Named("ChargeCard").
            WithRetries(paymentGatewayPolicy).
            WithIdempotencyKey("charge-{{.orderID}}").
            WithCompensation(refundCharge),

        orchestrator.Task(sendEmail).
            Named("SendConfirmationEmail").
            WithRetries(emailServicePolicy)

    // Global settings for the whole workflow are applied to the root orchestration.
    // These are inherited by all children.
    ).WithID("workflow-instance-123").
      WithLogger(mainLogger).
      WithStateProvider(redisProvider)
)

// Run it
result, err := workflow.Run(context.Background())
```
