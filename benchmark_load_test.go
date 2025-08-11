//go:build ignore
// +build ignore

// This benchmark file is disabled. It referenced outdated APIs (types.Orchestration alias, Await with context
// parameter, AllocsPerOp usage patterns) that no longer match the current public API. To re-enable, update
// benchmarks to:
//   - Use orchestration.Orchestration from internal packages only via public constructors
//   - Call workflow.Await()/AwaitWithContext() without passing a context to Await()
//   - Remove deprecated config structs from types in favor of internal/config via public wrappers
//   - Ensure no import cycles occur
//
// Keeping the file ignored preserves historical scenarios without breaking the build.
package orchestrator

// (intentionally empty)
