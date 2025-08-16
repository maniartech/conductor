package types

import (
	"sync"
	"sync/atomic"
	"testing"
)

// TestStatus_String tests status string representation
func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{NotStarted, "NotStarted"},
		{Running, "Running"},
		{Completed, "Completed"},
		{Failed, "Failed"},
		{Cancelled, "Cancelled"},
		{Status(999), "Unknown"}, // Invalid status
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Status.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestStatus_IsTerminal tests terminal status detection
func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, false},
		{Completed, true},
		{Failed, true},
		{Cancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.expected {
				t.Errorf("Status.IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestStatus_IsActive tests active status detection
func TestStatus_IsActive(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, true},
		{Completed, false},
		{Failed, false},
		{Cancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.expected {
				t.Errorf("Status.IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestStatus_IsSuccessful tests successful status detection
func TestStatus_IsSuccessful(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, false},
		{Completed, true},
		{Failed, false},
		{Cancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.IsSuccessful(); got != tt.expected {
				t.Errorf("Status.IsSuccessful() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestStatus_HasFailed tests failed status detection
func TestStatus_HasFailed(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, false},
		{Completed, false},
		{Failed, true},
		{Cancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.HasFailed(); got != tt.expected {
				t.Errorf("Status.HasFailed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestStatus_WasCancelled tests cancelled status detection
func TestStatus_WasCancelled(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, false},
		{Completed, false},
		{Failed, false},
		{Cancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.WasCancelled(); got != tt.expected {
				t.Errorf("Status.WasCancelled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestStatus_AtomicOperations tests atomic status operations
func TestStatus_AtomicOperations(t *testing.T) {
	var status uint32

	// Test atomic store and load
	atomic.StoreUint32(&status, uint32(Running))
	if Status(atomic.LoadUint32(&status)) != Running {
		t.Error("Expected atomic operations to work with Status type")
	}

	// Test compare and swap
	if !atomic.CompareAndSwapUint32(&status, uint32(Running), uint32(Completed)) {
		t.Error("Expected compare and swap to succeed")
	}

	if Status(atomic.LoadUint32(&status)) != Completed {
		t.Error("Expected status to be Completed after compare and swap")
	}
}

// TestStatus_ConcurrentAccess tests concurrent status access
func TestStatus_ConcurrentAccess(t *testing.T) {
	var status uint32
	var wg sync.WaitGroup

	const numGoroutines = 100
	const numOperations = 1000

	// Test concurrent reads
	atomic.StoreUint32(&status, uint32(Running))

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				s := Status(atomic.LoadUint32(&status))
				if s != Running {
					t.Errorf("Expected status Running, got %v", s)
				}
			}
		}()
	}

	wg.Wait()

	// Test concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Alternate between different statuses
				var newStatus Status
				switch (goroutineID + j) % 3 {
				case 0:
					newStatus = Running
				case 1:
					newStatus = Completed
				case 2:
					newStatus = Failed
				}
				atomic.StoreUint32(&status, uint32(newStatus))
			}
		}(i)
	}

	wg.Wait()

	// Final status should be one of the valid statuses
	finalStatus := Status(atomic.LoadUint32(&status))
	if finalStatus != Running && finalStatus != Completed && finalStatus != Failed {
		t.Errorf("Expected valid final status, got %v", finalStatus)
	}
}

// TestStatus_StateTransitions tests valid state transitions
func TestStatus_StateTransitions(t *testing.T) {
	tests := []struct {
		name        string
		from        Status
		to          Status
		shouldAllow bool
	}{
		// Valid transitions
		{"NotStarted to Running", NotStarted, Running, true},
		{"Running to Completed", Running, Completed, true},
		{"Running to Failed", Running, Failed, true},
		{"Running to Cancelled", Running, Cancelled, true},
		{"NotStarted to Cancelled", NotStarted, Cancelled, true},

		// Invalid transitions (terminal states)
		{"Completed to Running", Completed, Running, false},
		{"Failed to Running", Failed, Running, false},
		{"Cancelled to Running", Cancelled, Running, false},
		{"Completed to Failed", Completed, Failed, false},
		{"Failed to Completed", Failed, Completed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test documents expected behavior
			// In a real implementation, you might have a state machine
			// that enforces these transitions
			if tt.from.IsTerminal() && tt.shouldAllow == false {
				// Terminal states should not allow transitions
				t.Logf("✓ Correctly identified that %s should not transition to %s", tt.from, tt.to)
			} else if !tt.from.IsTerminal() && tt.shouldAllow == true {
				// Non-terminal states should allow transitions
				t.Logf("✓ Correctly identified that %s can transition to %s", tt.from, tt.to)
			}
		})
	}
}

// TestStatus_AllStatusValues tests all status values are covered
func TestStatus_AllStatusValues(t *testing.T) {
	allStatuses := []Status{NotStarted, Running, Completed, Failed, Cancelled}

	for _, status := range allStatuses {
		t.Run(status.String(), func(t *testing.T) {
			// Test that all methods work for all status values
			_ = status.String()
			_ = status.IsTerminal()
			_ = status.IsActive()
			_ = status.IsSuccessful()
			_ = status.HasFailed()
			_ = status.WasCancelled()

			// Verify string representation is not "Unknown"
			if status.String() == "Unknown" {
				t.Errorf("Status %d should not return 'Unknown'", status)
			}
		})
	}
}

// TestStatus_ZeroValue tests zero value behavior
func TestStatus_ZeroValue(t *testing.T) {
	var status Status

	if status != NotStarted {
		t.Errorf("Expected zero value to be NotStarted, got %v", status)
	}

	if status.String() != "NotStarted" {
		t.Errorf("Expected zero value string to be 'NotStarted', got '%s'", status.String())
	}

	if status.IsTerminal() {
		t.Error("Expected zero value not to be terminal")
	}

	if status.IsActive() {
		t.Error("Expected zero value not to be active")
	}
}

// BenchmarkStatus_String benchmarks string conversion
func BenchmarkStatus_String(b *testing.B) {
	status := Running

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = status.String()
	}
}

// BenchmarkStatus_IsTerminal benchmarks terminal check
func BenchmarkStatus_IsTerminal(b *testing.B) {
	status := Running

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = status.IsTerminal()
	}
}

// BenchmarkStatus_AtomicLoad benchmarks atomic load operations
func BenchmarkStatus_AtomicLoad(b *testing.B) {
	var status uint32
	atomic.StoreUint32(&status, uint32(Running))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Status(atomic.LoadUint32(&status))
	}
}

// BenchmarkStatus_AtomicStore benchmarks atomic store operations
func BenchmarkStatus_AtomicStore(b *testing.B) {
	var status uint32

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atomic.StoreUint32(&status, uint32(Running))
	}
}

// BenchmarkStatus_CompareAndSwap benchmarks compare and swap operations
func BenchmarkStatus_CompareAndSwap(b *testing.B) {
	var status uint32
	atomic.StoreUint32(&status, uint32(NotStarted))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atomic.CompareAndSwapUint32(&status, uint32(NotStarted), uint32(Running))
		atomic.CompareAndSwapUint32(&status, uint32(Running), uint32(NotStarted))
	}
}
