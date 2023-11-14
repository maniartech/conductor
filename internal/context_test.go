package internal_test

import (
	"context"
	"sync"
	"testing"

	"github.com/maniartech/async/internal"
)

func TestSetAndGet(t *testing.T) {
	ctx := internal.NewContext(context.Background())
	key := "testKey"
	value := "testValue"

	ctx.Set(key, value)
	gotValue, ok := ctx.Get(key)

	if !ok {
		t.Errorf("Expected to get a value for key %s", key)
	}

	if gotValue != value {
		t.Errorf("Expected value %v, got %v", value, gotValue)
	}
}

func TestThreadSafety(t *testing.T) {
	ctx := internal.NewContext(context.Background())
	key := "concurrency"
	value := 0
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.Set(key, value)
			_, _ = ctx.Get(key)
		}()
	}

	wg.Wait()

	// Check if the final value is set correctly
	finalValue, ok := ctx.Get(key)
	if !ok || finalValue != value {
		t.Errorf("Thread safety test failed, expected final value %d, got %v", value, finalValue)
	}
}

func TestNewContextWithNilParent(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when creating context with nil parent, but did not panic")
		}
	}()

	_ = internal.NewContext(nil)
}

func TestContextInheritance(t *testing.T) {
	parent := context.WithValue(context.Background(), "parentKey", "parentValue")
	ctx := internal.NewContext(parent)

	// Test if the value from the parent context is accessible
	if v, ok := ctx.Value("parentKey").(string); !ok || v != "parentValue" {
		t.Errorf("Expected to inherit value from parent context")
	}
}
