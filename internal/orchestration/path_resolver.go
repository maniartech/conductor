// Package orchestration provides path-based orchestration resolution for all orchestration types.
// This system allows finding any orchestration in the tree using hierarchical paths
// without maintaining centralized maps, keeping the system allocation-efficient.
package orchestration

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

// PathResolverBase provides a base implementation for path resolution.
// This can be embedded in orchestration types to provide common path resolution functionality.
type PathResolverBase struct {
	// These functions are provided by the embedding orchestration type
	getCurrentPath func() string
	getChildren    func() []Orchestration
	getChildName   func(child Orchestration, index int) string
}

// NewPathResolverBase creates a new base path resolver.
func NewPathResolverBase() *PathResolverBase {
	return &PathResolverBase{}
}

// SetCallbacks sets the callback functions for the path resolver.
// This is called by the embedding orchestration type to provide access to its methods.
func (prb *PathResolverBase) SetCallbacks(
	getCurrentPath func() string,
	getChildren func() []Orchestration,
	getChildName func(child Orchestration, index int) string,
) {
	prb.getCurrentPath = getCurrentPath
	prb.getChildren = getChildren
	prb.getChildName = getChildName
}

// ParsePath parses a hierarchical path into segments.
func (prb *PathResolverBase) ParsePath(path string) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	if path == "." || strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path format: %s", path)
	}

	// Check for invalid characters
	if strings.Contains(path, " ") {
		return nil, fmt.Errorf("invalid path format: spaces not allowed in %s", path)
	}
	if strings.Contains(path, "/") {
		return nil, fmt.Errorf("invalid path format: slashes not allowed in %s", path)
	}

	segments := strings.Split(path, ".")
	for _, segment := range segments {
		if segment == "" {
			return nil, fmt.Errorf("invalid path format: empty segment in %s", path)
		}
	}

	return segments, nil
}

// ValidatePath validates that a path has correct format.
func (prb *PathResolverBase) ValidatePath(path string) bool {
	_, err := prb.ParsePath(path)
	return err == nil
}

// NormalizePath normalizes a path to a standard format.
func (prb *PathResolverBase) NormalizePath(path string) string {
	// Convert to lowercase and replace underscores with hyphens
	normalized := strings.ToLower(path)
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return normalized
}

// GetCurrentPath returns the current orchestration's path using the callback.
func (prb *PathResolverBase) GetCurrentPath() string {
	if prb.getCurrentPath != nil {
		return prb.getCurrentPath()
	}
	return ""
}

// GetByPath finds an orchestration by its hierarchical path.
func (prb *PathResolverBase) GetByPath(path string) (Orchestration, error) {
	if prb.getCurrentPath == nil || prb.getChildren == nil {
		return nil, fmt.Errorf("path resolver not properly initialized")
	}

	currentPath := prb.getCurrentPath()

	// If this is the exact path, return self
	if path == currentPath {
		// We can't return self since we don't have a reference to the parent orchestration
		// This should be handled by the embedding type
		return nil, fmt.Errorf("cannot return self from base resolver")
	}

	// Check if the path is a child path
	if !strings.HasPrefix(path, currentPath+".") {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	// Get the next segment after current path
	remainingPath := strings.TrimPrefix(path, currentPath+".")
	segments := strings.Split(remainingPath, ".")
	targetChildName := segments[0]

	// Find the child with the target name
	children := prb.getChildren()
	for i, child := range children {
		childName := prb.getChildName(child, i)
		if childName == targetChildName {
			// If this is the final segment, return the child
			if len(segments) == 1 {
				return child, nil
			}

			// Otherwise, delegate to the child if it implements PathResolver
			if pathResolver, ok := child.(PathResolver); ok {
				return pathResolver.GetByPath(path)
			}

			return nil, fmt.Errorf("child does not support path resolution: %s", targetChildName)
		}
	}

	return nil, fmt.Errorf("path not found: %s", path)
}

// ListAllPaths returns all paths in the orchestration subtree.
func (prb *PathResolverBase) ListAllPaths() []string {
	if prb.getCurrentPath == nil || prb.getChildren == nil {
		return []string{}
	}

	var paths []string
	currentPath := prb.getCurrentPath()
	paths = append(paths, currentPath)

	// Add child paths
	children := prb.getChildren()
	for i, child := range children {
		childName := prb.getChildName(child, i)
		childPath := currentPath + "." + childName
		paths = append(paths, childPath)

		// If child implements PathResolver, get its paths recursively
		if pathResolver, ok := child.(PathResolver); ok {
			childPaths := pathResolver.ListAllPaths()
			paths = append(paths, childPaths...)
		}
	}

	return paths
}

// FindByName searches for orchestrations by name in the subtree.
func (prb *PathResolverBase) FindByName(name string) []PathMatch {
	if prb.getCurrentPath == nil || prb.getChildren == nil {
		return []PathMatch{}
	}

	var matches []PathMatch
	currentPath := prb.getCurrentPath()

	// Check children
	children := prb.getChildren()
	for i, child := range children {
		childName := prb.getChildName(child, i)
		childPath := currentPath + "." + childName

		if childName == name {
			matches = append(matches, PathMatch{
				Path:          childPath,
				Orchestration: child,
				Depth:         strings.Count(childPath, "."),
				Parent:        currentPath,
				Type:          "unknown", // Type detection would need to be implemented
			})
		}

		// If child implements PathResolver, search recursively
		if pathResolver, ok := child.(PathResolver); ok {
			childMatches := pathResolver.FindByName(name)
			matches = append(matches, childMatches...)
		}
	}

	return matches
}

// GetOrchestrationTree returns a tree representation of the orchestration hierarchy.
func (prb *PathResolverBase) GetOrchestrationTree() *OrchestrationTree {
	if prb.getCurrentPath == nil || prb.getChildren == nil {
		return nil
	}

	currentPath := prb.getCurrentPath()
	segments := strings.Split(currentPath, ".")
	name := segments[len(segments)-1]

	tree := &OrchestrationTree{
		Name:     name,
		Path:     currentPath,
		Type:     "unknown", // Type detection would need to be implemented
		Depth:    strings.Count(currentPath, "."),
		Children: make([]*OrchestrationTree, 0),
	}

	// Add children to tree
	children := prb.getChildren()
	for i, child := range children {
		childName := prb.getChildName(child, i)
		childPath := currentPath + "." + childName

		childTree := &OrchestrationTree{
			Name:          childName,
			Path:          childPath,
			Type:          "unknown",
			Depth:         strings.Count(childPath, "."),
			Parent:        tree,
			Orchestration: child,
		}

		// If child implements PathResolver, get its tree recursively
		if pathResolver, ok := child.(PathResolver); ok {
			childTree = pathResolver.GetOrchestrationTree()
			childTree.Parent = tree
		}

		tree.Children = append(tree.Children, childTree)
	}

	return tree
}

// Query returns a PathQuery instance for advanced path-based queries.
func (prb *PathResolverBase) Query() *PathQuery {
	tree := prb.GetOrchestrationTree()
	return NewPathQuery(tree)
}
