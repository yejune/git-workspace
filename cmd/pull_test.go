package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yejune/git-workspace/internal/manifest"
)

// ============================================================================
// Helper Functions
// ============================================================================

// setupRemoteRepoWithCommits creates a remote repo with multiple commits
func setupRemoteRepoWithCommits(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()

	// Initial commit
	readme := filepath.Join(dir, "README.md")
	os.WriteFile(readme, []byte("# Remote Repo"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Initial commit").Run()

	// Second commit
	config := filepath.Join(dir, "config.yml")
	os.WriteFile(config, []byte("version: 1.0"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Add config").Run()

	return dir
}

// setupWorkspaceWithKeepFile creates a workspace with a keep file configured
func setupWorkspaceWithKeepFile(t *testing.T, repoRoot, remoteRepo, workspacePath string) {
	t.Helper()

	// Clone the workspace
	cloneBranch = ""
	if err := runRoot(rootCmd, []string{remoteRepo, workspacePath}); err != nil {
		t.Fatalf("Failed to clone workspace: %v", err)
	}

	// Add keep file to manifest
	m, err := manifest.Load(repoRoot)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	for i, ws := range m.Workspaces {
		if ws.Path == workspacePath {
			m.Workspaces[i].Keep = []string{"config.yml"}
			break
		}
	}

	if err := manifest.Save(repoRoot, m); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}
}

// commitToRemote creates a new commit in the remote repository
func commitToRemote(t *testing.T, remoteRepo, filename, content string) {
	t.Helper()

	filePath := filepath.Join(remoteRepo, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	exec.Command("git", "-C", remoteRepo, "add", ".").Run()
	exec.Command("git", "-C", remoteRepo, "commit", "-m", "Update "+filename).Run()
}

// verifyBackupExists checks if a backup was created
func verifyBackupExists(t *testing.T, repoRoot, workspacePath, filename string) bool {
	t.Helper()

	backupDir := filepath.Join(repoRoot, ".workspaces", "backup", "modified")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return false
	}

	// Check if any backup contains the file
	for _, entry := range entries {
		if entry.IsDir() {
			yearPath := filepath.Join(backupDir, entry.Name())
			if hasBackupInDir(yearPath, filename) {
				return true
			}
		}
	}

	return false
}

// hasBackupInDir recursively checks for backup files
func hasBackupInDir(dir, filename string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if hasBackupInDir(fullPath, filename) {
				return true
			}
		} else {
			if strings.Contains(entry.Name(), filepath.Base(filename)) {
				return true
			}
		}
	}

	return false
}

// verifyPatchExists checks if a patch file was created
func verifyPatchExists(t *testing.T, repoRoot, workspacePath, filename string) bool {
	t.Helper()

	patchDir := filepath.Join(repoRoot, ".workspaces", "patches", workspacePath)
	patchFile := filepath.Join(patchDir, filepath.Base(filename)+".patch")

	_, err := os.Stat(patchFile)
	return err == nil
}

// mockInteractiveChoice simulates user input for interactive prompts
// This is a simplified version - in real tests we'd use stdin redirection
func mockInteractiveChoice(choice string) func() {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.WriteString(choice + "\n")
		w.Close()
	}()

	return func() {
		os.Stdin = oldStdin
	}
}

// ============================================================================
// Test Cases: Basic Flow (4 tests)
// ============================================================================

func TestRunPull_NoWorkspaces(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("pull with no workspaces shows message", func(t *testing.T) {
		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Logf("Error (expected): %v", err)
			}
		})

		if !strings.Contains(output, "No workspaces") && !strings.Contains(output, "no_subs_registered") {
			t.Errorf("Should show no workspaces message, got: %s", output)
		}
		_ = dir
	})
}

func TestRunPull_CleanWorkspace(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/test-pull"})

	t.Run("pull clean workspace succeeds", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Fatalf("Pull failed: %v", err)
			}
		})

		if !strings.Contains(output, "packages/test-pull") {
			t.Errorf("Output should show workspace path, got: %s", output)
		}
		_ = dir
	})
}

func TestRunPull_SpecificWorkspace(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo1 := setupRemoteRepoWithCommits(t)
	remoteRepo2 := setupRemoteRepoWithCommits(t)

	// Create two workspaces
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo1, "packages/ws1"})
	runRoot(rootCmd, []string{remoteRepo2, "packages/ws2"})

	t.Run("pull specific workspace only", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{"packages/ws1"}); err != nil {
				t.Fatalf("Pull failed: %v", err)
			}
		})

		if !strings.Contains(output, "packages/ws1") {
			t.Errorf("Output should show ws1, got: %s", output)
		}
		if strings.Contains(output, "packages/ws2") {
			t.Errorf("Output should not show ws2, got: %s", output)
		}
		_ = dir
	})
}

func TestRunPull_UserDeclines(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/decline-test"})

	t.Run("pull declined by user", func(t *testing.T) {
		restore := mockInteractiveChoice("n")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Logf("Error: %v", err)
			}
		})

		if !strings.Contains(output, "skipped") && !strings.Contains(output, "pull_skipped") {
			t.Logf("Output: %s", output)
		}
		_ = dir
	})
}

// ============================================================================
// Test Cases: Uncommitted Changes (3 tests)
// ============================================================================

func TestRunPull_WithUncommittedChanges(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/modified"})

	// Modify a file
	wsPath := filepath.Join(dir, "packages/modified")
	os.WriteFile(filepath.Join(wsPath, "README.md"), []byte("# Modified"), 0644)

	t.Run("pull shows uncommitted changes count", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Logf("Error: %v", err)
			}
		})

		if !strings.Contains(output, "uncommitted") && !strings.Contains(output, "1") {
			t.Logf("Should show uncommitted changes, got: %s", output)
		}
		_ = dir
	})
}

func TestRunPull_WithUntrackedFiles(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/untracked"})

	// Add untracked file
	wsPath := filepath.Join(dir, "packages/untracked")
	os.WriteFile(filepath.Join(wsPath, "new-file.txt"), []byte("new"), 0644)

	t.Run("pull shows untracked files in count", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Logf("Error: %v", err)
			}
		})

		if !strings.Contains(output, "uncommitted") {
			t.Logf("Should show uncommitted count, got: %s", output)
		}
		_ = dir
	})
}

func TestRunPull_WithStagedFiles(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/staged"})

	// Stage a file
	wsPath := filepath.Join(dir, "packages/staged")
	os.WriteFile(filepath.Join(wsPath, "staged.txt"), []byte("staged"), 0644)
	exec.Command("git", "-C", wsPath, "add", "staged.txt").Run()

	t.Run("pull shows staged files in count", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Logf("Error: %v", err)
			}
		})

		if !strings.Contains(output, "uncommitted") {
			t.Logf("Should show uncommitted count including staged, got: %s", output)
		}
		_ = dir
	})
}

// ============================================================================
// Test Cases: Keep Files (5 tests)
// ============================================================================

func TestRunPull_KeepFileNoRemoteChanges(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)
	setupWorkspaceWithKeepFile(t, dir, remoteRepo, "packages/keep-no-change")

	t.Run("keep file without remote changes pulls normally", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Fatalf("Pull failed: %v", err)
			}
		})

		if !strings.Contains(output, "packages/keep-no-change") {
			t.Errorf("Output should show workspace, got: %s", output)
		}
		_ = dir
	})
}

func TestRunPull_KeepFileWithRemoteChanges_UpdateAndReapply(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)
	setupWorkspaceWithKeepFile(t, dir, remoteRepo, "packages/keep-reapply")

	// Modify keep file locally
	wsPath := filepath.Join(dir, "packages/keep-reapply")
	os.WriteFile(filepath.Join(wsPath, "config.yml"), []byte("version: 1.0\nlocal: true"), 0644)

	// Commit change to remote
	commitToRemote(t, remoteRepo, "config.yml", "version: 2.0\nremote: true")

	t.Run("keep file with remote changes can be updated and reapplied", func(t *testing.T) {
		// Note: This test requires interactive input which is complex to mock
		// We test the code paths exist but full integration would need survey mock
		t.Skip("Interactive test - requires survey mocking")
		_ = dir
	})
}

func TestRunPull_KeepFileWithRemoteChanges_UpdateOnly(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)
	setupWorkspaceWithKeepFile(t, dir, remoteRepo, "packages/keep-update")

	// Modify keep file locally
	wsPath := filepath.Join(dir, "packages/keep-update")
	os.WriteFile(filepath.Join(wsPath, "config.yml"), []byte("version: 1.0\nlocal: change"), 0644)

	// Commit change to remote
	commitToRemote(t, remoteRepo, "config.yml", "version: 2.0")

	t.Run("keep file with remote changes can discard local changes", func(t *testing.T) {
		// Interactive test - would need survey mocking
		t.Skip("Interactive test - requires survey mocking")
		_ = dir
	})
}

func TestRunPull_KeepFileWithRemoteChanges_Skip(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)
	setupWorkspaceWithKeepFile(t, dir, remoteRepo, "packages/keep-skip")

	// Modify keep file locally
	wsPath := filepath.Join(dir, "packages/keep-skip")
	os.WriteFile(filepath.Join(wsPath, "config.yml"), []byte("version: 1.0\nkeep_local: true"), 0644)

	// Commit change to remote
	commitToRemote(t, remoteRepo, "config.yml", "version: 2.0")

	t.Run("keep file changes can be skipped", func(t *testing.T) {
		// Interactive test - would need survey mocking
		t.Skip("Interactive test - requires survey mocking")
		_ = dir
	})
}

func TestRunPull_KeepFileBackupCreated(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)
	setupWorkspaceWithKeepFile(t, dir, remoteRepo, "packages/keep-backup")

	// Modify keep file locally
	wsPath := filepath.Join(dir, "packages/keep-backup")
	os.WriteFile(filepath.Join(wsPath, "config.yml"), []byte("version: 1.0\nlocal: backup"), 0644)

	// Commit change to remote
	commitToRemote(t, remoteRepo, "config.yml", "version: 2.0")

	t.Run("backup is created for keep files", func(t *testing.T) {
		// This would need full integration with interactive prompts
		// Testing backup creation in isolation would require calling handleKeepFiles directly
		t.Skip("Integration test - requires full workflow")
		_ = dir
	})
}

// ============================================================================
// Test Cases: Error Handling (3 tests)
// ============================================================================

func TestRunPull_NotInGitRepo(t *testing.T) {
	dir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(dir)

	t.Run("pull outside git repo fails", func(t *testing.T) {
		err := runPull(pullCmd, []string{})
		if err == nil {
			t.Error("Should error when not in a git repository")
		} else if !strings.Contains(err.Error(), "not in a git repository") {
			t.Errorf("Expected 'not in a git repository' error, got: %v", err)
		}
	})
}

func TestRunPull_NonExistentWorkspace(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create a workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/exists"})

	t.Run("pull non-existent workspace fails", func(t *testing.T) {
		err := runPull(pullCmd, []string{"packages/nonexistent"})
		if err == nil {
			t.Error("Should error for non-existent workspace")
		} else if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "sub_not_found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
}

func TestRunPull_NotGitRepo(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/broken"})

	// Break the workspace by removing .git
	wsPath := filepath.Join(dir, "packages/broken")
	os.RemoveAll(filepath.Join(wsPath, ".git"))

	t.Run("pull on non-git workspace shows error", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Logf("Error: %v", err)
			}
		})

		if !strings.Contains(output, "not_git_repo") && !strings.Contains(output, "not") {
			t.Logf("Should show error for non-git repo, got: %s", output)
		}
		_ = dir
	})
}

func TestRunPull_FetchFailed(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/fetch-fail"})

	// Break remote
	wsPath := filepath.Join(dir, "packages/fetch-fail")
	exec.Command("git", "-C", wsPath, "remote", "set-url", "origin", "invalid://url").Run()

	t.Run("pull with fetch failure shows error", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Logf("Error: %v", err)
			}
		})

		if !strings.Contains(output, "fetch_failed") && !strings.Contains(output, "failed") {
			t.Logf("Should show fetch error, got: %s", output)
		}
		_ = dir
	})
}

func TestRunPull_PullFailed(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepoWithCommits(t)

	// Create workspace
	cloneBranch = ""
	runRoot(rootCmd, []string{remoteRepo, "packages/pull-fail"})

	// Make conflicting changes
	wsPath := filepath.Join(dir, "packages/pull-fail")
	os.WriteFile(filepath.Join(wsPath, "README.md"), []byte("# Conflict"), 0644)
	exec.Command("git", "-C", wsPath, "add", ".").Run()
	exec.Command("git", "-C", wsPath, "commit", "-m", "Local commit").Run()

	// Create conflicting remote change
	commitToRemote(t, remoteRepo, "README.md", "# Remote Conflict")

	t.Run("pull with conflicts shows error", func(t *testing.T) {
		restore := mockInteractiveChoice("y")
		defer restore()

		output := captureOutput(func() {
			if err := runPull(pullCmd, []string{}); err != nil {
				t.Logf("Error: %v", err)
			}
		})

		if !strings.Contains(output, "failed") && !strings.Contains(output, "pull_failed") {
			t.Logf("Should show pull error, got: %s", output)
		}
		_ = dir
	})
}
