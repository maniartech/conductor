// Package types provides core interfaces and types for the orchestrator library.
// This file contains the ContainerOrchestration interface and supporting types
// for orchestrations that manage child orchestrations.
package types

import (
	"errors"
)

// ContainerOrchestration extends the base Orchestration interface to provide
// comprehensive child orchestration management capabilities.
// This interface is implemented by orchestrations that contain other orchestrations
// such as Sequential, Concurrent, and Conditional orchestrations.
//
// Thread Safety:
//
//	All methods are thread-safe and can be called concurrently from multiple goroutines.
//	Child access operations use atomic operations and immutable snapshots where appropriate.
//
// Performance:
//   - O(1) operations: GetChildCount, GetChildAt, IsEmpty, ContainsChild
//   - O(n) operations: GetChildByName, FindChildrenByType (where n = child count)
//   - O(k) operations: GetChildrenByName (where k = matching children count)
//   - O(n*d) operations: Deep search methods (where n = total orchestrations, d = depth)
//
// Error Handling:
//
//	All methods return meaningful errors with context information.
//	Invalid indices return ErrChildNotFound with position details.
//	Name-based lookups return empty slices (not errors) when no matches found.
type ContainerOrchestration interface {
	// Embed the base Orchestration interface
	Orchestration

	// === Child Count and Validation ===

	// GetChildCount returns the total number of direct child orchestrations.
	// This operation is thread-safe and has O(1) complexity.
	//
	// Returns:
	//   - int: Number of direct child orchestrations
	//
	// Example:
	//   count := sequential.GetChildCount() // Returns 3 for Sequential(task1, task2, task3)
	GetChildCount() int

	// IsEmpty returns true if the container has no child orchestrations.
	// This operation is thread-safe and has O(1) complexity.
	//
	// Returns:
	//   - bool: true if no children, false otherwise
	IsEmpty() bool

	// === Index-based Child Access ===

	// GetChildAt returns the child orchestration at the specified index.
	// Indices are zero-based and follow Go conventions.
	// This operation is thread-safe and has O(1) complexity.
	//
	// Parameters:
	//   - index: Zero-based index of the child orchestration
	//
	// Returns:
	//   - Orchestration: The child orchestration at the specified index
	//   - error: ErrChildNotFound if index is out of bounds
	//
	// Example:
	//   child, err := sequential.GetChildAt(0) // Gets first child
	//   if err != nil {
	//       // Handle index out of bounds
	//   }
	GetChildAt(index int) (Orchestration, error)

	// GetFirstChild returns the first child orchestration if any exists.
	// This is a convenience method equivalent to GetChildAt(0).
	// This operation is thread-safe and has O(1) complexity.
	//
	// Returns:
	//   - Orchestration: The first child orchestration
	//   - error: ErrChildNotFound if container is empty
	GetFirstChild() (Orchestration, error)

	// GetLastChild returns the last child orchestration if any exists.
	// This is a convenience method for accessing the final child.
	// This operation is thread-safe and has O(1) complexity.
	//
	// Returns:
	//   - Orchestration: The last child orchestration
	//   - error: ErrChildNotFound if container is empty
	GetLastChild() (Orchestration, error)

	// === Name-based Child Access ===

	// GetChildByName returns the first child orchestration with the specified name.
	// If multiple children have the same name, only the first one is returned.
	// Use GetChildrenByName to retrieve all matches.
	// This operation is thread-safe and has O(n) complexity.
	//
	// Parameters:
	//   - name: Name of the child orchestration to find
	//
	// Returns:
	//   - Orchestration: The first matching child orchestration
	//   - error: ErrChildNotFound if no child with the specified name exists
	//
	// Example:
	//   auth, err := sequential.GetChildByName("user-authentication")
	//   if err != nil {
	//       // Handle child not found
	//   }
	GetChildByName(name string) (Orchestration, error)

	// GetChildByOperationID returns the child orchestration with the specified operation ID.
	// Operation IDs are unique within the orchestration tree, providing O(1) lookup performance.
	// This operation is thread-safe and has O(1) complexity with index caching.
	//
	// Parameters:
	//   - operationID: Unique operation ID of the child orchestration to find
	//
	// Returns:
	//   - Orchestration: The matching child orchestration
	//   - error: ErrChildNotFound if no child with the specified operation ID exists
	//
	// Example:
	//   child, err := sequential.GetChildByOperationID("task-user-authentication-001")
	//   if err != nil {
	//       // Handle child not found
	//   }
	GetChildByOperationID(operationID string) (Orchestration, error)

	// GetChildrenByName returns all child orchestrations with the specified name.
	// Returns empty slice if no children match the name.
	// This operation is thread-safe and has O(n) complexity.
	//
	// Parameters:
	//   - name: Name of the child orchestrations to find
	//
	// Returns:
	//   - []Orchestration: All matching child orchestrations (empty if none found)
	//
	// Example:
	//   validators := sequential.GetChildrenByName("data-validator")
	//   for _, validator := range validators {
	//       // Process each validator
	//   }
	GetChildrenByName(name string) []Orchestration

	// === Deep Nested Search ===

	// FindDescendantByName searches for the first orchestration with the specified name
	// at any depth level in the orchestration tree (recursive search).
	// This performs a depth-first search through all nested containers.
	// This operation is thread-safe and has O(n*d) complexity where n is total orchestrations and d is depth.
	//
	// Parameters:
	//   - name: Name of the orchestration to find in the entire tree
	//
	// Returns:
	//   - Orchestration: The first matching orchestration found at any depth
	//   - error: ErrChildNotFound if no orchestration with the specified name exists in the tree
	//
	// Example:
	//   deepTask, err := mainWorkflow.FindDescendantByName("user-authentication")
	//   if err != nil {
	//       // Handle orchestration not found anywhere in the tree
	//   }
	FindDescendantByName(name string) (Orchestration, error)

	// FindDescendantByOperationID searches for the orchestration with the specified operation ID
	// at any depth level in the orchestration tree (recursive search).
	// This performs a depth-first search through all nested containers.
	// This operation is thread-safe and has O(n*d) complexity where n is total orchestrations and d is depth.
	//
	// Parameters:
	//   - operationID: Unique operation ID of the orchestration to find in the entire tree
	//
	// Returns:
	//   - Orchestration: The matching orchestration found at any depth
	//   - error: ErrChildNotFound if no orchestration with the specified operation ID exists in the tree
	//
	// Example:
	//   deepTask, err := mainWorkflow.FindDescendantByOperationID("task-user-auth-validate-001")
	//   if err != nil {
	//       // Handle orchestration not found anywhere in the tree
	//   }
	FindDescendantByOperationID(operationID string) (Orchestration, error)

	// FindAllDescendantsByType searches for all orchestrations with the specified type
	// at any depth level in the orchestration tree (recursive search).
	// This performs a depth-first search through all nested containers.
	// This operation is thread-safe and has O(n*d) complexity where n is total orchestrations and d is depth.
	//
	// Parameters:
	//   - orchestrationType: Type of orchestrations to find in the entire tree (e.g., "task", "sequential")
	//
	// Returns:
	//   - []Orchestration: All matching orchestrations found at any depth (empty if none found)
	//
	// Example:
	//   allTasks := mainWorkflow.FindAllDescendantsByType("task")
	//   allSequentials := mainWorkflow.FindAllDescendantsByType("sequential")
	FindAllDescendantsByType(orchestrationType string) []Orchestration

	// === Type-based Child Access ===

	// FindChildrenByType returns all child orchestrations of the specified type.
	// Type matching is based on the orchestration's GetType() method.
	// Returns empty slice if no children match the type.
	// This operation is thread-safe and has O(n) complexity.
	//
	// Parameters:
	//   - orchestrationType: Type of orchestrations to find (e.g., "task", "sequential")
	//
	// Returns:
	//   - []Orchestration: All matching child orchestrations (empty if none found)
	//
	// Example:
	//   tasks := sequential.FindChildrenByType("task")
	//   concurrentBlocks := sequential.FindChildrenByType("concurrent")
	FindChildrenByType(orchestrationType string) []Orchestration

	// === Child Validation and Queries ===

	// ContainsChild checks if the specified orchestration is a direct child.
	// Comparison is based on orchestration identity (pointer equality).
	// This operation is thread-safe and has O(n) complexity.
	//
	// Parameters:
	//   - orchestration: The orchestration to search for
	//
	// Returns:
	//   - bool: true if the orchestration is a direct child, false otherwise
	//
	// Example:
	//   if sequential.ContainsChild(task1) {
	//       // task1 is a direct child of sequential
	//   }
	ContainsChild(orchestration Orchestration) bool

	// ContainsChildWithName checks if any direct child has the specified name.
	// This operation is thread-safe and has O(n) complexity.
	//
	// Parameters:
	//   - name: Name to search for among child orchestrations
	//
	// Returns:
	//   - bool: true if any child has the specified name, false otherwise
	//
	// Example:
	//   if sequential.ContainsChildWithName("authentication") {
	//       // At least one child is named "authentication"
	//   }
	ContainsChildWithName(name string) bool

	// === Child Collection Access ===

	// GetChildren returns a copy of all direct child orchestrations.
	// The returned slice is a snapshot and safe for concurrent access.
	// Modifications to the returned slice do not affect the container.
	// This operation is thread-safe and has O(n) complexity.
	//
	// Returns:
	//   - []Orchestration: Copy of all direct child orchestrations
	//
	// Example:
	//   children := sequential.GetChildren()
	//   for i, child := range children {
	//       fmt.Printf("Child %d: %s (%s)\n", i, child.GetName(), child.GetType())
	//   }
	GetChildren() []Orchestration
}

// Common error types for container operations
var (
	ErrChildNotFound          = errors.New("child orchestration not found")
	ErrInvalidIndex           = errors.New("child index out of bounds")
	ErrEmptyContainer         = errors.New("container has no children")
	ErrInvalidChildType       = errors.New("invalid child orchestration type")
	ErrInvalidOrchestration   = errors.New("invalid orchestration")
	ErrDuplicateOrchestration = errors.New("duplicate orchestration")
)
