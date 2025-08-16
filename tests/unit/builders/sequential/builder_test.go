package sequential

// This file intentionally left as an index after refactor.
// Granular tests for SequentialBuilder have been split into focused files:
//   - builder_constructor_test.go        (constructor & basic invariants)
//   - builder_fluent_api_test.go         (Named / With / ErrorBoundary chaining)
//   - builder_execute_failfast_test.go   (FailFast success & error scenarios)
//   - builder_execute_collectall_test.go (CollectAll, multi-error, panic, timeout, cancellation)
//   - builder_children_access_test.go    (GetChildAt / GetChildNames / FindChildByName / GetChildren immutability)
//   - builder_path_resolution_test.go    (GetCurrentPath / GetByPath / ListAllPaths / FindByName / GetOrchestrationTree)
//   - builder_bench_test.go              (benchmarks)
// Additional existing files cover dynamic naming, hierarchical naming, error handling, etc.
// This structure improves maintainability and targeted coverage.
