package types

import (
	"fmt"
	"strings"
)

// PathResolver provides path-based orchestration lookup capabilities.
// Each orchestration maintains its own path context and can resolve paths dynamically.
// This interface can be implemented by orchestration types that support path resolution.
type PathResolver interface {
	// GetByPath finds an orchestration by its hierarchical path
	GetByPath(path string) (Orchestration, error)

	// GetCurrentPath returns the current orchestration's full path
	GetCurrentPath() string

	// ListAllPaths returns all available paths in the orchestration tree
	ListAllPaths() []string

	// FindByName searches for orchestrations by name (may return multiple matches)
	FindByName(name string) []PathMatch

	// GetOrchestrationTree returns a tree representation of the orchestration hierarchy
	GetOrchestrationTree() *OrchestrationTree

	// Query returns a PathQuery instance for advanced path-based queries
	Query() *PathQuery
}

// PathMatch represents a matched orchestration with its path information.
type PathMatch struct {
	Path          string        // Full hierarchical path
	Orchestration Orchestration // The matched orchestration
	Depth         int           // Nesting depth
	Parent        string        // Parent path
	Type          string        // Orchestration type (sequential, concurrent, task, etc.)
	Index         int           // Index within parent (if applicable)
}

// GetSegments returns the path segments.
func (pm *PathMatch) GetSegments() []string {
	if pm.Path == "" {
		return []string{}
	}
	return strings.Split(pm.Path, ".")
}

// GetName returns the name of the matched orchestration (last segment).
func (pm *PathMatch) GetName() string {
	segments := pm.GetSegments()
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

// IsRoot returns true if this is a root orchestration.
func (pm *PathMatch) IsRoot() bool {
	return pm.Depth == 0
}

// OrchestrationTree represents a tree structure of orchestrations.
type OrchestrationTree struct {
	Name          string               // Name of this orchestration
	Path          string               // Full path to this orchestration
	Type          string               // Type of orchestration
	Depth         int                  // Nesting depth
	Root          Orchestration        // Root orchestration (nil for root itself)
	Parent        *OrchestrationTree   // Parent tree node
	Children      []*OrchestrationTree // Child tree nodes
	Orchestration Orchestration        // The actual orchestration
}

// GetAllPaths returns all paths in the tree.
func (ot *OrchestrationTree) GetAllPaths() []string {
	var paths []string
	ot.collectPaths(&paths)
	return paths
}

// collectPaths recursively collects all paths in the tree.
func (ot *OrchestrationTree) collectPaths(paths *[]string) {
	*paths = append(*paths, ot.Path)
	for _, child := range ot.Children {
		child.collectPaths(paths)
	}
}

// FindByName finds all nodes with the specified name.
func (ot *OrchestrationTree) FindByName(name string) []PathMatch {
	var matches []PathMatch
	ot.findByName(name, &matches)
	return matches
}

// findByName recursively searches for nodes with the specified name.
func (ot *OrchestrationTree) findByName(name string, matches *[]PathMatch) {
	if ot.Name == name {
		*matches = append(*matches, PathMatch{
			Path:          ot.Path,
			Orchestration: ot.Orchestration,
			Depth:         ot.Depth,
			Parent:        ot.getParentPath(),
			Type:          ot.Type,
		})
	}
	for _, child := range ot.Children {
		child.findByName(name, matches)
	}
}

// FindByType finds all nodes of the specified type.
func (ot *OrchestrationTree) FindByType(orchestrationType string) []PathMatch {
	var matches []PathMatch
	ot.findByType(orchestrationType, &matches)
	return matches
}

// findByType recursively searches for nodes of the specified type.
func (ot *OrchestrationTree) findByType(orchestrationType string, matches *[]PathMatch) {
	if ot.Type == orchestrationType {
		*matches = append(*matches, PathMatch{
			Path:          ot.Path,
			Orchestration: ot.Orchestration,
			Depth:         ot.Depth,
			Parent:        ot.getParentPath(),
			Type:          ot.Type,
		})
	}
	for _, child := range ot.Children {
		child.findByType(orchestrationType, matches)
	}
}

// FindByDepth finds all nodes at the specified depth.
func (ot *OrchestrationTree) FindByDepth(depth int) []PathMatch {
	var matches []PathMatch
	ot.findByDepth(depth, &matches)
	return matches
}

// findByDepth recursively searches for nodes at the specified depth.
func (ot *OrchestrationTree) findByDepth(depth int, matches *[]PathMatch) {
	if ot.Depth == depth {
		*matches = append(*matches, PathMatch{
			Path:          ot.Path,
			Orchestration: ot.Orchestration,
			Depth:         ot.Depth,
			Parent:        ot.getParentPath(),
			Type:          ot.Type,
		})
	}
	for _, child := range ot.Children {
		child.findByDepth(depth, matches)
	}
}

// GetLeafNodes returns all leaf nodes (nodes with no children).
func (ot *OrchestrationTree) GetLeafNodes() []PathMatch {
	var matches []PathMatch
	ot.getLeafNodes(&matches)
	return matches
}

// getLeafNodes recursively collects all leaf nodes.
func (ot *OrchestrationTree) getLeafNodes(matches *[]PathMatch) {
	if len(ot.Children) == 0 {
		*matches = append(*matches, PathMatch{
			Path:          ot.Path,
			Orchestration: ot.Orchestration,
			Depth:         ot.Depth,
			Parent:        ot.getParentPath(),
			Type:          ot.Type,
		})
	}
	for _, child := range ot.Children {
		child.getLeafNodes(matches)
	}
}

// Print prints the tree structure (for debugging).
func (ot *OrchestrationTree) Print() {
	ot.printWithIndent("")
}

// printWithIndent prints the tree with indentation.
func (ot *OrchestrationTree) printWithIndent(indent string) {
	fmt.Printf("%s%s (%s) - %s\n", indent, ot.Name, ot.Type, ot.Path)
	for _, child := range ot.Children {
		child.printWithIndent(indent + "  ")
	}
}

// getParentPath returns the parent path.
func (ot *OrchestrationTree) getParentPath() string {
	if ot.Parent == nil {
		return ""
	}
	return ot.Parent.Path
}

// PathQuery provides advanced querying capabilities for orchestration paths.
type PathQuery struct {
	tree *OrchestrationTree
}

// NewPathQuery creates a new path query instance.
func NewPathQuery(tree *OrchestrationTree) *PathQuery {
	return &PathQuery{tree: tree}
}

// FindByPattern finds orchestrations matching a pattern.
// Supports wildcards (*) and recursive search (**).
//
// Parameters:
//   - pattern: Search pattern (e.g., "*.auth.*", "**validate**")
//
// Returns:
//   - []PathMatch: Matching orchestrations
//
// Example:
//
//	query := NewPathQuery(tree)
//	matches := query.FindByPattern("*.auth.*")
//	// Finds all orchestrations with "auth" in their path
func (pq *PathQuery) FindByPattern(pattern string) []PathMatch {
	var matches []PathMatch
	pq.findByPattern(pq.tree, pattern, &matches)
	return matches
}

// findByPattern recursively searches for nodes matching the pattern.
func (pq *PathQuery) findByPattern(node *OrchestrationTree, pattern string, matches *[]PathMatch) {
	if pq.matchesPattern(node.Path, pattern) {
		*matches = append(*matches, PathMatch{
			Path:          node.Path,
			Orchestration: node.Orchestration,
			Depth:         node.Depth,
			Parent:        node.getParentPath(),
			Type:          node.Type,
		})
	}
	for _, child := range node.Children {
		pq.findByPattern(child, pattern, matches)
	}
}

// FindByType finds orchestrations by their type.
// This method examines the orchestration interface to determine type.
//
// Parameters:
//   - orchestrationType: Type to search for (e.g., "sequential", "concurrent", "task")
//
// Returns:
//   - []PathMatch: Matching orchestrations
func (pq *PathQuery) FindByType(orchestrationType string) []PathMatch {
	return pq.tree.FindByType(orchestrationType)
}

// FindByDepth finds orchestrations at a specific nesting depth.
//
// Parameters:
//   - depth: Target nesting depth (0 for root)
//
// Returns:
//   - []PathMatch: Orchestrations at the specified depth
func (pq *PathQuery) FindByDepth(depth int) []PathMatch {
	return pq.tree.FindByDepth(depth)
}

// FindLeafNodes finds all leaf nodes (orchestrations with no children).
//
// Returns:
//   - []PathMatch: All leaf orchestrations
func (pq *PathQuery) FindLeafNodes() []PathMatch {
	return pq.tree.GetLeafNodes()
}

// matchesPattern checks if a path matches a wildcard pattern.
func (pq *PathQuery) matchesPattern(path, pattern string) bool {
	// Simple wildcard matching - can be enhanced with more sophisticated patterns
	if pattern == "*" || pattern == "**" {
		return true
	}

	// Convert pattern to regex-like matching
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "**", ".*")

	// Simple contains check for now - can be enhanced with proper regex
	if strings.Contains(pattern, ".*") {
		// Remove .* and check if remaining parts are in the path
		parts := strings.Split(pattern, ".*")
		for _, part := range parts {
			if part != "" && !strings.Contains(path, part) {
				return false
			}
		}
		return true
	}

	return path == pattern
}
