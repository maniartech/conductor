package orchestration

import "github.com/maniartech/orchestrator/types"

// NamingContext is an alias to the types.NamingContext struct for backward compatibility.
// All naming context operations should use types.NamingContext directly.
type NamingContext = types.NamingContext

// HierarchicalNamer is an alias to the types.HierarchicalNamer struct for backward compatibility.
// All hierarchical namer operations should use types.HierarchicalNamer directly.
type HierarchicalNamer = types.HierarchicalNamer

// NewRootNamingContext creates a new root naming context.
// This is a convenience function that delegates to types.NewRootNamingContext.
func NewRootNamingContext(rootName string, orchestrationType string) *types.NamingContext {
	return types.NewRootNamingContext(rootName, orchestrationType)
}

// NewChildNamingContext creates a child naming context.
// This is a convenience function that delegates to types.NewChildNamingContext.
func NewChildNamingContext(parent *types.NamingContext, name string, orchestrationType string) *types.NamingContext {
	return types.NewChildNamingContext(parent, name, orchestrationType)
}

// NewHierarchicalNamer creates a new hierarchical namer.
// This is a convenience function that delegates to types.NewHierarchicalNamer.
func NewHierarchicalNamer(parentContext *types.NamingContext, name string, orchestrationType string, index int) *types.HierarchicalNamer {
	return types.NewHierarchicalNamer(parentContext, name, orchestrationType, index)
}
