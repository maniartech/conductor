package errors

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
