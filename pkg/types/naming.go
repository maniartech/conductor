package types

import (
	"fmt"
	"strings"
	"sync"
)

// NamingContext provides context for hierarchical naming within orchestrations.
// It maintains the path from root to the current orchestration and handles
// name generation for unnamed orchestrations.
type NamingContext struct {
	mu                sync.RWMutex
	path              []string        // Path segments from root to current
	childCounts       map[string]int  // Count of children by type for auto-naming
	namedChildren     map[string]bool // Track which children have explicit names
	parent            *NamingContext  // Parent context (nil for root)
	orchestrationType string          // Type of the orchestration this context belongs to
}

// NewRootNamingContext creates a new root naming context.
func NewRootNamingContext(rootName string, orchestrationType string) *NamingContext {
	return &NamingContext{
		path:              []string{rootName},
		childCounts:       make(map[string]int),
		namedChildren:     make(map[string]bool),
		orchestrationType: orchestrationType,
	}
}

// NewChildNamingContext creates a child naming context.
func NewChildNamingContext(parent *NamingContext, name string, orchestrationType string) *NamingContext {
	parent.mu.RLock()
	parentPath := make([]string, len(parent.path))
	copy(parentPath, parent.path)
	parent.mu.RUnlock()

	return &NamingContext{
		path:              append(parentPath, name),
		childCounts:       make(map[string]int),
		namedChildren:     make(map[string]bool),
		parent:            parent,
		orchestrationType: orchestrationType,
	}
}

// GetPath returns the full hierarchical path as a dot-separated string.
func (nc *NamingContext) GetPath() string {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return strings.Join(nc.path, ".")
}

// GetName returns the name of this orchestration (last segment of path).
func (nc *NamingContext) GetName() string {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	if len(nc.path) == 0 {
		return ""
	}
	return nc.path[len(nc.path)-1]
}

// GetDepth returns the nesting depth (0 for root).
func (nc *NamingContext) GetDepth() int {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return len(nc.path) - 1
}

// GenerateChildName generates a name for a child orchestration.
// If the child has an explicit name, it uses that. Otherwise, it generates
// a name based on the orchestration type and index.
func (nc *NamingContext) GenerateChildName(childName string, orchestrationType string, index int) string {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// If child has an explicit name, use it
	if childName != "" {
		nc.namedChildren[childName] = true
		return childName
	}

	// Generate a name based on type and index (1-based for display)
	displayIndex := index + 1

	// Use different naming patterns based on type
	switch orchestrationType {
	case "task":
		return fmt.Sprintf("task-%d", displayIndex)
	case "sequential":
		return fmt.Sprintf("seq-%d", displayIndex)
	case "concurrent":
		return fmt.Sprintf("conc-%d", displayIndex)
	default:
		return fmt.Sprintf("step-%d", displayIndex)
	}
}

// IsNamed returns true if the given name was explicitly set (not auto-generated).
func (nc *NamingContext) IsNamed(name string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.namedChildren[name]
}

// GetParent returns the parent naming context.
func (nc *NamingContext) GetParent() *NamingContext {
	return nc.parent
}

// GetRoot returns the root naming context.
func (nc *NamingContext) GetRoot() *NamingContext {
	current := nc
	for current.parent != nil {
		current = current.parent
	}
	return current
}

// GetParentPath returns the parent's path, or empty string if this is root.
func (nc *NamingContext) GetParentPath() string {
	if nc.parent == nil {
		return ""
	}
	return nc.parent.GetPath()
}

// HierarchicalNamer provides hierarchical naming capabilities for orchestrations.
// It manages the naming context and provides methods for path generation and resolution.
type HierarchicalNamer struct {
	context           *NamingContext
	name              string
	orchestrationType string
	index             int
}

// NewHierarchicalNamer creates a new hierarchical namer.
func NewHierarchicalNamer(parentContext *NamingContext, name string, orchestrationType string, index int) *HierarchicalNamer {
	var context *NamingContext

	if parentContext == nil {
		// This is a root orchestration
		if name == "" {
			name = "root"
		}
		context = NewRootNamingContext(name, orchestrationType)
	} else {
		// This is a child orchestration
		actualName := parentContext.GenerateChildName(name, orchestrationType, index)
		context = NewChildNamingContext(parentContext, actualName, orchestrationType)
	}

	return &HierarchicalNamer{
		context:           context,
		name:              name,
		orchestrationType: orchestrationType,
		index:             index,
	}
}

// GetOperationID returns the full hierarchical path as an operation ID.
func (hn *HierarchicalNamer) GetOperationID() string {
	return hn.context.GetPath()
}

// GetName returns the name of this orchestration.
func (hn *HierarchicalNamer) GetName() string {
	return hn.context.GetName()
}

// GetContext returns the naming context.
func (hn *HierarchicalNamer) GetContext() *NamingContext {
	return hn.context
}

// GetDepth returns the nesting depth.
func (hn *HierarchicalNamer) GetDepth() int {
	return hn.context.GetDepth()
}

// IsRoot returns true if this is a root orchestration.
func (hn *HierarchicalNamer) IsRoot() bool {
	return hn.context.parent == nil
}

// GetParentPath returns the parent's path, or empty string for root.
func (hn *HierarchicalNamer) GetParentPath() string {
	if hn.context.parent == nil {
		return ""
	}
	return hn.context.parent.GetPath()
}

// CreateChildNamer creates a hierarchical namer for a child orchestration.
func (hn *HierarchicalNamer) CreateChildNamer(childName string, childType string, index int) *HierarchicalNamer {
	return NewHierarchicalNamer(hn.context, childName, childType, index)
}

// GetPathSegments returns the path as individual segments.
func (hn *HierarchicalNamer) GetPathSegments() []string {
	hn.context.mu.RLock()
	defer hn.context.mu.RUnlock()

	segments := make([]string, len(hn.context.path))
	copy(segments, hn.context.path)
	return segments
}

// IsNamed returns true if this orchestration has an explicit name.
func (hn *HierarchicalNamer) IsNamed() bool {
	if hn.context.parent == nil {
		return hn.name != ""
	}
	return hn.context.parent.IsNamed(hn.context.GetName())
}

// GetType returns the orchestration type.
func (hn *HierarchicalNamer) GetType() string {
	return hn.orchestrationType
}

// GetIndex returns the index within the parent.
func (hn *HierarchicalNamer) GetIndex() int {
	return hn.index
}
