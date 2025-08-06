package context

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/internal/config"
)

func TestNewContext(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewContext(cfg)

	if ctx == nil {
		t.Fatal("NewContext() returned nil")
	}

	if ctx.Config().ErrorStrategy != cfg.ErrorStrategy {
		t.Error("Context should inherit config")
	}

	if ctx.Done() == nil {
		t.Error("Done() should return a channel")
	}
}

func TestContextSetAndGet(t *testing.T) {
	config := config.DefaultConfig()
	ctx := NewContext(config)

	// Test setting and getting values
	ctx.Set("test", "value")
	ctx.Set("number", 42)

	if got := ctx.Get("test"); got != "value" {
		t.Errorf("Expected 'value', got %v", got)
	}

	if got := ctx.Get("number"); got != 42 {
		t.Errorf("Expected 42, got %v", got)
	}

	if got := ctx.Get("nonexistent"); got != nil {
		t.Errorf("Expected nil for nonexistent key, got %v", got)
	}
}

func TestContextCancel(t *testing.T) {
	config := config.DefaultConfig()
	ctx := NewContext(config)

	done := ctx.Done()

	// Should not be cancelled initially
	select {
	case <-done:
		t.Error("Context should not be cancelled initially")
	default:
		// Expected
	}

	// Cancel the context
	ctx.Cancel()

	// Should be cancelled now
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after Cancel()")
	}
}

func TestContextConcurrentAccess(t *testing.T) {
	config := config.DefaultConfig()
	ctx := NewContext(config)

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			ctx.Set(key, id)
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			ctx.Get(key)
			ctx.Config()
		}(i)
	}

	wg.Wait()
}

func TestContextInterface(t *testing.T) {
	// Test that contextImpl implements Context interface
	var _ Context = (*contextImpl)(nil)

	config := config.DefaultConfig()
	ctx := NewContext(config)

	// Test interface methods
	ctx.Set("test", "value")
	if got := ctx.Get("test"); got != "value" {
		t.Errorf("Expected 'value', got %v", got)
	}

	if cfg := ctx.Config(); cfg.ErrorStrategy != config.ErrorStrategy {
		t.Error("Config should match")
	}

	// Test cancellation
	done := ctx.Done()
	select {
	case <-done:
		t.Error("Should not be cancelled initially")
	default:
		// Expected
	}

	ctx.Cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Should be cancelled after Cancel()")
	}
}

// Benchmark tests for Context
func BenchmarkContextSet(b *testing.B) {
	config := config.DefaultConfig()
	ctx := NewContext(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Set("key", i)
	}
}

func BenchmarkContextGet(b *testing.B) {
	config := config.DefaultConfig()
	ctx := NewContext(config)
	ctx.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Get("key")
	}
}

func BenchmarkConcurrentContextAccess(b *testing.B) {
	config := config.DefaultConfig()
	ctx := NewContext(config)
	ctx.Set("key", "value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx.Get("key")
		}
	})
}
