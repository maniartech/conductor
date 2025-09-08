// Package orchestration provides base abstractions for orchestration implementations.
// This file contains the BaseContainerOrchestration struct which provides
// foundational implementation for all container orchestration types.
package orchestration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/maniartech/orchestrator/pkg/config"
	"github.com/maniartech/orchestrator/pkg/errors"
	"github.com/maniartech/orchestrator/pkg/result"
	"github.com/maniartech/orchestrator/pkg/types"
)

// BaseContainerOrchestration provides the foundational implementation for all
// container orchestration types. It handles common functionality like child
// management, thread safety, and performance optimizations.
type BaseContainerOrchestration struct {
	*BaseOrchestrationBuilder

	// Thread-safe child management
	children       atomic.Value // []types.Orchestration - immutable slice
	childrenMutex  sync.RWMutex // Protects write operations
	childNameIndex atomic.Value // map[string][]int - name to indices mapping
	childTypeIndex atomic.Value // map[string][]int - type to indices mapping
	childOpIDIndex atomic.Value // map[string]int - operation ID to index mapping

	// Container metadata
	containerType string // Container type identifier

	// Performance optimizations
	childCount atomic.Int64 // Cached child count
	isEmpty    atomic.Bool  // Cached empty state

	// Path resolution support
	pathResolver *PathResolverBase
}

// NewBaseContainerOrchestration creates a new base container orchestration.
func NewBaseContainerOrchestration(containerType string, orchestrations []types.Orchestration) *BaseContainerOrchestration {
	base := &BaseContainerOrchestration{
		BaseOrchestrationBuilder: NewBaseOrchestrationBuilder(containerType),
		containerType:            containerType,
		pathResolver:             NewPathResolverBase(),
	}

	// Initialize with children
	base.setChildren(orchestrations)

	return base
}

// Thread-safe child management implementation
func (bco *BaseContainerOrchestration) setChildren(orchestrations []types.Orchestration) {
	bco.childrenMutex.Lock()
	defer bco.childrenMutex.Unlock()

	// Create immutable copy
	children := make([]types.Orchestration, len(orchestrations))
	copy(children, orchestrations)

	// Update atomic values
	bco.children.Store(children)
	bco.childCount.Store(int64(len(children)))
	bco.isEmpty.Store(len(children) == 0)

	// Rebuild indices
	bco.rebuildIndices(children)
}

func (bco *BaseContainerOrchestration) rebuildIndices(children []types.Orchestration) {
	nameIndex := make(map[string][]int)
	typeIndex := make(map[string][]int)
	opIDIndex := make(map[string]int)

	for i, child := range children {
		// Name index
		if name := child.GetName(); name != "" {
			nameIndex[name] = append(nameIndex[name], i)
		}

		// Type index
		if childType := child.GetType(); childType != "" {
			typeIndex[childType] = append(typeIndex[childType], i)
		}

		// Operation ID index (unique identifiers)
		if opID := child.GetOperationID(); opID != "" {
			opIDIndex[opID] = i
		}
	}

	bco.childNameIndex.Store(nameIndex)
	bco.childTypeIndex.Store(typeIndex)
	bco.childOpIDIndex.Store(opIDIndex)
}

// Lock-free child access using immutable snapshots
func (bco *BaseContainerOrchestration) getChildrenSnapshot() []types.Orchestration {
	children := bco.children.Load()
	if children == nil {
		return []types.Orchestration{}
	}
	return children.([]types.Orchestration)
}

// Thread-safe index access with atomic operations
func (bco *BaseContainerOrchestration) getNameIndices(name string) []int {
	nameIndex := bco.childNameIndex.Load()
	if nameIndex == nil {
		return []int{}
	}

	indices, exists := nameIndex.(map[string][]int)[name]
	if !exists {
		return []int{}
	}

	// Return copy to prevent concurrent modification
	result := make([]int, len(indices))
	copy(result, indices)
	return result
}

// Thread-safe type index access
func (bco *BaseContainerOrchestration) getTypeIndices(orchestrationType string) []int {
	typeIndex := bco.childTypeIndex.Load()
	if typeIndex == nil {
		return []int{}
	}

	indices, exists := typeIndex.(map[string][]int)[orchestrationType]
	if !exists {
		return []int{}
	}

	// Return copy to prevent concurrent modification
	result := make([]int, len(indices))
	copy(result, indices)
	return result
}

// Thread-safe operation ID index access
func (bco *BaseContainerOrchestration) getOperationIDIndex(operationID string) (int, bool) {
	opIDIndex := bco.childOpIDIndex.Load()
	if opIDIndex == nil {
		return -1, false
	}

	index, exists := opIDIndex.(map[string]int)[operationID]
	return index, exists
}

// ContainerOrchestration interface implementation

// GetChildCount returns the total number of direct child orchestrations.
func (bco *BaseContainerOrchestration) GetChildCount() int {
	return int(bco.childCount.Load())
}

// IsEmpty returns true if the container has no child orchestrations.
func (bco *BaseContainerOrchestration) IsEmpty() bool {
	return bco.isEmpty.Load()
}

// GetChildAt returns the child orchestration at the specified index.
func (bco *BaseContainerOrchestration) GetChildAt(index int) (types.Orchestration, error) {
	children := bco.getChildrenSnapshot()
	if index < 0 || index >= len(children) {
		return nil, fmt.Errorf("%w: index %d out of bounds [0, %d)",
			types.ErrInvalidIndex, index, len(children))
	}
	return children[index], nil
}

// GetFirstChild returns the first child orchestration if any exists.
func (bco *BaseContainerOrchestration) GetFirstChild() (types.Orchestration, error) {
	return bco.GetChildAt(0)
}

// GetLastChild returns the last child orchestration if any exists.
func (bco *BaseContainerOrchestration) GetLastChild() (types.Orchestration, error) {
	count := bco.GetChildCount()
	if count == 0 {
		return nil, types.ErrEmptyContainer
	}
	return bco.GetChildAt(count - 1)
}

// GetChildByName returns the first child orchestration with the specified name.
func (bco *BaseContainerOrchestration) GetChildByName(name string) (types.Orchestration, error) {
	children := bco.GetChildrenByName(name)
	if len(children) == 0 {
		return nil, fmt.Errorf("%w: no child found with name '%s'",
			types.ErrChildNotFound, name)
	}
	return children[0], nil
}

// GetChildrenByName returns all child orchestrations with the specified name.
func (bco *BaseContainerOrchestration) GetChildrenByName(name string) []types.Orchestration {
	indices := bco.getNameIndices(name)
	if len(indices) == 0 {
		return []types.Orchestration{}
	}

	children := bco.getChildrenSnapshot()
	result := make([]types.Orchestration, len(indices))
	for i, idx := range indices {
		result[i] = children[idx]
	}
	return result
}

// FindChildrenByType returns all child orchestrations of the specified type.
func (bco *BaseContainerOrchestration) FindChildrenByType(orchestrationType string) []types.Orchestration {
	indices := bco.getTypeIndices(orchestrationType)
	if len(indices) == 0 {
		return []types.Orchestration{}
	}

	children := bco.getChildrenSnapshot()
	result := make([]types.Orchestration, len(indices))
	for i, idx := range indices {
		result[i] = children[idx]
	}
	return result
}

// FindChildByName searches for a child orchestration by name and returns its index and the orchestration.
// Returns (-1, nil) if no child with the specified name is found.
func (bco *BaseContainerOrchestration) FindChildByName(name string) (int, types.Orchestration) {
	children := bco.getChildrenSnapshot()
	for i, child := range children {
		if child.GetName() == name {
			return i, child
		}
	}
	return -1, nil
}

// ContainsChild checks if the specified orchestration is a direct child.
func (bco *BaseContainerOrchestration) ContainsChild(orchestration types.Orchestration) bool {
	children := bco.getChildrenSnapshot()
	for _, child := range children {
		if child == orchestration {
			return true
		}
	}
	return false
}

// ContainsChildWithName checks if any direct child has the specified name.
func (bco *BaseContainerOrchestration) ContainsChildWithName(name string) bool {
	indices := bco.getNameIndices(name)
	return len(indices) > 0
}

// GetChildren returns a copy of all direct child orchestrations.
func (bco *BaseContainerOrchestration) GetChildren() []types.Orchestration {
	children := bco.getChildrenSnapshot()
	// Return a copy to ensure immutability
	result := make([]types.Orchestration, len(children))
	copy(result, children)
	return result
}

// GetContainerType returns the specific container type.
func (bco *BaseContainerOrchestration) GetContainerType() string {
	return bco.containerType
}

// GetPathResolver returns the path resolver for this container.
func (bco *BaseContainerOrchestration) GetPathResolver() *PathResolverBase {
	return bco.pathResolver
}

// GetChildByOperationID returns the child orchestration with the specified operation ID.
func (bco *BaseContainerOrchestration) GetChildByOperationID(operationID string) (types.Orchestration, error) {
	// Use optimized index lookup for O(1) performance
	if index, exists := bco.getOperationIDIndex(operationID); exists {
		children := bco.getChildrenSnapshot()
		if index < len(children) {
			return children[index], nil
		}
	}

	return nil, fmt.Errorf("%w: no child found with operation ID '%s'",
		types.ErrChildNotFound, operationID)
}

// FindDescendantByName searches for the first orchestration with the specified name
// at any depth level in the orchestration tree (recursive search).
func (bco *BaseContainerOrchestration) FindDescendantByName(name string) (types.Orchestration, error) {
	// First check direct children
	if child, err := bco.GetChildByName(name); err == nil {
		return child, nil
	}

	// Recursively search in child containers
	children := bco.getChildrenSnapshot()
	for _, child := range children {
		// Check if child is also a container
		if container, ok := child.(types.ContainerOrchestration); ok {
			if descendant, err := container.FindDescendantByName(name); err == nil {
				return descendant, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: no descendant found with name '%s'",
		types.ErrChildNotFound, name)
}

// FindDescendantByOperationID searches for the orchestration with the specified operation ID
// at any depth level in the orchestration tree (recursive search).
func (bco *BaseContainerOrchestration) FindDescendantByOperationID(operationID string) (types.Orchestration, error) {
	// First check direct children
	if child, err := bco.GetChildByOperationID(operationID); err == nil {
		return child, nil
	}

	// Recursively search in child containers
	children := bco.getChildrenSnapshot()
	for _, child := range children {
		// Check if child is also a container
		if container, ok := child.(types.ContainerOrchestration); ok {
			if descendant, err := container.FindDescendantByOperationID(operationID); err == nil {
				return descendant, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: no descendant found with operation ID '%s'",
		types.ErrChildNotFound, operationID)
}

// FindAllDescendantsByType searches for all orchestrations with the specified type
// at any depth level in the orchestration tree (recursive search).
func (bco *BaseContainerOrchestration) FindAllDescendantsByType(orchestrationType string) []types.Orchestration {
	var result []types.Orchestration

	// Add direct children of the specified type
	directChildren := bco.FindChildrenByType(orchestrationType)
	result = append(result, directChildren...)

	// Recursively search in child containers
	children := bco.getChildrenSnapshot()
	for _, child := range children {
		// Check if child is also a container
		if container, ok := child.(types.ContainerOrchestration); ok {
			descendants := container.FindAllDescendantsByType(orchestrationType)
			result = append(result, descendants...)
		}
	}

	return result
}

// AddChild adds a new child orchestration to this container.
func (bco *BaseContainerOrchestration) AddChild(child types.Orchestration) error {
	if child == nil {
		return fmt.Errorf("%w: cannot add nil orchestration", types.ErrInvalidOrchestration)
	}

	bco.childrenMutex.Lock()
	defer bco.childrenMutex.Unlock()

	children := bco.getChildrenSnapshot()

	// Check for duplicate operation ID
	if opID := child.GetOperationID(); opID != "" {
		for _, existing := range children {
			if existing.GetOperationID() == opID {
				return fmt.Errorf("%w: orchestration with operation ID '%s' already exists",
					types.ErrDuplicateOrchestration, opID)
			}
		}
	}

	// Create new slice with added child
	newChildren := make([]types.Orchestration, len(children)+1)
	copy(newChildren, children)
	newChildren[len(children)] = child

	// Update atomic values
	bco.children.Store(newChildren)
	bco.childCount.Store(int64(len(newChildren)))
	bco.isEmpty.Store(false)

	// Rebuild indices
	bco.rebuildIndices(newChildren)

	return nil
}

// RemoveChild removes the specified child orchestration from this container.
func (bco *BaseContainerOrchestration) RemoveChild(child types.Orchestration) error {
	if child == nil {
		return fmt.Errorf("%w: cannot remove nil orchestration", types.ErrInvalidOrchestration)
	}

	bco.childrenMutex.Lock()
	defer bco.childrenMutex.Unlock()

	children := bco.getChildrenSnapshot()

	// Find the child to remove
	removeIndex := -1
	for i, existing := range children {
		if existing == child {
			removeIndex = i
			break
		}
	}

	if removeIndex == -1 {
		return fmt.Errorf("%w: orchestration not found in container", types.ErrChildNotFound)
	}

	// Create new slice without the removed child
	newChildren := make([]types.Orchestration, len(children)-1)
	copy(newChildren[:removeIndex], children[:removeIndex])
	copy(newChildren[removeIndex:], children[removeIndex+1:])

	// Update atomic values
	bco.children.Store(newChildren)
	bco.childCount.Store(int64(len(newChildren)))
	bco.isEmpty.Store(len(newChildren) == 0)

	// Rebuild indices
	bco.rebuildIndices(newChildren)

	return nil
}

// RemoveChildAt removes the child orchestration at the specified index.
func (bco *BaseContainerOrchestration) RemoveChildAt(index int) error {
	bco.childrenMutex.Lock()
	defer bco.childrenMutex.Unlock()

	children := bco.getChildrenSnapshot()
	if index < 0 || index >= len(children) {
		return fmt.Errorf("%w: index %d out of bounds [0, %d)",
			types.ErrInvalidIndex, index, len(children))
	}

	// Create new slice without the removed child
	newChildren := make([]types.Orchestration, len(children)-1)
	copy(newChildren[:index], children[:index])
	copy(newChildren[index:], children[index+1:])

	// Update atomic values
	bco.children.Store(newChildren)
	bco.childCount.Store(int64(len(newChildren)))
	bco.isEmpty.Store(len(newChildren) == 0)

	// Rebuild indices
	bco.rebuildIndices(newChildren)

	return nil
}

// ClearChildren removes all child orchestrations from this container.
func (bco *BaseContainerOrchestration) ClearChildren() {
	bco.childrenMutex.Lock()
	defer bco.childrenMutex.Unlock()

	// Reset to empty state
	emptyChildren := []types.Orchestration{}
	bco.children.Store(emptyChildren)
	bco.childCount.Store(0)
	bco.isEmpty.Store(true)

	// Clear indices
	bco.childNameIndex.Store(make(map[string][]int))
	bco.childTypeIndex.Store(make(map[string][]int))
	bco.childOpIDIndex.Store(make(map[string]int))
}

// ReplaceChild replaces an existing child with a new orchestration.
func (bco *BaseContainerOrchestration) ReplaceChild(oldChild, newChild types.Orchestration) error {
	if oldChild == nil || newChild == nil {
		return fmt.Errorf("%w: cannot replace with nil orchestration", types.ErrInvalidOrchestration)
	}

	bco.childrenMutex.Lock()
	defer bco.childrenMutex.Unlock()

	children := bco.getChildrenSnapshot()

	// Find the child to replace
	replaceIndex := -1
	for i, existing := range children {
		if existing == oldChild {
			replaceIndex = i
			break
		}
	}

	if replaceIndex == -1 {
		return fmt.Errorf("%w: orchestration not found in container", types.ErrChildNotFound)
	}

	// Check for operation ID conflicts with other children
	if newOpID := newChild.GetOperationID(); newOpID != "" {
		for i, existing := range children {
			if i != replaceIndex && existing.GetOperationID() == newOpID {
				return fmt.Errorf("%w: orchestration with operation ID '%s' already exists",
					types.ErrDuplicateOrchestration, newOpID)
			}
		}
	}

	// Create new slice with replaced child
	newChildren := make([]types.Orchestration, len(children))
	copy(newChildren, children)
	newChildren[replaceIndex] = newChild

	// Update atomic values
	bco.children.Store(newChildren)

	// Rebuild indices
	bco.rebuildIndices(newChildren)

	return nil
}

// GetChildIndex returns the index of the specified child orchestration.
func (bco *BaseContainerOrchestration) GetChildIndex(child types.Orchestration) (int, error) {
	if child == nil {
		return -1, fmt.Errorf("%w: cannot get index of nil orchestration", types.ErrInvalidOrchestration)
	}

	children := bco.getChildrenSnapshot()
	for i, existing := range children {
		if existing == child {
			return i, nil
		}
	}

	return -1, fmt.Errorf("%w: orchestration not found in container", types.ErrChildNotFound)
}

// ValidateChildren performs comprehensive validation of all child orchestrations.
// This includes checking for duplicate operation IDs, invalid references, and other
// container-specific validation rules.
func (bco *BaseContainerOrchestration) ValidateChildren() error {
	children := bco.getChildrenSnapshot()

	if len(children) == 0 {
		return nil // Empty container is valid
	}

	// Track operation IDs to detect duplicates
	operationIDs := make(map[string]int)

	for i, child := range children {
		if child == nil {
			return fmt.Errorf("%w: child at index %d is nil", types.ErrInvalidOrchestration, i)
		}

		// Check for duplicate operation IDs
		if opID := child.GetOperationID(); opID != "" {
			if existingIndex, exists := operationIDs[opID]; exists {
				return fmt.Errorf("%w: operation ID '%s' is used by children at indices %d and %d",
					types.ErrDuplicateOrchestration, opID, existingIndex, i)
			}
			operationIDs[opID] = i
		}

		// Validate child type
		if childType := child.GetType(); childType == "" {
			return fmt.Errorf("%w: child at index %d has empty type", types.ErrInvalidChildType, i)
		}
	}

	return nil
}

// GetChildrenSummary returns a detailed summary of all child orchestrations.
// This is useful for debugging, logging, and monitoring purposes.
func (bco *BaseContainerOrchestration) GetChildrenSummary() map[string]interface{} {
	children := bco.getChildrenSnapshot()

	summary := make(map[string]interface{})
	summary["count"] = len(children)
	summary["is_empty"] = len(children) == 0

	if len(children) == 0 {
		return summary
	}

	// Collect child information
	childrenInfo := make([]map[string]interface{}, len(children))
	typeCount := make(map[string]int)

	for i, child := range children {
		childInfo := map[string]interface{}{
			"index":        i,
			"name":         child.GetName(),
			"type":         child.GetType(),
			"operation_id": child.GetOperationID(),
		}
		childrenInfo[i] = childInfo

		// Count types
		childType := child.GetType()
		typeCount[childType]++
	}

	summary["children"] = childrenInfo
	summary["type_distribution"] = typeCount

	// Add first and last child info for quick reference
	if len(children) > 0 {
		summary["first_child"] = map[string]interface{}{
			"name":         children[0].GetName(),
			"type":         children[0].GetType(),
			"operation_id": children[0].GetOperationID(),
		}

		lastIndex := len(children) - 1
		summary["last_child"] = map[string]interface{}{
			"name":         children[lastIndex].GetName(),
			"type":         children[lastIndex].GetType(),
			"operation_id": children[lastIndex].GetOperationID(),
		}
	}

	return summary
}

// String provides a string representation of the container orchestration.
// This implements the fmt.Stringer interface for easy debugging and logging.
func (bco *BaseContainerOrchestration) String() string {
	children := bco.getChildrenSnapshot()
	return fmt.Sprintf("%s[%d children]", bco.GetContainerType(), len(children))
}

// Named sets a name for the orchestration for observability and debugging.
// Implements types.Orchestration interface.
func (bco *BaseContainerOrchestration) Named(name string) types.Orchestration {
	bco.BaseOrchestrationBuilder.SetName(name)
	return bco
}

// With applies configuration to the orchestration.
// Implements types.Orchestration interface.
func (bco *BaseContainerOrchestration) With(config config.Config) types.Orchestration {
	bco.BaseOrchestrationBuilder.SetConfig(config)
	return bco
}

// ErrorBoundary sets the error handling strategy for this orchestration.
// Implements types.Orchestration interface.
func (bco *BaseContainerOrchestration) ErrorBoundary(strategy errors.ErrorStrategy) types.Orchestration {
	bco.BaseOrchestrationBuilder.SetErrorBoundary(strategy)
	return bco
}

// Execute is a placeholder for concrete container implementations.
// Each container type must implement its own execution logic.
func (bco *BaseContainerOrchestration) Execute(ctx context.Context, config config.Config) (*result.Result, error) {
	panic(fmt.Sprintf("Execute method must be implemented by concrete container type: %s", bco.containerType))
}

// PathResolver interface methods - delegate to pathResolver

// GetByPath finds an orchestration by its hierarchical path
func (bco *BaseContainerOrchestration) GetByPath(path string) (types.Orchestration, error) {
	return bco.pathResolver.GetByPath(path)
}

// GetCurrentPath returns the current orchestration's full path
func (bco *BaseContainerOrchestration) GetCurrentPath() string {
	return bco.pathResolver.GetCurrentPath()
}

// ListAllPaths returns all available paths in the orchestration tree
func (bco *BaseContainerOrchestration) ListAllPaths() []string {
	return bco.pathResolver.ListAllPaths()
}

// FindByName searches for orchestrations by name (may return multiple matches)
func (bco *BaseContainerOrchestration) FindByName(name string) []types.PathMatch {
	return bco.pathResolver.FindByName(name)
}

// GetOrchestrationTree returns a tree representation of the orchestration hierarchy
func (bco *BaseContainerOrchestration) GetOrchestrationTree() *types.OrchestrationTree {
	return bco.pathResolver.GetOrchestrationTree()
}

// Query returns a PathQuery instance for advanced path-based queries
func (bco *BaseContainerOrchestration) Query() *types.PathQuery {
	return bco.pathResolver.Query()
}

// GetOperationID returns a unique operation ID for this orchestration instance.
// This delegates to the BaseOrchestrationBuilder but provides the interface-expected signature.
func (bco *BaseContainerOrchestration) GetOperationID() string {
	return bco.BaseOrchestrationBuilder.GetOperationID(bco)
}
