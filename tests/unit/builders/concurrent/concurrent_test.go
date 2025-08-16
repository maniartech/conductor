package concurrent

// This file now serves as an index after refactor.
// Granular tests for ConcurrentBuilder have been split into focused files:
//   - concurrent_constructor_test.go        (constructor & validation panics)
//   - concurrent_fluent_api_test.go         (Named / With / ErrorBoundary chaining)
//   - concurrent_execute_failfast_test.go   (FailFast success, error, cancellation)
//   - concurrent_execute_collectall_test.go (CollectAll multi-error aggregation)
//   - concurrent_timeout_cancel_test.go     (Timeout & context cancellation)
//   - concurrent_concurrency_limit_test.go  (MaxConcurrency enforcement & semaphore behavior)
//   - concurrent_child_access_test.go       (GetChildAt / GetChildNames / FindChildByName / GetChildren immutability)
//   - concurrent_path_resolution_test.go    (GetCurrentPath / GetByPath / ListAllPaths / FindByName / GetOrchestrationTree)
//   - concurrent_panic_recovery_test.go     (panic recovery behavior)
//   - concurrent_race_and_leak_test.go      (race detection & goroutine leak heuristic)  [skips in short mode]
//   - benchmark_test.go                     (benchmarks – existing)
//   - sync_test.go                          (advanced synchronization & allocation tests – slated for future split)
// This structure improves maintainability and targeted coverage.
