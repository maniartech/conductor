// Package sequential provides advanced error handling strategies for sequential orchestrations.
// This file contains specialized error handling utilities that provide rich error context,
// error boundary containment, and comprehensive error reporting capabilities.
package sequential

import (
	"github.com/maniartech/orchestrator/internal/errors"
)

// Sequential-specific helper functions for working with the common ErrorBoundaryHandler

// CreateSequentialErrorContext creates a rich error context for a sequential orchestration.
// This uses the common ErrorContext from the errors package.
//
// Parameters:
//   - sequentialName: Name of the sequential orchestration
//   - sequentialID: Unique ID of the sequential orchestration
//   - totalSteps: Total number of steps in the sequential
//   - errorStrategy: Error strategy being used
//   - errorBoundary: Error boundary if set
//
// Returns:
//   - *errors.ErrorContext: Rich error context information
func CreateSequentialErrorContext(sequentialName, sequentialID string, totalSteps int,
	errorStrategy errors.ErrorStrategy, errorBoundary *errors.ErrorStrategy) *errors.ErrorContext {

	return errors.CreateErrorContext(sequentialName, sequentialID, "sequential", totalSteps, errorStrategy, errorBoundary)
}
