package common

import (
	"fmt"

	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/i18n"
	"github.com/yejune/git-workspace/internal/manifest"
)

// WorkspaceContext holds the repository root and manifest for workspace operations
type WorkspaceContext struct {
	RepoRoot string
	Manifest *manifest.Manifest
}

// LoadWorkspaceContext initializes workspace context by loading repository root and manifest
func LoadWorkspaceContext() (*WorkspaceContext, error) {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	i18n.SetLanguage(m.GetLanguage())

	return &WorkspaceContext{
		RepoRoot: repoRoot,
		Manifest: m,
	}, nil
}

// SaveManifest saves the current manifest to disk
func (ctx *WorkspaceContext) SaveManifest() error {
	return manifest.Save(ctx.RepoRoot, ctx.Manifest)
}
