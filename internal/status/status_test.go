package status

import (
	"sync"
	"testing"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{NotStarted, "NotStarted"},
		{Running, "Running"},
		{Completed, "Completed"},
		{Cancelled, "Cancelled"},
		{Failed, "Failed"},
		{Status(999), "Unknown"},
	}

	for _, test := range tests {
		if got := test.status.String(); got != test.expected {
			t.Errorf("Status(%d).String() = %q, want %q", test.status, got, test.expected)
		}
	}
}

func TestStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, false},
		{Completed, true},
		{Cancelled, true},
		{Failed, true},
	}

	for _, test := range tests {
		if got := test.status.IsTerminal(); got != test.expected {
			t.Errorf("Status(%s).IsTerminal() = %v, want %v", test.status, got, test.expected)
		}
	}
}

func TestStatusIsActive(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{NotStarted, false},
		{Running, true},
		{Completed, false},
		{Cancelled, false},
		{Failed, false},
	}

	for _, test := range tests {
		if got := test.status.IsActive(); got != test.expected {
			t.Errorf("Status(%s).IsActive() = %v, want %v", test.status, got, test.expected)
		}
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.Get() != NotStarted {
		t.Errorf("New manager should start with NotStarted status, got %s", manager.Get())
	}
}

func TestManagerGet(t *testing.T) {
	manager := NewManager()

	if status := manager.Get(); status != NotStarted {
		t.Errorf("Expected NotStarted, got %s", status)
	}
}

func TestManagerSet(t *testing.T) {
	manager := NewManager()

	manager.Set(Running)
	if status := manager.Get(); status != Running {
		t.Errorf("Expected Running, got %s", status)
	}

	manager.Set(Completed)
	if status := manager.Get(); status != Completed {
		t.Errorf("Expected Completed, got %s", status)
	}
}

func TestManagerCompareAndSwap(t *testing.T) {
	manager := NewManager()

	// Successful swap
	if !manager.CompareAndSwap(NotStarted, Running) {
		t.Error("CompareAndSwap should have succeeded")
	}

	if status := manager.Get(); status != Running {
		t.Errorf("Expected Running, got %s", status)
	}

	// Failed swap
	if manager.CompareAndSwap(NotStarted, Completed) {
		t.Error("CompareAndSwap should have failed")
	}

	if status := manager.Get(); status != Running {
		t.Errorf("Status should still be Running, got %s", status)
	}
}

func TestManagerTryTransition(t *testing.T) {
	manager := NewManager()

	// Valid transition
	if !manager.TryTransition(NotStarted, Running) {
		t.Error("TryTransition should have succeeded")
	}

	// Invalid transition (wrong expected state)
	if manager.TryTransition(NotStarted, Completed) {
		t.Error("TryTransition should have failed")
	}
}

func TestManagerTransitionToRunning(t *testing.T) {
	manager := NewManager()

	// Should succeed from NotStarted
	if !manager.TransitionToRunning() {
		t.Error("TransitionToRunning should have succeeded")
	}

	if status := manager.Get(); status != Running {
		t.Errorf("Expected Running, got %s", status)
	}

	// Should fail from Running
	if manager.TransitionToRunning() {
		t.Error("TransitionToRunning should have failed from Running state")
	}
}

func TestManagerTransitionToCompleted(t *testing.T) {
	manager := NewManager()

	// Should fail from NotStarted
	if manager.TransitionToCompleted() {
		t.Error("TransitionToCompleted should have failed from NotStarted")
	}

	// Set to Running first
	manager.Set(Running)

	// Should succeed from Running
	if !manager.TransitionToCompleted() {
		t.Error("TransitionToCompleted should have succeeded from Running")
	}

	if status := manager.Get(); status != Completed {
		t.Errorf("Expected Completed, got %s", status)
	}
}

func TestManagerTransitionToCancelled(t *testing.T) {
	manager := NewManager()

	// Should succeed from NotStarted
	if !manager.TransitionToCancelled() {
		t.Error("TransitionToCancelled should have succeeded from NotStarted")
	}

	if status := manager.Get(); status != Cancelled {
		t.Errorf("Expected Cancelled, got %s", status)
	}

	// Reset and test from Running
	manager.Set(Running)
	if !manager.TransitionToCancelled() {
		t.Error("TransitionToCancelled should have succeeded from Running")
	}

	// Should fail from terminal state
	if manager.TransitionToCancelled() {
		t.Error("TransitionToCancelled should have failed from terminal state")
	}
}

func TestManagerTransitionToFailed(t *testing.T) {
	manager := NewManager()

	// Should succeed from NotStarted
	if !manager.TransitionToFailed() {
		t.Error("TransitionToFailed should have succeeded from NotStarted")
	}

	if status := manager.Get(); status != Failed {
		t.Errorf("Expected Failed, got %s", status)
	}

	// Should fail from terminal state
	if manager.TransitionToFailed() {
		t.Error("TransitionToFailed should have failed from terminal state")
	}
}

func TestManagerStatusCheckers(t *testing.T) {
	manager := NewManager()

	// Test NotStarted
	if !manager.IsNotStarted() {
		t.Error("IsNotStarted should return true")
	}

	if manager.IsRunning() {
		t.Error("IsRunning should return false")
	}

	if manager.IsCompleted() {
		t.Error("IsCompleted should return false")
	}

	if manager.IsCancelled() {
		t.Error("IsCancelled should return false")
	}

	if manager.IsFailed() {
		t.Error("IsFailed should return false")
	}

	if manager.IsTerminal() {
		t.Error("IsTerminal should return false")
	}

	if manager.IsActive() {
		t.Error("IsActive should return false")
	}

	// Test Running
	manager.Set(Running)

	if manager.IsNotStarted() {
		t.Error("IsNotStarted should return false")
	}

	if !manager.IsRunning() {
		t.Error("IsRunning should return true")
	}

	if manager.IsTerminal() {
		t.Error("IsTerminal should return false")
	}

	if !manager.IsActive() {
		t.Error("IsActive should return true")
	}

	// Test Completed
	manager.Set(Completed)

	if !manager.IsCompleted() {
		t.Error("IsCompleted should return true")
	}

	if !manager.IsTerminal() {
		t.Error("IsTerminal should return true")
	}

	if manager.IsActive() {
		t.Error("IsActive should return false")
	}
}

func TestManagerReset(t *testing.T) {
	manager := NewManager()

	manager.Set(Completed)
	manager.Reset()

	if status := manager.Get(); status != NotStarted {
		t.Errorf("Expected NotStarted after reset, got %s", status)
	}
}

func TestConcurrentStatusOperations(t *testing.T) {
	manager := NewManager()
	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent transitions
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			// Try to transition to running
			manager.TransitionToRunning()

			// Try various operations
			manager.Get()
			manager.IsRunning()
			manager.IsTerminal()
		}()
	}

	wg.Wait()

	// Should be in Running state (only one goroutine should succeed)
	if status := manager.Get(); status != Running {
		t.Errorf("Expected Running after concurrent operations, got %s", status)
	}
}

func TestConcurrentTransitionToCancelled(t *testing.T) {
	manager := NewManager()
	const numGoroutines = 50

	manager.Set(Running)

	var wg sync.WaitGroup

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			if manager.TransitionToCancelled() {
				// Use atomic operation to count successes
				// For simplicity in this test, we'll just check the final state
			}
		}()
	}

	wg.Wait()

	// Should be in Cancelled state
	if status := manager.Get(); status != Cancelled {
		t.Errorf("Expected Cancelled after concurrent transitions, got %s", status)
	}
}

// Benchmark tests
func BenchmarkManagerGet(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Get()
	}
}

func BenchmarkManagerSet(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Set(Running)
	}
}

func BenchmarkManagerCompareAndSwap(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.CompareAndSwap(NotStarted, Running)
		manager.Set(NotStarted) // Reset for next iteration
	}
}

func BenchmarkConcurrentStatusAccess(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.Get()
		}
	})
}

func BenchmarkConcurrentStatusTransitions(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.TransitionToRunning()
			manager.Reset()
		}
	})
}
