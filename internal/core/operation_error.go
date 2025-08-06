// Package core provides the fundamental types and interfaces for the
// orchestrator library following Go best practices and KISS principles.
package core

import (
	"time"
)

// OperationError provides rich error information with metadata
type OperationError struct {
	Error     error
	Index     int
	Duration  time.Duration
	Timestamp time.Time
	OpID      string
	Stack     []byte
}
