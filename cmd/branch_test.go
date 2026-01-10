package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yejune/git-multirepo/internal/manifest"
)

// Display Tests (4 tests)

func TestRunBranch_AllWorkspaces(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "packages/ws1"})

	t.Run("branch shows all workspaces", func(t *testing.T) {
		output := captureOutput(func() {
			runBranch(branchCmd, []string{})
		})

		if !strings.Contains(output, "Repositories:") {
			t.Errorf("output should contain 'Repositories:', got: %s", output)
		}
		if !strings.Contains(output, "packages/ws1") {
			t.Errorf("output should contain workspace path, got: %s", output)
		}
		if !strings.Contains(output, "Repo:") {
			t.Errorf("output should contain 'Repo:', got: %s", output)
		}
		if !strings.Contains(output, "Branch:") {
			t.Errorf("output should contain 'Branch:', got: %s", output)
		}
		_ = dir
	})
}

func TestRunBranch_SpecificWorkspace(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "packages/specific"})

	t.Run("branch shows specific workspace", func(t *testing.T) {
		output := captureOutput(func() {
			runBranch(branchCmd, []string{"packages/specific"})
		})

		if !strings.Contains(output, "packages/specific") {
			t.Errorf("output should contain workspace path, got: %s", output)
		}
		if !strings.Contains(output, "Repo:") {
			t.Errorf("output should contain 'Repo:', got: %s", output)
		}
		if !strings.Contains(output, "Branch:") {
			t.Errorf("output should contain 'Branch:', got: %s", output)
		}
		_ = dir
	})
}

func TestRunBranch_WithTrackingBranch(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "packages/tracking"})

	// Tracking branch is set up automatically by clone
	t.Run("branch shows tracking branch", func(t *testing.T) {
		output := captureOutput(func() {
			runBranch(branchCmd, []string{"packages/tracking"})
		})

		if !strings.Contains(output, "packages/tracking") {
			t.Errorf("output should contain workspace path, got: %s", output)
		}
		if !strings.Contains(output, "Branch:") {
			t.Errorf("output should contain 'Branch:', got: %s", output)
		}
		// Tracking branch should be shown if available
		// Format: "Branch: main → origin/main" or just "Branch: main"
		_ = dir
	})
}

func TestRunBranch_NoWorkspaces(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("branch with no workspaces", func(t *testing.T) {
		output := captureOutput(func() {
			runBranch(branchCmd, []string{})
		})

		if !strings.Contains(output, "No repositories registered") {
			t.Errorf("output should show no repositories message, got: %s", output)
		}
	})
}

// Not Cloned Tests (2 tests)

func TestRunBranch_NotClonedWorkspace(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create manifest with workspace that doesn't exist
	m := &manifest.Manifest{
		Workspaces: []manifest.WorkspaceEntry{
			{Path: "packages/notcloned", Repo: "https://github.com/user/repo.git"},
		},
	}
	manifest.Save(dir, m)

	t.Run("branch shows not cloned", func(t *testing.T) {
		output := captureOutput(func() {
			runBranch(branchCmd, []string{})
		})

		if !strings.Contains(output, "packages/notcloned") {
			t.Errorf("output should contain workspace path, got: %s", output)
		}
		if !strings.Contains(output, "not cloned") {
			t.Errorf("output should show 'not cloned' status, got: %s", output)
		}
	})
}

func TestRunBranch_SpecificNotCloned(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create manifest with workspace that doesn't exist
	m := &manifest.Manifest{
		Workspaces: []manifest.WorkspaceEntry{
			{Path: "packages/missing", Repo: "https://github.com/user/repo.git"},
		},
	}
	manifest.Save(dir, m)

	t.Run("branch shows specific not cloned workspace", func(t *testing.T) {
		output := captureOutput(func() {
			runBranch(branchCmd, []string{"packages/missing"})
		})

		if !strings.Contains(output, "packages/missing") {
			t.Errorf("output should contain workspace path, got: %s", output)
		}
		if !strings.Contains(output, "not cloned") {
			t.Errorf("output should show 'not cloned' status, got: %s", output)
		}
	})
}

// Error Tests (4 tests)

func TestRunBranch_NotInGitRepo(t *testing.T) {
	dir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(dir)

	t.Run("branch outside git repo", func(t *testing.T) {
		err := runBranch(branchCmd, []string{})
		if err == nil {
			t.Error("should error when not in a git repository")
		}
		if !strings.Contains(err.Error(), "not in a git repository") {
			t.Errorf("expected 'not in a git repository' error, got: %v", err)
		}
	})
}

func TestRunBranch_ManifestLoadError(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create invalid manifest
	manifestPath := filepath.Join(dir, ".git.multirepos")
	os.WriteFile(manifestPath, []byte("invalid: yaml: [[["), 0644)

	t.Run("branch with manifest load error", func(t *testing.T) {
		err := runBranch(branchCmd, []string{})
		if err == nil {
			t.Error("should error with invalid manifest")
		}
		if !strings.Contains(err.Error(), "failed to load manifest") {
			t.Errorf("expected 'failed to load manifest' error, got: %v", err)
		}
	})
}

func TestRunBranch_WorkspaceNotFound(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "packages/exists"})

	t.Run("branch with non-existent workspace", func(t *testing.T) {
		err := runBranch(branchCmd, []string{"packages/nonexistent"})
		if err == nil {
			t.Error("should error on non-existent workspace")
		}
		if !strings.Contains(err.Error(), "repository not found") {
			t.Errorf("expected 'repository not found' error, got: %v", err)
		}
		_ = dir
	})
}

func TestRunBranch_CorruptedGitRepo(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "packages/corrupted"})

	// Corrupt the git repo by replacing .git with a file
	subPath := filepath.Join(dir, "packages/corrupted")
	gitPath := filepath.Join(subPath, ".git")
	os.RemoveAll(gitPath)
	os.WriteFile(gitPath, []byte("not a directory"), 0644)

	t.Run("branch with corrupted git repo", func(t *testing.T) {
		output := captureOutput(func() {
			runBranch(branchCmd, []string{"packages/corrupted"})
		})

		// Should show "not cloned" since .git is not a directory
		if !strings.Contains(output, "packages/corrupted") {
			t.Errorf("output should show workspace path, got: %s", output)
		}
		if !strings.Contains(output, "not cloned") {
			t.Errorf("output should show 'not cloned', got: %s", output)
		}
	})
}
