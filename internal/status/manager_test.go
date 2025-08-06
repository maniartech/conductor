package status

import (
	"sync"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.Get() != NotStarted {
		t.Error("New manager should start with NotStarted status")
	}
}

func TestManagerGetSet(t *testing.T) {
	manager := NewManager()

	// Test initial state
	if status := manager.Get(); status != NotStarted {
		t.Errorf("Expected NotStarted, got %v", status)
	}

	// Test setting status
	manager.Set(Running)
	if status := manager.Get(); status != Running {
		t.Errorf("Expected Running, got %v", status)
	}

	manager.Set(Completed)
	if status := manager.Get(); status != Completed {
		t.Errorf("Expected Completed, got %v", status)
	}
}

func TestManagerCompareAndSwap(t *testing.T) {
	manager := NewManager()

	// Should succeed when expected matches current
	if !manager.CompareAndSwap(NotStarted, Running) {
		t.Error("CompareAndSwap should succeed when expected matches current")
	}

	if status := manager.Get(); status != Running {
		t.Errorf("Status should be Running after successful swap, got %v", status)
	}

	// Should fail when expected doesn't match current
	if manager.CompareAndSwap(NotStarted, Completed) {
		t.Error("CompareAndSwap should fail when expected doesn't match current")
	}

	if status := manager.Get(); status != Running {
		t.Errorf("Status should remain Running after failed swap, got %v", status)
	}
}

func TestManagerTransitions(t *testing.T) {
	manager := NewManager()

	// Test TransitionToRunning
	if !manager.TransitionToRunning() {
		t.Error("TransitionToRunning should succeed from NotStarted")
	}

	if status := manager.Get(); status != Running {
		t.Errorf("Status should be Running, got %v", status)
	}

	// Should fail if already running
	if manager.TransitionToRunning() {
		t.Error("TransitionToRunning should fail when already running")
	}

	// Test TransitionToCompleted
	if !manager.TransitionToCompleted() {
		t.Error("TransitionToCompleted should succeed from Running")
	}

	if status := manager.Get(); status != Completed {
		t.Errorf("Status should be Completed, got %v", status)
	}

	// Should fail if already completed
	if manager.TransitionToCompleted() {
		t.Error("TransitionToCompleted should fail when already completed")
	}
}

func TestManagerTransitionToCancelled(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus Status
		expectSuccess bool
	}{
		{"from NotStarted", NotStarted, true},
		{"from Running", Running, true},
		{"from Completed", Completed, false},
		{"from Cancelled", Cancelled, false},
		{"from Failed", Failed, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := NewManager()
			manager.Set(test.initialStatus)

			success := manager.TransitionToCancelled()

			if success != test.expectSuccess {
				t.Errorf("Expected success=%v, got %v", test.expectSuccess, success)
			}

			if test.expectSuccess {
				if status := manager.Get(); status != Cancelled {
					t.Errorf("Status should be Cancelled, got %v", status)
				}
			}
		})
	}
}

func TestManagerTransitionToFailed(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus Status
		expectSuccess bool
	}{
		{"from NotStarted", NotStarted, true},
		{"from Running", Running, true},
		{"from Completed", Completed, false},
		{"from Cancelled", Cancelled, false},
		{"from Failed", Failed, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := NewManager()
			manager.Set(test.initialStatus)

			success := manager.TransitionToFailed()

			if success != test.expectSuccess {
				t.Errorf("Expected success=%v, got %v", test.expectSuccess, success)
			}

			if test.expectSuccess {
				if status := manager.Get(); status != Failed {
					t.Errorf("Status should be Failed, got %v", status)
				}
			}
		})
	}
}

func TestManagerStatusChecks(t *testing.T) {
	manager := NewManager()

	// Test NotStarted
	if !manager.IsNotStarted() {
		t.Error("Should be NotStarted initially")
	}

	// Test Running
	manager.Set(Running)
	if !manager.IsRunning() {
		t.Error("Should be Running")
	}
	if !manager.IsActive() {
		t.Error("Running should be active")
	}

	// Test Completed
	manager.Set(Completed)
	if !manager.IsCompleted() {
		t.Error("Should be Completed")
	}
	if !manager.IsTerminal() {
		t.Error("Completed should be terminal")
	}

	// Test Cancelled
	manager.Set(Cancelled)
	if !manager.IsCancelled() {
		t.Error("Should be Cancelled")
	}

	// Test Failed
	manager.Set(Failed)
	if !manager.IsFailed() {
		t.Error("Should be Failed")
	}
}

func TestManagerReset(t *testing.T) {
	manager := NewManager()

	// Change status
	manager.Set(Completed)

	// Reset should return to NotStarted
	manager.Reset()

	if status := manager.Get(); status != NotStarted {
		t.Errorf("Reset should return to NotStarted, got %v", status)
	}
}

func TestManagerConcurrentAccess(t *testing.T) {
	manager := NewManager()
	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines trying to transition to running
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			if manager.TransitionToRunning() {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Only one should succeed
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful transition, got %d", successCount)
	}

	if status := manager.Get(); status != Running {
		t.Errorf("Final status should be Running, got %v", status)
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
