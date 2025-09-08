package orchestration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/pkg/types"
)

// MockOrchestration implements types.Orchestration for testing
type MockOrchestration struct {
	name          string
	orchType      string
	operationID   string
	config        *config.Config
	status        types.Status
	executed      bool
	result        any
	err           error
	execDelay     time.Duration
	errorStrategy errors.ErrorStrategy
}

func NewMockOrchestration(name, orchType string) *MockOrchestration {
	return &MockOrchestration{
		name:          name,
		orchType:      orchType,
		operationID:   fmt.Sprintf("mock-%s-%d", name, time.Now().UnixNano()),
		status:        types.NotStarted,
		executed:      false,
		errorStrategy: errors.FailFast,
	}
}

func (m *MockOrchestration) GetName() string               { return m.name }
func (m *MockOrchestration) GetType() string               { return m.orchType }
func (m *MockOrchestration) GetOperationID() string        { return m.operationID }
func (m *MockOrchestration) GetStatus() types.Status       { return m.status }
func (m *MockOrchestration) GetConfig() *config.Config     { return m.config }
func (m *MockOrchestration) SetName(name string)           { m.name = name }
func (m *MockOrchestration) SetConfig(cfg config.Config)   { m.config = &cfg }
func (m *MockOrchestration) SetStatus(status types.Status) { m.status = status }
func (m *MockOrchestration) CompareAndSwapStatus(old, new types.Status) bool {
	if m.status == old {
		m.status = new
		return true
	}
	return false
}

// Implement the builder pattern methods required by Orchestration interface
func (m *MockOrchestration) Named(name string) types.Orchestration {
	m.name = name
	return m
}

func (m *MockOrchestration) With(cfg config.Config) types.Orchestration {
	m.config = &cfg
	return m
}

func (m *MockOrchestration) ErrorBoundary(strategy errors.ErrorStrategy) types.Orchestration {
	m.errorStrategy = strategy
	return m
}

// PathResolver methods - minimal implementation for testing
func (m *MockOrchestration) GetByPath(path string) (types.Orchestration, error) {
	if path == m.name || path == m.operationID {
		return m, nil
	}
	return nil, fmt.Errorf("path not found: %s", path)
}

func (m *MockOrchestration) GetCurrentPath() string {
	return m.name
}

func (m *MockOrchestration) ListAllPaths() []string {
	return []string{m.name}
}

func (m *MockOrchestration) FindByName(name string) []types.PathMatch {
	if m.name == name {
		return []types.PathMatch{
			{
				Path:          m.name,
				Orchestration: m,
				Depth:         0,
				Parent:        "",
				Type:          m.orchType,
				Index:         0,
			},
		}
	}
	return []types.PathMatch{}
}

func (m *MockOrchestration) GetOrchestrationTree() *types.OrchestrationTree {
	return &types.OrchestrationTree{
		Name:          m.name,
		Path:          m.name,
		Type:          m.orchType,
		Depth:         0,
		Root:          nil,
		Parent:        nil,
		Children:      []*types.OrchestrationTree{},
		Orchestration: m,
	}
}

func (m *MockOrchestration) Query() *types.PathQuery {
	return types.NewPathQuery(m.GetOrchestrationTree())
}

func (m *MockOrchestration) WithDelay(delay time.Duration) *MockOrchestration {
	m.execDelay = delay
	return m
}

func (m *MockOrchestration) WithResult(result any) *MockOrchestration {
	m.result = result
	return m
}

func (m *MockOrchestration) WithError(err error) *MockOrchestration {
	m.err = err
	return m
}

func (m *MockOrchestration) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
	m.executed = true
	m.status = types.Running

	if m.execDelay > 0 {
		select {
		case <-time.After(m.execDelay):
		case <-ctx.Done():
			m.status = types.Cancelled
			return nil, ctx.Err()
		}
	}

	if m.err != nil {
		m.status = types.Completed
		return nil, m.err
	}

	m.status = types.Completed
	res := result.NewResult()
	if m.result != nil {
		res.Set("task_result", m.result)
	}
	return res, nil
}

// TestBaseContainerOrchestration_BasicOperations tests basic container operations
func TestBaseContainerOrchestration_BasicOperations(t *testing.T) {
	// Create test orchestrations
	orchestrations := []types.Orchestration{
		NewMockOrchestration("task1", "task").WithResult("result1"),
		NewMockOrchestration("task2", "task").WithResult("result2"),
		NewMockOrchestration("task3", "task").WithResult("result3"),
	}

	// Create BaseContainerOrchestration
	container := NewBaseContainerOrchestration("test-container", orchestrations)

	t.Run("GetChildCount", func(t *testing.T) {
		if count := container.GetChildCount(); count != 3 {
			t.Errorf("Expected child count 3, got %d", count)
		}
	})

	t.Run("IsEmpty", func(t *testing.T) {
		if container.IsEmpty() {
			t.Error("Expected container not to be empty")
		}

		// Test empty container
		empty := NewBaseContainerOrchestration("empty", []types.Orchestration{})
		if !empty.IsEmpty() {
			t.Error("Expected empty container to be empty")
		}
	})

	t.Run("GetChildAt", func(t *testing.T) {
		// Test valid indices
		for i := 0; i < 3; i++ {
			child, err := container.GetChildAt(i)
			if err != nil {
				t.Errorf("Unexpected error at index %d: %v", i, err)
			}
			if child.GetName() != fmt.Sprintf("task%d", i+1) {
				t.Errorf("Expected name task%d, got %s", i+1, child.GetName())
			}
		}

		// Test invalid indices
		_, err := container.GetChildAt(-1)
		if err == nil {
			t.Error("Expected error for negative index")
		}

		_, err = container.GetChildAt(3)
		if err == nil {
			t.Error("Expected error for out of bounds index")
		}
	})

	t.Run("GetFirstChild and GetLastChild", func(t *testing.T) {
		first, err := container.GetFirstChild()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if first.GetName() != "task1" {
			t.Errorf("Expected first child name task1, got %s", first.GetName())
		}

		last, err := container.GetLastChild()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if last.GetName() != "task3" {
			t.Errorf("Expected last child name task3, got %s", last.GetName())
		}

		// Test empty container
		empty := NewBaseContainerOrchestration("empty", []types.Orchestration{})
		_, err = empty.GetFirstChild()
		if err == nil {
			t.Error("Expected error for empty container GetFirstChild")
		}

		_, err = empty.GetLastChild()
		if err == nil {
			t.Error("Expected error for empty container GetLastChild")
		}
	})

	t.Run("GetContainerType", func(t *testing.T) {
		if containerType := container.GetContainerType(); containerType != "test-container" {
			t.Errorf("Expected container type 'test-container', got %s", containerType)
		}
	})

	t.Run("GetChildren", func(t *testing.T) {
		children := container.GetChildren()
		if len(children) != 3 {
			t.Errorf("Expected 3 children, got %d", len(children))
		}

		// Verify it returns a copy
		children[0] = nil
		originalChild, _ := container.GetChildAt(0)
		if originalChild == nil {
			t.Error("Original children were modified, copy not returned")
		}
	})
}

// TestBaseContainerOrchestration_NameBasedOperations tests name-based child access
func TestBaseContainerOrchestration_NameBasedOperations(t *testing.T) {
	orchestrations := []types.Orchestration{
		NewMockOrchestration("auth", "task"),
		NewMockOrchestration("validate", "validator"),
		NewMockOrchestration("process", "task"),
		NewMockOrchestration("validate", "validator"), // Duplicate name
	}

	container := NewBaseContainerOrchestration("test-container", orchestrations)

	t.Run("GetChildByName", func(t *testing.T) {
		// Test existing name
		child, err := container.GetChildByName("auth")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if child.GetName() != "auth" {
			t.Errorf("Expected name 'auth', got %s", child.GetName())
		}

		// Test duplicate name (should return first)
		child, err = container.GetChildByName("validate")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if child.GetType() != "validator" {
			t.Errorf("Expected first validator, got type %s", child.GetType())
		}

		// Test non-existing name
		_, err = container.GetChildByName("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent name")
		}
	})

	t.Run("GetChildrenByName", func(t *testing.T) {
		// Test single match
		children := container.GetChildrenByName("auth")
		if len(children) != 1 {
			t.Errorf("Expected 1 child, got %d", len(children))
		}

		// Test multiple matches
		children = container.GetChildrenByName("validate")
		if len(children) != 2 {
			t.Errorf("Expected 2 children, got %d", len(children))
		}

		// Test no matches
		children = container.GetChildrenByName("nonexistent")
		if len(children) != 0 {
			t.Errorf("Expected 0 children, got %d", len(children))
		}
	})

	t.Run("ContainsChildWithName", func(t *testing.T) {
		if !container.ContainsChildWithName("auth") {
			t.Error("Expected to contain child with name 'auth'")
		}

		if container.ContainsChildWithName("nonexistent") {
			t.Error("Expected not to contain child with name 'nonexistent'")
		}
	})
}

// TestBaseContainerOrchestration_OperationIDOperations tests operation ID-based access
func TestBaseContainerOrchestration_OperationIDOperations(t *testing.T) {
	orchestrations := []types.Orchestration{
		NewMockOrchestration("task1", "task"),
		NewMockOrchestration("task2", "task"),
		NewMockOrchestration("task3", "task"),
	}

	container := NewBaseContainerOrchestration("test-container", orchestrations)

	t.Run("GetChildByOperationID", func(t *testing.T) {
		// Get a valid operation ID
		expectedChild, _ := container.GetChildAt(1)
		expectedOpID := expectedChild.GetOperationID()

		// Test retrieval by operation ID
		child, err := container.GetChildByOperationID(expectedOpID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if child.GetOperationID() != expectedOpID {
			t.Errorf("Expected operation ID %s, got %s", expectedOpID, child.GetOperationID())
		}

		// Test non-existing operation ID
		_, err = container.GetChildByOperationID("nonexistent-op-id")
		if err == nil {
			t.Error("Expected error for non-existent operation ID")
		}
	})
}

// TestBaseContainerOrchestration_TypeBasedOperations tests type-based child access
func TestBaseContainerOrchestration_TypeBasedOperations(t *testing.T) {
	orchestrations := []types.Orchestration{
		NewMockOrchestration("task1", "task"),
		NewMockOrchestration("validator1", "validator"),
		NewMockOrchestration("task2", "task"),
		NewMockOrchestration("processor1", "processor"),
		NewMockOrchestration("validator2", "validator"),
	}

	container := NewBaseContainerOrchestration("test-container", orchestrations)

	t.Run("FindChildrenByType", func(t *testing.T) {
		// Test tasks
		tasks := container.FindChildrenByType("task")
		if len(tasks) != 2 {
			t.Errorf("Expected 2 tasks, got %d", len(tasks))
		}

		// Test validators
		validators := container.FindChildrenByType("validator")
		if len(validators) != 2 {
			t.Errorf("Expected 2 validators, got %d", len(validators))
		}

		// Test processors
		processors := container.FindChildrenByType("processor")
		if len(processors) != 1 {
			t.Errorf("Expected 1 processor, got %d", len(processors))
		}

		// Test non-existing type
		unknown := container.FindChildrenByType("unknown")
		if len(unknown) != 0 {
			t.Errorf("Expected 0 unknown types, got %d", len(unknown))
		}
	})
}

// TestBaseContainerOrchestration_ContainsOperations tests containment checks
func TestBaseContainerOrchestration_ContainsOperations(t *testing.T) {
	orchestrations := []types.Orchestration{
		NewMockOrchestration("task1", "task"),
		NewMockOrchestration("task2", "task"),
	}

	container := NewBaseContainerOrchestration("test-container", orchestrations)

	t.Run("ContainsChild", func(t *testing.T) {
		// Test existing child
		child, _ := container.GetChildAt(0)
		if !container.ContainsChild(child) {
			t.Error("Expected to contain existing child")
		}

		// Test non-existing child
		external := NewMockOrchestration("external", "task")
		if container.ContainsChild(external) {
			t.Error("Expected not to contain external child")
		}

		// Test nil
		if container.ContainsChild(nil) {
			t.Error("Expected not to contain nil child")
		}
	})
}

// TestBaseContainerOrchestration_DeepSearch tests recursive search operations
func TestBaseContainerOrchestration_DeepSearch(t *testing.T) {
	// Create nested structure
	subContainer := NewBaseContainerOrchestration("sub", []types.Orchestration{
		NewMockOrchestration("deep-task", "task"),
		NewMockOrchestration("deep-validator", "validator"),
	})

	orchestrations := []types.Orchestration{
		NewMockOrchestration("task1", "task"),
		subContainer,
		NewMockOrchestration("task2", "task"),
	}

	container := NewBaseContainerOrchestration("main", orchestrations)

	t.Run("FindDescendantByName", func(t *testing.T) {
		// Test finding direct child
		descendant, err := container.FindDescendantByName("task1")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if descendant.GetName() != "task1" {
			t.Errorf("Expected name 'task1', got %s", descendant.GetName())
		}

		// Test finding nested child
		descendant, err = container.FindDescendantByName("deep-task")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if descendant.GetName() != "deep-task" {
			t.Errorf("Expected name 'deep-task', got %s", descendant.GetName())
		}

		// Test non-existing descendant
		_, err = container.FindDescendantByName("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent descendant")
		}
	})

	t.Run("FindDescendantByOperationID", func(t *testing.T) {
		// Get operation ID from nested child
		subChild, _ := subContainer.GetChildAt(0)
		opID := subChild.GetOperationID()

		// Test finding by operation ID
		descendant, err := container.FindDescendantByOperationID(opID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if descendant.GetOperationID() != opID {
			t.Errorf("Expected operation ID %s, got %s", opID, descendant.GetOperationID())
		}

		// Test non-existing operation ID
		_, err = container.FindDescendantByOperationID("nonexistent-op-id")
		if err == nil {
			t.Error("Expected error for non-existent operation ID")
		}
	})

	t.Run("FindAllDescendantsByType", func(t *testing.T) {
		// Test finding all tasks (should include direct and nested)
		tasks := container.FindAllDescendantsByType("task")
		expectedTasks := 3 // task1, deep-task, task2
		if len(tasks) != expectedTasks {
			t.Errorf("Expected %d tasks, got %d", expectedTasks, len(tasks))
		}

		// Test finding all validators (nested only)
		validators := container.FindAllDescendantsByType("validator")
		if len(validators) != 1 {
			t.Errorf("Expected 1 validator, got %d", len(validators))
		}

		// Test non-existing type
		unknown := container.FindAllDescendantsByType("unknown")
		if len(unknown) != 0 {
			t.Errorf("Expected 0 unknown types, got %d", len(unknown))
		}
	})
}

// TestBaseContainerOrchestration_ChildModification tests child modification operations
func TestBaseContainerOrchestration_ChildModification(t *testing.T) {
	initialChildren := []types.Orchestration{
		NewMockOrchestration("task1", "task"),
		NewMockOrchestration("task2", "task"),
	}

	container := NewBaseContainerOrchestration("test-container", initialChildren)

	t.Run("AddChild", func(t *testing.T) {
		newTask := NewMockOrchestration("task3", "task")
		err := container.AddChild(newTask)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if container.GetChildCount() != 3 {
			t.Errorf("Expected 3 children after add, got %d", container.GetChildCount())
		}

		// Verify the child was added
		child, err := container.GetChildAt(2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if child.GetName() != "task3" {
			t.Errorf("Expected added child name 'task3', got %s", child.GetName())
		}

		// Test adding nil child
		err = container.AddChild(nil)
		if err == nil {
			t.Error("Expected error when adding nil child")
		}
	})

	t.Run("RemoveChild", func(t *testing.T) {
		// Get a child to remove
		childToRemove, _ := container.GetChildAt(0)

		err := container.RemoveChild(childToRemove)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if container.GetChildCount() != 2 {
			t.Errorf("Expected 2 children after remove, got %d", container.GetChildCount())
		}

		// Verify the child was removed
		if container.ContainsChild(childToRemove) {
			t.Error("Child was not removed")
		}

		// Test removing non-existing child
		external := NewMockOrchestration("external", "task")
		err = container.RemoveChild(external)
		if err == nil {
			t.Error("Expected error when removing non-existing child")
		}

		// Test removing nil child
		err = container.RemoveChild(nil)
		if err == nil {
			t.Error("Expected error when removing nil child")
		}
	})

	t.Run("RemoveChildAt", func(t *testing.T) {
		initialCount := container.GetChildCount()

		err := container.RemoveChildAt(0)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if container.GetChildCount() != initialCount-1 {
			t.Errorf("Expected %d children after remove at index, got %d", initialCount-1, container.GetChildCount())
		}

		// Test removing at invalid index
		err = container.RemoveChildAt(-1)
		if err == nil {
			t.Error("Expected error when removing at negative index")
		}

		err = container.RemoveChildAt(container.GetChildCount())
		if err == nil {
			t.Error("Expected error when removing at out-of-bounds index")
		}
	})

	t.Run("ReplaceChild", func(t *testing.T) {
		oldChild, _ := container.GetChildAt(0)
		newChild := NewMockOrchestration("replacement", "task")

		err := container.ReplaceChild(oldChild, newChild)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify replacement
		replacedChild, _ := container.GetChildAt(0)
		if replacedChild.GetName() != "replacement" {
			t.Errorf("Expected replaced child name 'replacement', got %s", replacedChild.GetName())
		}

		if container.ContainsChild(oldChild) {
			t.Error("Old child still present after replacement")
		}

		// Test replacing with nil
		err = container.ReplaceChild(newChild, nil)
		if err == nil {
			t.Error("Expected error when replacing with nil child")
		}

		// Test replacing nil child
		err = container.ReplaceChild(nil, NewMockOrchestration("another", "task"))
		if err == nil {
			t.Error("Expected error when replacing nil child")
		}
	})

	t.Run("ClearChildren", func(t *testing.T) {
		container.ClearChildren()

		if !container.IsEmpty() {
			t.Error("Expected container to be empty after clear")
		}

		if container.GetChildCount() != 0 {
			t.Errorf("Expected 0 children after clear, got %d", container.GetChildCount())
		}
	})

	t.Run("GetChildIndex", func(t *testing.T) {
		// Repopulate for this test
		newChildren := []types.Orchestration{
			NewMockOrchestration("a", "task"),
			NewMockOrchestration("b", "task"),
		}
		container = NewBaseContainerOrchestration("test", newChildren)

		child, _ := container.GetChildAt(1)
		index, err := container.GetChildIndex(child)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if index != 1 {
			t.Errorf("Expected index 1, got %d", index)
		}

		// Test non-existing child
		external := NewMockOrchestration("external", "task")
		_, err = container.GetChildIndex(external)
		if err == nil {
			t.Error("Expected error for non-existing child index")
		}

		// Test nil child
		_, err = container.GetChildIndex(nil)
		if err == nil {
			t.Error("Expected error for nil child index")
		}
	})
}

// TestBaseContainerOrchestration_ThreadSafety tests thread safety of container operations
func TestBaseContainerOrchestration_ThreadSafety(t *testing.T) {
	const numGoroutines = 10
	const numOperations = 100

	orchestrations := []types.Orchestration{
		NewMockOrchestration("task1", "task"),
		NewMockOrchestration("task2", "task"),
		NewMockOrchestration("task3", "task"),
	}

	container := NewBaseContainerOrchestration("test-container", orchestrations)

	t.Run("ConcurrentReadOperations", func(t *testing.T) {
		var wg sync.WaitGroup

		// Start multiple goroutines performing read operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					// Perform various read operations
					container.GetChildCount()
					container.IsEmpty()
					container.GetChildren()
					container.FindChildrenByType("task")
					if j%2 == 0 {
						container.GetChildAt(0)
					} else {
						container.GetChildByName("task1")
					}
				}
			}()
		}

		wg.Wait()
		// If we reach here without deadlock or panic, the test passes
	})

	t.Run("ConcurrentWriteOperations", func(t *testing.T) {
		var wg sync.WaitGroup

		// Start multiple goroutines performing write operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ { // Fewer operations to prevent excessive children
					taskName := fmt.Sprintf("concurrent-task-%d-%d", id, j)
					newTask := NewMockOrchestration(taskName, "task")

					// Add child
					container.AddChild(newTask)

					// Try to find it
					found, _ := container.GetChildByName(taskName)
					if found != nil && found.GetName() == taskName {
						// Success - remove it
						container.RemoveChild(found)
					}
				}
			}(i)
		}

		wg.Wait()
		// Container should still be functional
		if container.GetChildCount() < 3 {
			t.Error("Original children should still be present")
		}
	})
}

// TestBaseContainerOrchestration_Validation tests validation operations
func TestBaseContainerOrchestration_Validation(t *testing.T) {
	t.Run("ValidateChildren_ValidContainer", func(t *testing.T) {
		orchestrations := []types.Orchestration{
			NewMockOrchestration("task1", "task"),
			NewMockOrchestration("task2", "validator"),
			NewMockOrchestration("task3", "processor"),
		}

		container := NewBaseContainerOrchestration("test-container", orchestrations)
		err := container.ValidateChildren()
		if err != nil {
			t.Errorf("Unexpected validation error: %v", err)
		}
	})

	t.Run("ValidateChildren_EmptyContainer", func(t *testing.T) {
		container := NewBaseContainerOrchestration("empty", []types.Orchestration{})
		err := container.ValidateChildren()
		if err != nil {
			t.Errorf("Empty container should be valid: %v", err)
		}
	})

	t.Run("GetChildrenSummary", func(t *testing.T) {
		orchestrations := []types.Orchestration{
			NewMockOrchestration("task1", "task"),
			NewMockOrchestration("validator1", "validator"),
			NewMockOrchestration("task2", "task"),
		}

		container := NewBaseContainerOrchestration("test-container", orchestrations)
		summary := container.GetChildrenSummary()

		// Check summary structure
		if summary["count"] != 3 {
			t.Errorf("Expected count 3, got %v", summary["count"])
		}

		if summary["is_empty"] != false {
			t.Errorf("Expected is_empty false, got %v", summary["is_empty"])
		}

		// Check type distribution
		typeDistribution, ok := summary["type_distribution"].(map[string]int)
		if !ok {
			t.Error("Expected type_distribution to be map[string]int")
		} else {
			if typeDistribution["task"] != 2 {
				t.Errorf("Expected 2 tasks in distribution, got %d", typeDistribution["task"])
			}
			if typeDistribution["validator"] != 1 {
				t.Errorf("Expected 1 validator in distribution, got %d", typeDistribution["validator"])
			}
		}

		// Check first and last child info
		firstChild, ok := summary["first_child"].(map[string]interface{})
		if !ok {
			t.Error("Expected first_child to be map[string]interface{}")
		} else {
			if firstChild["name"] != "task1" {
				t.Errorf("Expected first child name 'task1', got %v", firstChild["name"])
			}
		}

		lastChild, ok := summary["last_child"].(map[string]interface{})
		if !ok {
			t.Error("Expected last_child to be map[string]interface{}")
		} else {
			if lastChild["name"] != "task2" {
				t.Errorf("Expected last child name 'task2', got %v", lastChild["name"])
			}
		}
	})

	t.Run("GetChildrenSummary_EmptyContainer", func(t *testing.T) {
		container := NewBaseContainerOrchestration("empty", []types.Orchestration{})
		summary := container.GetChildrenSummary()

		if summary["count"] != 0 {
			t.Errorf("Expected count 0, got %v", summary["count"])
		}

		if summary["is_empty"] != true {
			t.Errorf("Expected is_empty true, got %v", summary["is_empty"])
		}

		// Should not have first_child or last_child keys
		if _, exists := summary["first_child"]; exists {
			t.Error("Empty container should not have first_child")
		}

		if _, exists := summary["last_child"]; exists {
			t.Error("Empty container should not have last_child")
		}
	})

	t.Run("String", func(t *testing.T) {
		orchestrations := []types.Orchestration{
			NewMockOrchestration("task1", "task"),
			NewMockOrchestration("task2", "task"),
			NewMockOrchestration("task3", "task"),
		}

		container := NewBaseContainerOrchestration("test-container", orchestrations)
		str := container.String()

		expected := "test-container[3 children]"
		if str != expected {
			t.Errorf("Expected string '%s', got '%s'", expected, str)
		}

		// Test empty container
		empty := NewBaseContainerOrchestration("empty", []types.Orchestration{})
		emptyStr := empty.String()
		expectedEmpty := "empty[0 children]"
		if emptyStr != expectedEmpty {
			t.Errorf("Expected string '%s', got '%s'", expectedEmpty, emptyStr)
		}
	})
}

// TestBaseContainerOrchestration_Performance tests performance characteristics
func TestBaseContainerOrchestration_Performance(t *testing.T) {
	// Create a large container for performance testing
	const numChildren = 10000
	orchestrations := make([]types.Orchestration, numChildren)
	for i := 0; i < numChildren; i++ {
		orchestrations[i] = NewMockOrchestration(fmt.Sprintf("task-%d", i), "task")
	}

	container := NewBaseContainerOrchestration("large-container", orchestrations)

	t.Run("GetChildCount_Performance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 10000; i++ {
			container.GetChildCount()
		}
		duration := time.Since(start)

		// Should be very fast (O(1))
		if duration > time.Millisecond*10 {
			t.Errorf("GetChildCount took too long: %v", duration)
		}
	})

	t.Run("GetChildAt_Performance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 1000; i++ {
			container.GetChildAt(i % numChildren)
		}
		duration := time.Since(start)

		// Should be very fast (O(1))
		if duration > time.Millisecond*10 {
			t.Errorf("GetChildAt took too long: %v", duration)
		}
	})

	t.Run("GetChildByOperationID_Performance", func(t *testing.T) {
		// Get some operation IDs to test with
		testOpIDs := make([]string, 100)
		for i := 0; i < 100; i++ {
			child, _ := container.GetChildAt(i * (numChildren / 100))
			testOpIDs[i] = child.GetOperationID()
		}

		start := time.Now()
		for _, opID := range testOpIDs {
			container.GetChildByOperationID(opID)
		}
		duration := time.Since(start)

		// Should be fast with indexing (O(1) per lookup)
		if duration > time.Millisecond*100 {
			t.Errorf("GetChildByOperationID lookups took too long: %v", duration)
		}
	})

	t.Run("FindChildrenByType_Performance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 100; i++ {
			container.FindChildrenByType("task")
		}
		duration := time.Since(start)

		// Should be reasonably fast with indexing
		if duration > time.Millisecond*500 {
			t.Errorf("FindChildrenByType took too long: %v", duration)
		}
	})
}

// TestBaseContainerOrchestration_EdgeCases tests edge cases and error conditions
func TestBaseContainerOrchestration_EdgeCases(t *testing.T) {
	t.Run("NilConstructorParameters", func(t *testing.T) {
		// Test with nil slice
		container := NewBaseContainerOrchestration("test", nil)
		if !container.IsEmpty() {
			t.Error("Container with nil slice should be empty")
		}

		// Test with empty slice
		container = NewBaseContainerOrchestration("test", []types.Orchestration{})
		if !container.IsEmpty() {
			t.Error("Container with empty slice should be empty")
		}
	})

	t.Run("OutOfBoundsAccess", func(t *testing.T) {
		container := NewBaseContainerOrchestration("test", []types.Orchestration{
			NewMockOrchestration("single", "task"),
		})

		// Test negative index
		_, err := container.GetChildAt(-1)
		if err == nil {
			t.Error("Expected error for negative index")
		}

		// Test too large index
		_, err = container.GetChildAt(1)
		if err == nil {
			t.Error("Expected error for too large index")
		}

		// Test edge case: exactly at boundary
		_, err = container.GetChildAt(0)
		if err != nil {
			t.Errorf("Valid index should not return error: %v", err)
		}
	})

	t.Run("EmptyContainerOperations", func(t *testing.T) {
		container := NewBaseContainerOrchestration("empty", []types.Orchestration{})

		// All these should handle empty containers gracefully
		if container.GetChildCount() != 0 {
			t.Error("Empty container should have 0 children")
		}

		children := container.GetChildren()
		if len(children) != 0 {
			t.Error("Empty container should return empty slice")
		}

		byType := container.FindChildrenByType("any")
		if len(byType) != 0 {
			t.Error("Empty container should return empty results for type search")
		}

		byName := container.GetChildrenByName("any")
		if len(byName) != 0 {
			t.Error("Empty container should return empty results for name search")
		}

		if container.ContainsChildWithName("any") {
			t.Error("Empty container should not contain any names")
		}
	})
}
