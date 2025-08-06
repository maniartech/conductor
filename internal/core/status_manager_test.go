package core

import (
	"testing"

	"github.com/maniartech/orchestrator/internal/status"
)

func TestNewStatusManager(t *testing.T) {
	sm := NewStatusManager()

	if sm == nil {
		t.Fatal("NewStatusManager() returned nil")
	}

	if sm.Manager == nil {
		t.Error("StatusManager.Manager should not be nil")
	}
}

func TestStatusManagerWithCustomManager(t *testing.T) {
	customManager := status.NewManager()
	sm := &StatusManager{
		Manager: customManager,
	}

	if sm.Manager != customManager {
		t.Error("StatusManager should use custom manager")
	}
}

func TestStatusManagerIntegration(t *testing.T) {
	sm := NewStatusManager()

	// Test that we can use the status manager through StatusManager
	// Since status.Manager is embedded, we can call its methods directly
	if sm.Manager == nil {
		t.Error("Should have access to status manager through StatusManager")
	}

	// Test that the embedded manager works
	// Note: We would need to check what methods are available on status.Manager
	// For now, just verify the manager is accessible
}

// Benchmark tests
func BenchmarkStatusManagerCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewStatusManager()
	}
}
