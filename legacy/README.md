# Legacy Orchestrator Code

This directory contains the original orchestrator implementation that was replaced by the new modular design in the `internal/` directory.

## Legacy Files

- `choreography.go` - Original orchestrator implementation with race conditions
- `activities_test.go` - Tests for the original orchestrator (contains race conditions)
- `processors.go` - Legacy processor implementations
- `choreographers.go` - Legacy choreographer implementations  
- `handlers.go` - Legacy handler implementations
- `go.go` - Legacy go routine management
- `status.go` - Legacy status management (replaced by `internal/orchestration/status.go`)
- `consts.go` - Legacy constants

## Why Moved to Legacy

These files were moved to the legacy directory because:

1. **Race Conditions**: The original implementation had multiple race conditions detected by `go test -race`
2. **Architecture**: The new modular architecture in `internal/` provides better separation of concerns
3. **Thread Safety**: The new implementation uses proper atomic operations and thread-safe patterns
4. **Maintainability**: The new code follows Go best practices and is easier to maintain

## New Implementation

The new orchestrator implementation can be found in:
- `internal/task/` - Task execution engine with atomic status management
- `internal/orchestration/` - Core orchestration interfaces and types
- `internal/config/` - Configuration management
- `internal/errors/` - Error handling and reporting
- `internal/result/` - Result management
- `internal/context/` - Context management
- `internal/pool/` - Object pooling for performance
- `internal/status/` - Status management

## Migration

If you need to reference the old implementation for any reason, the files are preserved here. However, all new development should use the new implementation in the `internal/` directory.