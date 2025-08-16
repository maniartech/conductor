package orchestration

import (
	"fmt"
	"strings"

	"github.com/maniartech/orchestrator/pkg/types"
)

// PathResolverBase provides a base implementation for path resolution.
// This can be embedded in orchestration types to provide common path resolution functionality.
type PathResolverBase struct {
	// These functions are provided by the embedding orchestration type
	getCurrentPath func() string
	getChildren    func() []types.Orchestration
	getChildName   func(child types.Orchestration, index int) string
}

// NewPathResolverBase creates a new base path resolver.
func NewPathResolverBase() *PathResolverBase {
	return &PathResolverBase{}
}

// SetCallbacks sets the callback functions for the path resolver.
// This is called by the embedding orchestration type to provide access to its methods.
func (prb *PathResolverBase) SetCallbacks(
	getCurrentPath func() string,
	getChildren func() []types.Orchestration,
	getChildName func(child types.Orchestration, index int) string,
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
func (prb *PathResolverBase) GetByPath(path string) (types.Orchestration, error) {
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
			if pathResolver, ok := child.(types.PathResolver); ok {
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
		if pathResolver, ok := child.(types.PathResolver); ok {
			childPaths := pathResolver.ListAllPaths()
			paths = append(paths, childPaths...)
		}
	}

	return paths
}

// FindByName searches for orchestrations by name in the subtree.
func (prb *PathResolverBase) FindByName(name string) []types.PathMatch {
	if prb.getCurrentPath == nil || prb.getChildren == nil {
		return []types.PathMatch{}
	}

	var matches []types.PathMatch
	currentPath := prb.getCurrentPath()

	// Check children
	children := prb.getChildren()
	for i, child := range children {
		childName := prb.getChildName(child, i)
		childPath := currentPath + "." + childName

		if childName == name {
			matches = append(matches, types.PathMatch{
				Path:          childPath,
				Orchestration: child,
				Depth:         strings.Count(childPath, "."),
				Parent:        currentPath,
				Type:          "unknown", // Type detection would need to be implemented
			})
		}

		// If child implements PathResolver, search recursively
		if pathResolver, ok := child.(types.PathResolver); ok {
			childMatches := pathResolver.FindByName(name)
			matches = append(matches, childMatches...)
		}
	}

	return matches
}

// GetOrchestrationTree returns a tree representation of the orchestration hierarchy.
func (prb *PathResolverBase) GetOrchestrationTree() *types.OrchestrationTree {
	if prb.getCurrentPath == nil || prb.getChildren == nil {
		return nil
	}

	currentPath := prb.getCurrentPath()
	segments := strings.Split(currentPath, ".")
	name := segments[len(segments)-1]

	tree := &types.OrchestrationTree{
		Name:     name,
		Path:     currentPath,
		Type:     "unknown", // Type detection would need to be implemented
		Depth:    strings.Count(currentPath, "."),
		Children: make([]*types.OrchestrationTree, 0),
	}

	// Add children to tree
	children := prb.getChildren()
	for i, child := range children {
		childName := prb.getChildName(child, i)
		childPath := currentPath + "." + childName

		childTree := &types.OrchestrationTree{
			Name:          childName,
			Path:          childPath,
			Type:          "unknown",
			Depth:         strings.Count(childPath, "."),
			Parent:        tree,
			Orchestration: child,
		}

		// If child implements PathResolver, get its tree recursively
		if pathResolver, ok := child.(types.PathResolver); ok {
			childTree = pathResolver.GetOrchestrationTree()
			childTree.Parent = tree
		}

		tree.Children = append(tree.Children, childTree)
	}

	return tree
}

// Query returns a PathQuery instance for advanced path-based queries.
func (prb *PathResolverBase) Query() *types.PathQuery {
	tree := prb.GetOrchestrationTree()
	return types.NewPathQuery(tree)
}
