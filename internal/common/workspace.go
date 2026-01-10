package common

import (
	"fmt"
	"path/filepath"

	"github.com/yejune/git-workspace/internal/manifest"
)

// WorkspaceHandler is a function that processes a workspace entry
type WorkspaceHandler func(ws *manifest.WorkspaceEntry, fullPath string) error

// ForEachWorkspace iterates over all workspaces and applies the handler function
// Returns error immediately if handler returns error
func (ctx *WorkspaceContext) ForEachWorkspace(handler WorkspaceHandler) error {
	for i := range ctx.Manifest.Workspaces {
		ws := &ctx.Manifest.Workspaces[i]
		fullPath := filepath.Join(ctx.RepoRoot, ws.Path)

		if err := handler(ws, fullPath); err != nil {
			return err
		}
	}
	return nil
}

// ForEachWorkspaceWithContinue iterates over all workspaces and applies the handler function
// Continues execution even if handler returns error
func (ctx *WorkspaceContext) ForEachWorkspaceWithContinue(handler WorkspaceHandler) {
	for i := range ctx.Manifest.Workspaces {
		ws := &ctx.Manifest.Workspaces[i]
		fullPath := filepath.Join(ctx.RepoRoot, ws.Path)

		_ = handler(ws, fullPath)
	}
}

// FilterWorkspaces returns workspaces filtered by command-line arguments
// If no args provided, returns all workspaces
// If args provided, returns only the matching workspace by path
func (ctx *WorkspaceContext) FilterWorkspaces(args []string) ([]manifest.WorkspaceEntry, error) {
	if len(args) == 0 {
		return ctx.Manifest.Workspaces, nil
	}

	targetPath := args[0]
	for _, ws := range ctx.Manifest.Workspaces {
		if ws.Path == targetPath {
			return []manifest.WorkspaceEntry{ws}, nil
		}
	}

	return nil, fmt.Errorf("workspace not found: %s", targetPath)
}
