package git

// WorkspaceStatus holds git status information for a workspace
type WorkspaceStatus struct {
	ModifiedFiles    []string
	UntrackedFiles   []string
	StagedFiles      []string
	TotalUncommitted int
}

// GetWorkspaceStatus retrieves comprehensive git status for a workspace
// Temporarily disables skip-worktree for keepFiles during status check
func GetWorkspaceStatus(workspacePath string, keepFiles []string) (*WorkspaceStatus, error) {
	status := &WorkspaceStatus{}

	// Check modified files with skip-worktree transaction
	err := WithSkipWorktreeTransaction(workspacePath, keepFiles, func() error {
		var err error
		status.ModifiedFiles, err = GetModifiedFiles(workspacePath)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Check untracked files
	untracked, err := GetUntrackedFiles(workspacePath)
	if err != nil {
		return nil, err
	}
	status.UntrackedFiles = untracked

	// Check staged files
	staged, err := GetStagedFiles(workspacePath)
	if err != nil {
		return nil, err
	}
	status.StagedFiles = staged

	// Calculate total uncommitted changes
	status.TotalUncommitted = len(status.ModifiedFiles) + len(status.UntrackedFiles) + len(status.StagedFiles)

	return status, nil
}
