// Package orchestration provides path-based orchestration resolution for all orchestration types.
// This system allows finding any orchestration in the tree using hierarchical paths
// without maintaining centralized maps, keeping the system allocation-efficient.
package orchestration

import "github.com/maniartech/orchestrator/pkg/types"

// PathResolver is an alias to the types.PathResolver interface for backward compatibility.
// All path resolver implementations should use types.PathResolver directly.
type PathResolver = types.PathResolver

// PathMatch is an alias to the types.PathMatch struct for backward compatibility.
// All path match operations should use types.PathMatch directly.
type PathMatch = types.PathMatch

// OrchestrationTree is an alias to the types.OrchestrationTree struct for backward compatibility.
// All orchestration tree operations should use types.OrchestrationTree directly.
type OrchestrationTree = types.OrchestrationTree

// PathQuery is an alias to the types.PathQuery struct for backward compatibility.
// All path query operations should use types.PathQuery directly.
type PathQuery = types.PathQuery

// NewPathQuery creates a new path query instance.
// This is a convenience function that delegates to types.NewPathQuery.
func NewPathQuery(tree *types.OrchestrationTree) *types.PathQuery {
	return types.NewPathQuery(tree)
}
